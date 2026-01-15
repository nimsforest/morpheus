package customer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCustomerConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "customers.yaml")

	configContent := `customers:
  - id: acme
    name: ACME Corp
    domain: acme.example.com
    ventures:
      - retail
      - wholesale
    hetzner:
      project_id: proj-123
      api_token: secret-token
  - id: globex
    name: Globex Corporation
    domain: globex.example.com
    ventures:
      - manufacturing
    hetzner:
      project_id: proj-456
      api_token: ${GLOBEX_API_TOKEN}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadCustomerConfig(configPath)
	if err != nil {
		t.Fatalf("LoadCustomerConfig failed: %v", err)
	}

	if len(cfg.Customers) != 2 {
		t.Errorf("expected 2 customers, got %d", len(cfg.Customers))
	}

	// Check first customer
	if cfg.Customers[0].ID != "acme" {
		t.Errorf("expected customer ID 'acme', got %q", cfg.Customers[0].ID)
	}
	if cfg.Customers[0].Name != "ACME Corp" {
		t.Errorf("expected customer name 'ACME Corp', got %q", cfg.Customers[0].Name)
	}
	if cfg.Customers[0].Domain != "acme.example.com" {
		t.Errorf("expected domain 'acme.example.com', got %q", cfg.Customers[0].Domain)
	}
	if len(cfg.Customers[0].Ventures) != 2 {
		t.Errorf("expected 2 ventures for acme, got %d", len(cfg.Customers[0].Ventures))
	}
	if cfg.Customers[0].Hetzner.ProjectID != "proj-123" {
		t.Errorf("expected project_id 'proj-123', got %q", cfg.Customers[0].Hetzner.ProjectID)
	}
	if cfg.Customers[0].Hetzner.APIToken != "secret-token" {
		t.Errorf("expected api_token 'secret-token', got %q", cfg.Customers[0].Hetzner.APIToken)
	}
}

func TestLoadCustomerConfig_FileNotFound(t *testing.T) {
	_, err := LoadCustomerConfig("/nonexistent/path/customers.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadCustomerConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `customers:
  - id: [invalid yaml structure
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadCustomerConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestGetCustomer(t *testing.T) {
	cfg := &CustomerConfig{
		Customers: []Customer{
			{ID: "acme", Name: "ACME Corp", Domain: "acme.com"},
			{ID: "globex", Name: "Globex Corp", Domain: "globex.com"},
		},
	}

	// Test finding existing customer
	cust, err := GetCustomer(cfg, "acme")
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}
	if cust.ID != "acme" {
		t.Errorf("expected ID 'acme', got %q", cust.ID)
	}
	if cust.Name != "ACME Corp" {
		t.Errorf("expected name 'ACME Corp', got %q", cust.Name)
	}

	// Test finding another customer
	cust, err = GetCustomer(cfg, "globex")
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}
	if cust.ID != "globex" {
		t.Errorf("expected ID 'globex', got %q", cust.ID)
	}
}

func TestGetCustomer_NotFound(t *testing.T) {
	cfg := &CustomerConfig{
		Customers: []Customer{
			{ID: "acme", Name: "ACME Corp", Domain: "acme.com"},
		},
	}

	_, err := GetCustomer(cfg, "unknown")
	if err == nil {
		t.Error("expected error for unknown customer, got nil")
	}
}

func TestGetCustomer_NilConfig(t *testing.T) {
	_, err := GetCustomer(nil, "acme")
	if err == nil {
		t.Error("expected error for nil config, got nil")
	}
}

func TestGetCustomer_EmptyCustomers(t *testing.T) {
	cfg := &CustomerConfig{
		Customers: []Customer{},
	}

	_, err := GetCustomer(cfg, "acme")
	if err == nil {
		t.Error("expected error for empty customers, got nil")
	}
}

func TestResolveToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "plain token",
			token:    "my-secret-token",
			expected: "my-secret-token",
		},
		{
			name:     "token with whitespace",
			token:    "  my-secret-token  ",
			expected: "my-secret-token",
		},
		{
			name:     "env var reference",
			token:    "${TEST_DNS_TOKEN}",
			envKey:   "TEST_DNS_TOKEN",
			envValue: "env-token-value",
			expected: "env-token-value",
		},
		{
			name:     "env var with whitespace in value",
			token:    "${TEST_TOKEN_WHITESPACE}",
			envKey:   "TEST_TOKEN_WHITESPACE",
			envValue: "  token-with-spaces  ",
			expected: "token-with-spaces",
		},
		{
			name:     "env var not set",
			token:    "${UNSET_ENV_VAR}",
			expected: "",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "partial env var syntax - prefix only",
			token:    "${PARTIAL",
			expected: "${PARTIAL",
		},
		{
			name:     "partial env var syntax - suffix only",
			token:    "PARTIAL}",
			expected: "PARTIAL}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variable if needed
			if tt.envKey != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := ResolveToken(tt.token)
			if result != tt.expected {
				t.Errorf("ResolveToken(%q) = %q, expected %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Test with HOME set
	os.Setenv("HOME", "/home/testuser")
	path := GetDefaultConfigPath()
	expected := "/home/testuser/.morpheus/customers.yaml"
	if path != expected {
		t.Errorf("GetDefaultConfigPath() = %q, expected %q", path, expected)
	}

	// Test with HOME unset
	os.Unsetenv("HOME")
	path = GetDefaultConfigPath()
	expected = "/tmp/.morpheus/customers.yaml"
	if path != expected {
		t.Errorf("GetDefaultConfigPath() with no HOME = %q, expected %q", path, expected)
	}
}

func TestListCustomers(t *testing.T) {
	cfg := &CustomerConfig{
		Customers: []Customer{
			{ID: "acme"},
			{ID: "globex"},
			{ID: "initech"},
		},
	}

	ids := ListCustomers(cfg)
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	if ids[0] != "acme" {
		t.Errorf("expected first ID 'acme', got %q", ids[0])
	}
	if ids[1] != "globex" {
		t.Errorf("expected second ID 'globex', got %q", ids[1])
	}
	if ids[2] != "initech" {
		t.Errorf("expected third ID 'initech', got %q", ids[2])
	}
}

func TestListCustomers_Nil(t *testing.T) {
	ids := ListCustomers(nil)
	if ids != nil {
		t.Errorf("expected nil for nil config, got %v", ids)
	}
}

func TestValidateCustomer(t *testing.T) {
	tests := []struct {
		name        string
		customer    *Customer
		expectError bool
	}{
		{
			name:        "nil customer",
			customer:    nil,
			expectError: true,
		},
		{
			name:        "empty ID",
			customer:    &Customer{ID: "", Domain: "example.com"},
			expectError: true,
		},
		{
			name:        "empty domain",
			customer:    &Customer{ID: "acme", Domain: ""},
			expectError: true,
		},
		{
			name:        "valid customer",
			customer:    &Customer{ID: "acme", Domain: "example.com"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomer(tt.customer)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateCustomerConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *CustomerConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid config",
			config: &CustomerConfig{
				Customers: []Customer{
					{ID: "acme", Domain: "acme.com"},
					{ID: "globex", Domain: "globex.com"},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate IDs",
			config: &CustomerConfig{
				Customers: []Customer{
					{ID: "acme", Domain: "acme.com"},
					{ID: "acme", Domain: "acme2.com"},
				},
			},
			expectError: true,
		},
		{
			name: "invalid customer in list",
			config: &CustomerConfig{
				Customers: []Customer{
					{ID: "acme", Domain: "acme.com"},
					{ID: "", Domain: "noname.com"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomerConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
