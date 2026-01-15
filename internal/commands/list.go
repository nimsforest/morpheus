package commands

import (
	"fmt"
	"os"
)

// HandleList handles the list command.
func HandleList() {
	storageProv, err := CreateStorage()
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
