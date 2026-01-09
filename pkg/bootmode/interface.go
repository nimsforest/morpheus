package bootmode

import (
	"context"
	"fmt"
)

// Manager defines the interface for boot mode management on VR nodes
type Manager interface {
	// ListModes returns all available boot modes (linux, windows)
	ListModes(ctx context.Context) ([]Mode, error)

	// GetMode returns a specific mode by name ("linux" or "windows")
	GetMode(ctx context.Context, name string) (*Mode, error)

	// GetCurrentMode returns the currently running mode, or nil if none
	GetCurrentMode(ctx context.Context) (*Mode, error)

	// Switch changes from the current mode to the target mode
	Switch(ctx context.Context, targetMode string, opts SwitchOptions) (*SwitchResult, error)

	// GetModeInfo returns detailed information about a mode
	GetModeInfo(ctx context.Context, name string) (*ModeInfo, error)

	// Ping checks if the Proxmox host is reachable
	Ping(ctx context.Context) error
}

// ModeNotFoundError is returned when a requested mode doesn't exist
type ModeNotFoundError struct {
	Mode string
}

func (e *ModeNotFoundError) Error() string {
	return fmt.Sprintf("mode not found: %s (valid modes: linux, windows)", e.Mode)
}

// AlreadyActiveError is returned when trying to switch to the already active mode
type AlreadyActiveError struct {
	Mode string
}

func (e *AlreadyActiveError) Error() string {
	return fmt.Sprintf("already in %s mode", e.Mode)
}

// SwitchError is returned when mode switching fails
type SwitchError struct {
	FromMode string
	ToMode   string
	Reason   string
}

func (e *SwitchError) Error() string {
	if e.FromMode == "" {
		return fmt.Sprintf("failed to switch to %s: %s", e.ToMode, e.Reason)
	}
	return fmt.Sprintf("failed to switch from %s to %s: %s", e.FromMode, e.ToMode, e.Reason)
}
