package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ServerStats represents NATS server statistics from the monitoring endpoint
type ServerStats struct {
	ServerID    string  `json:"server_id"`
	ServerName  string  `json:"server_name"`
	Version     string  `json:"version"`
	CPU         float64 `json:"cpu"`
	Mem         int64   `json:"mem"`
	Connections int     `json:"connections"`
	InMsgs      int64   `json:"in_msgs"`
	OutMsgs     int64   `json:"out_msgs"`
	InBytes     int64   `json:"in_bytes"`
	OutBytes    int64   `json:"out_bytes"`
	Uptime      string  `json:"uptime"`
	MaxPayload  int     `json:"max_payload"`
	// JetStream stats
	JetStreamEnabled bool `json:"jetstream"`
}

// ClusterInfo represents NATS cluster information
type ClusterInfo struct {
	Name   string `json:"name"`
	Leader bool   `json:"leader"`
	// Add more fields as needed
}

// VarzResponse represents the response from NATS /varz endpoint
type VarzResponse struct {
	ServerID    string       `json:"server_id"`
	ServerName  string       `json:"server_name"`
	Version     string       `json:"version"`
	CPU         float64      `json:"cpu"`
	Mem         int64        `json:"mem"`
	Connections int          `json:"connections"`
	InMsgs      int64        `json:"in_msgs"`
	OutMsgs     int64        `json:"out_msgs"`
	InBytes     int64        `json:"in_bytes"`
	OutBytes    int64        `json:"out_bytes"`
	Uptime      string       `json:"uptime"`
	MaxPayload  int          `json:"max_payload"`
	Cluster     *ClusterInfo `json:"cluster,omitempty"`
	JetStream   interface{}  `json:"jetstream,omitempty"`
}

// Monitor provides access to NATS monitoring endpoints
type Monitor struct {
	client *http.Client
}

// NewMonitor creates a new NATS monitor
func NewMonitor() *Monitor {
	return &Monitor{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetServerStats retrieves stats from a single NATS server
func (m *Monitor) GetServerStats(ctx context.Context, nodeIP string) (*ServerStats, error) {
	// NATS exposes stats at http://[ip]:8222/varz
	url := fmt.Sprintf("http://[%s]:8222/varz", nodeIP)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS monitoring: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NATS monitoring returned status %d", resp.StatusCode)
	}

	var varz VarzResponse
	if err := json.NewDecoder(resp.Body).Decode(&varz); err != nil {
		return nil, fmt.Errorf("failed to parse NATS stats: %w", err)
	}

	stats := &ServerStats{
		ServerID:    varz.ServerID,
		ServerName:  varz.ServerName,
		Version:     varz.Version,
		CPU:         varz.CPU,
		Mem:         varz.Mem,
		Connections: varz.Connections,
		InMsgs:      varz.InMsgs,
		OutMsgs:     varz.OutMsgs,
		InBytes:     varz.InBytes,
		OutBytes:    varz.OutBytes,
		Uptime:      varz.Uptime,
		MaxPayload:  varz.MaxPayload,
	}

	// Check if JetStream is enabled
	if varz.JetStream != nil {
		stats.JetStreamEnabled = true
	}

	return stats, nil
}

// GetClusterStats retrieves stats from all nodes in a cluster
func (m *Monitor) GetClusterStats(ctx context.Context, nodeIPs []string) ([]*ServerStats, error) {
	var stats []*ServerStats

	for _, ip := range nodeIPs {
		s, err := m.GetServerStats(ctx, ip)
		if err != nil {
			// Log but continue - node might be temporarily unavailable
			fmt.Printf("   ⚠️  Could not reach node %s: %s\n", ip, err)
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// ClusterHealth represents overall cluster health
type ClusterHealth struct {
	TotalNodes      int
	ReachableNodes  int
	TotalCPU        float64
	TotalMem        int64
	TotalConns      int
	AvgCPU          float64
	AvgMemPercent   float64
	HighCPU         bool
	HighMemory      bool
	CPUThreshold    float64
	MemoryThreshold float64
}

// GetClusterHealth calculates aggregate health metrics for a cluster
func (m *Monitor) GetClusterHealth(ctx context.Context, nodeIPs []string, cpuThreshold, memThreshold float64) (*ClusterHealth, error) {
	stats, _ := m.GetClusterStats(ctx, nodeIPs)

	health := &ClusterHealth{
		TotalNodes:      len(nodeIPs),
		ReachableNodes:  len(stats),
		CPUThreshold:    cpuThreshold,
		MemoryThreshold: memThreshold,
	}

	if len(stats) == 0 {
		return health, nil
	}

	// Calculate totals and averages
	for _, s := range stats {
		health.TotalCPU += s.CPU
		health.TotalMem += s.Mem
		health.TotalConns += s.Connections
	}

	health.AvgCPU = health.TotalCPU / float64(len(stats))

	// Check if any node exceeds thresholds
	for _, s := range stats {
		if s.CPU > cpuThreshold {
			health.HighCPU = true
		}
		// For memory, we need to know the total available memory
		// For now, we'll flag if any node reports high memory
		// NATS reports memory in bytes, we'll check against a reasonable threshold
	}

	return health, nil
}

// HealthStatus represents a health check result
type HealthStatus struct {
	Healthy     bool
	NodeIP      string
	Error       string
	Stats       *ServerStats
	CPUPercent  float64
	MemMB       int64
	Connections int
}

// CheckNodeHealth performs a health check on a single node
func (m *Monitor) CheckNodeHealth(ctx context.Context, nodeIP string) *HealthStatus {
	status := &HealthStatus{
		NodeIP:  nodeIP,
		Healthy: false,
	}

	stats, err := m.GetServerStats(ctx, nodeIP)
	if err != nil {
		status.Error = err.Error()
		return status
	}

	status.Healthy = true
	status.Stats = stats
	status.CPUPercent = stats.CPU
	status.MemMB = stats.Mem / (1024 * 1024)
	status.Connections = stats.Connections

	return status
}
