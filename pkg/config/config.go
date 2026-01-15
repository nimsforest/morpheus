package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Morpheus configuration
type Config struct {
	// New structure
	Machine      MachineConfig      `yaml:"machine"`
	DNS          DNSConfig          `yaml:"dns"`
	Storage      StorageConfig      `yaml:"storage"`
	Secrets      SecretsConfig      `yaml:"secrets"`
	Provisioning ProvisioningConfig `yaml:"provisioning"`

	// Legacy structure (for backward compatibility)
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`
	Integration    IntegrationConfig    `yaml:"integration"`
	Registry       RegistryConfig       `yaml:"registry"`
}

// MachineConfig defines machine provider settings
type MachineConfig struct {
	Provider string        `yaml:"provider"` // hetzner, local, none
	Hetzner  HetznerConfig `yaml:"hetzner"`
	SSH      SSHConfig     `yaml:"ssh"`
	IPv4     IPv4Config    `yaml:"ipv4"`
}

// HetznerConfig defines Hetzner-specific machine settings
type HetznerConfig struct {
	ServerType         string   `yaml:"server_type"`          // e.g., cx22
	ServerTypeFallback []string `yaml:"server_type_fallback"` // e.g., [cpx11, cx32]
	Image              string   `yaml:"image"`                // e.g., ubuntu-24.04
	Location           string   `yaml:"location"`             // e.g., fsn1
}

// IPv4Config defines IPv4 settings
type IPv4Config struct {
	Enabled bool `yaml:"enabled"` // Enable IPv4 (costs extra on Hetzner)
}

// DNSConfig defines DNS provider settings
type DNSConfig struct {
	Provider string `yaml:"provider"` // hetzner, hosts, none
	Domain   string `yaml:"domain"`   // Base domain for DNS records
	TTL      int    `yaml:"ttl"`      // TTL for DNS records
}

// StorageConfig defines storage provider settings
type StorageConfig struct {
	Provider   string             `yaml:"provider"` // storagebox, local, none
	StorageBox StorageBoxConfig   `yaml:"storagebox"`
	Local      LocalStorageConfig `yaml:"local"`
}

// StorageBoxConfig defines Hetzner StorageBox settings
type StorageBoxConfig struct {
	Host     string `yaml:"host"`     // uXXXXX.your-storagebox.de
	Username string `yaml:"username"` // uXXXXX
	Password string `yaml:"password"` // or ${STORAGEBOX_PASSWORD}
}

// LocalStorageConfig defines local storage settings
type LocalStorageConfig struct {
	Path string `yaml:"path"` // Path to local registry file
}

// RegistryConfig defines remote registry settings for multi-device access
// DEPRECATED: Use StorageConfig instead
type RegistryConfig struct {
	Type           string `yaml:"type"`            // "storagebox", "local", or "none"
	URL            string `yaml:"url"`             // WebDAV URL for storagebox (CLI access), or file path for local
	Username       string `yaml:"username"`        // Authentication username
	Password       string `yaml:"password"`        // Authentication password (or ${STORAGEBOX_PASSWORD} env var)
	StorageBoxHost string `yaml:"storagebox_host"` // CIFS host for nodes to mount: uXXXXX.your-storagebox.de
}

// ProvisioningConfig defines settings for the provisioning process
type ProvisioningConfig struct {
	// ReadinessTimeout is how long to wait for infrastructure to be ready (default: 5m)
	ReadinessTimeout string `yaml:"readiness_timeout"`
	// ReadinessInterval is how often to check readiness (default: 10s)
	ReadinessInterval string `yaml:"readiness_interval"`
	// SSHPort is the port to check for SSH connectivity (default: 22)
	SSHPort int `yaml:"ssh_port"`
}

// InfrastructureConfig defines infrastructure provider settings
// DEPRECATED: Use MachineConfig instead
type InfrastructureConfig struct {
	Provider string    `yaml:"provider"`
	SSH      SSHConfig `yaml:"ssh"`

	// IPv4 fallback configuration
	// By default, Morpheus uses IPv6-only to save costs (IPv4 costs extra on Hetzner)
	// Enable IPv4 fallback if your network doesn't have IPv6 connectivity
	EnableIPv4Fallback bool `yaml:"enable_ipv4_fallback"`

	// DEPRECATED: Legacy fields for backward compatibility
	Defaults  *DefaultServerConfig `yaml:"defaults,omitempty"`
	Locations []string             `yaml:"locations,omitempty"`
}

// SSHConfig defines SSH key settings
type SSHConfig struct {
	KeyName string `yaml:"key_name"` // Name of the SSH key (will be uploaded if needed)
	KeyPath string `yaml:"key_path"` // Optional: Path to SSH public key file
}

// IntegrationConfig defines integration with NimsForest
type IntegrationConfig struct {
	NimsForestURL string `yaml:"nimsforest_url"` // URL for NimsForest bootstrap callbacks
	RegistryURL   string `yaml:"registry_url"`   // Optional: Morpheus registry URL

	// NimsForest auto-installation settings (NimsForest includes embedded NATS)
	// By default, Morpheus will install NimsForest on all provisioned machines
	NimsForestInstall     bool   `yaml:"nimsforest_install"`      // Auto-install NimsForest on provisioned machines (default: true)
	NimsForestDownloadURL string `yaml:"nimsforest_download_url"` // URL to download binary (default: latest from GitHub)
	NimsForestVersion     string `yaml:"nimsforest_version"`      // Version to download (default: latest)
}

const (
	// DefaultNimsForestDownloadURL is the base URL for NimsForest releases
	DefaultNimsForestDownloadURL = "https://github.com/nimsforest/nimsforest2/releases/latest/download/forest-linux-amd64"
	// DefaultNimsForestVersion is the default version (empty means latest)
	DefaultNimsForestVersion = ""
)

// DefaultsConfig defines default server settings (DEPRECATED)
type DefaultsConfig struct {
	ServerType string `yaml:"server_type"`
	Image      string `yaml:"image"`
	SSHKey     string `yaml:"ssh_key"`      // Name of the SSH key in Hetzner Cloud
	SSHKeyPath string `yaml:"ssh_key_path"` // Optional: Path to local SSH public key file for auto-upload
}

// DefaultServerConfig is an alias for backward compatibility (DEPRECATED)
type DefaultServerConfig = DefaultsConfig

// SecretsConfig contains API tokens and credentials
type SecretsConfig struct {
	HetznerAPIToken string `yaml:"hetzner_api_token"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Trim whitespace/newlines from tokens that may be present in the config
	config.Secrets.HetznerAPIToken = strings.TrimSpace(config.Secrets.HetznerAPIToken)

	// Override with environment variables if set
	// Trim whitespace/newlines that may be present in the token
	if token := strings.TrimSpace(os.Getenv("HETZNER_API_TOKEN")); token != "" {
		config.Secrets.HetznerAPIToken = token
	}

	// Expand environment variables in storage password
	config.expandStoragePassword()

	// Apply defaults and migrate legacy config
	config.applyDefaults()
	config.migrateLegacyConfig()

	return &config, nil
}

