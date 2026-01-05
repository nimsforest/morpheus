package none

import (
	"context"
	"fmt"

	"github.com/nimsforest/morpheus/pkg/machine"
)

// Provider is a no-op machine provider used when no infrastructure management is needed
type Provider struct{}

// NewProvider creates a new no-op provider
func NewProvider() (*Provider, error) {
	return &Provider{}, nil
}

// CreateServer is a no-op that returns an error
func (p *Provider) CreateServer(ctx context.Context, req machine.CreateServerRequest) (*machine.Server, error) {
	return nil, fmt.Errorf("none provider does not support creating servers")
}

// GetServer is a no-op that returns an error
func (p *Provider) GetServer(ctx context.Context, serverID string) (*machine.Server, error) {
	return nil, fmt.Errorf("none provider does not support getting servers")
}

// DeleteServer is a no-op that returns nil
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	return nil // No-op - always succeeds
}

// WaitForServer is a no-op that returns nil
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state machine.ServerState) error {
	return nil // No-op - always succeeds
}

// ListServers is a no-op that returns an empty list
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*machine.Server, error) {
	return []*machine.Server{}, nil
}
