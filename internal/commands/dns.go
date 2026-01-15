package commands

import (
	"fmt"
	"os"
)

// HandleDNS handles the dns command group
func HandleDNS() {
	if len(os.Args) < 3 {
		printDNSHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]
	switch subcommand {
	// Simple commands (like "plant")
	case "add":
		HandleDNSAdd()
	case "remove":
		HandleDNSRemove()
	case "status":
		HandleDNSStatus()

	// Advanced commands
	case "zone":
		handleDNSZone()
	case "record":
		handleDNSRecord()

	case "help", "--help", "-h":
		printDNSHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown dns subcommand: %s\n\n", subcommand)
		printDNSHelp()
		os.Exit(1)
	}
}

func printDNSHelp() {
	fmt.Println("🌐 DNS Management - Manage DNS via Hetzner")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus dns <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  add <domain>             Create a DNS zone")
	fmt.Println("  remove <domain>          Delete zone and all records")
	fmt.Println("  status [domain]          Show zones or zone details")
	fmt.Println()
	fmt.Println("Advanced:")
	fmt.Println("  zone <cmd>               Zone management (create/list/get/delete)")
	fmt.Println("  record <cmd>             Record management (create/list/delete)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns add nimsforest.com")
	fmt.Println("  morpheus dns status")
	fmt.Println("  morpheus dns status nimsforest.com")
	fmt.Println()
	fmt.Println("Use 'morpheus dns <command> --help' for more info.")
}
