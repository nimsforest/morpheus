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
		RunningMode: "linuxvrstreaming",
		TargetMode:  "windowsvrstreaming",
		Message:     "GPU already in use by linuxvrstreaming",
	}

	expected := "GPU already in use by linuxvrstreaming"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestModeConflictError(t *testing.T) {
	err := &ModeConflictError{
		TargetMode: "nimsforestsharedgpu",
		Conflicts: []ConflictInfo{
			{ConflictingMode: "linuxvrstreaming"},
			{ConflictingMode: "windowsvrstreaming"},
		},
	}

	result := err.Error()
	if result != "cannot switch to nimsforestsharedgpu: conflicts with linuxvrstreaming, windowsvrstreaming" {
		t.Errorf("unexpected error message: %s", result)
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
	err := &AlreadyActiveError{Mode: "linuxvrstreaming"}

	expected := "mode already active: linuxvrstreaming"
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

func TestGPUMode_Values(t *testing.T) {
	// Ensure all GPU mode values are defined correctly
	modes := []GPUMode{
		GPUModeExclusive,
		GPUModeShared,
		GPUModeNone,
	}

	for _, m := range modes {
		if m == "" {
			t.Error("GPU mode should not be empty")
		}
	}
}

func TestSwitchResult(t *testing.T) {
	result := &SwitchResult{
		FromMode:    "linuxvrstreaming",
		ToMode:      "windowsvrstreaming",
		Success:     true,
		Duration:    15 * time.Second,
		IPAddresses: []string{"192.168.1.150"},
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.FromMode != "linuxvrstreaming" {
		t.Errorf("expected FromMode 'linuxvrstreaming', got %s", result.FromMode)
	}

	if result.ToMode != "windowsvrstreaming" {
		t.Errorf("expected ToMode 'windowsvrstreaming', got %s", result.ToMode)
	}

	if len(result.IPAddresses) != 1 || result.IPAddresses[0] != "192.168.1.150" {
		t.Errorf("unexpected IPAddresses: %v", result.IPAddresses)
	}
}

func TestMode(t *testing.T) {
	mode := Mode{
		Name:        "linuxvrstreaming",
		Description: "Linux VR streaming (CachyOS + WiVRN)",
		Provider:    "proxmox",
		ProviderID:  "101",
		GPUMode:     GPUModeExclusive,
		Status:      ModeStatusRunning,
		IPAddresses: []string{"192.168.1.150"},
		Uptime:      2 * time.Hour,
	}

	if mode.Name != "linuxvrstreaming" {
		t.Errorf("expected Name 'linuxvrstreaming', got %s", mode.Name)
	}

	if mode.Status != ModeStatusRunning {
		t.Errorf("expected Status 'running', got %s", mode.Status)
	}

	if mode.GPUMode != GPUModeExclusive {
		t.Errorf("expected GPUMode 'exclusive', got %s", mode.GPUMode)
	}

	if !mode.NeedsGPU() {
		t.Error("expected NeedsGPU() to be true")
	}

	if !mode.NeedsExclusiveGPU() {
		t.Error("expected NeedsExclusiveGPU() to be true")
	}
}

func TestMode_NeedsGPU(t *testing.T) {
	tests := []struct {
		gpuMode        GPUMode
		needsGPU       bool
		needsExclusive bool
	}{
		{GPUModeExclusive, true, true},
		{GPUModeShared, true, false},
		{GPUModeNone, false, false},
	}

	for _, tt := range tests {
		mode := Mode{GPUMode: tt.gpuMode}
		if mode.NeedsGPU() != tt.needsGPU {
			t.Errorf("NeedsGPU() for %s: expected %v", tt.gpuMode, tt.needsGPU)
		}
		if mode.NeedsExclusiveGPU() != tt.needsExclusive {
			t.Errorf("NeedsExclusiveGPU() for %s: expected %v", tt.gpuMode, tt.needsExclusive)
		}
	}
}

func TestModeInfo(t *testing.T) {
	info := &ModeInfo{
		Mode: Mode{
			Name:        "linuxvrstreaming",
			Description: "Linux VR streaming",
			Provider:    "proxmox",
			ProviderID:  "101",
			GPUMode:     GPUModeExclusive,
			Status:      ModeStatusRunning,
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

func TestConflictInfo(t *testing.T) {
	conflict := ConflictInfo{
		TargetMode:      "nimsforestsharedgpu",
		ConflictingMode: "linuxvrstreaming",
		Reason:          "GPU resource conflict",
		Alternatives:    []string{"nimsforestnogpu"},
	}

	if conflict.TargetMode != "nimsforestsharedgpu" {
		t.Errorf("expected TargetMode 'nimsforestsharedgpu', got %s", conflict.TargetMode)
	}

	if len(conflict.Alternatives) != 1 || conflict.Alternatives[0] != "nimsforestnogpu" {
		t.Errorf("unexpected Alternatives: %v", conflict.Alternatives)
	}
}
