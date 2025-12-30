package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/provider"
)

// Provider implements the Provider interface for local Docker-based deployment
type Provider struct {
	// networkName is the Docker network used for forest containers
	networkName string
}

// dockerContainer represents a Docker container from inspect output
type dockerContainer struct {
	ID              string            `json:"Id"`
	Name            string            `json:"Name"`
	State           dockerState       `json:"State"`
	NetworkSettings dockerNetwork     `json:"NetworkSettings"`
	Config          dockerConfig      `json:"Config"`
	Created         string            `json:"Created"`
}

type dockerState struct {
	Status  string `json:"Status"`
	Running bool   `json:"Running"`
}

type dockerNetwork struct {
	IPAddress string                       `json:"IPAddress"`
	Networks  map[string]dockerNetworkInfo `json:"Networks"`
}

type dockerNetworkInfo struct {
	IPAddress string `json:"IPAddress"`
}

type dockerConfig struct {
	Labels map[string]string `json:"Labels"`
}

// NewProvider creates a new local Docker provider
func NewProvider() (*Provider, error) {
	// Check if Docker is available
	if err := checkDockerAvailable(); err != nil {
		return nil, err
	}

	return &Provider{
		networkName: "morpheus-local",
	}, nil
}

// checkDockerAvailable verifies that Docker is installed and running
func checkDockerAvailable() error {
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker is not available or not running: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ensureNetwork creates the Docker network if it doesn't exist
func (p *Provider) ensureNetwork(ctx context.Context) error {
	// Check if network exists
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", p.networkName)
	if err := cmd.Run(); err == nil {
		// Network already exists
		return nil
	}

	// Create network
	fmt.Printf("Creating Docker network: %s\n", p.networkName)
	cmd = exec.CommandContext(ctx, "docker", "network", "create", p.networkName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create Docker network: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// CreateServer provisions a new Docker container as a "server"
func (p *Provider) CreateServer(ctx context.Context, req provider.CreateServerRequest) (*provider.Server, error) {
	// Ensure network exists
	if err := p.ensureNetwork(ctx); err != nil {
		return nil, err
	}

	// Build docker run command
	args := []string{
		"run", "-d",
		"--name", req.Name,
		"--network", p.networkName,
		"--hostname", req.Name,
		// Expose standard ports for NATS
		"-p", "4222",  // NATS client port
		"-p", "6222",  // NATS cluster port
		"-p", "8222",  // NATS monitoring port
		"-p", "7777",  // Application port
	}

	// Add labels
	for key, value := range req.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add standard morpheus labels
	args = append(args, "--label", "morpheus.managed=true")
	args = append(args, "--label", fmt.Sprintf("morpheus.name=%s", req.Name))

	// Select image - use Ubuntu by default for compatibility with cloud-init scripts
	// For local mode, we use a simplified approach without full cloud-init
	image := "ubuntu:24.04"
	if req.Image != "" {
		image = req.Image
	}

	// For local development, we run a simple init process that keeps the container alive
	// and allows SSH-like access via docker exec
	args = append(args, image)
	// Use sleep infinity which works on both alpine and ubuntu
	args = append(args, "sleep", "infinity")

	fmt.Printf("Creating local container: %s\n", req.Name)
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return nil, fmt.Errorf("docker returned empty container ID")
	}

	// Use short ID (12 chars) if the full ID is returned
	shortID := containerID
	if len(containerID) > 12 {
		shortID = containerID[:12]
	}

	// Get container info
	return p.GetServer(ctx, shortID)
}

// GetServer retrieves container information by ID
func (p *Provider) GetServer(ctx context.Context, serverID string) (*provider.Server, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", serverID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("container not found: %s", serverID)
	}

	var containers []dockerContainer
	if err := json.Unmarshal(output, &containers); err != nil {
		return nil, fmt.Errorf("failed to parse container info: %w", err)
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("container not found: %s", serverID)
	}

	return convertContainer(&containers[0], p.networkName), nil
}

// DeleteServer removes a container
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	// Stop the container first
	stopCmd := exec.CommandContext(ctx, "docker", "stop", serverID)
	stopCmd.Run() // Ignore error if already stopped

	// Remove the container
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", serverID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete container: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// WaitForServer waits until the container is in the specified state
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state provider.ServerState) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute) // Local containers start quickly

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to reach state: %s", state)
		case <-ticker.C:
			server, err := p.GetServer(ctx, serverID)
			if err != nil {
				// If we're waiting for deletion and can't find it, that's success
				if state == provider.ServerStateDeleting || state == provider.ServerStateStopped {
					return nil
				}
				return err
			}

			if server.State == state {
				return nil
			}

			fmt.Printf("Container %s current state: %s, waiting for: %s\n",
				serverID, server.State, state)
		}
	}
}

// ListServers lists all containers with optional filters
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*provider.Server, error) {
	// Build filter args
	args := []string{"ps", "-a", "--format", "{{.ID}}"}
	
	// Add label filters
	args = append(args, "--filter", "label=morpheus.managed=true")
	for key, value := range filters {
		args = append(args, "--filter", fmt.Sprintf("label=%s=%s", key, value))
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containerIDs) == 1 && containerIDs[0] == "" {
		return []*provider.Server{}, nil
	}

	var servers []*provider.Server
	for _, id := range containerIDs {
		if id == "" {
			continue
		}
		server, err := p.GetServer(ctx, id)
		if err != nil {
			continue // Skip containers we can't inspect
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// convertContainer converts a Docker container to a provider.Server
func convertContainer(container *dockerContainer, networkName string) *provider.Server {
	// Get IP address from the morpheus network
	ipAddress := container.NetworkSettings.IPAddress
	if networkInfo, ok := container.NetworkSettings.Networks[networkName]; ok {
		ipAddress = networkInfo.IPAddress
	}

	// Parse container name (remove leading /)
	name := strings.TrimPrefix(container.Name, "/")

	// Get labels
	labels := container.Config.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	return &provider.Server{
		ID:         container.ID[:12], // Use short ID
		Name:       name,
		PublicIPv4: ipAddress, // For local, use the Docker network IP
		PublicIPv6: "",
		Location:   "local",
		State:      convertContainerState(container.State),
		Labels:     labels,
		CreatedAt:  container.Created,
	}
}

// convertContainerState converts Docker container state to provider.ServerState
func convertContainerState(state dockerState) provider.ServerState {
	switch state.Status {
	case "created":
		return provider.ServerStateStarting
	case "running":
		return provider.ServerStateRunning
	case "paused", "exited", "dead":
		return provider.ServerStateStopped
	case "removing":
		return provider.ServerStateDeleting
	default:
		return provider.ServerStateUnknown
	}
}

// GetContainerIP returns the IP address of a container on the morpheus network
func (p *Provider) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	server, err := p.GetServer(ctx, containerID)
	if err != nil {
		return "", err
	}
	return server.PublicIPv4, nil
}

// ExecInContainer runs a command inside a container (useful for local testing)
func (p *Provider) ExecInContainer(ctx context.Context, containerID string, command []string) (string, error) {
	args := append([]string{"exec", containerID}, command...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec failed: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// CleanupNetwork removes the morpheus network if no containers are using it
func (p *Provider) CleanupNetwork(ctx context.Context) error {
	// Check if any morpheus containers exist
	servers, err := p.ListServers(ctx, nil)
	if err != nil {
		return err
	}

	if len(servers) > 0 {
		// Still have containers, don't remove network
		return nil
	}

	// Remove network
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", p.networkName)
	cmd.Run() // Ignore errors (network might not exist or be in use)

	return nil
}
