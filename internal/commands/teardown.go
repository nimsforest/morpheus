package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/nimsforest/morpheus/pkg/forest"
)

// HandleTeardown handles the teardown command.
func HandleTeardown() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus teardown <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	// First, get the forest info to determine the provider
	storageProv, err := CreateStorage()
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

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		os.Exit(1)
	}

	// Create provider
	machineProv, _, err := CreateMachineProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// Create DNS provider if configured
	dnsProv := CreateDNSProvider(cfg)

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
