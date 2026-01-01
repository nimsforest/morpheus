package provider

import (
	"context"
)

// Provider defines the interface for cloud infrastructure providers
type Provider interface {
	// CreateServer provisions a new server
	CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error)

	// GetServer retrieves server information by ID
	GetServer(ctx context.Context, serverID string) (*Server, error)

	// DeleteServer removes a server
	DeleteServer(ctx context.Context, serverID string) error

	// WaitForServer waits until the server is in the specified state
	WaitForServer(ctx context.Context, serverID string, state ServerState) error

	// ListServers lists all servers with optional filters
	ListServers(ctx context.Context, filters map[string]string) ([]*Server, error)
}

// LocationAwareProvider extends Provider with location-specific functionality
// This is implemented by cloud providers that have multiple locations with
// different server type availability
type LocationAwareProvider interface {
	Provider

	// CheckLocationAvailability checks if a server type is available in a location
	CheckLocationAvailability(ctx context.Context, locationName, serverTypeName string) (bool, error)

	// GetAvailableLocations returns all locations where a server type is available
	GetAvailableLocations(ctx context.Context, serverTypeName string) ([]string, error)

	// FilterLocationsByServerType filters locations to only those supporting the server type
	// Returns (supported locations, unsupported locations, error)
	FilterLocationsByServerType(ctx context.Context, locations []string, serverTypeName string) ([]string, []string, error)
}

// CreateServerRequest contains parameters for server creation
type CreateServerRequest struct {
	Name       string
	ServerType string
	Image      string
	Location   string
	SSHKeys    []string
	UserData   string
	Labels     map[string]string
}

// Server represents a provisioned server
type Server struct {
	ID         string
	Name       string
	PublicIPv4 string
	PublicIPv6 string
	Location   string
	State      ServerState
	Labels     map[string]string
	CreatedAt  string
}

// ServerState represents the current state of a server
type ServerState string

const (
	ServerStateStarting ServerState = "starting"
	ServerStateRunning  ServerState = "running"
	ServerStateStopped  ServerState = "stopped"
	ServerStateDeleting ServerState = "deleting"
	ServerStateUnknown  ServerState = "unknown"
)
