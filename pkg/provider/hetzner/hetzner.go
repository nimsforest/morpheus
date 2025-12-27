package hetzner

import (
	"context"
	"fmt"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/yourusername/morpheus/pkg/provider"
)

// Provider implements the Provider interface for Hetzner Cloud
type Provider struct {
	client *hcloud.Client
}

// NewProvider creates a new Hetzner Cloud provider
func NewProvider(apiToken string) (*Provider, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	client := hcloud.NewClient(hcloud.WithToken(apiToken))
	
	return &Provider{
		client: client,
	}, nil
}

// CreateServer provisions a new Hetzner Cloud server
func (p *Provider) CreateServer(ctx context.Context, req provider.CreateServerRequest) (*provider.Server, error) {
	// Resolve server type
	serverType, _, err := p.client.ServerType.GetByName(ctx, req.ServerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %w", err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", req.ServerType)
	}

	// Resolve image
	image, _, err := p.client.Image.GetByName(ctx, req.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return nil, fmt.Errorf("image not found: %s", req.Image)
	}

	// Resolve location
	location, _, err := p.client.Location.GetByName(ctx, req.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}
	if location == nil {
		return nil, fmt.Errorf("location not found: %s", req.Location)
	}

	// Resolve SSH keys
	var sshKeys []*hcloud.SSHKey
	for _, keyName := range req.SSHKeys {
		key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH key %s: %w", keyName, err)
		}
		if key == nil {
			return nil, fmt.Errorf("SSH key not found: %s", keyName)
		}
		sshKeys = append(sshKeys, key)
	}

	// Create server
	createOpts := hcloud.ServerCreateOpts{
		Name:       req.Name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		SSHKeys:    sshKeys,
		UserData:   req.UserData,
		Labels:     req.Labels,
		StartAfterCreate: hcloud.Ptr(true),
	}

	result, _, err := p.client.Server.Create(ctx, createOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return convertServer(result.Server), nil
}

// GetServer retrieves server information by ID
func (p *Provider) GetServer(ctx context.Context, serverID string) (*provider.Server, error) {
	server, _, err := p.client.Server.GetByID(ctx, parseServerID(serverID))
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	return convertServer(server), nil
}

// DeleteServer removes a server
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	server, _, err := p.client.Server.GetByID(ctx, parseServerID(serverID))
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverID)
	}

	_, _, err = p.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}

// WaitForServer waits until the server is in the specified state
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state provider.ServerState) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for server to reach state: %s", state)
		case <-ticker.C:
			server, err := p.GetServer(ctx, serverID)
			if err != nil {
				return err
			}

			if server.State == state {
				return nil
			}

			// Log progress
			fmt.Printf("Server %s current state: %s, waiting for: %s\n", 
				serverID, server.State, state)
		}
	}
}

// ListServers lists all servers with optional filters
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*provider.Server, error) {
	opts := hcloud.ServerListOpts{}
	
	if len(filters) > 0 {
		opts.LabelSelector = formatLabelSelector(filters)
	}

	servers, err := p.client.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	result := make([]*provider.Server, len(servers))
	for i, server := range servers {
		result[i] = convertServer(server)
	}

	return result, nil
}

// Helper functions

func convertServer(server *hcloud.Server) *provider.Server {
	var publicIPv4, publicIPv6 string
	if server.PublicNet.IPv4.IP != nil {
		publicIPv4 = server.PublicNet.IPv4.IP.String()
	}
	if server.PublicNet.IPv6.IP != nil {
		publicIPv6 = server.PublicNet.IPv6.IP.String()
	}

	return &provider.Server{
		ID:         fmt.Sprintf("%d", server.ID),
		Name:       server.Name,
		PublicIPv4: publicIPv4,
		PublicIPv6: publicIPv6,
		Location:   server.Datacenter.Location.Name,
		State:      convertServerState(server.Status),
		Labels:     server.Labels,
		CreatedAt:  server.Created.Format(time.RFC3339),
	}
}

func convertServerState(status hcloud.ServerStatus) provider.ServerState {
	switch status {
	case hcloud.ServerStatusStarting:
		return provider.ServerStateStarting
	case hcloud.ServerStatusRunning:
		return provider.ServerStateRunning
	case hcloud.ServerStatusStopping, hcloud.ServerStatusOff:
		return provider.ServerStateStopped
	case hcloud.ServerStatusDeleting:
		return provider.ServerStateDeleting
	default:
		return provider.ServerStateUnknown
	}
}

func parseServerID(id string) int64 {
	var serverID int64
	fmt.Sscanf(id, "%d", &serverID)
	return serverID
}

func formatLabelSelector(filters map[string]string) string {
	selector := ""
	first := true
	for key, value := range filters {
		if !first {
			selector += ","
		}
		selector += fmt.Sprintf("%s=%s", key, value)
		first = false
	}
	return selector
}
