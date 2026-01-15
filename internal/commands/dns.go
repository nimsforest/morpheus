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
	fmt.Println("DNS Management - Manage DNS zones and records via Hetzner")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus dns <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  zone     Manage DNS zones")
	fmt.Println("  record   Manage DNS records")
	fmt.Println()
	fmt.Println("Use 'morpheus dns <command> help' for more information.")
}
