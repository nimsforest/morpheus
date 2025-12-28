package forest

import (
	"context"
	"fmt"
	"time"

	"github.com/nimsforest/morpheus/pkg/cloudinit"
	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/provider"
)

// Provisioner handles forest provisioning
type Provisioner struct {
	provider provider.Provider
	registry *Registry
	config   *config.Config
}

// NewProvisioner creates a new forest provisioner
func NewProvisioner(p provider.Provider, r *Registry, cfg *config.Config) *Provisioner {
	return &Provisioner{
		provider: p,
		registry: r,
		config:   cfg,
	}
}

// ProvisionRequest contains parameters for provisioning a forest
type ProvisionRequest struct {
	ForestID string
	Size     string // wood, forest, jungle
	Location string
	Role     cloudinit.NodeRole
}

// Provision creates a new forest with the specified configuration
func (p *Provisioner) Provision(ctx context.Context, req ProvisionRequest) error {
	fmt.Printf("Starting forest provisioning: %s (size: %s, location: %s)\n",
		req.ForestID, req.Size, req.Location)

	// Register forest
	forest := &Forest{
		ID:       req.ForestID,
		Size:     req.Size,
		Location: req.Location,
		Provider: p.config.Infrastructure.Provider,
		Status:   "provisioning",
	}

	if err := p.registry.RegisterForest(forest); err != nil {
		return fmt.Errorf("failed to register forest: %w", err)
	}

	// Determine number of nodes based on size
	nodeCount := getNodeCount(req.Size)

	fmt.Printf("Provisioning %d node(s)...\n", nodeCount)

	// Provision nodes
	var provisionedServers []*provider.Server
	for i := 0; i < nodeCount; i++ {
		nodeName := fmt.Sprintf("%s-node-%d", req.ForestID, i+1)

		server, err := p.provisionNode(ctx, req, nodeName, i)
		if err != nil {
			// Rollback on failure
			fmt.Printf("Provisioning failed: %s. Rolling back...\n", err)
			p.rollback(ctx, req.ForestID, provisionedServers)
			return fmt.Errorf("failed to provision node %s: %w", nodeName, err)
		}

		provisionedServers = append(provisionedServers, server)

		// Register node in registry
		node := &Node{
			ID:       server.ID,
			ForestID: req.ForestID,
			Role:     string(req.Role),
			IP:       server.PublicIPv4,
			Location: server.Location,
			Status:   "active",
			Metadata: server.Labels,
		}

		if err := p.registry.RegisterNode(node); err != nil {
			fmt.Printf("Warning: failed to register node in registry: %s\n", err)
		}

		fmt.Printf("✓ Node %s provisioned successfully (IP: %s)\n", nodeName, server.PublicIPv4)
	}

	// Update forest status
	if err := p.registry.UpdateForestStatus(req.ForestID, "active"); err != nil {
		fmt.Printf("Warning: failed to update forest status: %s\n", err)
	}

	fmt.Printf("✓ Forest %s provisioned successfully!\n", req.ForestID)
	return nil
}

// provisionNode provisions a single node
func (p *Provisioner) provisionNode(ctx context.Context, req ProvisionRequest, nodeName string, index int) (*provider.Server, error) {
	// Generate cloud-init script
	cloudInitData := cloudinit.TemplateData{
		NodeRole:    req.Role,
		ForestID:    req.ForestID,
		RegistryURL: p.config.Integration.RegistryURL,
		CallbackURL: p.config.Integration.NimsForestURL,
	}

	userData, err := cloudinit.Generate(req.Role, cloudInitData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	// Create server
	createReq := provider.CreateServerRequest{
		Name:       nodeName,
		ServerType: p.config.Infrastructure.Defaults.ServerType,
		Image:      p.config.Infrastructure.Defaults.Image,
		Location:   req.Location,
		SSHKeys:    []string{p.config.Infrastructure.Defaults.SSHKey},
		UserData:   userData,
		Labels: map[string]string{
			"managed-by": "morpheus",
			"forest-id":  req.ForestID,
			"role":       string(req.Role),
		},
	}

	server, err := p.provider.CreateServer(ctx, createReq)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Server %s created, waiting for it to be ready...\n", server.ID)

	// Wait for server to be running
	if err := p.provider.WaitForServer(ctx, server.ID, provider.ServerStateRunning); err != nil {
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	// Give cloud-init some time to run
	fmt.Printf("Server running, waiting for cloud-init to complete...\n")
	time.Sleep(30 * time.Second)

	return server, nil
}

// Teardown removes a forest and all its resources
func (p *Provisioner) Teardown(ctx context.Context, forestID string) error {
	fmt.Printf("Tearing down forest: %s\n", forestID)

	// Get all nodes for this forest
	nodes, err := p.registry.GetNodes(forestID)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	// Delete all servers
	for _, node := range nodes {
		fmt.Printf("Deleting server %s (IP: %s)...\n", node.ID, node.IP)

		if err := p.provider.DeleteServer(ctx, node.ID); err != nil {
			fmt.Printf("Warning: failed to delete server %s: %s\n", node.ID, err)
		} else {
			fmt.Printf("✓ Server %s deleted\n", node.ID)
		}
	}

	// Remove from registry
	if err := p.registry.DeleteForest(forestID); err != nil {
		fmt.Printf("Warning: failed to remove forest from registry: %s\n", err)
	}

	fmt.Printf("✓ Forest %s torn down successfully\n", forestID)
	return nil
}

// rollback removes all provisioned servers on failure
func (p *Provisioner) rollback(ctx context.Context, forestID string, servers []*provider.Server) {
	fmt.Printf("Rolling back %d server(s)...\n", len(servers))

	for _, server := range servers {
		if err := p.provider.DeleteServer(ctx, server.ID); err != nil {
			fmt.Printf("Warning: failed to delete server %s during rollback: %s\n",
				server.ID, err)
		}
	}

	// Remove from registry
	p.registry.DeleteForest(forestID)
}

// getNodeCount returns the number of nodes for a given forest size
func getNodeCount(size string) int {
	switch size {
	case "wood":
		return 1
	case "forest":
		return 3
	case "jungle":
		return 5
	default:
		return 1
	}
}
