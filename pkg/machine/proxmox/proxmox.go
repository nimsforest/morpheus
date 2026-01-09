package proxmox

import (
	"context"
	"fmt"
	"strconv"

	"github.com/nimsforest/morpheus/pkg/machine"
)

// Provider implements machine.Provider for Proxmox VE
type Provider struct {
	client *Client
	config ProviderConfig
}

// NewProvider creates a new Proxmox provider
func NewProvider(config ProviderConfig) (*Provider, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	return &Provider{
		client: client,
		config: config,
	}, nil
}

// CreateServer creates a new VM (not typically used for boot mode switching)
// For boot modes, VMs should be pre-created in Proxmox
func (p *Provider) CreateServer(ctx context.Context, req machine.CreateServerRequest) (*machine.Server, error) {
	return nil, fmt.Errorf("proxmox provider: VM creation not supported, VMs should be pre-created in Proxmox")
}

// GetServer retrieves a VM by its VMID
func (p *Provider) GetServer(ctx context.Context, serverID string) (*machine.Server, error) {
	vmid, err := strconv.Atoi(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid VMID: %s", serverID)
	}

	vm, err := p.client.GetVM(ctx, vmid)
	if err != nil {
		return nil, err
	}

	return p.vmToServer(vm), nil
}

// DeleteServer stops a VM (does not destroy it)
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	vmid, err := strconv.Atoi(serverID)
	if err != nil {
		return fmt.Errorf("invalid VMID: %s", serverID)
	}

	// Graceful shutdown with 60s timeout
	upid, err := p.client.ShutdownVM(ctx, vmid, 60)
	if err != nil {
		return err
	}

	// Wait for shutdown to complete
	status, err := p.client.WaitForTask(ctx, upid, 0)
	if err != nil {
		return err
	}

	if !status.IsSuccessful() {
		return fmt.Errorf("shutdown failed: %s", status.ExitStatus)
	}

	return nil
}

// WaitForServer waits for a VM to reach a specific state
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state machine.ServerState) error {
	vmid, err := strconv.Atoi(serverID)
	if err != nil {
		return fmt.Errorf("invalid VMID: %s", serverID)
	}

	targetStatus := p.stateToVMStatus(state)
	return p.client.WaitForVMStatus(ctx, vmid, targetStatus, 0)
}

// ListServers returns all VMs on the node
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*machine.Server, error) {
	vms, err := p.client.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	servers := make([]*machine.Server, 0, len(vms))
	for _, vm := range vms {
		// Skip templates
		if vm.Template {
			continue
		}

		server := p.vmToServer(vm)

		// Apply filters
		if !p.matchFilters(server, filters) {
			continue
		}

		servers = append(servers, server)
	}

	return servers, nil
}

// vmToServer converts a Proxmox VM to a machine.Server
func (p *Provider) vmToServer(vm *VM) *machine.Server {
	// Get IP if running
	var ipv4 string
	if vm.Status == VMStatusRunning && len(vm.IPs) > 0 {
		ipv4 = vm.IPs[0]
	}

	return &machine.Server{
		ID:         strconv.Itoa(vm.VMID),
		Name:       vm.Name,
		PublicIPv4: ipv4,
		Location:   vm.Node,
		State:      p.vmStatusToState(vm.Status),
		Labels: map[string]string{
			"vmid":   strconv.Itoa(vm.VMID),
			"node":   vm.Node,
			"status": string(vm.Status),
		},
	}
}

// vmStatusToState converts Proxmox VM status to machine.ServerState
func (p *Provider) vmStatusToState(status VMStatus) machine.ServerState {
	switch status {
	case VMStatusRunning:
		return machine.ServerStateRunning
	case VMStatusStopped:
		return machine.ServerStateStopped
	case VMStatusPaused:
		return machine.ServerStateStopped
	default:
		return machine.ServerStateUnknown
	}
}

// stateToVMStatus converts machine.ServerState to Proxmox VM status
func (p *Provider) stateToVMStatus(state machine.ServerState) VMStatus {
	switch state {
	case machine.ServerStateRunning:
		return VMStatusRunning
	case machine.ServerStateStopped:
		return VMStatusStopped
	default:
		return VMStatusUnknown
	}
}

