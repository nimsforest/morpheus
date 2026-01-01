package forest

import (
	"testing"
)

func TestFormatSSHAddress(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		port     int
		expected string
	}{
		{
			name:     "IPv4 address",
			ip:       "95.217.0.1",
			port:     22,
			expected: "95.217.0.1:22",
		},
		{
			name:     "IPv6 address",
			ip:       "2001:db8::1",
			port:     22,
			expected: "[2001:db8::1]:22",
		},
		{
			name:     "IPv6 full address",
			ip:       "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			port:     22,
			expected: "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:22",
		},
		{
			name:     "IPv6 with custom port",
			ip:       "2001:db8::1",
			port:     2222,
			expected: "[2001:db8::1]:2222",
		},
		{
			name:     "IPv4 with custom port",
			ip:       "192.168.1.1",
			port:     2222,
			expected: "192.168.1.1:2222",
		},
		{
			name:     "localhost IPv4",
			ip:       "127.0.0.1",
			port:     22,
			expected: "127.0.0.1:22",
		},
		{
			name:     "localhost IPv6",
			ip:       "::1",
			port:     22,
			expected: "[::1]:22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSSHAddress(tt.ip, tt.port)
			if result != tt.expected {
				t.Errorf("formatSSHAddress(%s, %d) = %s; want %s",
					tt.ip, tt.port, result, tt.expected)
			}
		})
	}
}

func TestContainsColon(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "IPv4 no colon",
			input:    "95.217.0.1",
			expected: false,
		},
		{
			name:     "IPv6 has colon",
			input:    "2001:db8::1",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "String with colon",
			input:    "abc:def",
			expected: true,
		},
		{
			name:     "String without colon",
			input:    "abcdef",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsColon(tt.input)
			if result != tt.expected {
				t.Errorf("containsColon(%s) = %v; want %v",
					tt.input, result, tt.expected)
			}
		})
	}
}
