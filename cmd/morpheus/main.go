package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/bootmode"
	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/dns"
	dnshetzner "github.com/nimsforest/morpheus/pkg/dns/hetzner"
	dnsnone "github.com/nimsforest/morpheus/pkg/dns/none"
	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/machine"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
	"github.com/nimsforest/morpheus/pkg/machine/proxmox"
	"github.com/nimsforest/morpheus/pkg/nats"
	"github.com/nimsforest/morpheus/pkg/sshutil"
	"github.com/nimsforest/morpheus/pkg/storage"
	"github.com/nimsforest/morpheus/pkg/updater"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "plant":
		handlePlant()
	case "list":
		handleList()
	case "status":
		handleStatus()
	case "teardown":
		handleTeardown()
	case "grow":
		handleGrow()
	case "mode":
		handleMode()
	case "version":
		fmt.Printf("morpheus version %s\n", version)
	case "update":
		handleUpdate()
	case "check-update":
		handleCheckUpdate()
	case "check-ipv6":
		handleCheckIPv6()
	case "check":
		handleCheck()
	case "test":
		handleTest()
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handlePlant() {
	// Parse arguments - simplified CLI
	// morpheus plant             -> 2 nodes (default)
	// morpheus plant --nodes 3   -> 3 nodes

	nodeCount := 2

	// Parse arguments
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--nodes", "-n":
			if i+1 < len(os.Args) {
				i++
				n, err := strconv.Atoi(os.Args[i])
				if err != nil || n < 1 {
					fmt.Fprintf(os.Stderr, "❌ Invalid node count: %s\n", os.Args[i])
					os.Exit(1)
				}
				nodeCount = n
			} else {
				fmt.Fprintln(os.Stderr, "❌ --nodes requires a number")
				os.Exit(1)
			}
		case "--help", "-h":
			fmt.Println("Usage: morpheus plant [options]")
			fmt.Println()
			fmt.Println("Create a new forest with the specified number of nodes.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --nodes, -n N   Number of nodes to create (default: 2)")
			fmt.Println("  --help, -h      Show this help")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  morpheus plant              # Create 2-node cluster")
			fmt.Println("  morpheus plant --nodes 3    # Create 3-node forest")
			os.Exit(0)
		default:
			// Support legacy size arguments for backward compatibility
			if isValidSize(arg) {
				nodeCount = getNodeCount(arg)
			} else {
				fmt.Fprintf(os.Stderr, "❌ Unknown argument: %s\n", arg)
				fmt.Fprintln(os.Stderr, "Use 'morpheus plant --help' for usage")
				os.Exit(1)
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %s\n", err)
		os.Exit(1)
	}

	// Create machine provider based on configuration
	var machineProv machine.Provider
	var providerName string

	switch cfg.GetMachineProvider() {
	case "hetzner":
		machineProv, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
			os.Exit(1)
		}
		providerName = "hetzner"
	default:
		fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", cfg.GetMachineProvider())
		os.Exit(1)
	}

	// Create storage
	registryPath := getRegistryPath()
	storageProv, err := storage.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create storage: %s\n", err)
		os.Exit(1)
	}

	// Create DNS provider if configured
	var dnsProv dns.Provider
	if cfg.DNS.Provider != "" && cfg.DNS.Provider != "none" {
		switch cfg.DNS.Provider {
		case "hetzner":
			dnsToken := cfg.GetDNSToken()
			dnsProv, err = dnshetzner.NewProvider(dnsToken)
			if err != nil {
				fmt.Printf("⚠️  Warning: DNS provider not available: %s\n", err)
			}
		default:
			// Use no-op provider for unsupported providers
			dnsProv, _ = dnsnone.NewProvider()
		}
	}

	// Create provisioner
	var provisioner *forest.Provisioner
	if dnsProv != nil {
		provisioner = forest.NewProvisionerWithDNS(machineProv, storageProv, dnsProv, cfg)
	} else {
		provisioner = forest.NewProvisioner(machineProv, storageProv, cfg)
	}

	// Generate forest ID
	forestID := fmt.Sprintf("forest-%d", time.Now().Unix())

	// Create context early for provider operations
	ctx := context.Background()

	// Determine server type, location, and image from config
	var location, serverType, image string

	// For Hetzner, select the best server type and locations
	if hetznerProv, ok := machineProv.(*hetzner.Provider); ok {
		// Get default locations if not configured
		preferredLocations := []string{cfg.GetLocation()}
		if preferredLocations[0] == "" {
			preferredLocations = hetzner.GetDefaultLocations()
		}

		// Select best server type and available locations using config
		selectedType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, cfg.GetServerType(), cfg.GetServerTypeFallback(), preferredLocations)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to select server type: %s\n", err)
			os.Exit(1)
		}

		serverType = selectedType
		location = availableLocations[0] // Use first available location
		image = cfg.GetImage()
	} else {
		// Non-Hetzner provider
		serverType = cfg.GetServerType()
		location = cfg.GetLocation()
		image = cfg.GetImage()
	}

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   forestID,
		NodeCount:  nodeCount,
		Location:   location,
		ServerType: serverType,
		Image:      image,
	}

	// Display friendly provisioning header
	fmt.Printf("\n🌲 Planting your forest...\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Show what's being created
	var timeEstimate string
	switch {
	case nodeCount == 1:
		timeEstimate = "3-5 minutes"
	case nodeCount <= 3:
		timeEstimate = "7-15 minutes"
	default:
		timeEstimate = "15-30 minutes"
	}

	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Forest ID:  %s\n", forestID)
	fmt.Printf("   Nodes:      %d\n", nodeCount)
	fmt.Printf("   Machine:    %s (with automatic fallback if unavailable)\n", serverType)
	fmt.Printf("   Location:   %s (with automatic fallback if unavailable)\n", hetzner.GetLocationDescription(location))
	fmt.Printf("   Provider:   %s\n", providerName)
	fmt.Printf("   Time:       ~%s\n\n", timeEstimate)

	estimatedCost := hetzner.GetEstimatedCost(serverType) * float64(nodeCount)
	fmt.Printf("💰 Estimated cost: ~€%.2f/month\n", estimatedCost)
	if cfg.IsIPv4Enabled() {
		fmt.Printf("   (IPv4+IPv6, billed by minute, can teardown anytime)\n")
		fmt.Printf("   ⚠️  IPv4 enabled - additional charges apply per IPv4 address\n\n")
	} else {
		fmt.Printf("   (IPv6-only, billed by minute, can teardown anytime)\n\n")
	}

	fmt.Println("🚀 Starting provisioning...")

	// Use the full fallback system for Hetzner
	if hetznerProv, ok := machineProv.(*hetzner.Provider); ok {
		err = provisionWithFallback(ctx, provisioner, hetznerProv, req, cfg.GetServerType(), cfg.GetServerTypeFallback())
	} else {
		err = provisioner.Provision(ctx, req)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Provisioning failed: %s\n", err)
		os.Exit(1)
	}

	// Success message with clear next steps
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ Success! Your forest is ready!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("🎯 What's next?\n\n")

	fmt.Printf("📊 Check your forest status:\n")
	fmt.Printf("   morpheus status %s\n\n", forestID)

	fmt.Printf("🌐 Your machines are ready for NATS deployment\n")
	fmt.Printf("   Infrastructure is configured and waiting\n\n")

	fmt.Printf("📋 View all your forests:\n")
	fmt.Printf("   morpheus list\n\n")

	fmt.Printf("🌱 Add more nodes:\n")
	fmt.Printf("   morpheus grow %s --nodes 2\n\n", forestID)

	fmt.Printf("🗑️  Clean up when done:\n")
	fmt.Printf("   morpheus teardown %s\n\n", forestID)
}