// matchFilters checks if a server matches the given filters
func (p *Provider) matchFilters(server *machine.Server, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "name":
			if server.Name != value {
				return false
			}
		case "status", "state":
			if string(server.State) != value {
				return false
			}
		case "vmid":
			if server.ID != value {
				return false
			}
		}
	}
	return true
}

// StartVM starts a VM
func (p *Provider) StartVM(ctx context.Context, vmid int) error {
	upid, err := p.client.StartVM(ctx, vmid)
	if err != nil {
		return err
	}

	status, err := p.client.WaitForTask(ctx, upid, 0)
	if err != nil {
		return err
	}

	if !status.IsSuccessful() {
		return fmt.Errorf("start failed: %s", status.ExitStatus)
	}

	return nil
}

// StopVM stops a VM gracefully
func (p *Provider) StopVM(ctx context.Context, vmid int) error {
	upid, err := p.client.ShutdownVM(ctx, vmid, 60)
	if err != nil {
		return err
	}

	status, err := p.client.WaitForTask(ctx, upid, 0)
	if err != nil {
		return err
	}

	if !status.IsSuccessful() {
		return fmt.Errorf("stop failed: %s", status.ExitStatus)
	}

	return nil
}

// GetVM returns a VM by VMID
func (p *Provider) GetVM(ctx context.Context, vmid int) (*VM, error) {
	return p.client.GetVM(ctx, vmid)
}

// GetVMConfig returns the configuration of a VM
func (p *Provider) GetVMConfig(ctx context.Context, vmid int) (*VMConfig, error) {
	return p.client.GetVMConfig(ctx, vmid)
}

// HasGPUPassthrough checks if a VM has GPU passthrough configured
func (p *Provider) HasGPUPassthrough(ctx context.Context, vmid int) (bool, error) {
	config, err := p.client.GetVMConfig(ctx, vmid)
	if err != nil {
		return false, err
	}
	return len(config.HostPCI) > 0, nil
}

// GetRunningGPUVMs returns all running VMs with GPU passthrough
func (p *Provider) GetRunningGPUVMs(ctx context.Context) ([]*VM, error) {
	vms, err := p.client.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	var gpuVMs []*VM
	for _, vm := range vms {
		if vm.Status != VMStatusRunning {
			continue
		}

		hasGPU, err := p.HasGPUPassthrough(ctx, vm.VMID)
		if err != nil {
			continue // Skip VMs we can't inspect
		}

		if hasGPU {
			gpuVMs = append(gpuVMs, vm)
		}
	}

	return gpuVMs, nil
}

// GetModes returns all configured boot modes
func (p *Provider) GetModes() []BootMode {
	modes := make([]BootMode, 0, len(p.config.Modes))
	for name, spec := range p.config.Modes {
		modes = append(modes, BootMode{
			Name:           name,
			VMID:           spec.VMID,
			Description:    spec.Description,
			GPUPassthrough: spec.GPUPassthrough,
		})
	}
	return modes
}

// GetMode returns a specific boot mode by name
func (p *Provider) GetMode(name string) (*BootMode, error) {
	spec, ok := p.config.Modes[name]
	if !ok {
		return nil, fmt.Errorf("boot mode not found: %s", name)
	}

	return &BootMode{
		Name:           name,
		VMID:           spec.VMID,
		Description:    spec.Description,
		GPUPassthrough: spec.GPUPassthrough,
	}, nil
}

// GetCurrentMode returns the currently active boot mode (running VM)
func (p *Provider) GetCurrentMode(ctx context.Context) (*BootMode, error) {
	for name, spec := range p.config.Modes {
		vm, err := p.client.GetVM(ctx, spec.VMID)
		if err != nil {
			continue
		}
		if vm.Status == VMStatusRunning {
			return &BootMode{
				Name:           name,
				VMID:           spec.VMID,
				Description:    spec.Description,
				GPUPassthrough: spec.GPUPassthrough,
			}, nil
		}
	}
	return nil, nil // No mode active
}

// Ping checks connectivity to the Proxmox API
func (p *Provider) Ping(ctx context.Context) error {
	return p.client.Ping(ctx)
}
