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

func TestModeNotFoundError(t *testing.T) {
	err := &ModeNotFoundError{Mode: "macos"}

	expected := "mode not found: macos (valid modes: linux, windows)"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAlreadyActiveError(t *testing.T) {
	err := &AlreadyActiveError{Mode: "linux"}

	expected := "already in linux mode"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSwitchError(t *testing.T) {
	// With from mode
	err := &SwitchError{
		FromMode: "linux",
		ToMode:   "windows",
		Reason:   "VM failed to start",
	}

	expected := "failed to switch from linux to windows: VM failed to start"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	// Without from mode
	err2 := &SwitchError{
		ToMode: "windows",
		Reason: "VM not found",
	}

	expected2 := "failed to switch to windows: VM not found"
	if err2.Error() != expected2 {
		t.Errorf("expected %q, got %q", expected2, err2.Error())
	}
}

func TestModeStatus_Values(t *testing.T) {
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

func TestOSType_Values(t *testing.T) {
	osTypes := []OSType{
		OSTypeLinux,
		OSTypeWindows,
	}

	expected := []string{"linux", "windows"}
	for i, os := range osTypes {
		if string(os) != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], os)
		}
	}
}

func TestSwitchResult(t *testing.T) {
	result := &SwitchResult{
		FromMode:  "linux",
		ToMode:    "windows",
		Success:   true,
		Duration:  15 * time.Second,
		IPAddress: "192.168.1.150",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.FromMode != "linux" {
		t.Errorf("expected FromMode 'linux', got %s", result.FromMode)
	}

	if result.ToMode != "windows" {
		t.Errorf("expected ToMode 'windows', got %s", result.ToMode)
	}

	if result.IPAddress != "192.168.1.150" {
		t.Errorf("expected IPAddress '192.168.1.150', got %s", result.IPAddress)
	}
}

func TestMode(t *testing.T) {
	mode := Mode{
		Name:        "linux",
		OS:          OSTypeLinux,
		Description: "CachyOS + WiVRN",
		VMID:        101,
		Status:      ModeStatusRunning,
		IPAddress:   "192.168.1.150",
		Uptime:      2 * time.Hour,
		VRSoftware:  "wivrn",
		Services: []Service{
			{Name: "wivrn", Status: "active"},
			{Name: "nimsforest", Status: "active"},
			{Name: "nats", Status: "active"},
		},
	}

	if mode.Name != "linux" {
		t.Errorf("expected Name 'linux', got %s", mode.Name)
	}

	if mode.OS != OSTypeLinux {
		t.Errorf("expected OS 'linux', got %s", mode.OS)
	}

	if mode.Status != ModeStatusRunning {
		t.Errorf("expected Status 'running', got %s", mode.Status)
	}

	if mode.VRSoftware != "wivrn" {
		t.Errorf("expected VRSoftware 'wivrn', got %s", mode.VRSoftware)
	}

	if len(mode.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(mode.Services))
	}
}

func TestVRNodeConfig(t *testing.T) {
	config := VRNodeConfig{
		Linux: VMConfig{
			VMID:     101,
			Name:     "nimsforest-vr-linux",
			Memory:   32768,
			Cores:    12,
			DiskSize: 100,
		},
		Windows: VMConfig{
			VMID:     102,
			Name:     "nimsforest-vr-windows",
			Memory:   32768,
			Cores:    12,
			DiskSize: 200,
		},
		GPUPCI: "0000:01:00",
	}

	if config.Linux.VMID != 101 {
		t.Errorf("expected Linux VMID 101, got %d", config.Linux.VMID)
	}

	if config.Windows.VMID != 102 {
		t.Errorf("expected Windows VMID 102, got %d", config.Windows.VMID)
	}

	if config.GPUPCI != "0000:01:00" {
		t.Errorf("expected GPUPCI '0000:01:00', got %s", config.GPUPCI)
	}
}

func TestModeInfo(t *testing.T) {
	info := &ModeInfo{
		Mode: Mode{
			Name:        "linux",
			OS:          OSTypeLinux,
			Description: "CachyOS + WiVRN",
			VMID:        101,
			Status:      ModeStatusRunning,
		},
		CPUUsage:    45.5,
		MemoryUsage: 62.3,
		MemoryTotal: 32 * 1024 * 1024 * 1024, // 32 GB
		GPUName:     "NVIDIA RTX 4090",
	}

	if info.CPUUsage != 45.5 {
		t.Errorf("expected CPUUsage 45.5, got %f", info.CPUUsage)
	}

	if info.GPUName != "NVIDIA RTX 4090" {
		t.Errorf("expected GPUName 'NVIDIA RTX 4090', got %s", info.GPUName)
	}
}

func TestService(t *testing.T) {
	service := Service{
		Name:   "wivrn",
		Status: "active",
	}

	if service.Name != "wivrn" {
		t.Errorf("expected Name 'wivrn', got %s", service.Name)
	}

	if service.Status != "active" {
		t.Errorf("expected Status 'active', got %s", service.Status)
	}
}
