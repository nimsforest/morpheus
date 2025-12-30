package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the Morpheus configuration
type Config struct {
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`
	Integration    IntegrationConfig    `yaml:"integration"`
	Secrets        SecretsConfig        `yaml:"secrets"`
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

	return &config, nil
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
