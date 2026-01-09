package bootmode

import (
	"context"
	"fmt"
	"time"

	"github.com/nimsforest/morpheus/pkg/machine/proxmox"
)

// ProxmoxManager implements Manager for Proxmox VE VR nodes
type ProxmoxManager struct {
	client *proxmox.Client
	config VRNodeConfig
	node   string
}

// NewProxmoxManager creates a new Proxmox boot mode manager
func NewProxmoxManager(proxmoxConfig proxmox.ProviderConfig, vrConfig VRNodeConfig) (*ProxmoxManager, error) {
	client, err := proxmox.NewClient(proxmoxConfig)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	return &ProxmoxManager{
		client: client,
		config: vrConfig,
		node:   proxmoxConfig.Node,
	}, nil
}

// ListModes returns the linux and windows modes
func (m *ProxmoxManager) ListModes(ctx context.Context) ([]Mode, error) {
	modes := make([]Mode, 0, 2)

	// Linux mode
	linuxMode, err := m.getMode(ctx, "linux", m.config.Linux.VMID)
	if err == nil {
		modes = append(modes, *linuxMode)
	}

	// Windows mode
	windowsMode, err := m.getMode(ctx, "windows", m.config.Windows.VMID)
	if err == nil {
		modes = append(modes, *windowsMode)
	}

	return modes, nil
}

// GetMode returns a specific mode by name
func (m *ProxmoxManager) GetMode(ctx context.Context, name string) (*Mode, error) {
	vmid, err := m.getVMID(name)
	if err != nil {
		return nil, err
	}
	return m.getMode(ctx, name, vmid)
}

// GetCurrentMode returns the currently running mode
func (m *ProxmoxManager) GetCurrentMode(ctx context.Context) (*Mode, error) {
	// Check Linux VM
	linuxVM, err := m.client.GetVM(ctx, m.config.Linux.VMID)
	if err == nil && linuxVM.Status == proxmox.VMStatusRunning {
		return m.getMode(ctx, "linux", m.config.Linux.VMID)
	}

	// Check Windows VM
	windowsVM, err := m.client.GetVM(ctx, m.config.Windows.VMID)
	if err == nil && windowsVM.Status == proxmox.VMStatusRunning {
		return m.getMode(ctx, "windows", m.config.Windows.VMID)
	}

	return nil, nil // No mode active
}

