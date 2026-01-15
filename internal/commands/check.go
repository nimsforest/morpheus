package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
	"github.com/nimsforest/morpheus/pkg/sshutil"
)

// HandleCheckIPv6 handles the check-ipv6 command.
func HandleCheckIPv6() {
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

// HandleCheck handles the check command.
func HandleCheck() {
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
	case "config":
		runConfigCheck(true)
	case "":
		// Run all checks
		fmt.Println("🔍 Running Morpheus Diagnostics")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		configOk := runConfigCheck(false)
		fmt.Println()
		ipv6Ok, ipv4Ok := runNetworkCheck(false)
		fmt.Println()
		sshOk := runSSHCheck(false)

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if configOk && ipv6Ok && sshOk {
			fmt.Println("✅ All checks passed! You're ready to use Morpheus.")
		} else if configOk && ipv4Ok && sshOk {
			fmt.Println("⚠️  IPv6 not available, but IPv4 works.")
			fmt.Println("   Enable IPv4 fallback in config.yaml:")
			fmt.Println("     machine:")
			fmt.Println("       ipv4:")
			fmt.Println("         enabled: true")
			fmt.Println("   Note: IPv4 costs extra on Hetzner.")
		} else if !configOk {
			fmt.Println("❌ Configuration issues detected. Please review the issues above.")
			os.Exit(1)
		} else {
			fmt.Println("⚠️  Some checks failed. Please review the issues above.")
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown check: %s\n\n", subcommand)
		fmt.Fprintln(os.Stderr, "Usage: morpheus check [config|ipv6|ipv4|network|ssh]")
		fmt.Fprintln(os.Stderr, "  morpheus check         Run all checks")
		fmt.Fprintln(os.Stderr, "  morpheus check config  Check config file and env variables")
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
			if IsValidSSHKey(content) {
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
	cfg, err := LoadConfig()
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
	reg, err := CreateStorage()
	if err == nil {
		forests := reg.ListForests()
		var activeNodes []*struct {
			IP string
		}
		for _, f := range forests {
			if f.Status == "active" {
				nodes, err := reg.GetNodes(f.ID)
				if err == nil {
					for _, n := range nodes {
						activeNodes = append(activeNodes, &struct{ IP string }{IP: n.IP})
					}
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
				fmt.Printf("   ⚠️  Cannot reach server %s: %s\n", node.IP, ClassifyNetError(err))
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

// runConfigCheck checks if config file exists and all required env variables are set
func runConfigCheck(exitOnResult bool) bool {
	fmt.Println("📋 Configuration")

	allOk := true
	var configPath string
	var cfg *config.Config
	var loadErr error

	// Check for config file in standard locations
	configPaths := []string{
		"./config.yaml",
		filepath.Join(os.Getenv("HOME"), ".morpheus", "config.yaml"),
		"/etc/morpheus/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			cfg, loadErr = config.LoadConfig(path)
			break
		}
	}

	if configPath == "" {
		fmt.Println("   ❌ No config file found")
		fmt.Println()
		fmt.Println("   Searched locations:")
		for _, p := range configPaths {
			fmt.Printf("      • %s\n", p)
		}
		fmt.Println()
		fmt.Println("   Create a config file:")
		fmt.Println("      cp config.example.yaml config.yaml")
		fmt.Println("      # Edit config.yaml with your settings")
		allOk = false
	} else if loadErr != nil {
		fmt.Printf("   ❌ Config file found but failed to load: %s\n", configPath)
		fmt.Printf("      Error: %s\n", loadErr)
		allOk = false
	} else {
		fmt.Printf("   ✅ Config file loaded: %s\n", configPath)
	}

	fmt.Println()
	fmt.Println("   Secrets & Credentials:")

	// Check required secrets (can come from env or config file)
	type secretVar struct {
		name        string
		description string
		required    bool
		hasValue    bool
		masked      string
		source      string // "env", "config", or ""
	}

	vars := []secretVar{
		{
			name:        "HETZNER_API_TOKEN",
			description: "Hetzner Cloud API token",
			required:    cfg == nil || cfg.GetMachineProvider() == "hetzner" || cfg.GetMachineProvider() == "",
		},
		{
			name:        "HETZNER_DNS_TOKEN",
			description: "Hetzner DNS API token (optional, uses HETZNER_API_TOKEN as fallback)",
			required:    false,
		},
		{
			name:        "STORAGEBOX_PASSWORD",
			description: "Hetzner StorageBox password (for shared registry)",
			required:    cfg != nil && cfg.GetStorageProvider() == "storagebox",
		},
		{
			name:        "PROXMOX_HOST",
			description: "Proxmox VE host (for VR mode management)",
			required:    false,
		},
		{
			name:        "PROXMOX_API_TOKEN",
			description: "Proxmox API token secret",
			required:    false,
		},
		{
			name:        "PROXMOX_TOKEN_ID",
			description: "Proxmox API token ID (e.g., morpheus@pam!token)",
			required:    false,
		},
	}

	// Helper to mask a value
	maskValue := func(val string) string {
		if len(val) > 8 {
			return val[:4] + "..." + val[len(val)-4:]
		}
		return "****"
	}

	// Check each variable - first check env, then config file
	for i := range vars {
		envVal := os.Getenv(vars[i].name)
		if envVal != "" {
			vars[i].hasValue = true
			vars[i].masked = maskValue(envVal)
			vars[i].source = "env"
		}
	}

	// Check config file for values (only if not already set from env)
	if cfg != nil {
		// HETZNER_API_TOKEN
		for i := range vars {
			if vars[i].name == "HETZNER_API_TOKEN" && vars[i].source == "" {
				if cfg.Secrets.HetznerAPIToken != "" {
					vars[i].hasValue = true
					vars[i].masked = maskValue(cfg.Secrets.HetznerAPIToken)
					vars[i].source = "config"
				}
				break
			}
		}
		// HETZNER_DNS_TOKEN
		for i := range vars {
			if vars[i].name == "HETZNER_DNS_TOKEN" && vars[i].source == "" {
				if cfg.Secrets.HetznerDNSToken != "" {
					vars[i].hasValue = true
					vars[i].masked = maskValue(cfg.Secrets.HetznerDNSToken)
					vars[i].source = "config"
				}
				break
			}
		}
		// STORAGEBOX_PASSWORD
		for i := range vars {
			if vars[i].name == "STORAGEBOX_PASSWORD" && vars[i].source == "" {
				if cfg.Storage.StorageBox.Password != "" {
					vars[i].hasValue = true
					vars[i].masked = maskValue(cfg.Storage.StorageBox.Password)
					vars[i].source = "config"
				}
				break
			}
		}
	}

	// Display results
	hasRequired := true
	for _, v := range vars {
		if v.required && !v.hasValue {
			hasRequired = false
		}

		if v.hasValue {
			sourceLabel := ""
			switch v.source {
			case "env":
				sourceLabel = " ← from environment variable"
			case "config":
				sourceLabel = " ← from config file (persistent)"
			}

			if v.required {
				fmt.Printf("      ✅ %s: %s%s\n", v.name, v.masked, sourceLabel)
			} else {
				fmt.Printf("      ✅ %s: %s (optional)%s\n", v.name, v.masked, sourceLabel)
			}
		} else {
			if v.required {
				fmt.Printf("      ❌ %s: not set (REQUIRED)\n", v.name)
				fmt.Printf("         %s\n", v.description)
				fmt.Printf("         Set with: morpheus config set %s <value>\n", strings.ToLower(strings.ReplaceAll(v.name, "_", "_")))
			} else {
				fmt.Printf("      ○  %s: not set (optional)\n", v.name)
			}
		}
	}

	if !hasRequired {
		allOk = false
	}

	// Validate configuration if loaded
	if cfg != nil {
		fmt.Println()
		fmt.Println("   Configuration Settings:")

		provider := cfg.GetMachineProvider()
		if provider != "" {
			fmt.Printf("      Machine provider: %s\n", provider)
		} else {
			fmt.Println("      ❌ Machine provider: not set")
			allOk = false
		}

		fmt.Printf("      Server type:      %s\n", cfg.GetServerType())
		fmt.Printf("      Image:            %s\n", cfg.GetImage())
		fmt.Printf("      Location:         %s\n", cfg.GetLocation())
		fmt.Printf("      SSH key name:     %s\n", cfg.GetSSHKeyName())
		fmt.Printf("      IPv4 enabled:     %v\n", cfg.IsIPv4Enabled())
		fmt.Printf("      Storage provider: %s\n", cfg.GetStorageProvider())

		if cfg.DNS.Provider != "" && cfg.DNS.Provider != "none" {
			fmt.Printf("      DNS provider:     %s (domain: %s)\n", cfg.DNS.Provider, cfg.DNS.Domain)
		}

		// Run validation
		if err := cfg.Validate(); err != nil {
			fmt.Println()
			fmt.Printf("   ❌ Config validation failed: %s\n", err)
			allOk = false
		} else {
			fmt.Println()
			fmt.Println("   ✅ Config validation passed")
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
