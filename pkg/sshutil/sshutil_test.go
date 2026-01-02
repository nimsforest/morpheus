package sshutil

import "testing"

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
