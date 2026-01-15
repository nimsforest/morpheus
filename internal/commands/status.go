package commands

import (
	"fmt"
	"os"

	"github.com/nimsforest/morpheus/internal/ui"
	"github.com/nimsforest/morpheus/pkg/sshutil"
)

// HandleStatus handles the status command.
func HandleStatus() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus status <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	storageProv, err := CreateStorage()
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
				ui.TruncateIP(node.IP, 24),
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
			fmt.Printf("   ... (%d more machine%s)\n", len(nodes)-2, ui.Plural(len(nodes)-2))
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
