package bootmode

import (
	"context"
	"fmt"
	"time"

	"github.com/nimsforest/morpheus/pkg/machine/proxmox"
)

// ProxmoxManager implements Manager for Proxmox VE
type ProxmoxManager struct {
	provider *proxmox.Provider
}

// NewProxmoxManager creates a new Proxmox boot mode manager
func NewProxmoxManager(config proxmox.ProviderConfig) (*ProxmoxManager, error) {
	provider, err := proxmox.NewProvider(config)
	if err != nil {
		return nil, fmt.Errorf("create proxmox provider: %w", err)
	}

	return &ProxmoxManager{
		provider: provider,
	}, nil
}

// ListModes returns all configured boot modes
func (m *ProxmoxManager) ListModes(ctx context.Context) ([]Mode, error) {
	providerModes := m.provider.GetModes()
	modes := make([]Mode, 0, len(providerModes))

	for _, pm := range providerModes {
		vm, err := m.provider.GetVM(ctx, pm.VMID)
		if err != nil {
			// Include mode even if we can't get VM status
			modes = append(modes, Mode{
				Name:           pm.Name,
				Description:    pm.Description,
				Provider:       "proxmox",
				ProviderID:     fmt.Sprintf("%d", pm.VMID),
				GPUPassthrough: pm.GPUPassthrough,
				Status:         ModeStatusUnknown,
			})
			continue
		}

		status := m.vmStatusToModeStatus(vm.Status)

		mode := Mode{
			Name:           pm.Name,
			Description:    pm.Description,
			Provider:       "proxmox",
			ProviderID:     fmt.Sprintf("%d", pm.VMID),
			GPUPassthrough: pm.GPUPassthrough,
			Status:         status,
		}

		if status == ModeStatusRunning {
			mode.Uptime = time.Duration(vm.Uptime) * time.Second
			mode.IPAddresses = vm.IPs
		}

		modes = append(modes, mode)
	}

	return modes, nil
}

// GetMode returns a specific boot mode by name
func (m *ProxmoxManager) GetMode(ctx context.Context, name string) (*Mode, error) {
	pm, err := m.provider.GetMode(name)
	if err != nil {
		return nil, &ModeNotFoundError{Mode: name}
	}

	vm, err := m.provider.GetVM(ctx, pm.VMID)
	if err != nil {
		return &Mode{
			Name:           pm.Name,
			Description:    pm.Description,
			Provider:       "proxmox",
			ProviderID:     fmt.Sprintf("%d", pm.VMID),
			GPUPassthrough: pm.GPUPassthrough,
			Status:         ModeStatusUnknown,
		}, nil
	}

	mode := &Mode{
		Name:           pm.Name,
		Description:    pm.Description,
		Provider:       "proxmox",
		ProviderID:     fmt.Sprintf("%d", pm.VMID),
		GPUPassthrough: pm.GPUPassthrough,
		Status:         m.vmStatusToModeStatus(vm.Status),
	}

	if mode.Status == ModeStatusRunning {
		mode.Uptime = time.Duration(vm.Uptime) * time.Second
		mode.IPAddresses = vm.IPs
	}

	return mode, nil
}

// GetCurrentMode returns the currently active boot mode
func (m *ProxmoxManager) GetCurrentMode(ctx context.Context) (*Mode, error) {
	pm, err := m.provider.GetCurrentMode(ctx)
	if err != nil {
		return nil, err
	}
	if pm == nil {
		return nil, nil // No mode active
	}

	return m.GetMode(ctx, pm.Name)
}

