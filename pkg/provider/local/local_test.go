package local

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/nimsforest/morpheus/pkg/provider"
)

// skipIfNoDocker skips the test if Docker is not available
func skipIfNoDocker(t *testing.T) {
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker is not available, skipping test")
	}
}

// skipIntegrationTest skips integration tests unless MORPHEUS_INTEGRATION_TESTS=1 is set
// Integration tests require a fully working Docker environment with network access
func skipIntegrationTest(t *testing.T) {
	if os.Getenv("MORPHEUS_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test (set MORPHEUS_INTEGRATION_TESTS=1 to run)")
	}
	skipIfNoDocker(t)
}

func TestNewProvider(t *testing.T) {
	skipIfNoDocker(t)

	p, err := NewProvider()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if p == nil {
		t.Fatal("Provider is nil")
	}

	if p.networkName != "morpheus-local" {
		t.Errorf("Expected network name 'morpheus-local', got '%s'", p.networkName)
	}
}

func TestCheckDockerAvailable(t *testing.T) {
	// This test depends on whether Docker is installed
	err := checkDockerAvailable()

	// Check if docker command exists
	_, cmdErr := exec.LookPath("docker")
	if cmdErr != nil {
		// Docker not installed, expect error
		if err == nil {
			t.Error("Expected error when Docker is not installed")
		}
	}
	// If Docker is installed, the test passes regardless of whether daemon is running
}

