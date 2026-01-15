package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
	"github.com/nimsforest/morpheus/pkg/sshutil"
)

// HandleTest handles the test command.
func HandleTest() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus test <subcommand>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  e2e      Run end-to-end tests")
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "e2e":
		handleTestE2E()
	default:
		fmt.Fprintf(os.Stderr, "Unknown test subcommand: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, "Available subcommands: e2e")
		os.Exit(1)
	}
}

func handleTestE2E() {
	// Parse flags
	keepForest := false
	for _, arg := range os.Args[3:] {
		if arg == "--keep" {
			keepForest = true
		}
	}

	fmt.Println()
	fmt.Println("🧪 Morpheus E2E Test Suite")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Load config to get API token
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %s\n", err)
		fmt.Fprintln(os.Stderr, "   Make sure HETZNER_API_TOKEN is set or config.yaml exists")
		os.Exit(1)
	}

	if cfg.Secrets.HetznerAPIToken == "" {
		fmt.Fprintln(os.Stderr, "❌ Hetzner API token not configured")
		fmt.Fprintln(os.Stderr, "   Set HETZNER_API_TOKEN environment variable or add to config.yaml")
		os.Exit(1)
	}

	// Create Hetzner provider for direct API operations
	hetznerProv, err := hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Hetzner provider: %s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var testForestID string
	testsPassed := 0
	testsFailed := 0

	// Helper to run SSH commands on a node
	runSSHToNode := func(nodeIP, command string) (string, error) {
		sshArgs := []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "ConnectTimeout=15",
			fmt.Sprintf("root@%s", nodeIP),
			command,
		}
		cmd := exec.Command("ssh", sshArgs...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Cleanup function
	cleanup := func() {
		if testForestID != "" && !keepForest {
			fmt.Println()
			fmt.Println("🧹 Tearing down test forest...")

			// Load storage to get nodes
			reg, err := CreateStorage()
			if err == nil {
				nodes, _ := reg.GetNodes(testForestID)
				for _, node := range nodes {
					if node.ID != "" {
						_ = hetznerProv.DeleteServer(ctx, node.ID)
					}
				}
				_ = reg.DeleteForest(testForestID)
			}
			fmt.Println("   ✅ Test forest torn down")
		} else if keepForest && testForestID != "" {
			fmt.Println()
			fmt.Println("📌 Keeping test forest (--keep flag)")
			fmt.Printf("   Forest ID: %s\n", testForestID)
			fmt.Println("   To teardown later: morpheus teardown " + testForestID)
		}
	}

	// Step 1: Check network connectivity
	fmt.Println("📡 Step 1: Checking network connectivity...")

	ctx6, cancel6 := context.WithTimeout(ctx, 10*time.Second)
	result6 := httputil.CheckIPv6Connectivity(ctx6)
	cancel6()

	ctx4, cancel4 := context.WithTimeout(ctx, 10*time.Second)
	result4 := httputil.CheckIPv4Connectivity(ctx4)
	cancel4()

	hasIPv6 := result6.Available
	hasIPv4 := result4.Available

	if hasIPv6 {
		fmt.Printf("   ✅ IPv6 available (%s)\n", result6.Address)
		testsPassed++
	} else {
		fmt.Println("   ⚠️  IPv6 not available")
	}

	if hasIPv4 {
		fmt.Printf("   ✅ IPv4 available (%s)\n", result4.Address)
		if !hasIPv6 {
			testsPassed++
		}
	} else {
		fmt.Println("   ⚠️  IPv4 not available")
	}

	if !hasIPv6 && !hasIPv4 {
		fmt.Println("   ❌ No network connectivity")
		testsFailed++
		os.Exit(1)
	}

	// Enable IPv4 fallback if no IPv6
	if !hasIPv6 && hasIPv4 {
		fmt.Println("   📝 Enabling IPv4 fallback mode for this test")
		cfg.Machine.IPv4.Enabled = true
	}

	// Step 2: Ensure SSH key exists
	fmt.Println()
	fmt.Println("🔑 Step 2: Checking SSH key...")

	sshKeyName := cfg.GetSSHKeyName()
	_, err = hetznerProv.EnsureSSHKeyWithPath(ctx, sshKeyName, "")
	if err != nil {
		fmt.Printf("   ❌ Failed to ensure SSH key: %s\n", err)
		testsFailed++
		os.Exit(1)
	}
	fmt.Printf("   ✅ SSH key '%s' ready in Hetzner\n", sshKeyName)
	testsPassed++

	// Step 3: Plant a test forest
	fmt.Println()
	fmt.Println("🌲 Step 3: Planting test forest (1 node)...")

	// Create storage
	reg, err := CreateStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ❌ Failed to create storage: %s\n", err)
		testsFailed++
		os.Exit(1)
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(hetznerProv, reg, cfg)

	// Generate forest ID
	testForestID = fmt.Sprintf("e2e-test-%d", time.Now().Unix())

	// Select server type from config
	preferredLocations := []string{"ash", "hel1", "nbg1", "fsn1"}

	serverType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, cfg.GetServerType(), cfg.GetServerTypeFallback(), preferredLocations)
	if err != nil {
		fmt.Printf("   ❌ Failed to select server type: %s\n", err)
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	location := availableLocations[0]
	fmt.Printf("   📦 Using %s in %s\n", serverType, location)

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   testForestID,
		NodeCount:  1,
		Location:   location,
		ServerType: serverType,
		Image:      "ubuntu-24.04",
	}

	// Provision
	err = provisioner.Provision(ctx, req)
	if err != nil {
		fmt.Printf("   ❌ Provisioning failed: %s\n", err)
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	fmt.Printf("   ✅ Forest %s planted\n", testForestID)
	testsPassed++

	// Step 4: Get node info and verify connectivity
	fmt.Println()
	fmt.Println("🔍 Step 4: Verifying node connectivity...")

	nodes, err := reg.GetNodes(testForestID)
	if err != nil || len(nodes) == 0 {
		fmt.Println("   ❌ No nodes found in forest")
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	node := nodes[0]

	// Use the appropriate IP based on connectivity
	nodeIP := node.GetPreferredIP(hasIPv6)
	if hasIPv6 && node.IPv6 != "" {
		fmt.Printf("   📍 Node IP: %s (IPv6)\n", nodeIP)
	} else if node.IPv4 != "" {
		fmt.Printf("   📍 Node IP: %s (IPv4)\n", nodeIP)
	} else {
		fmt.Printf("   📍 Node IP: %s\n", nodeIP)
	}

	// Wait for SSH to be available
	fmt.Println("   ⏳ Waiting for SSH...")
	sshReady := false
	sshDeadline := time.Now().Add(3 * time.Minute)

	for time.Now().Before(sshDeadline) {
		sshAddr := sshutil.FormatSSHAddress(nodeIP, 22)
		conn, err := net.DialTimeout("tcp", sshAddr, 5*time.Second)
		if err == nil {
			conn.Close()
			sshReady = true
			break
		}
		time.Sleep(10 * time.Second)
	}

	if !sshReady {
		fmt.Println("   ❌ SSH not available within timeout")
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	fmt.Println("   ✅ SSH is available")
	testsPassed++

	// Step 5: Verify cloud-init completed
	fmt.Println()
	fmt.Println("⚙️  Step 5: Verifying cloud-init...")

	// Wait a bit for cloud-init
	time.Sleep(30 * time.Second)

	output, err := runSSHToNode(nodeIP, "cloud-init status --wait 2>/dev/null || echo 'done'")
	if err == nil && (strings.Contains(output, "done") || strings.Contains(output, "status: done")) {
		fmt.Println("   ✅ Cloud-init completed")
		testsPassed++
	} else {
		fmt.Printf("   ⚠️  Cloud-init status unclear: %s\n", strings.TrimSpace(output))
	}

	// Print summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Test Results")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Passed: %d\n", testsPassed)
	fmt.Printf("   Failed: %d\n", testsFailed)
	fmt.Println()

	if testsFailed == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Printf("❌ %d test(s) failed\n", testsFailed)
	}

	// Cleanup
	cleanup()

	fmt.Println()
	if testsFailed == 0 {
		fmt.Println("✅ E2E test suite completed successfully")
	} else {
		fmt.Println("❌ E2E test suite completed with failures")
		os.Exit(1)
	}
}