// expandStoragePassword expands environment variables in storage password
func (c *Config) expandStoragePassword() {
	// Check for STORAGEBOX_PASSWORD env var first - it always overrides
	envPass := strings.TrimSpace(os.Getenv("STORAGEBOX_PASSWORD"))

	// New storage config
	if strings.HasPrefix(c.Storage.StorageBox.Password, "${") && strings.HasSuffix(c.Storage.StorageBox.Password, "}") {
		envVar := c.Storage.StorageBox.Password[2 : len(c.Storage.StorageBox.Password)-1]
		c.Storage.StorageBox.Password = strings.TrimSpace(os.Getenv(envVar))
	}
	if envPass != "" {
		c.Storage.StorageBox.Password = envPass
	}

	// Legacy registry config
	if strings.HasPrefix(c.Registry.Password, "${") && strings.HasSuffix(c.Registry.Password, "}") {
		envVar := c.Registry.Password[2 : len(c.Registry.Password)-1]
		c.Registry.Password = strings.TrimSpace(os.Getenv(envVar))
	}
	// STORAGEBOX_PASSWORD always overrides for legacy config too
	if envPass != "" {
		c.Registry.Password = envPass
	}
}

// applyDefaults sets default values for the configuration
func (c *Config) applyDefaults() {
	// Provisioning defaults
	if c.Provisioning.ReadinessTimeout == "" {
		c.Provisioning.ReadinessTimeout = "5m"
	}
	if c.Provisioning.ReadinessInterval == "" {
		c.Provisioning.ReadinessInterval = "5s"
	}
	if c.Provisioning.SSHPort == 0 {
		c.Provisioning.SSHPort = 22
	}

	// Machine defaults
	if c.Machine.SSH.KeyName == "" {
		c.Machine.SSH.KeyName = "morpheus"
	}
	if c.Machine.Hetzner.Image == "" {
		c.Machine.Hetzner.Image = "ubuntu-24.04"
	}
	if c.Machine.Hetzner.ServerType == "" {
		c.Machine.Hetzner.ServerType = "cx22"
	}
	if c.Machine.Hetzner.Location == "" {
		c.Machine.Hetzner.Location = "fsn1"
	}

	// DNS defaults
	if c.DNS.TTL == 0 {
		c.DNS.TTL = 300
	}
	if c.DNS.Provider == "" {
		c.DNS.Provider = "none"
	}

	// Storage defaults
	if c.Storage.Provider == "" {
		c.Storage.Provider = "local"
	}

	// NimsForest integration defaults - install by default
	// NimsForestInstall defaults to true (install NimsForest on all machines)
	if c.Integration.NimsForestDownloadURL == "" {
		c.Integration.NimsForestDownloadURL = DefaultNimsForestDownloadURL
		// If URL wasn't set, enable install by default
		c.Integration.NimsForestInstall = true
	}
}

