package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Morpheus configuration
type Config struct {
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`
	Integration    IntegrationConfig    `yaml:"integration"`
	Provisioning   ProvisioningConfig   `yaml:"provisioning"`
	Registry       RegistryConfig       `yaml:"registry"`
	Secrets        SecretsConfig        `yaml:"secrets"`
}

// RegistryConfig defines remote registry settings for multi-device access
type RegistryConfig struct {
	Type           string `yaml:"type"`            // "storagebox", "s3", "local", or "none"
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
type InfrastructureConfig struct {
	Provider string    `yaml:"provider"`
	SSH      SSHConfig `yaml:"ssh"`

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

	// NimsForest auto-installation settings
	NimsForestInstall     bool   `yaml:"nimsforest_install"`      // Auto-install NimsForest on provisioned machines
	NimsForestDownloadURL string `yaml:"nimsforest_download_url"` // URL to download binary (e.g., https://nimsforest.io/bin/nimsforest)

	// DEPRECATED: NimsForest now embeds NATS - these settings are ignored
	// Kept for backward compatibility with existing configs
	NATSInstall bool   `yaml:"nats_install"` // DEPRECATED: NimsForest embeds NATS now
	NATSVersion string `yaml:"nats_version"` // DEPRECATED: NimsForest embeds NATS now
}

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

	// Expand environment variables in registry password
	if strings.HasPrefix(config.Registry.Password, "${") && strings.HasSuffix(config.Registry.Password, "}") {
		envVar := config.Registry.Password[2 : len(config.Registry.Password)-1]
		config.Registry.Password = strings.TrimSpace(os.Getenv(envVar))
	}
	// Also check for STORAGEBOX_PASSWORD env var as override
	if pass := strings.TrimSpace(os.Getenv("STORAGEBOX_PASSWORD")); pass != "" {
		config.Registry.Password = pass
	}

	// Apply defaults
	config.applyProvisioningDefaults()
	config.applyInfrastructureDefaults()
	config.applyRegistryDefaults()

	return &config, nil
}

// applyProvisioningDefaults sets default values for provisioning config
func (c *Config) applyProvisioningDefaults() {
	if c.Provisioning.ReadinessTimeout == "" {
		c.Provisioning.ReadinessTimeout = "5m"
	}
	if c.Provisioning.ReadinessInterval == "" {
		// Check every 5 seconds for faster detection of SSH readiness
		c.Provisioning.ReadinessInterval = "5s"
	}
	if c.Provisioning.SSHPort == 0 {
		c.Provisioning.SSHPort = 22
	}
}

// applyInfrastructureDefaults sets default values for infrastructure config
func (c *Config) applyInfrastructureDefaults() {
	// IPv6-only by default (IPv4 costs extra on Hetzner)
	// No configuration needed - always uses IPv6

	// Migrate from legacy config format
	if c.Infrastructure.Defaults != nil {
		// Migrate SSH config
		if c.Infrastructure.SSH.KeyName == "" && c.Infrastructure.Defaults.SSHKey != "" {
			c.Infrastructure.SSH.KeyName = c.Infrastructure.Defaults.SSHKey
		}
		if c.Infrastructure.SSH.KeyPath == "" && c.Infrastructure.Defaults.SSHKeyPath != "" {
			c.Infrastructure.SSH.KeyPath = c.Infrastructure.Defaults.SSHKeyPath
		}
	}

	// Set default SSH key name if not provided
	if c.Infrastructure.SSH.KeyName == "" {
		c.Infrastructure.SSH.KeyName = "morpheus"
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
	if c.Infrastructure.Provider == "" {
		return fmt.Errorf("infrastructure.provider is required")
	}

	switch c.Infrastructure.Provider {
	case "hetzner":
		if c.Secrets.HetznerAPIToken == "" {
			return fmt.Errorf("hetzner_api_token is required (set via config or HETZNER_API_TOKEN env var)")
		}
		// SSH key name has a default, so no need to validate
	case "local":
		// Local provider has minimal requirements - Docker is checked at runtime
		// No API token or specific server type required
	default:
		return fmt.Errorf("unsupported provider: %s (supported: hetzner, local)", c.Infrastructure.Provider)
	}

	return nil
}

// GetSSHKeyName returns the SSH key name (with fallback to legacy config)
func (c *Config) GetSSHKeyName() string {
	if c.Infrastructure.SSH.KeyName != "" {
		return c.Infrastructure.SSH.KeyName
	}
	// Fallback to legacy config
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.SSHKey != "" {
		return c.Infrastructure.Defaults.SSHKey
	}
	return "morpheus"
}

// GetSSHKeyPath returns the SSH key path (with fallback to legacy config)
func (c *Config) GetSSHKeyPath() string {
	if c.Infrastructure.SSH.KeyPath != "" {
		return c.Infrastructure.SSH.KeyPath
	}
	// Fallback to legacy config
	if c.Infrastructure.Defaults != nil && c.Infrastructure.Defaults.SSHKeyPath != "" {
		return c.Infrastructure.Defaults.SSHKeyPath
	}
	return ""
}

// applyRegistryDefaults sets default values for registry config
func (c *Config) applyRegistryDefaults() {
	// Default to local registry if not specified
	if c.Registry.Type == "" {
		c.Registry.Type = "local"
	}
}

// IsRemoteRegistry returns true if the registry is configured to use remote storage
func (c *Config) IsRemoteRegistry() bool {
	return c.Registry.Type == "storagebox" || c.Registry.Type == "s3"
}

// GetRegistryType returns the registry type with fallback to "local"
func (c *Config) GetRegistryType() string {
	if c.Registry.Type == "" {
		return "local"
	}
	return c.Registry.Type
}
