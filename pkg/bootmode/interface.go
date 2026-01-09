package bootmode

import (
	"context"
	"fmt"
	"strings"
)

// Manager defines the interface for boot mode management
type Manager interface {
	// ListModes returns all available boot modes
	ListModes(ctx context.Context) ([]Mode, error)

	// GetMode returns a specific mode by name
	GetMode(ctx context.Context, name string) (*Mode, error)

	// GetCurrentMode returns the currently active mode, or nil if none
	GetCurrentMode(ctx context.Context) (*Mode, error)

	// CheckConflicts checks if switching to a target mode would conflict with running modes
	CheckConflicts(ctx context.Context, targetMode string) ([]ConflictInfo, error)

	// Switch changes from the current mode to the target mode
	// Returns the previous mode (if any) and any error
	Switch(ctx context.Context, targetMode string, opts SwitchOptions) (*SwitchResult, error)

	// GetModeInfo returns detailed information about a mode
	GetModeInfo(ctx context.Context, name string) (*ModeInfo, error)

	// Ping checks if the boot mode provider is reachable
	Ping(ctx context.Context) error
}

// ModeConflictError is returned when mode conflicts are detected
type ModeConflictError struct {
	TargetMode  string
	Conflicts   []ConflictInfo
}

func (e *ModeConflictError) Error() string {
	var names []string
	for _, c := range e.Conflicts {
		names = append(names, c.ConflictingMode)
	}
	return fmt.Sprintf("cannot switch to %s: conflicts with %s", e.TargetMode, strings.Join(names, ", "))
}

// GPUConflictError is returned when a GPU passthrough conflict is detected
type GPUConflictError struct {
	RunningMode string
	TargetMode  string
	Message     string
}

func (e *GPUConflictError) Error() string {
	return e.Message
}

// ModeNotFoundError is returned when a requested mode doesn't exist
type ModeNotFoundError struct {
	Mode string
}

func (e *ModeNotFoundError) Error() string {
	return "boot mode not found: " + e.Mode
}

// AlreadyActiveError is returned when trying to switch to the already active mode
type AlreadyActiveError struct {
	Mode string
}

func (e *AlreadyActiveError) Error() string {
	return "mode already active: " + e.Mode
}