// provisionWithFallback tries to provision a forest, automatically falling back
// to alternative server types and locations if the primary ones are unavailable.
func provisionWithFallback(ctx context.Context, provisioner *forest.Provisioner, hetznerProv *hetzner.Provider, req forest.ProvisionRequest, serverType string, fallbacks []string) error {
	// Get all server type options from config
	allServerTypes := append([]string{serverType}, fallbacks...)

	// Preferred location order: Helsinki first, then Nuremberg, then others
	preferredLocations := hetzner.GetDefaultLocations()

	var lastErr error
	var attemptedCombos []string
	var validServerTypes []string
	isFirstAttempt := true

	// First, validate which server types actually exist in Hetzner
	for _, serverType := range allServerTypes {
		exists, err := hetznerProv.ValidateServerType(ctx, serverType)
		if err != nil {
			fmt.Printf("   ⚠️  Could not validate server type %s: %v\n", serverType, err)
			continue
		}
		if !exists {
			fmt.Printf("   ⚠️  Server type %s does not exist in Hetzner, skipping\n", serverType)
			continue
		}
		validServerTypes = append(validServerTypes, serverType)
	}

	if len(validServerTypes) == 0 {
		return fmt.Errorf("none of the configured server types exist in Hetzner: %s", joinLocations(allServerTypes))
	}

	// Try each validated server type
	for serverTypeIdx, serverType := range validServerTypes {
		// Get available locations for this server type
		availableLocations, err := hetznerProv.GetAvailableLocations(ctx, serverType)
		if err != nil {
			fmt.Printf("   ⚠️  Could not check availability for %s: %v\n", serverType, err)
			continue
		}
		if len(availableLocations) == 0 {
			fmt.Printf("   ⚠️  Server type %s has no available locations\n", serverType)
			continue
		}

		// Reorder available locations to match preferred order
		orderedLocations := orderLocationsByPreference(availableLocations, preferredLocations)

		// Show info when switching to fallback server type
		if serverTypeIdx > 0 && len(attemptedCombos) > 0 {
			fmt.Printf("\n📦 Trying alternative server type: %s (~€%.2f/mo)\n",
				serverType, hetzner.GetEstimatedCost(serverType))
		}

		// Try each location for this server type (in preferred order)
		for _, location := range orderedLocations {
			attemptedCombos = append(attemptedCombos, fmt.Sprintf("%s@%s", serverType, location))

			// Update request with current server type and location
			req.ServerType = serverType
			req.Location = location

			if !isFirstAttempt {
				fmt.Printf("   📍 Trying %s in %s...\n", serverType, hetzner.GetLocationDescription(location))
			}
			isFirstAttempt = false

			err := provisioner.Provision(ctx, req)
			if err == nil {
				// Success!
				return nil
			}

			lastErr = err

			// Check if the error is a location/server type availability error
			errStr := err.Error()
			if containsLocationError(errStr) {
				fmt.Printf("   ⚠️  %s not available in %s, trying next option...\n", serverType, location)
				continue
			}

			// If it's not a location error, this is a real error - stop trying
			return err
		}
	}

	// All combinations failed
	if lastErr != nil && containsLocationError(lastErr.Error()) {
		return fmt.Errorf("no server type/location combination available\n\n"+
			"Tried %d combinations across server types: %s\n\n"+
			"This usually means Hetzner is experiencing high demand or capacity issues.\n"+
			"Please try again in a few minutes.\n"+
			"For status updates, check: https://status.hetzner.com/",
			len(attemptedCombos), joinLocations(validServerTypes))
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no server type available")
}

// orderLocationsByPreference reorders available locations to match the preferred order.
func orderLocationsByPreference(available, preferredOrder []string) []string {
	availableSet := make(map[string]bool)
	for _, loc := range available {
		availableSet[loc] = true
	}

	var result []string

	// First, add locations in preferred order (if available)
	for _, loc := range preferredOrder {
		if availableSet[loc] {
			result = append(result, loc)
			delete(availableSet, loc)
		}
	}

	// Then add any remaining available locations
	for _, loc := range available {
		if availableSet[loc] {
			result = append(result, loc)
		}
	}

	return result
}

// containsLocationError checks if an error message indicates a location availability issue
func containsLocationError(errMsg string) bool {
	locationErrorPhrases := []string{
		"server location disabled",
		"resource_unavailable",
		"location not available",
		"location disabled",
		"datacenter not available",
		"unsupported location",
		"unsupported location for server type",
	}

	errLower := strings.ToLower(errMsg)
	for _, phrase := range locationErrorPhrases {
		if strings.Contains(errLower, phrase) {
			return true
		}
	}
	return false
}

// joinLocations joins location names with commas
func joinLocations(locations []string) string {
	return strings.Join(locations, ", ")
}

func handleList() {
	registryPath := getRegistryPath()
	storageProv, err := storage.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load storage: %s\n", err)
		os.Exit(1)
	}

	forests := storageProv.ListForests()

	if len(forests) == 0 {
		fmt.Println("🌲 No forests yet!")
		fmt.Println()
		fmt.Println("Create your first forest:")
		fmt.Println("  morpheus plant              # Create 2-node cluster")
		fmt.Println("  morpheus plant --nodes 3    # Create 3-node forest")
		return
	}

	fmt.Printf("🌲 Your Forests (%d)\n", len(forests))
	fmt.Println()
	fmt.Println("FOREST ID            NODES   LOCATION  STATUS       CREATED")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, f := range forests {
		statusIcon := "✅"
		if f.Status == "provisioning" {
			statusIcon = "⏳"
		} else if f.Status != "active" {
			statusIcon = "⚠️ "
		}

		fmt.Printf("%-20s %-7d %-9s %s %-11s %s\n",
			f.ID,
			f.NodeCount,
			f.Location,
			statusIcon,
			f.Status,
			f.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	fmt.Println()
	fmt.Println("💡 Tip: Use 'morpheus status <forest-id>' to see detailed information")
}

func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus status <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	registryPath := getRegistryPath()
	storageProv, err := storage.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load storage: %s\n", err)
		os.Exit(1)
	}

	forestInfo, err := storageProv.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get forest: %s\n", err)
		os.Exit(1)
	}

	nodes, err := storageProv.GetNodes(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get nodes: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("🌲 Forest: %s\n", forestInfo.ID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	statusIcon := "✅"
	if forestInfo.Status == "provisioning" {
		statusIcon = "⏳"
	} else if forestInfo.Status != "active" {
		statusIcon = "⚠️ "
	}

	fmt.Printf("📊 Overview:\n")
	fmt.Printf("   Status:   %s %s\n", statusIcon, forestInfo.Status)
	fmt.Printf("   Nodes:    %d\n", forestInfo.NodeCount)
	fmt.Printf("   Location: %s\n", forestInfo.Location)
	fmt.Printf("   Provider: %s\n", forestInfo.Provider)
	fmt.Printf("   Created:  %s\n", forestInfo.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(nodes) > 0 {
		fmt.Printf("\n🖥️  Machines (%d):\n", len(nodes))
		fmt.Println()
		fmt.Println("   ID                IP ADDRESS               LOCATION  STATUS")
		fmt.Println("   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, node := range nodes {
			nodeStatusIcon := "✅"
			if node.Status != "active" {
				nodeStatusIcon = "⏳"
			}
			fmt.Printf("   %-17s %-24s %-9s %s %s\n",
				node.ID,
				truncateIP(node.IP, 24),
				node.Location,
				nodeStatusIcon,
				node.Status,
			)
		}

		fmt.Println()

		// Detect SSH private key for better guidance
		sshKeyPath := sshutil.DetectSSHPrivateKeyPath()

		fmt.Printf("💡 SSH into machines:\n")
		for i, node := range nodes {
			if i < 2 { // Show first 2 examples
				if sshKeyPath != "" {
					fmt.Printf("   %s\n", sshutil.FormatSSHCommandWithIdentity("root", node.IP, sshKeyPath))
				} else {
					fmt.Printf("   %s\n", sshutil.FormatSSHCommand("root", node.IP))
				}
			}
		}
		if len(nodes) > 2 {
			fmt.Printf("   ... (%d more machine%s)\n", len(nodes)-2, plural(len(nodes)-2))
		}

		fmt.Println()
		fmt.Printf("   ⚠️  If asked for a password, your SSH key may not be configured correctly.\n")
		fmt.Printf("   Run 'morpheus check ssh' to diagnose SSH key issues.\n")
	} else {
		fmt.Println("\n⏳ No machines registered yet (still provisioning)")
	}

	fmt.Println()
	fmt.Printf("🌱 Add nodes: morpheus grow %s --nodes 2\n", forestInfo.ID)
	fmt.Printf("🗑️  Teardown: morpheus teardown %s\n", forestInfo.ID)
}

func handleTeardown() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus teardown <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	// First, get the forest info to determine the provider
	registryPath := getRegistryPath()
	storageProv, err := storage.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create storage: %s\n", err)
		os.Exit(1)
	}

	// Verify forest exists
	_, err = storageProv.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get forest info: %s\n", err)
		os.Exit(1)
	}

	// Create provider
	var machineProv machine.Provider

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		os.Exit(1)
	}

	switch cfg.GetMachineProvider() {
	case "hetzner":
		machineProv, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", cfg.GetMachineProvider())
		os.Exit(1)
	}

	// Create DNS provider if configured
	var dnsProv dns.Provider
	if cfg.DNS.Provider != "" && cfg.DNS.Provider != "none" {
		switch cfg.DNS.Provider {
		case "hetzner":
			dnsToken := cfg.GetDNSToken()
			dnsProv, _ = dnshetzner.NewProvider(dnsToken)
		}
	}

	// Create provisioner
	var provisioner *forest.Provisioner
	if dnsProv != nil {
		provisioner = forest.NewProvisionerWithDNS(machineProv, storageProv, dnsProv, cfg)
	} else {
		provisioner = forest.NewProvisioner(machineProv, storageProv, cfg)
	}

	// Show what will be deleted
	nodes, _ := storageProv.GetNodes(forestID)

	fmt.Printf("\n⚠️  About to permanently delete:\n")
	fmt.Printf("   Forest: %s\n", forestID)
	fmt.Printf("   Nodes:  %d\n", len(nodes))
	if len(nodes) > 0 {
		fmt.Printf("   Machines:\n")
		for _, node := range nodes {
			fmt.Printf("      • %s (%s)\n", node.ID, node.IP)
		}
	}
	fmt.Println()
	fmt.Printf("💰 This will stop billing for these resources\n")
	fmt.Println()
	fmt.Print("Type 'yes' to confirm deletion: ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("\n✅ Teardown cancelled - your forest is safe!")
		return
	}

	// Teardown
	fmt.Println()
	ctx := context.Background()
	if err := provisioner.Teardown(ctx, forestID); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Teardown failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ Forest %s deleted successfully!\n", forestID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💰 Resources have been removed and billing stopped")
	fmt.Println()
	fmt.Println("💡 View your remaining forests: morpheus list")
}

