// Package proxmox provides a machine provider for Proxmox VE hypervisors.
// It enables remote management of VMs for boot mode switching scenarios.
package proxmox

import "time"

// VMStatus represents the status of a Proxmox VM
type VMStatus string

const (
	VMStatusRunning VMStatus = "running"
	VMStatusStopped VMStatus = "stopped"
	VMStatusPaused  VMStatus = "paused"
	VMStatusUnknown VMStatus = "unknown"
)

// VM represents a Proxmox virtual machine
type VM struct {
	VMID        int               `json:"vmid"`
	Name        string            `json:"name"`
	Status      VMStatus          `json:"status"`
	Node        string            `json:"node"`
	Memory      int64             `json:"maxmem"`      // Maximum memory in bytes
	MemoryUsed  int64             `json:"mem"`         // Used memory in bytes
	CPUs        int               `json:"cpus"`        // Number of CPUs
	CPUUsage    float64           `json:"cpu"`         // CPU usage (0.0-1.0)
	Uptime      int64             `json:"uptime"`      // Uptime in seconds
	NetIn       int64             `json:"netin"`       // Network bytes in
	NetOut      int64             `json:"netout"`      // Network bytes out
	DiskRead    int64             `json:"diskread"`    // Disk bytes read
	DiskWrite   int64             `json:"diskwrite"`   // Disk bytes written
	Template    bool              `json:"template"`    // Is this a template?
	Tags        string            `json:"tags"`        // Comma-separated tags
	Description string            `json:"description"` // VM description
	Config      *VMConfig         `json:"-"`           // Full VM configuration (loaded separately)
	IPs         []string          `json:"-"`           // IP addresses (from QEMU agent)
}

// VMConfig represents the full configuration of a VM
type VMConfig struct {
	Name        string   `json:"name"`
	Memory      int64    `json:"memory"`
	Cores       int      `json:"cores"`
	Sockets     int      `json:"sockets"`
	CPU         string   `json:"cpu"`
	Machine     string   `json:"machine"`
	BIOS        string   `json:"bios"`
	Boot        string   `json:"boot"`
	OSType      string   `json:"ostype"`
	Description string   `json:"description"`
	HostPCI     []string `json:"-"` // PCI passthrough devices (hostpci0, hostpci1, etc.)
}

// Node represents a Proxmox cluster node
type Node struct {
	Node           string  `json:"node"`
	Status         string  `json:"status"` // "online" or "offline"
	CPU            float64 `json:"cpu"`    // CPU usage (0.0-1.0)
	MaxCPU         int     `json:"maxcpu"`
	Memory         int64   `json:"mem"`    // Used memory
	MaxMemory      int64   `json:"maxmem"` // Total memory
	Disk           int64   `json:"disk"`   // Used disk
	MaxDisk        int64   `json:"maxdisk"`
	Uptime         int64   `json:"uptime"`
	SSLFingerprint string  `json:"ssl_fingerprint"`
}

// TaskStatus represents the status of an async Proxmox task
type TaskStatus struct {
	UPID       string `json:"upid"`
	Node       string `json:"node"`
	Status     string `json:"status"` // "running", "stopped"
	ExitStatus string `json:"exitstatus"`
	Type       string `json:"type"`
	User       string `json:"user"`
	StartTime  int64  `json:"starttime"`
	EndTime    int64  `json:"endtime"`
}

// IsRunning returns true if the task is still running
func (t *TaskStatus) IsRunning() bool {
	return t.Status == "running"
}

// IsSuccessful returns true if the task completed successfully
func (t *TaskStatus) IsSuccessful() bool {
	return t.Status == "stopped" && t.ExitStatus == "OK"
}

// BootMode represents a configured boot mode
type BootMode struct {
	Name           string `json:"name"`
	VMID           int    `json:"vmid"`
	Description    string `json:"description"`
	GPUPassthrough bool   `json:"gpu_passthrough"`
}

// ProviderConfig holds Proxmox provider configuration
type ProviderConfig struct {
	Host           string              `yaml:"host"`
	Port           int                 `yaml:"port"`
	Node           string              `yaml:"node"`
	APITokenID     string              `yaml:"api_token_id"`
	APITokenSecret string              `yaml:"api_token_secret"`
	VerifySSL      bool                `yaml:"verify_ssl"`
	Timeout        time.Duration       `yaml:"timeout"`
	Modes          map[string]ModeSpec `yaml:"modes"`
}

// ModeSpec defines a boot mode in configuration
type ModeSpec struct {
	VMID           int    `yaml:"vmid"`
	Description    string `yaml:"description"`
	GPUPassthrough bool   `yaml:"gpu_passthrough"`
}

// DefaultConfig returns a ProviderConfig with sensible defaults
func DefaultConfig() ProviderConfig {
	return ProviderConfig{
		Port:      8006,
		Node:      "pve",
		VerifySSL: false, // Common in home labs with self-signed certs
		Timeout:   30 * time.Second,
		Modes:     make(map[string]ModeSpec),
	}
}