// migrateLegacyConfig migrates from the old config format to the new one
func (c *Config) migrateLegacyConfig() {
	// Migrate from Infrastructure to Machine
	if c.Machine.Provider == "" && c.Infrastructure.Provider != "" {
		c.Machine.Provider = c.Infrastructure.Provider
	}
	if c.Machine.SSH.KeyName == "" || c.Machine.SSH.KeyName == "morpheus" {
		if c.Infrastructure.SSH.KeyName != "" {
			c.Machine.SSH.KeyName = c.Infrastructure.SSH.KeyName
		}
	}
	if c.Machine.SSH.KeyPath == "" && c.Infrastructure.SSH.KeyPath != "" {
		c.Machine.SSH.KeyPath = c.Infrastructure.SSH.KeyPath
	}
	if !c.Machine.IPv4.Enabled && c.Infrastructure.EnableIPv4Fallback {
		c.Machine.IPv4.Enabled = true
	}

	// Migrate from legacy Defaults
	if c.Infrastructure.Defaults != nil {
		if c.Machine.SSH.KeyName == "" || c.Machine.SSH.KeyName == "morpheus" {
			if c.Infrastructure.Defaults.SSHKey != "" {
				c.Machine.SSH.KeyName = c.Infrastructure.Defaults.SSHKey
			}
		}
		if c.Machine.SSH.KeyPath == "" && c.Infrastructure.Defaults.SSHKeyPath != "" {
			c.Machine.SSH.KeyPath = c.Infrastructure.Defaults.SSHKeyPath
		}
		if c.Machine.Hetzner.ServerType == "" || c.Machine.Hetzner.ServerType == "cx22" {
			if c.Infrastructure.Defaults.ServerType != "" {
				c.Machine.Hetzner.ServerType = c.Infrastructure.Defaults.ServerType
			}
		}
		if c.Machine.Hetzner.Image == "" || c.Machine.Hetzner.Image == "ubuntu-24.04" {
			if c.Infrastructure.Defaults.Image != "" {
				c.Machine.Hetzner.Image = c.Infrastructure.Defaults.Image
			}
		}
	}

	// Migrate from Registry to Storage
	if c.Storage.Provider == "" || c.Storage.Provider == "local" {
		if c.Registry.Type != "" && c.Registry.Type != "local" {
			c.Storage.Provider = c.Registry.Type
		}
	}
	if c.Storage.StorageBox.Host == "" && c.Registry.StorageBoxHost != "" {
		c.Storage.StorageBox.Host = c.Registry.StorageBoxHost
	}
	if c.Storage.StorageBox.Username == "" && c.Registry.Username != "" {
		c.Storage.StorageBox.Username = c.Registry.Username
	}
	if c.Storage.StorageBox.Password == "" && c.Registry.Password != "" {
		c.Storage.StorageBox.Password = c.Registry.Password
	}

	// Also keep legacy config updated for backward compatibility
	if c.Infrastructure.Provider == "" && c.Machine.Provider != "" {
		c.Infrastructure.Provider = c.Machine.Provider
	}
	if c.Infrastructure.SSH.KeyName == "" && c.Machine.SSH.KeyName != "" {
		c.Infrastructure.SSH.KeyName = c.Machine.SSH.KeyName
	}
	if c.Infrastructure.SSH.KeyPath == "" && c.Machine.SSH.KeyPath != "" {
		c.Infrastructure.SSH.KeyPath = c.Machine.SSH.KeyPath
	}
	if !c.Infrastructure.EnableIPv4Fallback && c.Machine.IPv4.Enabled {
		c.Infrastructure.EnableIPv4Fallback = true
	}
	if c.Registry.Type == "" && c.Storage.Provider != "" {
		c.Registry.Type = c.Storage.Provider
	}
}

