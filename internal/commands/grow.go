package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nimsforest/morpheus/internal/ui"
	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
	"github.com/nimsforest/morpheus/pkg/nats"
	"github.com/nimsforest/morpheus/pkg/storage"
)

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

// HandleGrow handles the grow command.
func HandleGrow() {
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
	reg, err := CreateStorage()
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
		reachableNodes, ui.Plural(reachableNodes),
		totalConns, ui.Plural(totalConns))
	fmt.Println()

	// Display resource usage with progress bars
	fmt.Printf("Resource Usage:\n")
	fmt.Printf("  CPU:    %5.1f%% %s\n", avgCPU, ui.ProgressBar(avgCPU, threshold))
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
				ui.TruncateID(info.NodeID, 13),
				ui.TruncateIP(info.IP, 23),
				info.CPU,
				info.MemMB,
				info.Connections,
				warning)
		} else {
			fmt.Printf("  %-13s %-23s    -        -       -  ❌ unreachable\n",
				ui.TruncateID(info.NodeID, 13),
				ui.TruncateIP(info.IP, 23))
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

// expandCluster adds new nodes to the cluster
func expandCluster(forestID string, forestInfo *storage.Forest, reg storage.Registry, nodeCount int) {
	fmt.Println()
	fmt.Printf("🌱 Adding %d node%s to cluster...\n", nodeCount, ui.Plural(nodeCount))

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		return
	}

	// Create provider
	machineProv, _, err := CreateMachineProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
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
