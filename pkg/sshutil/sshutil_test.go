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
