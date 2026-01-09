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

	if config.Modes == nil {
		t.Error("expected Modes map to be initialized")
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

func TestProvider_GetModes(t *testing.T) {
	config := ProviderConfig{
		Host:           "192.168.1.100",
		APITokenID:     "user@pam!token",
		APITokenSecret: "secret",
		Modes: map[string]ModeSpec{
			"linuxvrstreaming": {
				VMID:        101,
				Description: "Linux VR streaming",
				GPUMode:     GPUModeExclusive,
			},
			"windowsvrstreaming": {
				VMID:        102,
				Description: "Windows VR streaming",
				GPUMode:     GPUModeExclusive,
			},
			"nimsforestnogpu": {
				VMID:        103,
				Description: "NimsForest without GPU",
				GPUMode:     GPUModeNone,
			},
			"nimsforestsharedgpu": {
				VMID:          104,
				Description:   "NimsForest with shared GPU",
				GPUMode:       GPUModeShared,
				ConflictsWith: []string{"linuxvrstreaming", "windowsvrstreaming"},
			},
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	modes := provider.GetModes()
	if len(modes) != 4 {
		t.Errorf("expected 4 modes, got %d", len(modes))
	}

	// Check that all modes are present
	modeNames := make(map[string]bool)
	for _, m := range modes {
		modeNames[m.Name] = true
	}

	for _, name := range []string{"linuxvrstreaming", "windowsvrstreaming", "nimsforestnogpu", "nimsforestsharedgpu"} {
		if !modeNames[name] {
			t.Errorf("missing mode: %s", name)
		}
	}
}

func TestProvider_GetMode(t *testing.T) {
	config := ProviderConfig{
		Host:           "192.168.1.100",
		APITokenID:     "user@pam!token",
		APITokenSecret: "secret",
		Modes: map[string]ModeSpec{
			"linuxvrstreaming": {
				VMID:        101,
				Description: "Linux VR streaming",
				GPUMode:     GPUModeExclusive,
			},
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test existing mode
	mode, err := provider.GetMode("linuxvrstreaming")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mode.Name != "linuxvrstreaming" {
		t.Errorf("expected name 'linuxvrstreaming', got %s", mode.Name)
	}

	if mode.VMID != 101 {
		t.Errorf("expected VMID 101, got %d", mode.VMID)
	}

	if mode.GPUMode != GPUModeExclusive {
		t.Errorf("expected GPUMode 'exclusive', got %s", mode.GPUMode)
	}

	if !mode.NeedsExclusiveGPU() {
		t.Error("expected NeedsExclusiveGPU() to be true")
	}

	// Test non-existing mode
	_, err = provider.GetMode("nonexistent")
	if err == nil {
		t.Error("expected error for non-existing mode")
	}
}

func TestBootMode_NeedsGPU(t *testing.T) {
	tests := []struct {
		mode     GPUMode
		needsGPU bool
		needsExclusive bool
	}{
		{GPUModeExclusive, true, true},
		{GPUModeShared, true, false},
		{GPUModeNone, false, false},
	}

	for _, tt := range tests {
		bm := BootMode{GPUMode: tt.mode}
		if bm.NeedsGPU() != tt.needsGPU {
			t.Errorf("NeedsGPU() for %s: expected %v", tt.mode, tt.needsGPU)
		}
		if bm.NeedsExclusiveGPU() != tt.needsExclusive {
			t.Errorf("NeedsExclusiveGPU() for %s: expected %v", tt.mode, tt.needsExclusive)
		}
	}
}

func TestBootMode_ConflictsWithMode(t *testing.T) {
	bm := BootMode{
		Name:          "nimsforestsharedgpu",
		ConflictsWith: []string{"linuxvrstreaming", "windowsvrstreaming"},
	}

	if !bm.ConflictsWithMode("linuxvrstreaming") {
		t.Error("expected conflict with linuxvrstreaming")
	}

	if !bm.ConflictsWithMode("windowsvrstreaming") {
		t.Error("expected conflict with windowsvrstreaming")
	}

	if bm.ConflictsWithMode("nimsforestnogpu") {
		t.Error("should not conflict with nimsforestnogpu")
	}
}
