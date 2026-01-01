package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
					SSH: SSHConfig{
						KeyName: "main",
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
					SSH: SSHConfig{
						KeyName: "main",
					},
				},
				Secrets: SecretsConfig{
					HetznerAPIToken: "",
				},
			},
			expectErr: true,
		},
		{
			name: "valid config with legacy defaults",
			config: Config{
				Infrastructure: InfrastructureConfig{
					Provider: "hetzner",
					Defaults: &DefaultServerConfig{
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

func TestProvisioningConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without provisioning section - should get defaults
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
  hetzner_api_token: test-token
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check defaults are applied
	if cfg.Provisioning.ReadinessTimeout != "5m" {
		t.Errorf("Expected default readiness_timeout '5m', got '%s'", cfg.Provisioning.ReadinessTimeout)
	}

	if cfg.Provisioning.ReadinessInterval != "10s" {
		t.Errorf("Expected default readiness_interval '10s', got '%s'", cfg.Provisioning.ReadinessInterval)
	}

	if cfg.Provisioning.SSHPort != 22 {
		t.Errorf("Expected default ssh_port 22, got %d", cfg.Provisioning.SSHPort)
	}
}

func TestProvisioningConfigCustomValues(t *testing.T) {
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

provisioning:
  readiness_timeout: "10m"
  readiness_interval: "30s"
  ssh_port: 2222

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

	if cfg.Provisioning.ReadinessTimeout != "10m" {
		t.Errorf("Expected readiness_timeout '10m', got '%s'", cfg.Provisioning.ReadinessTimeout)
	}

	if cfg.Provisioning.ReadinessInterval != "30s" {
		t.Errorf("Expected readiness_interval '30s', got '%s'", cfg.Provisioning.ReadinessInterval)
	}

	if cfg.Provisioning.SSHPort != 2222 {
		t.Errorf("Expected ssh_port 2222, got %d", cfg.Provisioning.SSHPort)
	}
}

func TestProvisioningConfigGetDurations(t *testing.T) {
	tests := []struct {
		name             string
		timeout          string
		interval         string
		expectedTimeout  time.Duration
		expectedInterval time.Duration
	}{
		{
			name:             "valid durations",
			timeout:          "5m",
			interval:         "10s",
			expectedTimeout:  5 * time.Minute,
			expectedInterval: 10 * time.Second,
		},
		{
			name:             "different valid durations",
			timeout:          "15m30s",
			interval:         "1m",
			expectedTimeout:  15*time.Minute + 30*time.Second,
			expectedInterval: 1 * time.Minute,
		},
		{
			name:             "invalid timeout falls back to default",
			timeout:          "invalid",
			interval:         "10s",
			expectedTimeout:  5 * time.Minute, // default
			expectedInterval: 10 * time.Second,
		},
		{
			name:             "invalid interval falls back to default",
			timeout:          "5m",
			interval:         "not-a-duration",
			expectedTimeout:  5 * time.Minute,
			expectedInterval: 10 * time.Second, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := ProvisioningConfig{
				ReadinessTimeout:  tt.timeout,
				ReadinessInterval: tt.interval,
			}

			if got := pc.GetReadinessTimeout(); got != tt.expectedTimeout {
				t.Errorf("GetReadinessTimeout() = %v, want %v", got, tt.expectedTimeout)
			}

			if got := pc.GetReadinessInterval(); got != tt.expectedInterval {
				t.Errorf("GetReadinessInterval() = %v, want %v", got, tt.expectedInterval)
			}
		})
	}
}
