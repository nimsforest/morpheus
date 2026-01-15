// Package cli provides the command-line interface for morpheus.
package cli

import (
	"fmt"
	"os"

	"github.com/nimsforest/morpheus/internal/commands"
)

// Version is set at build time via -ldflags
var Version = "dev"

// Run executes the CLI with the given arguments.
func Run() {
	if len(os.Args) < 2 {
		PrintHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "plant":
		commands.HandlePlant()
	case "list":
		commands.HandleList()
	case "status":
		commands.HandleStatus()
	case "teardown":
		commands.HandleTeardown()
	case "grow":
		commands.HandleGrow()
	case "mode":
		commands.HandleMode()
	case "config":
		commands.HandleConfig()
	case "version":
		fmt.Printf("morpheus version %s\n", Version)
	case "update":
		commands.HandleUpdate(Version)
	case "check-update":
		commands.HandleCheckUpdate(Version)
	case "check-ipv6":
		commands.HandleCheckIPv6()
	case "check":
		commands.HandleCheck()
	case "test":
		commands.HandleTest()
	case "help", "--help", "-h":
		PrintHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		PrintHelp()
		os.Exit(1)
	}
}

// PrintHelp prints the main help message.
func PrintHelp() {
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
	fmt.Println("  config <subcommand>      Manage configuration")
	fmt.Println("    set <key> <value>      Set a config value (persists to file)")
	fmt.Println("    get <key>              Get a config value")
	fmt.Println("    list                   List all configurable keys")
	fmt.Println("    path                   Show config file location")
	fmt.Println()
	fmt.Println("  mode <subcommand>        VR node boot mode management")
	fmt.Println("    list                   List available modes")
	fmt.Println("    status                 Show current mode")
	fmt.Println("    linux                  Switch to Linux (CachyOS + WiVRN)")
	fmt.Println("    windows                Switch to Windows (SteamLink)")
	fmt.Println()
	fmt.Println("  check                    Run all diagnostics")
	fmt.Println("  check config             Check config file and env variables")
	fmt.Println("  check ipv6               Check IPv6 connectivity")
	fmt.Println("  check ssh                Check SSH key setup")
	fmt.Println("  check dns                Check DNS configuration and Hetzner DNS zones")
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
	fmt.Println("  morpheus config set hetzner_api_token YOUR_TOKEN")
	fmt.Println("  morpheus config list        # View all settings")
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
	fmt.Println("  Use 'morpheus config set' to persist settings to config file.")
	fmt.Println()
	fmt.Println("More info: https://github.com/nimsforest/morpheus")
}