func TestConvertContainerState(t *testing.T) {
	tests := []struct {
		name     string
		state    dockerState
		expected provider.ServerState
	}{
		{
			name:     "created state",
			state:    dockerState{Status: "created", Running: false},
			expected: provider.ServerStateStarting,
		},
		{
			name:     "running state",
			state:    dockerState{Status: "running", Running: true},
			expected: provider.ServerStateRunning,
		},
		{
			name:     "paused state",
			state:    dockerState{Status: "paused", Running: false},
			expected: provider.ServerStateStopped,
		},
		{
			name:     "exited state",
			state:    dockerState{Status: "exited", Running: false},
			expected: provider.ServerStateStopped,
		},
		{
			name:     "dead state",
			state:    dockerState{Status: "dead", Running: false},
			expected: provider.ServerStateStopped,
		},
		{
			name:     "removing state",
			state:    dockerState{Status: "removing", Running: false},
			expected: provider.ServerStateDeleting,
		},
		{
			name:     "unknown state",
			state:    dockerState{Status: "something-else", Running: false},
			expected: provider.ServerStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertContainerState(tt.state)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConvertContainer(t *testing.T) {
	container := &dockerContainer{
		ID:   "abc123def456789012345678901234567890123456789012",
		Name: "/test-container",
		State: dockerState{
			Status:  "running",
			Running: true,
		},
		NetworkSettings: dockerNetwork{
			IPAddress: "172.17.0.2",
			Networks: map[string]dockerNetworkInfo{
				"morpheus-local": {IPAddress: "172.18.0.5"},
			},
		},
		Config: dockerConfig{
			Labels: map[string]string{
				"forest_id": "forest-123",
			},
		},
		Created: "2025-01-01T00:00:00Z",
	}

	server := convertContainer(container, "morpheus-local")

	if server.ID != "abc123def456" {
		t.Errorf("Expected ID 'abc123def456', got '%s'", server.ID)
	}

	if server.Name != "test-container" {
		t.Errorf("Expected Name 'test-container', got '%s'", server.Name)
	}

	// Should use the morpheus-local network IP
	if server.PublicIPv4 != "172.18.0.5" {
		t.Errorf("Expected IP '172.18.0.5', got '%s'", server.PublicIPv4)
	}

	if server.Location != "local" {
		t.Errorf("Expected Location 'local', got '%s'", server.Location)
	}

	if server.State != provider.ServerStateRunning {
		t.Errorf("Expected state Running, got %s", server.State)
	}

	if server.Labels["forest_id"] != "forest-123" {
		t.Errorf("Expected label forest_id='forest-123', got '%s'", server.Labels["forest_id"])
	}
}

func TestConvertContainerFallbackIP(t *testing.T) {
	// Test that it falls back to default IP when network not found
	container := &dockerContainer{
		ID:   "abc123def456789012345678901234567890123456789012",
		Name: "/test-container",
		State: dockerState{
			Status:  "running",
			Running: true,
		},
		NetworkSettings: dockerNetwork{
			IPAddress: "172.17.0.2",
			Networks:  map[string]dockerNetworkInfo{},
		},
		Config: dockerConfig{
			Labels: map[string]string{},
		},
		Created: "2025-01-01T00:00:00Z",
	}

	server := convertContainer(container, "morpheus-local")

	// Should fall back to default IP
	if server.PublicIPv4 != "172.17.0.2" {
		t.Errorf("Expected fallback IP '172.17.0.2', got '%s'", server.PublicIPv4)
	}
}

func TestConvertContainerNilLabels(t *testing.T) {
	container := &dockerContainer{
		ID:   "abc123def456789012345678901234567890123456789012",
		Name: "/test-container",
		State: dockerState{
			Status:  "running",
			Running: true,
		},
		NetworkSettings: dockerNetwork{
			IPAddress: "172.17.0.2",
		},
		Config: dockerConfig{
			Labels: nil,
		},
		Created: "2025-01-01T00:00:00Z",
	}

	server := convertContainer(container, "morpheus-local")

	// Labels should be initialized to empty map, not nil
	if server.Labels == nil {
		t.Error("Expected Labels to be non-nil")
	}
}

// Integration tests - these require Docker to be running

func TestIntegrationCreateAndDeleteServer(t *testing.T) {
	skipIntegrationTest(t)

	p, err := NewProvider()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	// Create a test container - use busybox for fast, reliable CI tests
	req := provider.CreateServerRequest{
		Name:       "morpheus-test-container",
		ServerType: "local",
		Image:      "busybox:latest",
		Location:   "local",
		Labels: map[string]string{
			"forest_id": "test-forest",
			"test":      "true",
		},
	}

	server, err := p.CreateServer(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Verify server was created
	if server.Name != req.Name {
		t.Errorf("Expected name '%s', got '%s'", req.Name, server.Name)
	}

	if server.State != provider.ServerStateRunning {
		t.Errorf("Expected state Running, got %s", server.State)
	}

	if server.Location != "local" {
		t.Errorf("Expected location 'local', got '%s'", server.Location)
	}

	// Test GetServer
	retrieved, err := p.GetServer(ctx, server.ID)
	if err != nil {
		t.Fatalf("Failed to get server: %v", err)
	}

	if retrieved.ID != server.ID {
		t.Errorf("Expected ID '%s', got '%s'", server.ID, retrieved.ID)
	}

	// Test ListServers
	servers, err := p.ListServers(ctx, map[string]string{"forest_id": "test-forest"})
	if err != nil {
		t.Fatalf("Failed to list servers: %v", err)
	}

	found := false
	for _, s := range servers {
		if s.ID == server.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created server not found in list")
	}

	// Clean up - delete the server
	err = p.DeleteServer(ctx, server.ID)
	if err != nil {
		t.Fatalf("Failed to delete server: %v", err)
	}

	// Verify it's deleted
	_, err = p.GetServer(ctx, server.ID)
	if err == nil {
		t.Error("Server should not exist after deletion")
	}
}

func TestIntegrationListServersEmpty(t *testing.T) {
	skipIntegrationTest(t)

	p, err := NewProvider()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	// List with a filter that shouldn't match anything
	servers, err := p.ListServers(ctx, map[string]string{
		"nonexistent_label": "nonexistent_value",
	})
	if err != nil {
		t.Fatalf("Failed to list servers: %v", err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(servers))
	}
}
