package bootmode

import (
	"testing"
	"time"
)

func TestDefaultSwitchOptions(t *testing.T) {
	opts := DefaultSwitchOptions()

	if opts.Force {
		t.Error("expected Force to be false by default")
	}

	if opts.ShutdownTimeout != 60*time.Second {
		t.Errorf("expected ShutdownTimeout 60s, got %v", opts.ShutdownTimeout)
	}

	if opts.StartupTimeout != 120*time.Second {
		t.Errorf("expected StartupTimeout 120s, got %v", opts.StartupTimeout)
	}

	if !opts.WaitForNetwork {
		t.Error("expected WaitForNetwork to be true by default")
	}

	if opts.DryRun {
		t.Error("expected DryRun to be false by default")
	}
}

func TestGPUConflictError(t *testing.T) {
	err := &GPUConflictError{
		RunningMode: "cachyos",
		TargetMode:  "windows",
		Message:     "GPU already in use by cachyos",
	}

	expected := "GPU already in use by cachyos"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestModeNotFoundError(t *testing.T) {
	err := &ModeNotFoundError{Mode: "nonexistent"}

	expected := "boot mode not found: nonexistent"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAlreadyActiveError(t *testing.T) {
	err := &AlreadyActiveError{Mode: "cachyos"}

	expected := "mode already active: cachyos"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestModeStatus_Values(t *testing.T) {
	// Ensure all status values are defined correctly
	statuses := []ModeStatus{
		ModeStatusRunning,
		ModeStatusStopped,
		ModeStatusStarting,
		ModeStatusStopping,
		ModeStatusUnknown,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
	}
}

func TestSwitchResult(t *testing.T) {
	result := &SwitchResult{
		FromMode:    "cachyos",
		ToMode:      "windows",
		Success:     true,
		Duration:    15 * time.Second,
		IPAddresses: []string{"192.168.1.150"},
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.FromMode != "cachyos" {
		t.Errorf("expected FromMode 'cachyos', got %s", result.FromMode)
	}

	if result.ToMode != "windows" {
		t.Errorf("expected ToMode 'windows', got %s", result.ToMode)
	}

	if len(result.IPAddresses) != 1 || result.IPAddresses[0] != "192.168.1.150" {
		t.Errorf("unexpected IPAddresses: %v", result.IPAddresses)
	}
}

func TestMode(t *testing.T) {
	mode := Mode{
		Name:           "cachyos",
		Description:    "CachyOS with WiVRN",
		Provider:       "proxmox",
		ProviderID:     "101",
		GPUPassthrough: true,
		Status:         ModeStatusRunning,
		IPAddresses:    []string{"192.168.1.150"},
		Uptime:         2 * time.Hour,
	}

	if mode.Name != "cachyos" {
		t.Errorf("expected Name 'cachyos', got %s", mode.Name)
	}

	if mode.Status != ModeStatusRunning {
		t.Errorf("expected Status 'running', got %s", mode.Status)
	}

	if !mode.GPUPassthrough {
		t.Error("expected GPUPassthrough to be true")
	}
}

func TestModeInfo(t *testing.T) {
	info := &ModeInfo{
		Mode: Mode{
			Name:           "cachyos",
			Description:    "CachyOS with WiVRN",
			Provider:       "proxmox",
			ProviderID:     "101",
			GPUPassthrough: true,
			Status:         ModeStatusRunning,
		},
		CPUUsage:    45.5,
		MemoryUsage: 62.3,
		MemoryTotal: 32 * 1024 * 1024 * 1024, // 32 GB
		GPUDevices: []GPUDevice{
			{
				Address: "0000:01:00.0",
				Vendor:  "NVIDIA",
				Model:   "RTX 4090",
			},
		},
	}

	if info.CPUUsage != 45.5 {
		t.Errorf("expected CPUUsage 45.5, got %f", info.CPUUsage)
	}

	if len(info.GPUDevices) != 1 {
		t.Errorf("expected 1 GPU device, got %d", len(info.GPUDevices))
	}

	if info.GPUDevices[0].Model != "RTX 4090" {
		t.Errorf("expected GPU model 'RTX 4090', got %s", info.GPUDevices[0].Model)
	}
}