// GetReadinessTimeout returns the readiness timeout as a duration
func (p *ProvisioningConfig) GetReadinessTimeout() time.Duration {
	d, err := time.ParseDuration(p.ReadinessTimeout)
	if err != nil {
		return 5 * time.Minute // default
	}
	return d
}

// GetReadinessInterval returns the readiness check interval as a duration
func (p *ProvisioningConfig) GetReadinessInterval() time.Duration {
	d, err := time.ParseDuration(p.ReadinessInterval)
	if err != nil {
		return 5 * time.Second // default
	}
	return d
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	provider := c.GetMachineProvider()
	if provider == "" {
		return fmt.Errorf("machine.provider is required (or infrastructure.provider for legacy config)")
	}

	switch provider {
	case "hetzner":
		if c.Secrets.HetznerAPIToken == "" {
			return fmt.Errorf("hetzner_api_token is required (set via config or HETZNER_API_TOKEN env var)")
		}
	case "local":
		// Local provider has minimal requirements - Docker is checked at runtime
	case "none":
		// No-op provider has no requirements
	default:
		return fmt.Errorf("unsupported provider: %s (supported: hetzner, local, none)", provider)
	}

	// Validate DNS provider if specified
	if c.DNS.Provider != "" && c.DNS.Provider != "none" {
		switch c.DNS.Provider {
		case "hetzner":
			if c.Secrets.HetznerAPIToken == "" {
				return fmt.Errorf("hetzner_api_token is required for Hetzner DNS (set via config or HETZNER_API_TOKEN env var)")
			}
		case "hosts":
			// hosts provider uses /etc/hosts, no credentials needed
		default:
			return fmt.Errorf("unsupported DNS provider: %s (supported: hetzner, hosts, none)", c.DNS.Provider)
		}
	}

	return nil
}

// GetMachineProvider returns the machine provider (with legacy fallback)
func (c *Config) GetMachineProvider() string {
	if c.Machine.Provider != "" {
		return c.Machine.Provider
	}
	return c.Infrastructure.Provider
}

// GetSSHKeyName returns the SSH key name (with legacy fallback)
func (c *Config) GetSSHKeyName() string {
	if c.Machine.SSH.KeyName != "" {
		return c.Machine.SSH.KeyName
	}
	if c.Infrastructure.SSH.KeyName != "" {
		return c.Infrastructure.SSH.KeyName
	}
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.SSHKey != "" {
		return c.Infrastructure.Defaults.SSHKey
	}
	return "morpheus"
}

