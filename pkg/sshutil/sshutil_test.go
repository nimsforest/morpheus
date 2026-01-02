package sshutil

import (
	"strings"
	"testing"
)

func TestFormatSSHCommand(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		ip       string
		expected string
	}{
		{
			name:     "IPv4 address",
			user:     "root",
			ip:       "192.168.1.1",
			expected: "ssh root@192.168.1.1",
		},
		{
			name:     "IPv6 address",
			user:     "root",
			ip:       "2001:db8::1",
			expected: "ssh root@2001:db8::1",
		},
		{
			name:     "IPv6 full address",
			user:     "root",
			ip:       "2a01:4f9:c012:1576::1",
			expected: "ssh root@2a01:4f9:c012:1576::1",
		},
		{
			name:     "different user",
			user:     "ubuntu",
			ip:       "10.0.0.1",
			expected: "ssh ubuntu@10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSSHCommand(tt.user, tt.ip)
			if result != tt.expected {
				t.Errorf("FormatSSHCommand(%q, %q) = %q, want %q", tt.user, tt.ip, result, tt.expected)
			}
		})
	}
}

func TestFormatSSHAddress(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		port     int
		expected string
	}{
		{
			name:     "IPv4 address",
			ip:       "192.168.1.1",
			port:     22,
			expected: "192.168.1.1:22",
		},
		{
			name:     "IPv6 address needs brackets",
			ip:       "2001:db8::1",
			port:     22,
			expected: "[2001:db8::1]:22",
		},
		{
			name:     "IPv6 full address",
			ip:       "2a01:4f9:c012:1576::1",
			port:     22,
			expected: "[2a01:4f9:c012:1576::1]:22",
		},
		{
			name:     "custom port",
			ip:       "2001:db8::1",
			port:     2222,
			expected: "[2001:db8::1]:2222",
		},
		{
			name:     "already bracketed IPv6",
			ip:       "[2001:db8::1]",
			port:     22,
			expected: "[2001:db8::1]:22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSSHAddress(tt.ip, tt.port)
			if result != tt.expected {
				t.Errorf("FormatSSHAddress(%q, %d) = %q, want %q", tt.ip, tt.port, result, tt.expected)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"2001:db8::1", true},
		{"::1", true},
		{"2a01:4f9:c012:1576::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsIPv6(tt.ip)
			if result != tt.expected {
				t.Errorf("IsIPv6(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestFormatSSHCommandWithIdentity(t *testing.T) {
	tests := []struct {
		name         string
		user         string
		ip           string
		identityFile string
		expected     string
	}{
		{
			name:         "with identity file",
			user:         "root",
			ip:           "192.168.1.1",
			identityFile: "~/.ssh/id_ed25519",
			expected:     "ssh -i ~/.ssh/id_ed25519 root@192.168.1.1",
		},
		{
			name:         "with identity file IPv6",
			user:         "root",
			ip:           "2001:db8::1",
			identityFile: "~/.ssh/id_rsa",
			expected:     "ssh -i ~/.ssh/id_rsa root@2001:db8::1",
		},
		{
			name:         "empty identity file falls back to basic format",
			user:         "root",
			ip:           "192.168.1.1",
			identityFile: "",
			expected:     "ssh root@192.168.1.1",
		},
		{
			name:         "absolute path identity file",
			user:         "ubuntu",
			ip:           "10.0.0.1",
			identityFile: "/home/user/.ssh/custom_key",
			expected:     "ssh -i /home/user/.ssh/custom_key ubuntu@10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSSHCommandWithIdentity(tt.user, tt.ip, tt.identityFile)
			if result != tt.expected {
				t.Errorf("FormatSSHCommandWithIdentity(%q, %q, %q) = %q, want %q",
					tt.user, tt.ip, tt.identityFile, result, tt.expected)
			}
		})
	}
}

func TestGetSSHPrivateKeyForPublicKey(t *testing.T) {
	tests := []struct {
		name          string
		publicKeyPath string
		expectEmpty   bool // since we can't test actual file existence easily
	}{
		{
			name:          "empty path returns empty",
			publicKeyPath: "",
			expectEmpty:   true,
		},
		{
			name:          "non-pub file returns empty",
			publicKeyPath: "/path/to/key",
			expectEmpty:   true,
		},
		{
			name:          "pub file with non-existent private key returns empty",
			publicKeyPath: "/nonexistent/path/id_ed25519.pub",
			expectEmpty:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSSHPrivateKeyForPublicKey(tt.publicKeyPath)
			if tt.expectEmpty && result != "" {
				t.Errorf("GetSSHPrivateKeyForPublicKey(%q) = %q, want empty string",
					tt.publicKeyPath, result)
			}
		})
	}
}

func TestDetectSSHPrivateKeyPath(t *testing.T) {
	// This function depends on actual filesystem state,
	// so we just verify it doesn't panic and returns a valid format
	result := DetectSSHPrivateKeyPath()

	// Result should either be empty or start with ~/.ssh/
	if result != "" && !strings.HasPrefix(result, "~/.ssh/") {
		t.Errorf("DetectSSHPrivateKeyPath() = %q, expected empty or starting with ~/.ssh/", result)
	}
}

func TestCalculateSSHKeyFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		publicKey   string
		wantErr     bool
		fingerprint string // Only check if wantErr is false and this is non-empty
	}{
		{
			name:      "empty key",
			publicKey: "",
			wantErr:   true,
		},
		{
			name:      "invalid key - single part",
			publicKey: "not-valid",
			wantErr:   true,
		},
		{
			name:      "invalid base64",
			publicKey: "ssh-ed25519 not-valid-base64",
			wantErr:   true,
		},
		{
			name: "valid ed25519 key",
			// This is a sample ed25519 public key (not a real key)
			publicKey:   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com",
			wantErr:     false,
			fingerprint: "65:96:2d:fc:e8:d5:a9:11:64:0c:0f:ea:00:6e:5b:bd", // Expected MD5 fingerprint
		},
		{
			name: "valid rsa key",
			// Sample RSA public key (truncated for testing)
			publicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7q3MeXQy5H0+VwqQJNFijOqxqXcO2QkCuR3TtM+k9nWDZWvEVPvJMt2Z3xYz6LqNJ+8OvkX1nYJDaX6U2 test@example.com",
			wantErr:     false,
			fingerprint: "", // Don't check specific value since it's a truncated key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fingerprint, err := CalculateSSHKeyFingerprint(tt.publicKey)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateSSHKeyFingerprint(%q) expected error, got fingerprint %q", tt.publicKey, fingerprint)
				}
				return
			}
			if err != nil {
				t.Errorf("CalculateSSHKeyFingerprint(%q) unexpected error: %v", tt.publicKey, err)
				return
			}
			// Check fingerprint format (should be colon-separated hex bytes)
			if len(fingerprint) != 47 { // MD5 fingerprint is 32 hex chars + 15 colons = 47
				t.Errorf("CalculateSSHKeyFingerprint(%q) returned fingerprint with wrong length: %q (len=%d)", tt.publicKey, fingerprint, len(fingerprint))
			}
			// Check if fingerprint matches expected value (if provided)
			if tt.fingerprint != "" && fingerprint != tt.fingerprint {
				t.Errorf("CalculateSSHKeyFingerprint(%q) = %q, want %q", tt.publicKey, fingerprint, tt.fingerprint)
			}
		})
	}
}

func TestCalculateSSHKeyFingerprintFormat(t *testing.T) {
	// Test that the fingerprint format is correct (colon-separated hex)
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
	fingerprint, err := CalculateSSHKeyFingerprint(publicKey)
	if err != nil {
		t.Fatalf("CalculateSSHKeyFingerprint() unexpected error: %v", err)
	}

	// Check format: should be 32 hex bytes separated by colons
	parts := strings.Split(fingerprint, ":")
	if len(parts) != 16 {
		t.Errorf("Expected 16 parts in fingerprint, got %d: %s", len(parts), fingerprint)
	}

	// Each part should be exactly 2 hex characters
	for i, part := range parts {
		if len(part) != 2 {
			t.Errorf("Part %d should be 2 characters, got %d: %q", i, len(part), part)
		}
		// Check if it's valid hex
		for _, c := range part {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Part %d contains non-hex character: %q", i, part)
			}
		}
	}
}
