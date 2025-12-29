package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1
    - nbg1

integration:
  nimsforest_url: "https://nimsforest.example.com"
  registry_url: "https://registry.example.com"

secrets:
  hetzner_api_token: test-token
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Infrastructure.Provider != "hetzner" {
		t.Errorf("Expected provider 'hetzner', got '%s'", cfg.Infrastructure.Provider)
	}

	if cfg.Infrastructure.Defaults.ServerType != "cpx31" {
		t.Errorf("Expected server_type 'cpx31', got '%s'", cfg.Infrastructure.Defaults.ServerType)
	}

	if len(cfg.Infrastructure.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(cfg.Infrastructure.Locations))
	}

	if cfg.Secrets.HetznerAPIToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", cfg.Secrets.HetznerAPIToken)
	}

	if cfg.Integration.NimsForestURL != "https://nimsforest.example.com" {
		t.Errorf("Expected nimsforest_url 'https://nimsforest.example.com', got '%s'", cfg.Integration.NimsForestURL)
	}

	if cfg.Integration.RegistryURL != "https://registry.example.com" {
		t.Errorf("Expected registry_url 'https://registry.example.com', got '%s'", cfg.Integration.RegistryURL)
	}
}

func TestLoadConfigWithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1

secrets:
  hetzner_api_token: ""
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable
	os.Setenv("HETZNER_API_TOKEN", "env-token")
	defer os.Unsetenv("HETZNER_API_TOKEN")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Secrets.HetznerAPIToken != "env-token" {
		t.Errorf("Expected token from env 'env-token', got '%s'", cfg.Secrets.HetznerAPIToken)
	}
}

func TestLoadConfigTrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test token with surrounding whitespace in config file
	configContent := `
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1

secrets:
  hetzner_api_token: "  token-with-spaces  "
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Secrets.HetznerAPIToken != "token-with-spaces" {
		t.Errorf("Expected trimmed token 'token-with-spaces', got '%s'", cfg.Secrets.HetznerAPIToken)
	}
}

func TestLoadConfigTrimsWhitespaceFromEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1

secrets:
  hetzner_api_token: ""
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable with whitespace and newline
	os.Setenv("HETZNER_API_TOKEN", "  env-token-with-whitespace\n")
	defer os.Unsetenv("HETZNER_API_TOKEN")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Secrets.HetznerAPIToken != "env-token-with-whitespace" {
		t.Errorf("Expected trimmed token 'env-token-with-whitespace', got '%s'", cfg.Secrets.HetznerAPIToken)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
infrastructure:
  provider: hetzner
  invalid: [unclosed
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid hetzner config",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "hetzner",
					Defaults: DefaultServerConfig{
						ServerType: "cpx31",
						Image:      "ubuntu-24.04",
						SSHKey:     "main",
					},
				},
				Secrets: SecretsConfig{
					HetznerAPIToken: "token",
				},
			},
			expectErr: false,
		},
		{
			name: "missing provider",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "",
				},
			},
			expectErr: true,
		},
		{
			name: "missing api token",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "hetzner",
					Defaults: DefaultServerConfig{
						ServerType: "cpx31",
						Image:      "ubuntu-24.04",
					},
				},
				Secrets: SecretsConfig{
					HetznerAPIToken: "",
				},
			},
			expectErr: true,
		},
		{
			name: "missing server type",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "hetzner",
					Defaults: DefaultServerConfig{
						ServerType: "",
						Image:      "ubuntu-24.04",
					},
				},
				Secrets: SecretsConfig{
					HetznerAPIToken: "token",
				},
			},
			expectErr: true,
		},
		{
			name: "missing image",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "hetzner",
					Defaults: DefaultServerConfig{
						ServerType: "cpx31",
						Image:      "",
					},
				},
				Secrets: SecretsConfig{
					HetznerAPIToken: "token",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