// GetSSHKeyPath returns the SSH key path (with legacy fallback)
func (c *Config) GetSSHKeyPath() string {
	if c.Machine.SSH.KeyPath != "" {
		return c.Machine.SSH.KeyPath
	}
	if c.Infrastructure.SSH.KeyPath != "" {
		return c.Infrastructure.SSH.KeyPath
	}
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.SSHKeyPath != "" {
		return c.Infrastructure.Defaults.SSHKeyPath
	}
	return ""
}

// GetServerType returns the server type (with legacy fallback)
func (c *Config) GetServerType() string {
	if c.Machine.Hetzner.ServerType != "" {
		return c.Machine.Hetzner.ServerType
	}
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.ServerType != "" {
		return c.Infrastructure.Defaults.ServerType
	}
	return "cx22"
}

// GetServerTypeFallback returns the fallback server types
func (c *Config) GetServerTypeFallback() []string {
	return c.Machine.Hetzner.ServerTypeFallback
}

// GetImage returns the image (with legacy fallback)
func (c *Config) GetImage() string {
	if c.Machine.Hetzner.Image != "" {
		return c.Machine.Hetzner.Image
	}
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.Image != "" {
		return c.Infrastructure.Defaults.Image
	}
	return "ubuntu-24.04"
}

// GetLocation returns the location (with legacy fallback)
func (c *Config) GetLocation() string {
	if c.Machine.Hetzner.Location != "" {
		return c.Machine.Hetzner.Location
	}
	if len(c.Infrastructure.Locations) > 0 {
		return c.Infrastructure.Locations[0]
	}
	return "fsn1"
}

// IsIPv4Enabled returns whether IPv4 is enabled
func (c *Config) IsIPv4Enabled() bool {
	return c.Machine.IPv4.Enabled || c.Infrastructure.EnableIPv4Fallback
}

// GetStorageProvider returns the storage provider
func (c *Config) GetStorageProvider() string {
	if c.Storage.Provider != "" {
		return c.Storage.Provider
	}
	if c.Registry.Type != "" {
		return c.Registry.Type
	}
	return "local"
}

// IsRemoteRegistry returns true if the registry is configured to use remote storage
func (c *Config) IsRemoteRegistry() bool {
	provider := c.GetStorageProvider()
	return provider == "storagebox"
}

// GetRegistryType returns the registry type with fallback to "local"
// DEPRECATED: Use GetStorageProvider instead
func (c *Config) GetRegistryType() string {
	return c.GetStorageProvider()
}

// GetDNSToken returns the API token for DNS operations
// Hetzner uses the same token for both Cloud and DNS APIs
func (c *Config) GetDNSToken() string {
	return c.Secrets.HetznerAPIToken
}

// IsNimsForestInstallEnabled returns whether NimsForest should be installed
// By default, NimsForest is installed unless explicitly disabled via config
func (c *Config) IsNimsForestInstallEnabled() bool {
	return c.Integration.NimsForestInstall
}

// GetNimsForestDownloadURL returns the NimsForest download URL
func (c *Config) GetNimsForestDownloadURL() string {
	if c.Integration.NimsForestDownloadURL != "" {
		return c.Integration.NimsForestDownloadURL
	}
	return DefaultNimsForestDownloadURL
}

// applyProvisioningDefaults sets default values for provisioning config
// DEPRECATED: Use applyDefaults instead
func (c *Config) applyProvisioningDefaults() {
	c.applyDefaults()
}

// applyInfrastructureDefaults sets default values for infrastructure config
// DEPRECATED: Use applyDefaults instead
func (c *Config) applyInfrastructureDefaults() {
	c.applyDefaults()
}

// applyRegistryDefaults sets default values for registry config
// DEPRECATED: Use applyDefaults instead
func (c *Config) applyRegistryDefaults() {
	c.applyDefaults()
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, ".morpheus", "config.yaml")
}

