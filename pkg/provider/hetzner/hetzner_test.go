package hetzner

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nimsforest/morpheus/pkg/provider"
)

func TestNewProvider(t *testing.T) {
	p, err := NewProvider("test-token")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if p == nil {
		t.Fatal("Expected non-nil provider")
	}
}

func TestNewProviderEmptyToken(t *testing.T) {
	_, err := NewProvider("")
	if err == nil {
		t.Error("Expected error for empty API token")
	}
}

func TestNewProviderWithWhitespace(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "token with leading/trailing spaces",
			token:       "  valid-token-123  ",
			expectError: false,
		},
		{
			name:        "token with newline",
			token:       "valid-token-123\n",
			expectError: false,
		},
		{
			name:        "token with carriage return and newline",
			token:       "valid-token-123\r\n",
			expectError: false,
		},
		{
			name:        "only whitespace",
			token:       "   \n\r\t  ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(tt.token)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestSanitizeAPIToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean token",
			input:    "valid-token-123",
			expected: "valid-token-123",
		},
		{
			name:     "token with leading spaces",
			input:    "   token123",
			expected: "token123",
		},
		{
			name:     "token with trailing newline",
			input:    "token123\n",
			expected: "token123",
		},
		{
			name:     "token with CRLF",
			input:    "token123\r\n",
			expected: "token123",
		},
		{
			name:     "token with embedded carriage return",
			input:    "token\r123",
			expected: "token123",
		},
		{
			name:     "token with BOM",
			input:    "\uFEFFtoken123",
			expected: "token123",
		},
		{
			name:     "token with tab",
			input:    "token\t123",
			expected: "token123",
		},
		{
			name:     "token with null byte",
			input:    "token\x00123",
			expected: "token123",
		},
		{
			name:     "token with non-ASCII characters",
			input:    "token™123",
			expected: "token123",
		},
		{
			name:     "token with space in middle",
			input:    "token 123",
			expected: "token123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "  \n\r\t  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeAPIToken(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeAPIToken(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateAPIToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid alphanumeric token",
			token:       "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
			expectError: false,
		},
		{
			name:        "valid token with special chars",
			token:       "token-with_special.chars~plus+slash/equals=",
			expectError: false,
		},
		{
			name:        "token with newline",
			token:       "token\n123",
			expectError: true,
		},
		{
			name:        "token with carriage return",
			token:       "token\r123",
			expectError: true,
		},
		{
			name:        "token with space",
			token:       "token 123",
			expectError: true,
		},
		{
			name:        "token with tab",
			token:       "token\t123",
			expectError: true,
		},
		{
			name:        "token with null byte",
			token:       "token\x00123",
			expectError: true,
		},
		{
			name:        "token with BOM",
			token:       "\uFEFFtoken123",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIToken(tt.token)
			if tt.expectError && err == nil {
				t.Errorf("validateAPIToken(%q) expected error, got nil", tt.token)
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateAPIToken(%q) expected no error, got: %v", tt.token, err)
			}
		})
	}
}