// Switch changes from the current mode to the target mode
func (m *ProxmoxManager) Switch(ctx context.Context, targetMode string, opts SwitchOptions) (*SwitchResult, error) {
	startTime := time.Now()
	result := &SwitchResult{
		ToMode: targetMode,
	}

	// Validate target mode
	targetVMID, err := m.getVMID(targetMode)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Get current mode
	current, err := m.GetCurrentMode(ctx)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	if current != nil {
		result.FromMode = current.Name

		// Already on target mode?
		if current.Name == targetMode {
			result.Success = true
			result.Duration = time.Since(startTime)
			return result, &AlreadyActiveError{Mode: targetMode}
		}
	}

	// Dry run - just report what would happen
	if opts.DryRun {
		result.Success = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Stop current mode if running
	if current != nil {
		currentVMID, _ := m.getVMID(current.Name)
		if err := m.stopVM(ctx, currentVMID, opts); err != nil {
			result.Error = fmt.Sprintf("failed to stop %s: %v", current.Name, err)
			return result, &SwitchError{FromMode: current.Name, ToMode: targetMode, Reason: err.Error()}
		}
	}

	// Start target mode
	upid, err := m.client.StartVM(ctx, targetVMID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to start %s: %v", targetMode, err)
		return result, &SwitchError{FromMode: result.FromMode, ToMode: targetMode, Reason: err.Error()}
	}

	// Wait for start task to complete
	if _, err := m.client.WaitForTask(ctx, upid, time.Second); err != nil {
		result.Error = fmt.Sprintf("start task failed: %v", err)
		return result, &SwitchError{FromMode: result.FromMode, ToMode: targetMode, Reason: err.Error()}
	}

	// Wait for VM to be running
	waitCtx, cancel := context.WithTimeout(ctx, opts.StartupTimeout)
	defer cancel()

	if err := m.client.WaitForVMStatus(waitCtx, targetVMID, proxmox.VMStatusRunning, time.Second); err != nil {
		result.Error = fmt.Sprintf("timeout waiting for %s to start: %v", targetMode, err)
		return result, &SwitchError{FromMode: result.FromMode, ToMode: targetMode, Reason: err.Error()}
	}

	// Get IP address if waiting for network
	if opts.WaitForNetwork {
		ips, _ := m.client.GetVMIPs(ctx, targetVMID)
		if len(ips) > 0 {
			result.IPAddress = ips[0]
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result, nil
}

// GetModeInfo returns detailed information about a mode
func (m *ProxmoxManager) GetModeInfo(ctx context.Context, name string) (*ModeInfo, error) {
	vmid, err := m.getVMID(name)
	if err != nil {
		return nil, err
	}

	vm, err := m.client.GetVM(ctx, vmid)
	if err != nil {
		return nil, err
	}

	mode, err := m.getMode(ctx, name, vmid)
	if err != nil {
		return nil, err
	}

	info := &ModeInfo{
		Mode:        *mode,
		CPUUsage:    vm.CPUUsage * 100,
		MemoryUsage: float64(vm.MemoryUsed) / float64(vm.Memory) * 100,
		MemoryTotal: vm.Memory,
	}

	// GPU info would come from config
	info.GPUName = fmt.Sprintf("GPU at %s", m.config.GPUPCI)

	return info, nil
}

// Ping checks if Proxmox is reachable
func (m *ProxmoxManager) Ping(ctx context.Context) error {
	return m.client.Ping(ctx)
}

// Helper methods

func (m *ProxmoxManager) getVMID(mode string) (int, error) {
	switch mode {
	case "linux":
		return m.config.Linux.VMID, nil
	case "windows":
		return m.config.Windows.VMID, nil
	default:
		return 0, &ModeNotFoundError{Mode: mode}
	}
}

func (m *ProxmoxManager) getMode(ctx context.Context, name string, vmid int) (*Mode, error) {
	vm, err := m.client.GetVM(ctx, vmid)
	if err != nil {
		return nil, err
	}

	var osType OSType
	var vrSoftware string
	var description string
	var vmConfig VMConfig

	switch name {
	case "linux":
		osType = OSTypeLinux
		vrSoftware = "wivrn"
		description = "CachyOS + WiVRN"
		vmConfig = m.config.Linux
	case "windows":
		osType = OSTypeWindows
		vrSoftware = "steamlink"
		description = "Windows + SteamLink"
		vmConfig = m.config.Windows
	}

	mode := &Mode{
		Name:        name,
		OS:          osType,
		Description: description,
		VMID:        vmConfig.VMID,
		Status:      m.vmStatusToModeStatus(vm.Status),
		VRSoftware:  vrSoftware,
	}

	if mode.Status == ModeStatusRunning {
		mode.Uptime = time.Duration(vm.Uptime) * time.Second
		ips, _ := m.client.GetVMIPs(ctx, vmid)
		if len(ips) > 0 {
			mode.IPAddress = ips[0]
		}

		// Default services for each mode
		if name == "linux" {
			mode.Services = []Service{
				{Name: "wivrn", Status: "active"},
				{Name: "nimsforest", Status: "active"},
				{Name: "nats", Status: "active"},
			}
		} else {
			mode.Services = []Service{
				{Name: "steamlink", Status: "active"},
				{Name: "nimsforest", Status: "active"},
				{Name: "nats", Status: "active"},
			}
		}
	}

	return mode, nil
}

func (m *ProxmoxManager) stopVM(ctx context.Context, vmid int, opts SwitchOptions) error {
	stopCtx, cancel := context.WithTimeout(ctx, opts.ShutdownTimeout)
	defer cancel()

	// Use graceful shutdown
	upid, err := m.client.ShutdownVM(stopCtx, vmid, int(opts.ShutdownTimeout.Seconds()))
	if err != nil {
		return err
	}

	// Wait for shutdown task
	status, err := m.client.WaitForTask(stopCtx, upid, time.Second)
	if err != nil {
		return err
	}

	if !status.IsSuccessful() {
		return fmt.Errorf("shutdown failed: %s", status.ExitStatus)
	}

	return nil
}

func (m *ProxmoxManager) vmStatusToModeStatus(status proxmox.VMStatus) ModeStatus {
	switch status {
	case proxmox.VMStatusRunning:
		return ModeStatusRunning
	case proxmox.VMStatusStopped:
		return ModeStatusStopped
	case proxmox.VMStatusPaused:
		return ModeStatusStopped
	default:
		return ModeStatusUnknown
	}
}
