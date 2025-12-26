package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the Morpheus configuration
type Config struct {
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`
	Secrets        SecretsConfig        `yaml:"secrets"`
}

// InfrastructureConfig defines infrastructure provider settings
type InfrastructureConfig struct {
	Provider  string                 `yaml:"provider"`
	Defaults  DefaultServerConfig    `yaml:"defaults"`
	Locations []string               `yaml:"locations"`
}

// DefaultServerConfig defines default server settings
type DefaultServerConfig struct {
	ServerType string `yaml:"server_type"`
	Image      string `yaml:"image"`
	SSHKey     string `yaml:"ssh_key"`
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

	// Override with environment variables if set
	if token := os.Getenv("HETZNER_API_TOKEN"); token != "" {
		config.Secrets.HetznerAPIToken = token
	}

	return &config, nil
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
