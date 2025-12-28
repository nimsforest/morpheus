package hetzner

import (
	"net"
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
