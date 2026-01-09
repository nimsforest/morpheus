package proxmox

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Port != 8006 {
		t.Errorf("expected port 8006, got %d", config.Port)
	}

	if config.Node != "pve" {
		t.Errorf("expected node 'pve', got %s", config.Node)
	}

	if config.VerifySSL != false {
		t.Error("expected VerifySSL to be false by default")
	}
}

func TestTaskStatus_IsRunning(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"running", true},
		{"stopped", false},
		{"", false},
	}

	for _, tt := range tests {
		ts := TaskStatus{Status: tt.status}
		if ts.IsRunning() != tt.expected {
			t.Errorf("IsRunning() for status %q: expected %v, got %v",
				tt.status, tt.expected, ts.IsRunning())
		}
	}
}

func TestTaskStatus_IsSuccessful(t *testing.T) {
	tests := []struct {
		status     string
		exitStatus string
		expected   bool
	}{
		{"stopped", "OK", true},
		{"stopped", "ERROR", false},
		{"running", "OK", false},
		{"running", "", false},
	}

	for _, tt := range tests {
		ts := TaskStatus{Status: tt.status, ExitStatus: tt.exitStatus}
		if ts.IsSuccessful() != tt.expected {
			t.Errorf("IsSuccessful() for status=%q, exitStatus=%q: expected %v, got %v",
				tt.status, tt.exitStatus, tt.expected, ts.IsSuccessful())
		}
	}
}

func TestNewClient_MissingHost(t *testing.T) {
	config := ProviderConfig{
		APITokenID:     "user@pam!token",
		APITokenSecret: "secret",
	}

	_, err := NewClient(config)
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestNewClient_MissingToken(t *testing.T) {
	config := ProviderConfig{
		Host: "192.168.1.100",
	}

	_, err := NewClient(config)
	if err == nil {
		t.Error("expected error for missing token")
	}
}

func TestNewClient_Valid(t *testing.T) {
	config := ProviderConfig{
		Host:           "192.168.1.100",
		Port:           8006,
		Node:           "pve",
		APITokenID:     "user@pam!token",
		APITokenSecret: "secret",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.baseURL != "https://192.168.1.100:8006/api2/json" {
		t.Errorf("unexpected baseURL: %s", client.baseURL)
	}

	if client.node != "pve" {
		t.Errorf("unexpected node: %s", client.node)
	}
}

func TestNewProvider(t *testing.T) {
	config := ProviderConfig{
		Host:           "192.168.1.100",
		Port:           8006,
		Node:           "pve",
		APITokenID:     "user@pam!token",
		APITokenSecret: "secret",
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("expected provider to be non-nil")
	}
}

func TestVMStatus_Constants(t *testing.T) {
	// Ensure all status values are defined correctly
	statuses := []VMStatus{
		VMStatusRunning,
		VMStatusStopped,
		VMStatusPaused,
		VMStatusUnknown,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
	}
}
