package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/nimsforest/morpheus/pkg/config"
)

// HandleConfig handles the config command.
func HandleConfig() {
	if len(os.Args) < 3 {
		printConfigHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "set":
		handleConfigSet()
	case "get":
		handleConfigGet()
	case "list":
		handleConfigList()
	case "path":
		handleConfigPath()
	case "help", "--help", "-h":
		printConfigHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown config subcommand: %s\n\n", subcommand)
		printConfigHelp()
		os.Exit(1)
	}
}

func printConfigHelp() {
	fmt.Println("⚙️  Morpheus Config - Manage Configuration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus config <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  set <key> <value>    Set a configuration value (persists to file)")
	fmt.Println("  get <key>            Get a configuration value")
	fmt.Println("  list                 List all configurable keys")
	fmt.Println("  path                 Show config file location")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus config set hetzner_api_token YOUR_TOKEN_HERE")
	fmt.Println("  morpheus config set machine_provider hetzner")
	fmt.Println("  morpheus config set ipv4_enabled true")
	fmt.Println("  morpheus config get hetzner_api_token")
	fmt.Println("  morpheus config list")
	fmt.Println()
	fmt.Println("Common Keys:")
	fmt.Println("  hetzner_api_token    Hetzner API token (used for Cloud and DNS)")
	fmt.Println("  machine_provider     Machine provider (hetzner, local, none)")
	fmt.Println("  ipv4_enabled         Enable IPv4 (true/false)")
	fmt.Println("  server_type          Server type (e.g., cx22)")
	fmt.Println("  location             Datacenter location (e.g., fsn1)")
	fmt.Println()
	fmt.Println("Note:")
	fmt.Println("  Values set with 'config set' are persisted to the config file.")
	fmt.Println("  Environment variables still override config file values at runtime.")
}

func handleConfigSet() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus config set <key> <value>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus config set hetzner_api_token YOUR_TOKEN_HERE")
		fmt.Fprintln(os.Stderr, "  morpheus config set machine_provider hetzner")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Run 'morpheus config list' to see all available keys.")
		os.Exit(1)
	}

	key := os.Args[3]
	value := os.Args[4]

	// Find or create config path
	configPath := config.FindConfigPath()
	if configPath == "" {
		// Create default config path
		if err := config.EnsureConfigDir(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create config directory: %s\n", err)
			os.Exit(1)
		}
		configPath = config.GetDefaultConfigPath()
	}

	// Set the value
	if err := config.SetConfigValue(configPath, key, value); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to set config value: %s\n", err)
		os.Exit(1)
	}

	// Show success message
	maskedValue := value
	if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "password") {
		maskedValue = config.MaskToken(value)
	}

	fmt.Printf("✅ Set %s = %s\n", key, maskedValue)
	fmt.Printf("   Saved to: %s\n", configPath)

	// Verify the config is valid after the change
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("\n⚠️  Warning: Config file has issues: %s\n", err)
	} else if err := cfg.Validate(); err != nil {
		fmt.Printf("\n💡 Note: Config validation: %s\n", err)
		fmt.Println("   This may be okay if you haven't set all required values yet.")
	}
}

func handleConfigGet() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus config get <key>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus config get hetzner_api_token")
		fmt.Fprintln(os.Stderr, "  morpheus config get machine_provider")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Run 'morpheus config list' to see all available keys.")
		os.Exit(1)
	}

	key := os.Args[3]

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		// If no config file, show empty
		fmt.Printf("%s = (not configured)\n", key)
		fmt.Println()
		fmt.Println("No config file found. Create one with:")
		fmt.Printf("  morpheus config set %s <value>\n", key)
		os.Exit(0)
	}

	value, fromEnv := config.GetConfigValue(cfg, key)
	if value == "" {
		fmt.Printf("%s = (not set)\n", key)
	} else {
		// Mask tokens and passwords
		displayValue := value
		if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "password") {
			displayValue = config.MaskToken(value)
		}

		if fromEnv {
			envVar := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			fmt.Printf("%s = %s (from env: %s)\n", key, displayValue, envVar)
		} else {
			fmt.Printf("%s = %s\n", key, displayValue)
		}
	}
}

func handleConfigList() {
	fmt.Println("⚙️  Available Configuration Keys")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	keys := config.ListConfigKeys()

	// Group keys by category
	secretKeys := []string{}
	machineKeys := []string{}
	dnsKeys := []string{}
	otherKeys := []string{}

	for _, key := range keys {
		switch {
		case strings.Contains(key, "token") || strings.Contains(key, "password"):
			secretKeys = append(secretKeys, key)
		case strings.Contains(key, "machine") || strings.Contains(key, "ssh") || strings.Contains(key, "ipv4") || strings.Contains(key, "server") || strings.Contains(key, "location") || strings.Contains(key, "image"):
			machineKeys = append(machineKeys, key)
		case strings.Contains(key, "dns"):
			dnsKeys = append(dnsKeys, key)
		default:
			otherKeys = append(otherKeys, key)
		}
	}

	// Load config to show current values
	cfg, _ := LoadConfig()

	fmt.Println("🔐 Secrets:")
	for _, key := range secretKeys {
		printConfigKeyValue(cfg, key)
	}

	fmt.Println()
	fmt.Println("🖥️  Machine:")
	for _, key := range machineKeys {
		printConfigKeyValue(cfg, key)
	}

	if len(dnsKeys) > 0 {
		fmt.Println()
		fmt.Println("🌐 DNS:")
		for _, key := range dnsKeys {
			printConfigKeyValue(cfg, key)
		}
	}

	if len(otherKeys) > 0 {
		fmt.Println()
		fmt.Println("📋 Other:")
		for _, key := range otherKeys {
			printConfigKeyValue(cfg, key)
		}
	}

	fmt.Println()
	fmt.Println("💡 Set a value: morpheus config set <key> <value>")
}

func printConfigKeyValue(cfg *config.Config, key string) {
	if cfg == nil {
		fmt.Printf("   %-22s (not configured)\n", key)
		return
	}

	value, fromEnv := config.GetConfigValue(cfg, key)
	if value == "" {
		fmt.Printf("   %-22s (not set)\n", key)
	} else {
		// Mask tokens and passwords
		displayValue := value
		if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "password") {
			displayValue = config.MaskToken(value)
		}

		source := ""
		if fromEnv {
			source = " (env)"
		}
		fmt.Printf("   %-22s %s%s\n", key, displayValue, source)
	}
}

func handleConfigPath() {
	configPath := config.FindConfigPath()
	if configPath != "" {
		fmt.Printf("Config file: %s\n", configPath)
	} else {
		fmt.Println("No config file found.")
		fmt.Println()
		fmt.Println("Searched locations:")
		fmt.Println("  • ./config.yaml")
		fmt.Printf("  • %s\n", config.GetDefaultConfigPath())
		fmt.Println("  • /etc/morpheus/config.yaml")
		fmt.Println()
		fmt.Println("Create a config file with:")
		fmt.Println("  morpheus config set hetzner_api_token YOUR_TOKEN_HERE")
		fmt.Printf("\nThis will create: %s\n", config.GetDefaultConfigPath())
	}
}