// Switch changes from the current mode to the target mode
func (m *ProxmoxManager) Switch(ctx context.Context, targetMode string, opts SwitchOptions) (*SwitchResult, error) {
	startTime := time.Now()
	result := &SwitchResult{
		ToMode: targetMode,
	}

	// Get target mode
	target, err := m.provider.GetMode(targetMode)
	if err != nil {
		result.Error = err.Error()
		return result, &ModeNotFoundError{Mode: targetMode}
	}

	// Get current mode (if any)
	current, err := m.provider.GetCurrentMode(ctx)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	if current != nil {
		result.FromMode = current.Name

		// Check if already on target mode
		if current.Name == targetMode {
			result.Success = true
			result.Duration = time.Since(startTime)
			return result, &AlreadyActiveError{Mode: targetMode}
		}

		// GPU conflict check: if both modes need GPU, we need to stop the current one
		if current.GPUPassthrough && target.GPUPassthrough {
			// This is expected - we'll stop the current VM first
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
		if err := m.stopVM(ctx, current.VMID, opts); err != nil {
			result.Error = fmt.Sprintf("failed to stop %s: %v", current.Name, err)
			return result, err
		}
	}

	// Start target mode
	if err := m.provider.StartVM(ctx, target.VMID); err != nil {
		result.Error = fmt.Sprintf("failed to start %s: %v", targetMode, err)
		return result, err
	}

	// Wait for the VM to be running
	waitCtx, cancel := context.WithTimeout(ctx, opts.StartupTimeout)
	defer cancel()

	vm, err := m.waitForRunning(waitCtx, target.VMID)
	if err != nil {
		result.Error = fmt.Sprintf("timeout waiting for %s to start: %v", targetMode, err)
		return result, err
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	result.IPAddresses = vm.IPs

	return result, nil
}

// GetModeInfo returns detailed information about a mode
func (m *ProxmoxManager) GetModeInfo(ctx context.Context, name string) (*ModeInfo, error) {
	pm, err := m.provider.GetMode(name)
	if err != nil {
		return nil, &ModeNotFoundError{Mode: name}
	}

	vm, err := m.provider.GetVM(ctx, pm.VMID)
	if err != nil {
		return nil, err
	}

	config, err := m.provider.GetVMConfig(ctx, pm.VMID)
	if err != nil {
		return nil, err
	}

	info := &ModeInfo{
		Mode: Mode{
			Name:           pm.Name,
			Description:    pm.Description,
			Provider:       "proxmox",
			ProviderID:     fmt.Sprintf("%d", pm.VMID),
			GPUPassthrough: pm.GPUPassthrough,
			Status:         m.vmStatusToModeStatus(vm.Status),
			IPAddresses:    vm.IPs,
			Uptime:         time.Duration(vm.Uptime) * time.Second,
		},
		CPUUsage:    vm.CPUUsage * 100, // Convert to percentage
		MemoryUsage: float64(vm.MemoryUsed) / float64(vm.Memory) * 100,
		MemoryTotal: vm.Memory,
	}

	// Add GPU devices from config
	for _, pci := range config.HostPCI {
		info.GPUDevices = append(info.GPUDevices, GPUDevice{
			Address: pci,
		})
	}

	return info, nil
}

// Ping checks if Proxmox is reachable
func (m *ProxmoxManager) Ping(ctx context.Context) error {
	return m.provider.Ping(ctx)
}

// stopVM stops a VM with the given options
func (m *ProxmoxManager) stopVM(ctx context.Context, vmid int, opts SwitchOptions) error {
	stopCtx, cancel := context.WithTimeout(ctx, opts.ShutdownTimeout)
	defer cancel()

	if err := m.provider.StopVM(stopCtx, vmid); err != nil {
		return err
	}

	// Wait for stopped
	return m.waitForStopped(stopCtx, vmid)
}

// waitForRunning waits for a VM to be running
func (m *ProxmoxManager) waitForRunning(ctx context.Context, vmid int) (*proxmox.VM, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			vm, err := m.provider.GetVM(ctx, vmid)
			if err != nil {
				return nil, err
			}
			if vm.Status == proxmox.VMStatusRunning {
				return vm, nil
			}
		}
	}
}

// waitForStopped waits for a VM to be stopped
func (m *ProxmoxManager) waitForStopped(ctx context.Context, vmid int) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			vm, err := m.provider.GetVM(ctx, vmid)
			if err != nil {
				return err
			}
			if vm.Status == proxmox.VMStatusStopped {
				return nil
			}
		}
	}
}

// vmStatusToModeStatus converts Proxmox VM status to mode status
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
