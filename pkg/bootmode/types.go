// Package bootmode provides abstractions for managing machine boot modes.
// A boot mode represents a specific OS configuration that a NimsForest VR node
// can run - either Linux (CachyOS + WiVRN) or Windows (SteamLink).
package bootmode

import "time"

// OSType represents the operating system type
type OSType string

const (
	OSTypeLinux   OSType = "linux"
	OSTypeWindows OSType = "windows"
)

// Mode represents a bootable OS configuration on a VR node
type Mode struct {
	Name        string        `json:"name"`         // "linux" or "windows"
	OS          OSType        `json:"os"`           // Operating system type
	Description string        `json:"description"`  // Human-readable description
	VMID        int           `json:"vmid"`         // Proxmox VM ID
	Status      ModeStatus    `json:"status"`       // Current status
	IPAddress   string        `json:"ip_address"`   // VM IP address (when running)
	Uptime      time.Duration `json:"uptime"`       // How long running
	VRSoftware  string        `json:"vr_software"`  // "wivrn" or "steamlink"
	Services    []Service     `json:"services"`     // Running services
}

// Service represents a service running in a mode
type Service struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "active", "inactive", "failed"
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
	FromMode  string        `json:"from_mode,omitempty"`
	ToMode    string        `json:"to_mode"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	IPAddress string        `json:"ip_address,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// SwitchOptions configures the mode switch behavior
type SwitchOptions struct {
	// Force stops the current VM immediately instead of graceful shutdown
	Force bool

	// ShutdownTimeout for graceful shutdown (default: 60s)
	ShutdownTimeout time.Duration

	// StartupTimeout for VM startup (default: 120s)
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

// VRNodeConfig holds configuration for a VR-capable NimsForest node
type VRNodeConfig struct {
	// Linux VM configuration
	Linux VMConfig `yaml:"linux"`

	// Windows VM configuration
	Windows VMConfig `yaml:"windows"`

	// GPU PCI address for passthrough (e.g., "0000:01:00")
	GPUPCI string `yaml:"gpu_pci"`
}

// VMConfig holds configuration for a single VM
type VMConfig struct {
	VMID     int    `yaml:"vmid"`
	Name     string `yaml:"name"`
	Memory   int    `yaml:"memory"`    // MB
	Cores    int    `yaml:"cores"`
	DiskSize int    `yaml:"disk_size"` // GB
}

// ModeInfo contains detailed information about a mode
type ModeInfo struct {
	Mode
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	MemoryTotal int64   `json:"memory_total"`
	GPUName     string  `json:"gpu_name"`
	GPUUsage    float64 `json:"gpu_usage,omitempty"`
}
