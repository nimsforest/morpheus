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

	if cfg.Provisioning.ReadinessInterval != "5s" {
		t.Errorf("Expected default readiness_interval '5s', got '%s'", cfg.Provisioning.ReadinessInterval)
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
			interval:         "5s",
			expectedTimeout:  5 * time.Minute,
			expectedInterval: 5 * time.Second,
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
			expectedInterval: 5 * time.Second, // hardcoded fallback in GetReadinessInterval
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

func TestRegistryConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without registry section - should get defaults
	configContent := `
infrastructure:
  provider: hetzner

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

	// Check default registry type
	if cfg.Registry.Type != "local" {
		t.Errorf("Expected default registry type 'local', got '%s'", cfg.Registry.Type)
	}

	if cfg.GetRegistryType() != "local" {
		t.Errorf("Expected GetRegistryType() 'local', got '%s'", cfg.GetRegistryType())
	}

	if cfg.IsRemoteRegistry() {
		t.Error("Expected IsRemoteRegistry() false for local registry")
	}
}

func TestRegistryConfigStorageBox(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner

registry:
  type: storagebox
  url: "https://u12345.your-storagebox.de/morpheus/registry.json"
  username: "u12345"
  password: "mypassword"

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

	if cfg.Registry.Type != "storagebox" {
		t.Errorf("Expected registry type 'storagebox', got '%s'", cfg.Registry.Type)
	}

	if cfg.Registry.URL != "https://u12345.your-storagebox.de/morpheus/registry.json" {
		t.Errorf("Unexpected registry URL: %s", cfg.Registry.URL)
	}

	if cfg.Registry.Username != "u12345" {
		t.Errorf("Expected username 'u12345', got '%s'", cfg.Registry.Username)
	}

	if cfg.Registry.Password != "mypassword" {
		t.Errorf("Expected password 'mypassword', got '%s'", cfg.Registry.Password)
	}

	if !cfg.IsRemoteRegistry() {
		t.Error("Expected IsRemoteRegistry() true for storagebox registry")
	}
}

func TestRegistryConfigPasswordEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner

registry:
  type: storagebox
  url: "https://u12345.your-storagebox.de/morpheus/registry.json"
  username: "u12345"
  password: "${MY_STORAGEBOX_PASS}"

secrets:
  hetzner_api_token: test-token
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable
	os.Setenv("MY_STORAGEBOX_PASS", "env-password")
	defer os.Unsetenv("MY_STORAGEBOX_PASS")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Registry.Password != "env-password" {
		t.Errorf("Expected password from env 'env-password', got '%s'", cfg.Registry.Password)
	}
}

func TestRegistryConfigStorageBoxPasswordEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
infrastructure:
  provider: hetzner

registry:
  type: storagebox
  url: "https://u12345.your-storagebox.de/morpheus/registry.json"
  username: "u12345"
  password: "config-password"

secrets:
  hetzner_api_token: test-token
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set STORAGEBOX_PASSWORD env var - should override config value
	os.Setenv("STORAGEBOX_PASSWORD", "override-password")
	defer os.Unsetenv("STORAGEBOX_PASSWORD")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Registry.Password != "override-password" {
		t.Errorf("Expected password from STORAGEBOX_PASSWORD env 'override-password', got '%s'", cfg.Registry.Password)
	}
}

func TestIsRemoteRegistry(t *testing.T) {
	tests := []struct {
		name     string
		regType  string
		expected bool
	}{
		{"storagebox", "storagebox", true},
		{"local", "local", false},
		{"none", "none", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Registry: RegistryConfig{
					Type: tt.regType,
				},
			}
			if got := cfg.IsRemoteRegistry(); got != tt.expected {
				t.Errorf("IsRemoteRegistry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIPv4FallbackConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test with IPv4 fallback enabled
	configContent := `
infrastructure:
  provider: hetzner
  enable_ipv4_fallback: true

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

	if !cfg.Infrastructure.EnableIPv4Fallback {
		t.Error("Expected EnableIPv4Fallback to be true")
	}
}

