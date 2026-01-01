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
	Secrets        SecretsConfig        `yaml:"secrets"`
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
	Provider  string              `yaml:"provider"`
	Defaults  DefaultServerConfig `yaml:"defaults"`
	Locations []string            `yaml:"locations"`
}

// IntegrationConfig defines integration with NimsForest
type IntegrationConfig struct {
	NimsForestURL string `yaml:"nimsforest_url"` // URL for NimsForest bootstrap callbacks
	RegistryURL   string `yaml:"registry_url"`   // Optional: Morpheus registry URL
}

// DefaultsConfig defines default server settings
type DefaultsConfig struct {
	ServerType string `yaml:"server_type"`
	Image      string `yaml:"image"`
	SSHKey     string `yaml:"ssh_key"`      // Name of the SSH key in Hetzner Cloud
	SSHKeyPath string `yaml:"ssh_key_path"` // Optional: Path to local SSH public key file for auto-upload
	PreferIPv6 bool   `yaml:"prefer_ipv6"`  // Use IPv6 instead of IPv4 for connections (default: true)
	IPv6Only   bool   `yaml:"ipv6_only"`    // Strict IPv6-only mode: fail if no IPv6 (default: false)
}

// DefaultServerConfig is an alias for backward compatibility
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

	// Apply defaults
	config.applyProvisioningDefaults()
	config.applyInfrastructureDefaults()

	return &config, nil
}

// applyProvisioningDefaults sets default values for provisioning config
func (c *Config) applyProvisioningDefaults() {
	if c.Provisioning.ReadinessTimeout == "" {
		c.Provisioning.ReadinessTimeout = "5m"
	}
	if c.Provisioning.ReadinessInterval == "" {
		c.Provisioning.ReadinessInterval = "10s"
	}
	if c.Provisioning.SSHPort == 0 {
		c.Provisioning.SSHPort = 22
	}
}

// applyInfrastructureDefaults sets default values for infrastructure config
func (c *Config) applyInfrastructureDefaults() {
	// Default to IPv6-first (with IPv4 fallback)
	// This is only applied if the config file doesn't explicitly set prefer_ipv6
	// Note: YAML unmarshal will set false for boolean fields not present in config,
	// so we can't distinguish between "not set" and "explicitly false"
	// This default is mainly for in-code config creation
	if c.Infrastructure.Defaults.Image == "" {
		c.Infrastructure.Defaults.PreferIPv6 = true
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
		return 10 * time.Second // default
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
		if c.Infrastructure.Defaults.ServerType == "" {
			return fmt.Errorf("infrastructure.defaults.server_type is required")
		}
		if c.Infrastructure.Defaults.Image == "" {
			return fmt.Errorf("infrastructure.defaults.image is required")
		}
	case "local":
		// Local provider has minimal requirements - Docker is checked at runtime
		// No API token or specific server type required
	default:
		return fmt.Errorf("unsupported provider: %s (supported: hetzner, local)", c.Infrastructure.Provider)
	}

	return nil
}
