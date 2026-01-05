package machine

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
	// EnableIPv4 enables IPv4 in addition to IPv6
	// By default, servers are IPv6-only to save costs (IPv4 costs extra on Hetzner)
	EnableIPv4 bool
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

// GetPreferredIP returns the preferred IP address for connectivity.
// It prefers IPv6 over IPv4, falling back to IPv4 if IPv6 is not available.
func (s *Server) GetPreferredIP() string {
	if s.PublicIPv6 != "" {
		return s.PublicIPv6
	}
	return s.PublicIPv4
}

// GetFallbackIP returns the fallback IP address (IPv4 if IPv6 is preferred).
// Returns empty string if no fallback is available.
func (s *Server) GetFallbackIP() string {
	if s.PublicIPv6 != "" && s.PublicIPv4 != "" {
		return s.PublicIPv4
	}
	return ""
}

// HasIPv4 returns true if the server has an IPv4 address
func (s *Server) HasIPv4() bool {
	return s.PublicIPv4 != ""
}

// HasIPv6 returns true if the server has an IPv6 address
func (s *Server) HasIPv6() bool {
	return s.PublicIPv6 != ""
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