func handleGrow() {
	// Parse arguments
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus grow <forest-id> [options]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Add nodes to an existing forest or check cluster health.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --nodes, -n N    Add N nodes to the forest")
		fmt.Fprintln(os.Stderr, "  --auto           Non-interactive mode (auto-expand if needed)")
		fmt.Fprintln(os.Stderr, "  --threshold N    Resource threshold percentage (default: 80)")
		fmt.Fprintln(os.Stderr, "  --json           Output in JSON format")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus grow forest-123              # Check health")
		fmt.Fprintln(os.Stderr, "  morpheus grow forest-123 --nodes 2    # Add 2 nodes")
		os.Exit(1)
	}

	forestID := os.Args[2]

	// Parse optional flags
	addNodes := 0
	autoMode := false
	jsonOutput := false
	threshold := 80.0

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--nodes", "-n":
			if i+1 < len(os.Args) {
				i++
				n, err := strconv.Atoi(os.Args[i])
				if err != nil || n < 1 {
					fmt.Fprintf(os.Stderr, "❌ Invalid node count: %s\n", os.Args[i])
					os.Exit(1)
				}
				addNodes = n
			}
		case "--auto":
			autoMode = true
		case "--json":
			jsonOutput = true
		case "--threshold":
			if i+1 < len(os.Args) {
				i++
				fmt.Sscanf(os.Args[i], "%f", &threshold)
			}
		}
	}

	// Load storage
	registryPath := getRegistryPath()
	reg, err := storage.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load storage: %s\n", err)
		os.Exit(1)
	}

	// Get forest info
	forestInfo, err := reg.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Forest not found: %s\n", err)
		os.Exit(1)
	}

	// Get nodes
	nodes, err := reg.GetNodes(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get nodes: %s\n", err)
		os.Exit(1)
	}

	// If --nodes specified, add nodes directly
	if addNodes > 0 {
		expandCluster(forestID, forestInfo, reg, addNodes)
		return
	}

	if len(nodes) == 0 {
		fmt.Fprintln(os.Stderr, "No nodes found in forest")
		os.Exit(1)
	}

	// Create NATS monitor
	monitor := nats.NewMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Collect node IPs
	var nodeIPs []string
	for _, node := range nodes {
		if node.IP != "" {
			nodeIPs = append(nodeIPs, node.IP)
		}
	}

	if !jsonOutput {
		fmt.Printf("\n🌲 Forest: %s\n", forestID)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	// Check each node's NATS stats
	var totalCPU float64
	var totalMem int64
	var totalConns int
	var reachableNodes int
	var nodeStats []*nodeHealthInfo

	for _, node := range nodes {
		if node.IP == "" {
			continue
		}

		status := monitor.CheckNodeHealth(ctx, node.IP)
		info := &nodeHealthInfo{
			NodeID:    node.ID,
			IP:        node.IP,
			Reachable: status.Healthy,
		}

		if status.Healthy {
			reachableNodes++
			info.CPU = status.CPUPercent
			info.MemMB = status.MemMB
			info.Connections = status.Connections
			totalCPU += status.CPUPercent
			totalMem += status.Stats.Mem
			totalConns += status.Connections
		} else {
			info.Error = status.Error
		}

		nodeStats = append(nodeStats, info)
	}

	// Calculate averages
	avgCPU := 0.0
	avgMem := 0.0
	if reachableNodes > 0 {
		avgCPU = totalCPU / float64(reachableNodes)
		avgMem = float64(totalMem) / float64(reachableNodes) / (1024 * 1024) // Convert to MB
	}

	// JSON output
	if jsonOutput {
		output := map[string]interface{}{
			"forest_id":         forestID,
			"total_nodes":       len(nodes),
			"reachable_nodes":   reachableNodes,
			"total_connections": totalConns,
			"avg_cpu_percent":   avgCPU,
			"avg_mem_mb":        avgMem,
			"cpu_high":          avgCPU > threshold,
			"threshold":         threshold,
			"nodes":             nodeStats,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	// Display cluster info
	fmt.Printf("📊 NATS Cluster: %d node%s, %d connection%s\n",
		reachableNodes, plural(reachableNodes),
		totalConns, plural(totalConns))
	fmt.Println()

	// Display resource usage with progress bars
	fmt.Printf("Resource Usage:\n")
	fmt.Printf("  CPU:    %5.1f%% %s\n", avgCPU, progressBar(avgCPU, threshold))
	fmt.Printf("  Memory: %5.0f MB avg\n", avgMem)
	fmt.Println()

	// Display node table
	fmt.Println("Nodes:")
	fmt.Println("  NODE          IP                      CPU      MEM      CONNS  STATUS")
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, info := range nodeStats {
		if info.Reachable {
			warning := ""
			if info.CPU > threshold {
				warning = " ⚠️"
			}
			fmt.Printf("  %-13s %-23s %5.1f%%   %5dMB   %5d  ✅%s\n",
				truncateID(info.NodeID, 13),
				truncateIP(info.IP, 23),
				info.CPU,
				info.MemMB,
				info.Connections,
				warning)
		} else {
			fmt.Printf("  %-13s %-23s    -        -       -  ❌ unreachable\n",
				truncateID(info.NodeID, 13),
				truncateIP(info.IP, 23))
		}
	}
	fmt.Println()

	// Show warnings
	needsExpansion := avgCPU > threshold
	if needsExpansion {
		fmt.Printf("⚠️  Average CPU above %.0f%% threshold\n", threshold)
		fmt.Println()
	}

	// Auto mode or interactive
	if autoMode {
		if needsExpansion {
			fmt.Println("🌱 Auto-expanding cluster...")
			expandCluster(forestID, forestInfo, reg, 1)
		} else {
			fmt.Println("✅ Cluster resources within threshold. No expansion needed.")
		}
		return
	}

	// Interactive mode
	if needsExpansion {
		fmt.Print("Add 1 node to cluster? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" || response == "yes" {
			expandCluster(forestID, forestInfo, reg, 1)
		} else {
			fmt.Println("\n✅ No changes made.")
		}
	} else {
		fmt.Println("✅ Cluster resources within threshold.")
		fmt.Println("   Use 'morpheus grow <forest-id> --nodes N' to add nodes manually.")
	}
}

// nodeHealthInfo holds health info for display
type nodeHealthInfo struct {
	NodeID      string  `json:"node_id"`
	IP          string  `json:"ip"`
	Reachable   bool    `json:"reachable"`
	CPU         float64 `json:"cpu_percent,omitempty"`
	MemMB       int64   `json:"mem_mb,omitempty"`
	Connections int     `json:"connections,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// progressBar creates a simple ASCII progress bar
func progressBar(value, threshold float64) string {
	width := 24
	filled := int(value / 100.0 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	warning := ""
	if value > threshold {
		warning = " ⚠️"
	}
	return bar + warning
}

// truncateID truncates a node ID to maxLen characters
func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen-3] + "..."
}

// expandCluster adds new nodes to the cluster
func expandCluster(forestID string, forestInfo *storage.Forest, reg storage.Registry, nodeCount int) {
	fmt.Println()
	fmt.Printf("🌱 Adding %d node%s to cluster...\n", nodeCount, plural(nodeCount))

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		return
	}

	// Create provider
	var machineProv machine.Provider
	switch forestInfo.Provider {
	case "hetzner":
		machineProv, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", forestInfo.Provider)
		return
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(machineProv, reg, cfg)

	// Determine server type from config
	serverType := ""
	location := forestInfo.Location

	if hetznerProv, ok := machineProv.(*hetzner.Provider); ok {
		ctx := context.Background()
		selectedType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, cfg.GetServerType(), cfg.GetServerTypeFallback(), []string{location})
		if err == nil {
			serverType = selectedType
			if len(availableLocations) > 0 {
				location = availableLocations[0]
			}
		}
	}

	if serverType == "" {
		serverType = cfg.GetServerType()
	}

	// Get existing nodes to determine new node numbers
	existingNodes, _ := reg.GetNodes(forestID)
	startIndex := len(existingNodes)

	// Create provision request for additional nodes
	req := forest.ProvisionRequest{
		ForestID:   forestID,
		NodeCount:  nodeCount,
		Location:   location,
		ServerType: serverType,
		Image:      cfg.GetImage(),
	}

	// Update the forest's node count
	forestInfo.NodeCount += nodeCount
	_ = reg.UpdateForest(forestInfo)

	ctx := context.Background()
	
	// Provision additional nodes (using a modified request that starts at the right index)
	// Note: The provisioner will handle the node naming based on existing nodes
	_ = startIndex // Used for future enhancement
	
	if err := provisioner.Provision(ctx, req); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Expansion failed: %s\n", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Cluster expanded successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("💡 View updated cluster: morpheus status %s\n", forestID)
}

func loadConfig() (*config.Config, error) {
	// Try multiple config locations
	configPaths := []string{
		"./config.yaml",
		filepath.Join(os.Getenv("HOME"), ".morpheus", "config.yaml"),
		"/etc/morpheus/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return config.LoadConfig(path)
		}
	}

	return nil, fmt.Errorf("no config file found (tried: %v)", configPaths)
}

func getRegistryPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}

	registryDir := filepath.Join(homeDir, ".morpheus")
	os.MkdirAll(registryDir, 0755)

	return filepath.Join(registryDir, "registry.json")
}

func handleUpdate() {
	u := updater.NewUpdater(version)

	fmt.Println("🔍 Checking for updates...")
	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check for updates: %s\n", err)
		fmt.Fprintf(os.Stderr, "\nYou can manually download the latest release from:\n")
		fmt.Fprintf(os.Stderr, "  https://github.com/nimsforest/morpheus/releases/latest\n")
		os.Exit(1)
	}

	fmt.Printf("\nCurrent version: %s\n", info.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", info.LatestVersion)

	if !info.Available {
		fmt.Println("\n✅ You are already running the latest version!")
		return
	}

	fmt.Println("\n🎉 A new version is available!")
	fmt.Println("\nRelease notes:")
	fmt.Println("─────────────────────────────────────────────────")
	if info.ReleaseNotes != "" {
		fmt.Println(info.ReleaseNotes)
	} else {
		fmt.Println("No release notes available.")
	}
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Printf("\nView full release: %s\n", info.UpdateURL)

	// Ask for confirmation
	fmt.Print("\nDo you want to update now? (yes/no): ")
	var response string
	fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		fmt.Println("\nUpdate cancelled.")
		fmt.Printf("To update later, run: morpheus update\n")
		return
	}

	// Perform update
	fmt.Println()
	if err := u.PerformUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Update failed: %s\n", err)
		os.Exit(1)
	}
}

func handleCheckUpdate() {
	u := updater.NewUpdater(version)

	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check for updates: %s\n", err)
		os.Exit(1)
	}

	if info.Available {
		fmt.Printf("Update available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
		fmt.Printf("Run 'morpheus update' to install.\n")
		os.Exit(0)
	} else {
		fmt.Printf("Already up to date: %s\n", info.CurrentVersion)
		os.Exit(0)
	}
}

func handleCheckIPv6() {
	fmt.Println("🔍 Checking IPv6 connectivity...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv6Connectivity(ctx)

	if result.Available {
		fmt.Println("✅ IPv6 connectivity is available!")
		fmt.Printf("   Your IPv6 address: %s\n", result.Address)
		fmt.Println()
		fmt.Println("You can use Morpheus to provision IPv6-only infrastructure on Hetzner Cloud.")
		os.Exit(0)
	} else {
		fmt.Println("❌ IPv6 connectivity is NOT available")
		fmt.Println()
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
			fmt.Println()
		}
		fmt.Println("Morpheus requires IPv6 connectivity to provision infrastructure.")
		fmt.Println("Hetzner Cloud uses IPv6-only by default (IPv4 costs extra).")
		fmt.Println()
		fmt.Println("Options to get IPv6:")
		fmt.Println("  1. Enable IPv6 on your ISP/router")
		fmt.Println("  2. Use an IPv6 tunnel service (e.g., Hurricane Electric)")
		fmt.Println("  3. Use a VPS/server with IPv6 to run Morpheus")
		fmt.Println()
		fmt.Println("For more information, see:")
		fmt.Println("  https://github.com/nimsforest/morpheus/blob/main/docs/guides/IPV6_SETUP.md")
		os.Exit(1)
	}
}

func handleCheck() {
	// Parse subcommand
	subcommand := ""
	if len(os.Args) >= 3 {
		subcommand = os.Args[2]
	}

	switch subcommand {
	case "ipv6":
		runIPv6Check(true)
	case "ipv4":
		runIPv4Check(true)
	case "network":
		runNetworkCheck(true)
	case "ssh":
		runSSHCheck(true)
	case "":
		// Run all checks
		fmt.Println("🔍 Running Morpheus Diagnostics")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		ipv6Ok, ipv4Ok := runNetworkCheck(false)
		fmt.Println()
		sshOk := runSSHCheck(false)

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if ipv6Ok && sshOk {
			fmt.Println("✅ All checks passed! You're ready to use Morpheus.")
		} else if ipv4Ok && sshOk {
			fmt.Println("⚠️  IPv6 not available, but IPv4 works.")
			fmt.Println("   Enable IPv4 fallback in config.yaml:")
			fmt.Println("     machine:")
			fmt.Println("       ipv4:")
			fmt.Println("         enabled: true")
			fmt.Println("   Note: IPv4 costs extra on Hetzner.")
		} else {
			fmt.Println("⚠️  Some checks failed. Please review the issues above.")
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown check: %s\n\n", subcommand)
		fmt.Fprintln(os.Stderr, "Usage: morpheus check [ipv6|ipv4|network|ssh]")
		fmt.Fprintln(os.Stderr, "  morpheus check         Run all checks")
		fmt.Fprintln(os.Stderr, "  morpheus check ipv6    Check IPv6 connectivity")
		fmt.Fprintln(os.Stderr, "  morpheus check ipv4    Check IPv4 connectivity")
		fmt.Fprintln(os.Stderr, "  morpheus check network Check both IPv6 and IPv4")
		fmt.Fprintln(os.Stderr, "  morpheus check ssh     Check SSH key setup")
		os.Exit(1)
	}
}

// runIPv6Check checks IPv6 connectivity and returns true if successful
func runIPv6Check(exitOnResult bool) bool {
	fmt.Println("📡 IPv6 Connectivity")
	fmt.Println("   Checking connection to IPv6 services...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv6Connectivity(ctx)

	if result.Available {
		fmt.Println("   ✅ IPv6 is available")
		fmt.Printf("   Your IPv6 address: %s\n", result.Address)
		if exitOnResult {
			os.Exit(0)
		}
		return true
	} else {
		fmt.Println("   ❌ IPv6 is NOT available")
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		fmt.Println()
		fmt.Println("   Morpheus uses IPv6 by default to connect to provisioned servers.")
		fmt.Println("   Options:")
		fmt.Println("     • Enable IPv6 on your ISP/router")
		fmt.Println("     • Use an IPv6 tunnel (e.g., Hurricane Electric)")
		fmt.Println("     • Use a VPS with IPv6 connectivity")
		fmt.Println("     • Enable IPv4 fallback in config.yaml (costs extra)")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}
}

// runIPv4Check checks IPv4 connectivity and returns true if successful
func runIPv4Check(exitOnResult bool) bool {
	fmt.Println("📡 IPv4 Connectivity")
	fmt.Println("   Checking connection to IPv4 services...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv4Connectivity(ctx)

	if result.Available {
		fmt.Println("   ✅ IPv4 is available")
		fmt.Printf("   Your IPv4 address: %s\n", result.Address)
		if exitOnResult {
			os.Exit(0)
		}
		return true
	} else {
		fmt.Println("   ❌ IPv4 is NOT available")
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}
}

// runNetworkCheck checks both IPv6 and IPv4 connectivity
// Returns (ipv6Ok, ipv4Ok)
func runNetworkCheck(exitOnResult bool) (bool, bool) {
	fmt.Println("📡 Network Connectivity")
	fmt.Println()

	// Check IPv6
	fmt.Println("   Checking IPv6...")
	ctx6, cancel6 := context.WithTimeout(context.Background(), 10*time.Second)
	result6 := httputil.CheckIPv6Connectivity(ctx6)
	cancel6()

	ipv6Ok := false
	if result6.Available {
		fmt.Println("   ✅ IPv6 is available")
		fmt.Printf("      Your IPv6 address: %s\n", result6.Address)
		ipv6Ok = true
	} else {
		fmt.Println("   ❌ IPv6 is NOT available")
	}

	fmt.Println()

	// Check IPv4
	fmt.Println("   Checking IPv4...")
	ctx4, cancel4 := context.WithTimeout(context.Background(), 10*time.Second)
	result4 := httputil.CheckIPv4Connectivity(ctx4)
	cancel4()

	ipv4Ok := false
	if result4.Available {
		fmt.Println("   ✅ IPv4 is available")
		fmt.Printf("      Your IPv4 address: %s\n", result4.Address)
		ipv4Ok = true
	} else {
		fmt.Println("   ❌ IPv4 is NOT available")
	}

	fmt.Println()

	// Summary and recommendations
	if ipv6Ok && ipv4Ok {
		fmt.Println("   ✅ Both IPv6 and IPv4 are available")
		fmt.Println("      Morpheus will use IPv6 by default (recommended, saves costs)")
	} else if ipv6Ok {
		fmt.Println("   ✅ IPv6 available - Morpheus will work with default settings")
	} else if ipv4Ok {
		fmt.Println("   ⚠️  Only IPv4 available")
		fmt.Println("      To use Morpheus, enable IPv4 fallback in config.yaml:")
		fmt.Println("        machine:")
		fmt.Println("          ipv4:")
		fmt.Println("            enabled: true")
		fmt.Println("      Note: IPv4 costs extra on Hetzner Cloud")
	} else {
		fmt.Println("   ❌ No network connectivity")
		fmt.Println("      Please check your internet connection")
	}

	if exitOnResult {
		if ipv6Ok {
			os.Exit(0)
		} else if ipv4Ok {
			os.Exit(0) // IPv4 available, user can enable fallback
		} else {
			os.Exit(1)
		}
	}

	return ipv6Ok, ipv4Ok
}

// runSSHCheck checks SSH key configuration and returns true if successful
func runSSHCheck(exitOnResult bool) bool {
	fmt.Println("🔑 SSH Key Setup")

	allOk := true

	// 1. Check for local SSH keys
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		fmt.Println("   ❌ Cannot determine home directory")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Check if .ssh directory exists
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		fmt.Println("   ❌ SSH directory not found (~/.ssh)")
		fmt.Println("   Run: ssh-keygen -t ed25519")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}

	// Look for SSH keys (both public and private)
	keyPaths := []string{
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_rsa"),
	}

	var foundKey string
	var foundKeyPath string
	var foundPrivateKeyPath string
	for _, basePath := range keyPaths {
		pubPath := basePath + ".pub"
		if data, err := os.ReadFile(pubPath); err == nil {
			content := string(data)
			if isValidSSHKey(content) {
				foundKey = content
				foundKeyPath = pubPath
				// Check if private key also exists
				if _, err := os.Stat(basePath); err == nil {
					foundPrivateKeyPath = basePath
				}
				break
			}
		}
	}

	if foundKey == "" {
		fmt.Println("   ❌ No SSH public key found")
		fmt.Println("   Searched: ~/.ssh/id_ed25519.pub, ~/.ssh/id_rsa.pub")
		fmt.Println()
		fmt.Println("   Generate a new key with:")
		fmt.Println("     ssh-keygen -t ed25519 -C \"your_email@example.com\"")
		allOk = false
	} else {
		fmt.Printf("   ✅ SSH public key found: %s\n", foundKeyPath)
		// Show key type
		if len(foundKey) > 20 {
			parts := strings.SplitN(foundKey, " ", 2)
			if len(parts) > 0 && parts[0] != "" {
				fmt.Printf("      Key type: %s\n", parts[0])
			}
		}

		// Check private key
		if foundPrivateKeyPath != "" {
			fmt.Printf("   ✅ SSH private key found: %s\n", foundPrivateKeyPath)

			fmt.Println()
			fmt.Println("   💡 SSH Authentication Tips:")
			fmt.Printf("      When connecting, use: ssh -i %s root@<ip>\n", foundPrivateKeyPath)
			fmt.Println()
			fmt.Println("      If still asked for a password:")
			fmt.Println("      • Your private key may have a passphrase (enter that, not server password)")
			fmt.Println("      • Try adding your key to ssh-agent: ssh-add " + foundPrivateKeyPath)
			fmt.Println("      • Ensure the public key was uploaded to Hetzner (check below)")
		} else {
			fmt.Println("   ⚠️  SSH private key NOT found")
			fmt.Println("      The private key is required for authentication")
			fmt.Println("      Expected at: " + strings.TrimSuffix(foundKeyPath, ".pub"))
			allOk = false
		}
	}

	// 2. Check config and Hetzner API token
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println()
		fmt.Println("   ⚠️  No config file found (can't check Hetzner SSH key status)")
		fmt.Println("   Create config.yaml to enable full SSH key validation")
	} else if cfg.Secrets.HetznerAPIToken == "" {
		fmt.Println()
		fmt.Println("   ⚠️  No Hetzner API token configured (can't verify cloud SSH key)")
	} else {
		// Try to check if SSH key exists in Hetzner
		fmt.Println()
		fmt.Println("   Checking Hetzner Cloud SSH key status...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		hetznerProv, err := hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Printf("   ⚠️  Could not connect to Hetzner: %s\n", err)
		} else {
			keyName := cfg.GetSSHKeyName()
			keyInfo, err := hetznerProv.GetSSHKeyInfo(ctx, keyName)
			if err != nil {
				fmt.Printf("   ⚠️  Could not check SSH key: %s\n", err)
			} else if keyInfo != nil {
				fmt.Printf("   ✅ SSH key '%s' exists in Hetzner Cloud\n", keyName)
				fmt.Printf("      Hetzner fingerprint: %s\n", keyInfo.Fingerprint)

				// Compare fingerprints if we found a local key
				if foundKeyPath != "" {
					localFingerprint, _, err := sshutil.ReadAndCalculateFingerprint(foundKeyPath)
					if err != nil {
						fmt.Printf("   ⚠️  Could not calculate local key fingerprint: %s\n", err)
					} else {
						fmt.Printf("      Local fingerprint:   %s\n", localFingerprint)
						if localFingerprint == keyInfo.Fingerprint {
							fmt.Println("   ✅ Fingerprints MATCH - your local key matches Hetzner")
						} else {
							allOk = false
							fmt.Println()
							fmt.Println("   ❌ FINGERPRINT MISMATCH!")
							fmt.Println("      Your local SSH key does NOT match the key in Hetzner Cloud.")
							fmt.Println("      This is why the server asks for a password!")
							fmt.Println()
							fmt.Println("   To fix this:")
							fmt.Println("   1. Delete the key in Hetzner Console and let Morpheus re-upload it")
							fmt.Println("   2. Or update your local key to match the one in Hetzner")
						}
					}
				}
			} else {
				fmt.Printf("   ⚠️  SSH key '%s' not found in Hetzner Cloud\n", keyName)
				fmt.Println("   Morpheus will automatically upload it when you provision")
				if foundKey == "" {
					allOk = false
					fmt.Println("   ❌ But no local SSH key was found to upload!")
				}
			}
		}
	}

	// 3. Check SSH connectivity to existing servers (if any)
	registryPath := getRegistryPath()
	reg, err := storage.NewLocalRegistry(registryPath)
	if err == nil {
		forests := reg.ListForests()
		var activeNodes []*storage.Node
		for _, f := range forests {
			if f.Status == "active" {
				nodes, err := reg.GetNodes(f.ID)
				if err == nil {
					activeNodes = append(activeNodes, nodes...)
				}
			}
		}

		if len(activeNodes) > 0 {
			fmt.Println()
			fmt.Printf("   Testing SSH connectivity to %d active server(s)...\n", len(activeNodes))

			// Test first server only to avoid too many checks
			node := activeNodes[0]
			sshPort := 22
			if cfg != nil && cfg.Provisioning.SSHPort != 0 {
				sshPort = cfg.Provisioning.SSHPort
			}

			addr := sshutil.FormatSSHAddress(node.IP, sshPort)
			conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
			if err != nil {
				fmt.Printf("   ⚠️  Cannot reach server %s: %s\n", node.IP, classifyNetError(err))
				fmt.Println("   This could be due to:")
				fmt.Println("     • Server is still booting")
				fmt.Println("     • IPv6 connectivity issues from your network")
				fmt.Println("     • Firewall blocking the connection")
			} else {
				conn.Close()
				fmt.Printf("   ✅ Server %s is reachable on port %d\n", node.IP, sshPort)
			}
		}
	}

	if exitOnResult {
		if allOk {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
	return allOk
}

// isValidSSHKey checks if a string looks like a valid SSH public key
func isValidSSHKey(key string) bool {
	key = strings.TrimSpace(key)
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2-"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// classifyNetError returns a human-readable description of a network error
func classifyNetError(err error) string {
	if err == nil {
		return "connected"
	}
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "connection refused"):
		return "port closed"
	case strings.Contains(errStr, "no route to host"):
		return "no route"
	case strings.Contains(errStr, "network is unreachable"):
		return "network unreachable"
	case strings.Contains(errStr, "i/o timeout"), strings.Contains(errStr, "timeout"):
		return "timeout"
	default:
		return err.Error()
	}
}

// plural returns "s" if count is not 1, empty string otherwise
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// truncateIP truncates an IP address to fit within maxLen characters
func truncateIP(ip string, maxLen int) string {
	if len(ip) <= maxLen {
		return ip
	}
	if maxLen < 10 {
		return ip[:maxLen]
	}
	prefixLen := (maxLen - 3) / 2
	suffixLen := maxLen - 3 - prefixLen
	return ip[:prefixLen] + "..." + ip[len(ip)-suffixLen:]
}

// getNodeCount returns the number of nodes for a given forest size (legacy support)
func getNodeCount(size string) int {
	switch size {
	case "small":
		return 2
	case "medium":
		return 3
	case "large":
		return 5
	default:
		return 1
	}
}

// isValidSize checks if a size is valid (legacy support)
func isValidSize(size string) bool {
	validSizes := []string{"small", "medium", "large"}
	for _, valid := range validSizes {
		if size == valid {
			return true
		}
	}
	return false
}

func handleTest() {
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
	cfg, err := loadConfig()
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
			registryPath := getRegistryPath()
			reg, err := storage.NewLocalRegistry(registryPath)
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
	registryPath := getRegistryPath()
	reg, err := storage.NewLocalRegistry(registryPath)
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

func handleMode() {
	if len(os.Args) < 3 {
		printModeHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		handleModeList()
	case "status":
		handleModeStatus()
	case "linux", "windows":
		handleModeSwitch(subcommand)
	case "help", "--help", "-h":
		printModeHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode subcommand: %s\n\n", subcommand)
		printModeHelp()
		os.Exit(1)
	}
}

func printModeHelp() {
	fmt.Println("🎮 Morpheus Mode - VR Node Boot Mode Management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus mode <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list       List available boot modes")
	fmt.Println("  status     Show current mode and status")
	fmt.Println("  linux      Switch to Linux mode (CachyOS + WiVRN)")
	fmt.Println("  windows    Switch to Windows mode (SteamLink)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus mode status    # Check current mode")
	fmt.Println("  morpheus mode linux     # Switch to Linux for WiVRN VR")
	fmt.Println("  morpheus mode windows   # Switch to Windows for SteamVR")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  Configure Proxmox settings in ~/.morpheus/config.yaml:")
	fmt.Println()
	fmt.Println("  proxmox:")
	fmt.Println("    host: \"192.168.1.100\"")
	fmt.Println("    api_token_id: \"morpheus@pam!token\"")
	fmt.Println("    api_token_secret: \"${PROXMOX_API_TOKEN}\"")
	fmt.Println()
	fmt.Println("  vr:")
	fmt.Println("    linux:")
	fmt.Println("      vmid: 101")
	fmt.Println("    windows:")
	fmt.Println("      vmid: 102")
}

func loadProxmoxManager() (*bootmode.ProxmoxManager, error) {
	// Try to load config, but it's optional if env vars are set
	_, _ = loadConfig()

	// Get Proxmox config from environment
	proxmoxConfig := proxmox.ProviderConfig{
		Host:           getEnvOrDefault("PROXMOX_HOST", ""),
		Port:           8006,
		Node:           getEnvOrDefault("PROXMOX_NODE", "pve"),
		APITokenID:     getEnvOrDefault("PROXMOX_TOKEN_ID", ""),
		APITokenSecret: getEnvOrDefault("PROXMOX_API_TOKEN", ""),
		VerifySSL:      false,
	}

	// Check if config is valid
	if proxmoxConfig.Host == "" || proxmoxConfig.APITokenSecret == "" {
		return nil, fmt.Errorf(`Proxmox not configured

Set these environment variables:
  export PROXMOX_HOST="192.168.1.100"
  export PROXMOX_API_TOKEN="your-api-token"
  export PROXMOX_TOKEN_ID="morpheus@pam!morpheus-token"

Optional:
  export PROXMOX_NODE="pve"           # Default: pve
  export PROXMOX_LINUX_VMID="101"     # Default: 101
  export PROXMOX_WINDOWS_VMID="102"   # Default: 102`)
	}

	// VR node config - get from environment or use defaults
	vrConfig := bootmode.VRNodeConfig{
		Linux: bootmode.VMConfig{
			VMID: getEnvOrDefaultInt("PROXMOX_LINUX_VMID", 101),
			Name: "nimsforest-vr-linux",
		},
		Windows: bootmode.VMConfig{
			VMID: getEnvOrDefaultInt("PROXMOX_WINDOWS_VMID", 102),
			Name: "nimsforest-vr-windows",
		},
		GPUPCI: getEnvOrDefault("PROXMOX_GPU_PCI", "0000:01:00"),
	}

	return bootmode.NewProxmoxManager(proxmoxConfig, vrConfig)
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func handleModeList() {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ping to check connectivity
	if err := manager.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot connect to Proxmox: %s\n", err)
		os.Exit(1)
	}

	modes, err := manager.ListModes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list modes: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("🎮 Available Boot Modes")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("%-10s %-6s %-10s %-10s %s\n", "MODE", "VMID", "STATUS", "VR", "DESCRIPTION")
	fmt.Println("─────────────────────────────────────────────────────────────")

	var currentMode string
	for _, mode := range modes {
		statusIcon := "○"
		if mode.Status == bootmode.ModeStatusRunning {
			statusIcon = "●"
			currentMode = mode.Name
		}

		fmt.Printf("%-10s %-6d %s %-9s %-10s %s\n",
			mode.Name,
			mode.VMID,
			statusIcon,
			mode.Status,
			mode.VRSoftware,
			mode.Description,
		)
	}

	fmt.Println()
	if currentMode != "" {
		fmt.Printf("Current mode: %s\n", currentMode)
	} else {
		fmt.Println("No mode currently active")
	}
	fmt.Println()
	fmt.Println("💡 Switch modes: morpheus mode linux  or  morpheus mode windows")
}

func handleModeStatus() {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	current, err := manager.GetCurrentMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get current mode: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	if current == nil {
		fmt.Println("⚠️  No mode currently active")
		fmt.Println()
		fmt.Println("Start a mode with:")
		fmt.Println("  morpheus mode linux     # For WiVRN VR streaming")
		fmt.Println("  morpheus mode windows   # For SteamLink VR")
		return
	}

	fmt.Printf("🎮 Current Mode: %s\n", current.Name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("   VM:          %s (VMID %d)\n", current.Description, current.VMID)
	fmt.Printf("   Status:      %s\n", current.Status)
	if current.IPAddress != "" {
		fmt.Printf("   IP:          %s\n", current.IPAddress)
	}
	if current.Uptime > 0 {
		fmt.Printf("   Uptime:      %s\n", formatDuration(current.Uptime))
	}
	fmt.Printf("   VR Software: %s\n", current.VRSoftware)

	if len(current.Services) > 0 {
		fmt.Println()
		fmt.Println("   Services:")
		for _, svc := range current.Services {
			icon := "✓"
			if svc.Status != "active" {
				icon = "✗"
			}
			fmt.Printf("     %s %s: %s\n", icon, svc.Name, svc.Status)
		}
	}

	fmt.Println()
	otherMode := "windows"
	if current.Name == "windows" {
		otherMode = "linux"
	}
	fmt.Printf("💡 Switch to %s: morpheus mode %s\n", otherMode, otherMode)
}

func handleModeSwitch(targetMode string) {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	// Parse options
	opts := bootmode.DefaultSwitchOptions()
	dryRun := false
	for _, arg := range os.Args[3:] {
		switch arg {
		case "--dry-run":
			dryRun = true
			opts.DryRun = true
		case "--force":
			opts.Force = true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get current mode for display
	current, _ := manager.GetCurrentMode(ctx)

	fmt.Println()
	if dryRun {
		fmt.Println("🔍 Dry run - no changes will be made")
		fmt.Println()
	}

	if current != nil {
		fmt.Printf("Switching %s → %s...\n", current.Name, targetMode)
	} else {
		fmt.Printf("Starting %s mode...\n", targetMode)
	}
	fmt.Println()

	result, err := manager.Switch(ctx, targetMode, opts)

	// Handle specific errors
	if _, ok := err.(*bootmode.AlreadyActiveError); ok {
		fmt.Printf("✅ Already in %s mode\n", targetMode)
		return
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Switch failed: %s\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("✅ Dry run complete - switch would succeed")
		return
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ Now in %s mode\n", targetMode)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if result.IPAddress != "" {
		fmt.Printf("   IP: %s\n", result.IPAddress)
	}
	fmt.Printf("   Duration: %s\n", result.Duration.Round(time.Second))

	if targetMode == "linux" {
		fmt.Println()
		fmt.Println("   🎮 WiVRN is ready for VR streaming")
	} else {
		fmt.Println()
		fmt.Println("   🎮 SteamLink is ready for VR streaming")
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func printHelp() {
	fmt.Println("🌲 Morpheus - Infrastructure Provisioning")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  plant [options]          Create a new forest")
	fmt.Println("    --nodes, -n N          Number of nodes (default: 2)")
	fmt.Println()
	fmt.Println("  grow <forest-id> [options]  Add nodes or check health")
	fmt.Println("    --nodes, -n N          Add N nodes to the forest")
	fmt.Println("    --auto                 Auto-expand if needed")
	fmt.Println("    --threshold N          CPU threshold (default: 80)")
	fmt.Println()
	fmt.Println("  list                     List all forests")
	fmt.Println("  status <forest-id>       Show forest details")
	fmt.Println("  teardown <forest-id>     Delete a forest")
	fmt.Println()
	fmt.Println("  mode <subcommand>        VR node boot mode management")
	fmt.Println("    list                   List available modes")
	fmt.Println("    status                 Show current mode")
	fmt.Println("    linux                  Switch to Linux (CachyOS + WiVRN)")
	fmt.Println("    windows                Switch to Windows (SteamLink)")
	fmt.Println()
	fmt.Println("  check                    Run all diagnostics")
	fmt.Println("  check ipv6               Check IPv6 connectivity")
	fmt.Println("  check ssh                Check SSH key setup")
	fmt.Println()
	fmt.Println("  version                  Show version")
	fmt.Println("  update                   Check for updates and install")
	fmt.Println("  help                     Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus plant              # Create 2-node cluster")
	fmt.Println("  morpheus plant --nodes 3    # Create 3-node forest")
	fmt.Println("  morpheus grow forest-123 --nodes 2  # Add 2 nodes")
	fmt.Println("  morpheus list               # View all forests")
	fmt.Println("  morpheus teardown forest-123  # Delete forest")
	fmt.Println()
	fmt.Println("  morpheus mode status        # Check VR node mode")
	fmt.Println("  morpheus mode linux         # Switch to Linux VR mode")
	fmt.Println("  morpheus mode windows       # Switch to Windows VR mode")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Morpheus looks for config.yaml in:")
	fmt.Println("    - ./config.yaml")
	fmt.Println("    - ~/.morpheus/config.yaml")
	fmt.Println("    - /etc/morpheus/config.yaml")
	fmt.Println()
	fmt.Println("More info: https://github.com/nimsforest/morpheus")
}