func TestIPv4FallbackConfigDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test without IPv4 fallback specified (should default to false)
	configContent := `
infrastructure:
  provider: hetzner

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

	if cfg.Infrastructure.EnableIPv4Fallback {
		t.Error("Expected EnableIPv4Fallback to default to false")
	}
}

func TestSetConfigValue(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		checkFunc func(*Config) bool
	}{
		{
			name:  "set hetzner_api_token",
			key:   "hetzner_api_token",
			value: "test-token-123",
			checkFunc: func(c *Config) bool {
				return c.Secrets.HetznerAPIToken == "test-token-123"
			},
		},
		{
			name:  "set hetzner-api-token (with hyphens)",
			key:   "hetzner-api-token",
			value: "test-token-456",
			checkFunc: func(c *Config) bool {
				return c.Secrets.HetznerAPIToken == "test-token-456"
			},
		},
		{
			name:  "set machine_provider",
			key:   "machine_provider",
			value: "hetzner",
			checkFunc: func(c *Config) bool {
				return c.Machine.Provider == "hetzner"
			},
		},
		{
			name:  "set ipv4_enabled true",
			key:   "ipv4_enabled",
			value: "true",
			checkFunc: func(c *Config) bool {
				return c.Machine.IPv4.Enabled == true
			},
		},
		{
			name:  "set ipv4_enabled false",
			key:   "ipv4_enabled",
			value: "false",
			checkFunc: func(c *Config) bool {
				return c.Machine.IPv4.Enabled == false
			},
		},
		{
			name:  "set server_type",
			key:   "server_type",
			value: "cx22",
			checkFunc: func(c *Config) bool {
				return c.Machine.Hetzner.ServerType == "cx22"
			},
		},
		{
			name:  "set location",
			key:   "location",
			value: "nbg1",
			checkFunc: func(c *Config) bool {
				return c.Machine.Hetzner.Location == "nbg1"
			},
		},
		{
			name:  "set dns_provider",
			key:   "dns_provider",
			value: "hetzner",
			checkFunc: func(c *Config) bool {
				return c.DNS.Provider == "hetzner"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Set the value
			err := SetConfigValue(configPath, tt.key, tt.value)
			if err != nil {
				t.Fatalf("SetConfigValue failed: %v", err)
			}

			// Load and verify
			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if !tt.checkFunc(cfg) {
				t.Errorf("Config value not set correctly for key %s", tt.key)
			}
		})
	}
}

func TestSetConfigValueUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := SetConfigValue(configPath, "unknown_key", "value")
	if err == nil {
		t.Error("Expected error for unknown config key")
	}
}

func TestSetConfigValueUpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	initialConfig := `
machine:
  provider: hetzner
  hetzner:
    server_type: cx11
    location: fsn1

secrets:
  hetzner_api_token: old-token
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Update the token
	err := SetConfigValue(configPath, "hetzner_api_token", "new-token")
	if err != nil {
		t.Fatalf("SetConfigValue failed: %v", err)
	}

	// Load and verify
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check new value is set
	if cfg.Secrets.HetznerAPIToken != "new-token" {
		t.Errorf("Expected new-token, got %s", cfg.Secrets.HetznerAPIToken)
	}

	// Check that other values are preserved
	if cfg.Machine.Provider != "hetzner" {
		t.Errorf("Expected provider 'hetzner' to be preserved, got %s", cfg.Machine.Provider)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Machine: MachineConfig{
			Provider: "hetzner",
			Hetzner: HetznerConfig{
				ServerType: "cx22",
				Location:   "fsn1",
				Image:      "ubuntu-24.04",
			},
		},
		Secrets: SecretsConfig{
			HetznerAPIToken: "my-secret-token",
		},
	}

	err := SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Check permissions (should be 0600 for security)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %v", info.Mode().Perm())
	}

	// Load and verify contents
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Machine.Provider != "hetzner" {
		t.Errorf("Expected provider 'hetzner', got %s", loadedCfg.Machine.Provider)
	}

	if loadedCfg.Secrets.HetznerAPIToken != "my-secret-token" {
		t.Errorf("Expected token 'my-secret-token', got %s", loadedCfg.Secrets.HetznerAPIToken)
	}
}

func TestGetConfigValue(t *testing.T) {
	cfg := &Config{
		Machine: MachineConfig{
			Provider: "hetzner",
			Hetzner: HetznerConfig{
				ServerType: "cx22",
				Location:   "fsn1",
			},
			IPv4: IPv4Config{
				Enabled: true,
			},
		},
		Secrets: SecretsConfig{
			HetznerAPIToken: "test-token",
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"hetzner_api_token", "test-token"},
		{"machine_provider", "hetzner"},
		{"server_type", "cx22"},
		{"location", "fsn1"},
		{"ipv4_enabled", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, _ := GetConfigValue(cfg, tt.key)
			if value != tt.expected {
				t.Errorf("GetConfigValue(%s) = %s, want %s", tt.key, value, tt.expected)
			}
		})
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "(not set)"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"abcdefghijklmnop", "abcd...mnop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MaskToken(tt.input)
			if result != tt.expected {
				t.Errorf("MaskToken(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindConfigPath(t *testing.T) {
	// This test verifies the function doesn't crash
	// The actual path depends on the environment
	path := FindConfigPath()
	// Just check it returns something (could be empty if no config exists)
	_ = path
}

func TestGetDefaultConfigPath(t *testing.T) {
	path := GetDefaultConfigPath()
	if path == "" {
		t.Error("GetDefaultConfigPath() returned empty string")
	}
}