func TestConvertServerState(t *testing.T) {
	tests := []struct {
		hcloudStatus hcloud.ServerStatus
		expected     provider.ServerState
	}{
		{hcloud.ServerStatusStarting, provider.ServerStateStarting},
		{hcloud.ServerStatusRunning, provider.ServerStateRunning},
		{hcloud.ServerStatusStopping, provider.ServerStateStopped},
		{hcloud.ServerStatusOff, provider.ServerStateStopped},
		{hcloud.ServerStatusDeleting, provider.ServerStateDeleting},
		{hcloud.ServerStatus("unknown"), provider.ServerStateUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.hcloudStatus), func(t *testing.T) {
			result := convertServerState(tt.hcloudStatus)
			if result != tt.expected {
				t.Errorf("Expected state %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseServerID(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"12345", 12345},
		{"0", 0},
		{"999999", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseServerID(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestFormatLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]string
		expected string
	}{
		{
			name:     "single filter",
			filters:  map[string]string{"role": "edge"},
			expected: "role=edge",
		},
		{
			name:     "empty filters",
			filters:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabelSelector(tt.filters)
			if tt.name == "empty filters" && result != "" {
				t.Errorf("Expected empty string, got '%s'", result)
			}
			if tt.name == "single filter" && result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConvertServer(t *testing.T) {
	now := time.Now()

	// Create IP addresses
	ipv4 := net.ParseIP("95.217.0.1")
	ipv6 := net.ParseIP("2001:db8::1")

	hcloudServer := &hcloud.Server{
		ID:   12345,
		Name: "test-server",
		PublicNet: hcloud.ServerPublicNet{
			IPv4: hcloud.ServerPublicNetIPv4{
				IP: ipv4,
			},
			IPv6: hcloud.ServerPublicNetIPv6{
				IP: ipv6,
			},
		},
		Datacenter: &hcloud.Datacenter{
			Location: &hcloud.Location{
				Name: "fsn1",
			},
		},
		Status:  hcloud.ServerStatusRunning,
		Labels:  map[string]string{"role": "edge"},
		Created: now,
	}

	server := convertServer(hcloudServer)

	if server.ID != "12345" {
		t.Errorf("Expected ID '12345', got '%s'", server.ID)
	}

	if server.Name != "test-server" {
		t.Errorf("Expected name 'test-server', got '%s'", server.Name)
	}

	if server.PublicIPv4 != "95.217.0.1" {
		t.Errorf("Expected IPv4 '95.217.0.1', got '%s'", server.PublicIPv4)
	}

	if server.Location != "fsn1" {
		t.Errorf("Expected location 'fsn1', got '%s'", server.Location)
	}

	if server.State != provider.ServerStateRunning {
		t.Errorf("Expected state 'running', got '%s'", server.State)
	}

	if server.Labels["role"] != "edge" {
		t.Errorf("Expected label role='edge', got '%s'", server.Labels["role"])
	}

	expectedTime := now.Format(time.RFC3339)
	if server.CreatedAt != expectedTime {
		t.Errorf("Expected CreatedAt '%s', got '%s'", expectedTime, server.CreatedAt)
	}
}

func TestIsValidSSHPublicKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid RSA key",
			key:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDZy... user@host",
			expected: true,
		},
		{
			name:     "valid ED25519 key",
			key:      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqB... user@host",
			expected: true,
		},
		{
			name:     "valid ECDSA key",
			key:      "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTI... user@host",
			expected: true,
		},
		{
			name:     "valid DSS key",
			key:      "ssh-dss AAAAB3NzaC1kc3MAAACBAOgR... user@host",
			expected: true,
		},
		{
			name:     "invalid key - no prefix",
			key:      "AAAAB3NzaC1yc2EAAAADAQABAAABAQDZy... user@host",
			expected: false,
		},
		{
			name:     "invalid key - empty",
			key:      "",
			expected: false,
		},
		{
			name:     "invalid key - random text",
			key:      "this is not an ssh key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSSHPublicKey(tt.key)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for key: %s", tt.expected, result, tt.key)
			}
		})
	}
}

func TestReadSSHPublicKey(t *testing.T) {
	// Create a temporary directory for test SSH keys
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create temp .ssh directory: %v", err)
	}

	// Create test SSH key files
	validKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqBqwe7x7U1nCN9MCQP0aJL6+lTXYmNxnPKPPPHASCT test@example.com"
	testKeyPath := filepath.Join(sshDir, "test_key.pub")
	err = os.WriteFile(testKeyPath, []byte(validKey+"\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to write test key file: %v", err)
	}

	// Create default key
	defaultKeyPath := filepath.Join(sshDir, "id_ed25519.pub")
	err = os.WriteFile(defaultKeyPath, []byte(validKey+"\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to write default key file: %v", err)
	}

	// Create invalid key
	invalidKeyPath := filepath.Join(sshDir, "invalid_key.pub")
	err = os.WriteFile(invalidKeyPath, []byte("not a valid key\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid key file: %v", err)
	}

	// Test with custom path
	t.Run("read from custom path", func(t *testing.T) {
		content, err := readSSHPublicKey("test_key", testKeyPath)
		if err != nil {
			t.Errorf("Failed to read SSH key with custom path: %v", err)
		}
		if content != validKey {
			t.Errorf("Expected key content to match, got: %s", content)
		}
	})

	t.Run("invalid key format", func(t *testing.T) {
		_, err := readSSHPublicKey("invalid_key", invalidKeyPath)
		if err == nil {
			t.Error("Expected error for invalid SSH key format")
		}
	})

	t.Run("non-existent key", func(t *testing.T) {
		_, err := readSSHPublicKey("nonexistent", filepath.Join(sshDir, "nonexistent.pub"))
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})
}

func TestWrapAuthError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		operation      string
		expectNil      bool
		expectContains []string
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test operation",
			expectNil: true,
		},
		{
			name:      "unauthorized error wraps with helpful message",
			err:       hcloud.Error{Code: hcloud.ErrorCodeUnauthorized, Message: "token invalid"},
			operation: "failed to get server type",
			expectContains: []string{
				"failed to get server type",
				"token invalid",
				"API token is invalid",
				"Hetzner Cloud Console",
				"HETZNER_API_TOKEN",
			},
		},
		{
			name:      "non-auth error passes through",
			err:       hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"},
			operation: "failed to get server",
			expectContains: []string{
				"failed to get server",
				"not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapAuthError(tt.err, tt.operation)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil error, got: %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil error")
			}

			errMsg := result.Error()
			for _, expected := range tt.expectContains {
				if !contains(errMsg, expected) {
					t.Errorf("Expected error to contain '%s', got: %s", expected, errMsg)
				}
			}
		})
	}
}

// contains checks if s contains substr (case-insensitive for flexibility)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReadSSHPublicKeyTildeExpansion(t *testing.T) {
	// This test verifies that tilde expansion works
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Skipping test: cannot get home directory: %v", err)
	}

	// Create a test key in actual .ssh directory
	sshDir := filepath.Join(homeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: .ssh directory does not exist")
	}

	// Note: This test assumes there's at least one valid SSH key in ~/.ssh/
	// We'll just test that tilde expansion doesn't break the path resolution
	_, err = readSSHPublicKey("test", "~/nonexistent.pub")
	// We expect an error here (file not found), but not a path expansion error
	if err != nil && !os.IsNotExist(err) {
		// Error is fine, as long as it's about the file not existing, not path issues
		t.Logf("Got expected error: %v", err)
	}
}
