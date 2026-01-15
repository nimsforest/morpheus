// Package commands provides the command handlers for the CLI.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/dns"
	dnshetzner "github.com/nimsforest/morpheus/pkg/dns/hetzner"
	dnsnone "github.com/nimsforest/morpheus/pkg/dns/none"
	"github.com/nimsforest/morpheus/pkg/machine"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
	"github.com/nimsforest/morpheus/pkg/storage"
)

// LoadConfig loads the configuration from the default locations.
func LoadConfig() (*config.Config, error) {
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

// GetRegistryPath returns the path to the registry file.
func GetRegistryPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}

	registryDir := filepath.Join(homeDir, ".morpheus")
	os.MkdirAll(registryDir, 0755)

	return filepath.Join(registryDir, "registry.json")
}

// CreateMachineProvider creates a machine provider based on the configuration.
func CreateMachineProvider(cfg *config.Config) (machine.Provider, string, error) {
	var machineProv machine.Provider
	var err error
	var providerName string

	switch cfg.GetMachineProvider() {
	case "hetzner":
		machineProv, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create provider: %w", err)
		}
		providerName = "hetzner"
	default:
		return nil, "", fmt.Errorf("unsupported provider: %s", cfg.GetMachineProvider())
	}

	return machineProv, providerName, nil
}

// CreateDNSProvider creates a DNS provider based on the configuration.
// Auto-detects Hetzner if dns_domain and hetzner_dns_token are set.
func CreateDNSProvider(cfg *config.Config) dns.Provider {
	// If no domain configured, no DNS integration
	if cfg.DNS.Domain == "" {
		return nil
	}

	// If token is available, use Hetzner DNS
	dnsToken := cfg.GetDNSToken()
	if dnsToken != "" {
		dnsProv, err := dnshetzner.NewProvider(dnsToken)
		if err != nil {
			fmt.Printf("⚠️  Warning: DNS provider not available: %s\n", err)
			return nil
		}
		return dnsProv
	}

	// Explicit provider config (legacy)
	if cfg.DNS.Provider != "" && cfg.DNS.Provider != "none" {
		switch cfg.DNS.Provider {
		case "hetzner":
			// Token already checked above
			return nil
		default:
			dnsProv, _ := dnsnone.NewProvider()
			return dnsProv
		}
	}

	return nil
}

// CreateStorage creates a local registry storage.
func CreateStorage() (storage.Registry, error) {
	registryPath := GetRegistryPath()
	return storage.NewLocalRegistry(registryPath)
}

// GetEnvOrDefault returns the environment variable value or a default.
func GetEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetEnvOrDefaultInt returns the environment variable value as int or a default.
func GetEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// IsValidSSHKey checks if a string looks like a valid SSH public key.
func IsValidSSHKey(key string) bool {
	key = strings.TrimSpace(key)
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2-"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// ClassifyNetError returns a human-readable description of a network error.
func ClassifyNetError(err error) string {
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

// JoinLocations joins location names with commas.
func JoinLocations(locations []string) string {
	return strings.Join(locations, ", ")
}

// ContainsLocationError checks if an error message indicates a location availability issue.
func ContainsLocationError(errMsg string) bool {
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

// OrderLocationsByPreference reorders available locations to match the preferred order.
func OrderLocationsByPreference(available, preferredOrder []string) []string {
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
