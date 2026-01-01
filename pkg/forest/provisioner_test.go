package forest

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/provider"
)

func TestGetNodeCount(t *testing.T) {
	tests := []struct {
		size     string
		expected int
	}{
		{"small", 1},
		{"medium", 3},
		{"large", 5},
		{"unknown", 1}, // default
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			count := getNodeCount(tt.size)
			if count != tt.expected {
				t.Errorf("For size '%s', expected %d nodes, got %d",
					tt.size, tt.expected, count)
			}
		})
	}
}

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	servers map[string]*provider.Server
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		servers: make(map[string]*provider.Server),
	}
}

func (m *mockProvider) CreateServer(ctx context.Context, req provider.CreateServerRequest) (*provider.Server, error) {
	server := &provider.Server{
		ID:         fmt.Sprintf("server-%d", len(m.servers)+1),
		Name:       req.Name,
		PublicIPv6: "::1",
		Location:   req.Location,
		State:      provider.ServerStateStarting,
		Labels:     req.Labels,
	}
	m.servers[server.ID] = server
	return server, nil
}

func (m *mockProvider) GetServer(ctx context.Context, serverID string) (*provider.Server, error) {
	if server, ok := m.servers[serverID]; ok {
		return server, nil
	}
	return nil, fmt.Errorf("server not found: %s", serverID)
}

func (m *mockProvider) DeleteServer(ctx context.Context, serverID string) error {
	delete(m.servers, serverID)
	return nil
}

func (m *mockProvider) WaitForServer(ctx context.Context, serverID string, state provider.ServerState) error {
	if server, ok := m.servers[serverID]; ok {
		server.State = state
		return nil
	}
	return fmt.Errorf("server not found: %s", serverID)
}

func (m *mockProvider) ListServers(ctx context.Context, filters map[string]string) ([]*provider.Server, error) {
	var result []*provider.Server
	for _, s := range m.servers {
		result = append(result, s)
	}
	return result, nil
}

func TestCheckSSHConnectivity(t *testing.T) {
	// Start a test TCP server to simulate SSH
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	cfg := &config.Config{
		Provisioning: config.ProvisioningConfig{
			ReadinessTimeout:  "5s",
			ReadinessInterval: "1s",
			SSHPort:           22,
		},
	}

	p := NewProvisioner(newMockProvider(), nil, cfg)

	// Test successful connection
	addr := listener.Addr().String()
	status, err := p.checkSSHConnectivityWithStatus(addr)
	if err != nil {
		t.Errorf("Expected successful connection, got error: %v", err)
	}
	if status != "connected" {
		t.Errorf("Expected status 'connected', got '%s'", status)
	}

	// Test failed connection (no server listening)
	status, err = p.checkSSHConnectivityWithStatus("127.0.0.1:59999")
	if err == nil {
		t.Error("Expected connection error for non-listening port")
	}
	if status == "connected" {
		t.Error("Expected non-connected status for failed connection")
	}
}

func TestClassifySSHError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"nil error", "", "connected"},
		{"connection refused", "dial tcp: connection refused", "port closed"},
		{"no route to host", "dial tcp: no route to host", "no route"},
		{"network unreachable", "dial tcp: network is unreachable", "network unreachable"},
		{"i/o timeout", "dial tcp: i/o timeout", "timeout"},
		{"timeout", "dial tcp: timeout", "timeout"},
		{"connection reset", "dial tcp: connection reset by peer", "connection reset"},
		{"host down", "dial tcp: host is down", "host down"},
		{"unknown error", "some unknown error", "connecting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = fmt.Errorf(tt.errMsg)
			}
			result := classifySSHError(err)
			if result != tt.expected {
				t.Errorf("classifySSHError(%q) = %q, want %q", tt.errMsg, result, tt.expected)
			}
		})
	}
}

func TestWaitForInfrastructureReady_Success(t *testing.T) {
	// Start a test TCP server to simulate SSH on IPv6
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Get the port from listener
	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	cfg := &config.Config{
		Provisioning: config.ProvisioningConfig{
			ReadinessTimeout:  "5s",
			ReadinessInterval: "100ms",
			SSHPort:           port,
		},
	}

	p := NewProvisioner(newMockProvider(), nil, cfg)

	server := &provider.Server{
		ID:         "test-server",
		PublicIPv6: "::1",
	}

	ctx := context.Background()
	err = p.waitForInfrastructureReady(ctx, server)
	if err != nil {
		t.Errorf("Expected infrastructure to be ready, got error: %v", err)
	}
}

func TestWaitForInfrastructureReady_Timeout(t *testing.T) {
	cfg := &config.Config{
		Provisioning: config.ProvisioningConfig{
			ReadinessTimeout:  "500ms",
			ReadinessInterval: "100ms",
			SSHPort:           59998, // Port with nothing listening
		},
	}

	p := NewProvisioner(newMockProvider(), nil, cfg)

	server := &provider.Server{
		ID:         "test-server",
		PublicIPv6: "::1",
	}

	ctx := context.Background()
	start := time.Now()
	err := p.waitForInfrastructureReady(ctx, server)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	// Should have timed out around 500ms
	if elapsed < 400*time.Millisecond || elapsed > 1*time.Second {
		t.Errorf("Expected timeout around 500ms, got %v", elapsed)
	}
}

func TestWaitForInfrastructureReady_NoIPAddress(t *testing.T) {
	cfg := &config.Config{
		Provisioning: config.ProvisioningConfig{
			ReadinessTimeout:  "5s",
			ReadinessInterval: "100ms",
			SSHPort:           22,
		},
	}

	p := NewProvisioner(newMockProvider(), nil, cfg)

	server := &provider.Server{
		ID:         "test-server",
		PublicIPv6: "", // No IP address
	}

	ctx := context.Background()
	err := p.waitForInfrastructureReady(ctx, server)
	if err == nil {
		t.Error("Expected error for server with no IPv6 address")
	}
}

func TestWaitForInfrastructureReady_ContextCancelled(t *testing.T) {
	cfg := &config.Config{
		Provisioning: config.ProvisioningConfig{
			ReadinessTimeout:  "30s",
			ReadinessInterval: "100ms",
			SSHPort:           59997, // Port with nothing listening
		},
	}

	p := NewProvisioner(newMockProvider(), nil, cfg)

	server := &provider.Server{
		ID:         "test-server",
		PublicIPv6: "::1",
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := p.waitForInfrastructureReady(ctx, server)
	if err == nil {
		t.Error("Expected context cancelled error")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}
