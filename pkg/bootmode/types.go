// Package bootmode provides abstractions for managing machine boot modes.
// A boot mode represents a specific OS/workload configuration that a machine
// can run, typically implemented as different VMs in a hypervisor.
package bootmode

import "time"

// GPUMode represents how a boot mode uses the GPU
type GPUMode string

const (
	// GPUModeExclusive means the VM needs exclusive GPU access (e.g., VR streaming)
	GPUModeExclusive GPUMode = "exclusive"
	// GPUModeShared means the VM uses GPU but can potentially share (e.g., compute workloads)
	GPUModeShared GPUMode = "shared"
	// GPUModeNone means the VM doesn't need GPU access
	GPUModeNone GPUMode = "none"
)

// Mode represents a bootable configuration
type Mode struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Provider      string            `json:"provider"`    // e.g., "proxmox"
	ProviderID    string            `json:"provider_id"` // e.g., VM ID
	GPUMode       GPUMode           `json:"gpu_mode"`
	ConflictsWith []string          `json:"conflicts_with,omitempty"`
	Status        ModeStatus        `json:"status"`
	IPAddresses   []string          `json:"ip_addresses,omitempty"`
	Uptime        time.Duration     `json:"uptime,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// NeedsGPU returns true if this mode requires GPU access
func (m *Mode) NeedsGPU() bool {
	return m.GPUMode == GPUModeExclusive || m.GPUMode == GPUModeShared
}

// NeedsExclusiveGPU returns true if this mode requires exclusive GPU access
func (m *Mode) NeedsExclusiveGPU() bool {
	return m.GPUMode == GPUModeExclusive
}

// ModeStatus represents the current status of a boot mode
type ModeStatus string

const (
	ModeStatusRunning  ModeStatus = "running"
	ModeStatusStopped  ModeStatus = "stopped"
	ModeStatusStarting ModeStatus = "starting"
	ModeStatusStopping ModeStatus = "stopping"
	ModeStatusUnknown  ModeStatus = "unknown"
)

// SwitchResult contains the result of a mode switch operation
type SwitchResult struct {
	FromMode    string        `json:"from_mode,omitempty"`
	ToMode      string        `json:"to_mode"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	IPAddresses []string      `json:"ip_addresses,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// SwitchOptions configures the mode switch behavior
type SwitchOptions struct {
	// Force stops the current VM immediately instead of graceful shutdown
	Force bool

	// Timeout for graceful shutdown (default: 60s)
	ShutdownTimeout time.Duration

	// Timeout for startup (default: 120s)
	StartupTimeout time.Duration

	// WaitForNetwork waits for the VM to get an IP address
	WaitForNetwork bool

	// DryRun only shows what would happen without making changes
	DryRun bool
}

// DefaultSwitchOptions returns sensible default switch options
func DefaultSwitchOptions() SwitchOptions {
	return SwitchOptions{
		Force:           false,
		ShutdownTimeout: 60 * time.Second,
		StartupTimeout:  120 * time.Second,
		WaitForNetwork:  true,
		DryRun:          false,
	}
}

// ModeInfo contains detailed information about a mode
type ModeInfo struct {
	Mode
	CPUUsage    float64     `json:"cpu_usage,omitempty"`
	MemoryUsage float64     `json:"memory_usage,omitempty"`
	MemoryTotal int64       `json:"memory_total,omitempty"`
	GPUDevices  []GPUDevice `json:"gpu_devices,omitempty"`
}

// ConflictInfo describes a mode conflict
type ConflictInfo struct {
	TargetMode      string   `json:"target_mode"`
	ConflictingMode string   `json:"conflicting_mode"`
	Reason          string   `json:"reason"`
	Alternatives    []string `json:"alternatives,omitempty"`
}

// GPUDevice represents a GPU passed through to a VM
type GPUDevice struct {
	Address     string `json:"address"`     // PCI address (e.g., "0000:01:00.0")
	Vendor      string `json:"vendor"`      // e.g., "NVIDIA"
	Model       string `json:"model"`       // e.g., "RTX 4090"
	VRAMBytes   int64  `json:"vram_bytes"`
}
