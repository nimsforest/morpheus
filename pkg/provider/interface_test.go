package provider

import (
	"testing"
)

func TestServerGetPreferredIP(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		expected string
	}{
		{
			name: "IPv6 only",
			server: Server{
				PublicIPv6: "2001:db8::1",
				PublicIPv4: "",
			},
			expected: "2001:db8::1",
		},
		{
			name: "IPv4 only",
			server: Server{
				PublicIPv6: "",
				PublicIPv4: "192.168.1.1",
			},
			expected: "192.168.1.1",
		},
		{
			name: "Both IPv6 and IPv4",
			server: Server{
				PublicIPv6: "2001:db8::1",
				PublicIPv4: "192.168.1.1",
			},
			expected: "2001:db8::1", // IPv6 preferred
		},
		{
			name: "Neither",
			server: Server{
				PublicIPv6: "",
				PublicIPv4: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.GetPreferredIP()
			if result != tt.expected {
				t.Errorf("GetPreferredIP() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestServerGetFallbackIP(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		expected string
	}{
		{
			name: "IPv6 only - no fallback",
			server: Server{
				PublicIPv6: "2001:db8::1",
				PublicIPv4: "",
			},
			expected: "",
		},
		{
			name: "IPv4 only - no fallback",
			server: Server{
				PublicIPv6: "",
				PublicIPv4: "192.168.1.1",
			},
			expected: "",
		},
		{
			name: "Both - IPv4 is fallback",
			server: Server{
				PublicIPv6: "2001:db8::1",
				PublicIPv4: "192.168.1.1",
			},
			expected: "192.168.1.1",
		},
		{
			name: "Neither - no fallback",
			server: Server{
				PublicIPv6: "",
				PublicIPv4: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.GetFallbackIP()
			if result != tt.expected {
				t.Errorf("GetFallbackIP() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestServerHasIPv4(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		expected bool
	}{
		{
			name: "Has IPv4",
			server: Server{
				PublicIPv4: "192.168.1.1",
			},
			expected: true,
		},
		{
			name: "No IPv4",
			server: Server{
				PublicIPv4: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.HasIPv4()
			if result != tt.expected {
				t.Errorf("HasIPv4() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestServerHasIPv6(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		expected bool
	}{
		{
			name: "Has IPv6",
			server: Server{
				PublicIPv6: "2001:db8::1",
			},
			expected: true,
		},
		{
			name: "No IPv6",
			server: Server{
				PublicIPv6: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.HasIPv6()
			if result != tt.expected {
				t.Errorf("HasIPv6() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
