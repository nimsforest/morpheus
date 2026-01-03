package httputil

import (
	"context"
	"testing"
	"time"
)

func TestCheckIPv6Connectivity(t *testing.T) {
	// Set a reasonable timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := CheckIPv6Connectivity(ctx)

	// The test should complete without panicking
	// The result may be either success or failure depending on the environment

	if result.Available {
		t.Logf("IPv6 is available, detected address: %s", result.Address)

		// If available, address should not be empty
		if result.Address == "" {
			t.Error("IPv6 reported as available but address is empty")
		}

		// Error should be nil when available
		if result.Error != nil {
			t.Errorf("IPv6 reported as available but error is not nil: %v", result.Error)
		}

		// Verify the address is a valid IPv6 address
		if !isValidIPv6(result.Address) {
			t.Errorf("Invalid IPv6 address format: %s", result.Address)
		}
	} else {
		t.Logf("IPv6 is not available: %v", result.Error)

		// If not available, address should be empty
		if result.Address != "" {
			t.Errorf("IPv6 reported as unavailable but address is not empty: %s", result.Address)
		}

		// Error should not be nil when unavailable
		if result.Error == nil {
			t.Error("IPv6 reported as unavailable but error is nil")
		}
	}
}

func TestIsValidIPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid IPv6 full",
			input:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: true,
		},
		{
			name:     "Valid IPv6 compressed",
			input:    "2001:db8::1",
			expected: true,
		},
		{
			name:     "Valid IPv6 loopback",
			input:    "::1",
			expected: true,
		},
		{
			name:     "Valid IPv6 all zeros",
			input:    "::",
			expected: true,
		},
		{
			name:     "Invalid IPv4",
			input:    "192.168.1.1",
			expected: false,
		},
		{
			name:     "Invalid empty",
			input:    "",
			expected: false,
		},
		{
			name:     "Invalid text",
			input:    "not-an-ip",
			expected: false,
		},
		{
			name:     "Invalid IPv4-mapped IPv6",
			input:    "::ffff:192.168.1.1", // IPv4-mapped IPv6 - not considered pure IPv6
			expected: false,                // We want pure IPv6 for connectivity check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIPv6(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIPv6(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No whitespace",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "Leading newline",
			input:    "\n2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "Trailing newline",
			input:    "2001:db8::1\n",
			expected: "2001:db8::1",
		},
		{
			name:     "Both newlines",
			input:    "\n2001:db8::1\n",
			expected: "2001:db8::1",
		},
		{
			name:     "Multiple whitespace types",
			input:    " \t\n2001:db8::1\r\n ",
			expected: "2001:db8::1",
		},
		{
			name:     "Only whitespace",
			input:    " \t\n\r ",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("trimWhitespace(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsRestrictedEnvironment(t *testing.T) {
	// This test just ensures the function doesn't panic
	// The actual result depends on the environment
	result := IsRestrictedEnvironment()
	t.Logf("IsRestrictedEnvironment() = %v", result)
}

func TestCreateHTTPClient(t *testing.T) {
	// Test that creating an HTTP client doesn't panic
	timeout := 10 * time.Second
	client := CreateHTTPClient(timeout)

	if client == nil {
		t.Error("CreateHTTPClient returned nil")
	}

	if client.Timeout != timeout {
		t.Errorf("Client timeout = %v, expected %v", client.Timeout, timeout)
	}
}

func TestCheckIPv4Connectivity(t *testing.T) {
	// Set a reasonable timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := CheckIPv4Connectivity(ctx)

	// The test should complete without panicking
	// The result may be either success or failure depending on the environment

	if result.Available {
		t.Logf("IPv4 is available, detected address: %s", result.Address)

		// If available, address should not be empty
		if result.Address == "" {
			t.Error("IPv4 reported as available but address is empty")
		}

		// Error should be nil when available
		if result.Error != nil {
			t.Errorf("IPv4 reported as available but error is not nil: %v", result.Error)
		}

		// Verify the address is a valid IPv4 address
		if !isValidIPv4(result.Address) {
			t.Errorf("Invalid IPv4 address format: %s", result.Address)
		}
	} else {
		t.Logf("IPv4 is not available: %v", result.Error)

		// If not available, address should be empty
		if result.Address != "" {
			t.Errorf("IPv4 reported as unavailable but address is not empty: %s", result.Address)
		}

		// Error should not be nil when unavailable
		if result.Error == nil {
			t.Error("IPv4 reported as unavailable but error is nil")
		}
	}
}

func TestIsValidIPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid IPv4",
			input:    "192.168.1.1",
			expected: true,
		},
		{
			name:     "Valid IPv4 all zeros",
			input:    "0.0.0.0",
			expected: true,
		},
		{
			name:     "Valid IPv4 broadcast",
			input:    "255.255.255.255",
			expected: true,
		},
		{
			name:     "Valid IPv4 public",
			input:    "8.8.8.8",
			expected: true,
		},
		{
			name:     "Invalid IPv6",
			input:    "2001:db8::1",
			expected: false,
		},
		{
			name:     "Invalid empty",
			input:    "",
			expected: false,
		},
		{
			name:     "Invalid text",
			input:    "not-an-ip",
			expected: false,
		},
		{
			name:     "Invalid partial IP",
			input:    "192.168.1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIPv4(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIPv4(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrimWhitespaceWithIPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IPv4 no whitespace",
			input:    "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 with newline",
			input:    "192.168.1.1\n",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 with spaces",
			input:    " 8.8.8.8 ",
			expected: "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("trimWhitespace(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