// FindConfigPath searches for an existing config file
// Returns empty string if no config file is found
func FindConfigPath() string {
	configPaths := []string{
		"./config.yaml",
		filepath.Join(os.Getenv("HOME"), ".morpheus", "config.yaml"),
		"/etc/morpheus/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	configDir := filepath.Join(homeDir, ".morpheus")
	return os.MkdirAll(configDir, 0755)
}

// SaveConfig saves the configuration to a YAML file
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write with restrictive permissions since it may contain secrets
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetConfigValue sets a specific configuration value and saves to file
// Supported keys: hetzner_api_token, storagebox_password,
// machine_provider, ssh_key_name, ipv4_enabled, dns_provider, dns_domain
func SetConfigValue(configPath, key, value string) error {
	var config *Config
	var err error

	// Try to load existing config, or create new one
	if _, statErr := os.Stat(configPath); statErr == nil {
		config, err = LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load existing config: %w", err)
		}
	} else {
		config = &Config{}
		config.applyDefaults()
	}

	// Set the value based on key
	switch key {
	case "hetzner_api_token", "hetzner-api-token":
		config.Secrets.HetznerAPIToken = strings.TrimSpace(value)
	case "storagebox_password", "storagebox-password":
		config.Storage.StorageBox.Password = strings.TrimSpace(value)
	case "machine_provider", "machine-provider":
		config.Machine.Provider = strings.TrimSpace(value)
	case "ssh_key_name", "ssh-key-name":
		config.Machine.SSH.KeyName = strings.TrimSpace(value)
	case "ssh_key_path", "ssh-key-path":
		config.Machine.SSH.KeyPath = strings.TrimSpace(value)
	case "ipv4_enabled", "ipv4-enabled":
		config.Machine.IPv4.Enabled = strings.ToLower(strings.TrimSpace(value)) == "true"
	case "dns_provider", "dns-provider":
		config.DNS.Provider = strings.TrimSpace(value)
	case "dns_domain", "dns-domain":
		config.DNS.Domain = strings.TrimSpace(value)
	case "server_type", "server-type":
		config.Machine.Hetzner.ServerType = strings.TrimSpace(value)
	case "location":
		config.Machine.Hetzner.Location = strings.TrimSpace(value)
	case "image":
		config.Machine.Hetzner.Image = strings.TrimSpace(value)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return SaveConfig(configPath, config)
}

// GetConfigValue gets a specific configuration value
// Returns the value and whether it came from environment variable
func GetConfigValue(config *Config, key string) (value string, fromEnv bool) {
	switch key {
	case "hetzner_api_token", "hetzner-api-token":
		// Check if it came from env
		if envVal := strings.TrimSpace(os.Getenv("HETZNER_API_TOKEN")); envVal != "" && envVal == config.Secrets.HetznerAPIToken {
			return config.Secrets.HetznerAPIToken, true
		}
		return config.Secrets.HetznerAPIToken, false
	case "storagebox_password", "storagebox-password":
		if envVal := strings.TrimSpace(os.Getenv("STORAGEBOX_PASSWORD")); envVal != "" && envVal == config.Storage.StorageBox.Password {
			return config.Storage.StorageBox.Password, true
		}
		return config.Storage.StorageBox.Password, false
	case "machine_provider", "machine-provider":
		return config.GetMachineProvider(), false
	case "ssh_key_name", "ssh-key-name":
		return config.GetSSHKeyName(), false
	case "ssh_key_path", "ssh-key-path":
		return config.GetSSHKeyPath(), false
	case "ipv4_enabled", "ipv4-enabled":
		return fmt.Sprintf("%v", config.IsIPv4Enabled()), false
	case "dns_provider", "dns-provider":
		return config.DNS.Provider, false
	case "dns_domain", "dns-domain":
		return config.DNS.Domain, false
	case "server_type", "server-type":
		return config.GetServerType(), false
	case "location":
		return config.GetLocation(), false
	case "image":
		return config.GetImage(), false
	default:
		return "", false
	}
}

// MaskToken masks a token for display (shows first 4 and last 4 chars)
func MaskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) > 8 {
		return token[:4] + "..." + token[len(token)-4:]
	}
	return "****"
}

// ListConfigKeys returns all supported config keys
func ListConfigKeys() []string {
	return []string{
		"hetzner_api_token",
		"storagebox_password",
		"machine_provider",
		"ssh_key_name",
		"ssh_key_path",
		"ipv4_enabled",
		"dns_provider",
		"dns_domain",
		"server_type",
		"location",
		"image",
	}
}
