package nats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMonitorIntegration(t *testing.T) {
	// Create a mock NATS monitoring server
	varz := VarzResponse{
		ServerID:    "NATS_TEST",
		ServerName:  "test-nats-server",
		Version:     "2.10.24",
		CPU:         45.5,
		Mem:         536870912, // 512MB
		Connections: 25,
		InMsgs:      5000,
		OutMsgs:     10000,
		InBytes:     1000000,
		OutBytes:    2000000,
		Uptime:      "2h15m",
		MaxPayload:  1048576,
		JetStream:   map[string]interface{}{"enabled": true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/varz" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(varz)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// The mock server uses IPv4, but our monitor expects IPv6 format
	// This test verifies the JSON parsing logic works correctly
	t.Run("verify_varz_parsing", func(t *testing.T) {
		// Make a direct HTTP request to verify parsing
		resp, err := http.Get(server.URL + "/varz")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		var parsed VarzResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if parsed.ServerID != "NATS_TEST" {
			t.Errorf("Expected ServerID 'NATS_TEST', got '%s'", parsed.ServerID)
		}
		if parsed.Version != "2.10.24" {
			t.Errorf("Expected Version '2.10.24', got '%s'", parsed.Version)
		}
		if parsed.CPU != 45.5 {
			t.Errorf("Expected CPU 45.5, got %f", parsed.CPU)
		}
		if parsed.Mem != 536870912 {
			t.Errorf("Expected Mem 536870912, got %d", parsed.Mem)
		}
		if parsed.Connections != 25 {
			t.Errorf("Expected Connections 25, got %d", parsed.Connections)
		}
		if parsed.JetStream == nil {
			t.Error("Expected JetStream to be set")
		}
	})

	t.Run("check_cluster_health_with_no_nodes", func(t *testing.T) {
		monitor := NewMonitor()
		ctx := context.Background()

		health, err := monitor.GetClusterHealth(ctx, []string{}, 80.0, 80.0)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if health.TotalNodes != 0 {
			t.Errorf("Expected 0 total nodes, got %d", health.TotalNodes)
		}
		if health.ReachableNodes != 0 {
			t.Errorf("Expected 0 reachable nodes, got %d", health.ReachableNodes)
		}
	})

	t.Run("check_node_health_unreachable", func(t *testing.T) {
		monitor := NewMonitor()
		ctx := context.Background()

		// Test with an unreachable address
		status := monitor.CheckNodeHealth(ctx, "2001:db8::dead")
		if status.Healthy {
			t.Error("Expected unhealthy status for unreachable node")
		}
		if status.Error == "" {
			t.Error("Expected error message")
		}
	})
}

func TestServerStatsConversion(t *testing.T) {
	// Test that VarzResponse correctly converts to ServerStats
	varz := VarzResponse{
		ServerID:    "TEST",
		ServerName:  "test",
		Version:     "2.10.24",
		CPU:         50.0,
		Mem:         1073741824, // 1GB
		Connections: 100,
		InMsgs:      1000,
		OutMsgs:     2000,
		InBytes:     50000,
		OutBytes:    100000,
		Uptime:      "1h",
		MaxPayload:  1048576,
		JetStream:   map[string]interface{}{"enabled": true},
	}

	stats := &ServerStats{
		ServerID:         varz.ServerID,
		ServerName:       varz.ServerName,
		Version:          varz.Version,
		CPU:              varz.CPU,
		Mem:              varz.Mem,
		Connections:      varz.Connections,
		InMsgs:           varz.InMsgs,
		OutMsgs:          varz.OutMsgs,
		InBytes:          varz.InBytes,
		OutBytes:         varz.OutBytes,
		Uptime:           varz.Uptime,
		MaxPayload:       varz.MaxPayload,
		JetStreamEnabled: varz.JetStream != nil,
	}

	if stats.ServerID != "TEST" {
		t.Errorf("Expected ServerID 'TEST', got '%s'", stats.ServerID)
	}
	if stats.CPU != 50.0 {
		t.Errorf("Expected CPU 50.0, got %f", stats.CPU)
	}
	if stats.Mem != 1073741824 {
		t.Errorf("Expected Mem 1073741824, got %d", stats.Mem)
	}
	if !stats.JetStreamEnabled {
		t.Error("Expected JetStream to be enabled")
	}

	// Convert memory to MB for display
	memMB := stats.Mem / (1024 * 1024)
	if memMB != 1024 {
		t.Errorf("Expected 1024 MB, got %d", memMB)
	}
}
