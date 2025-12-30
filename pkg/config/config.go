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

// DefaultServerConfig defines default server settings
type DefaultServerConfig struct {
	ServerType string `yaml:"server_type"`
	Image      string `yaml:"image"`
	SSHKey     string `yaml:"ssh_key"`      // Name of the SSH key in Hetzner Cloud
	SSHKeyPath string `yaml:"ssh_key_path"` // Optional: Path to local SSH public key file for auto-upload
}

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

	// Apply provisioning defaults
	config.applyProvisioningDefaults()

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

	if c.Infrastructure.Provider == "hetzner" {
		if c.Secrets.HetznerAPIToken == "" {
			return fmt.Errorf("hetzner_api_token is required (set via config or HETZNER_API_TOKEN env var)")
		}
		if c.Infrastructure.Defaults.ServerType == "" {
			return fmt.Errorf("infrastructure.defaults.server_type is required")
		}
		if c.Infrastructure.Defaults.Image == "" {
			return fmt.Errorf("infrastructure.defaults.image is required")
		}
	}

	return nil
}
