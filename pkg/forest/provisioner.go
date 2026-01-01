package forest

import (
	"context"
	"fmt"
	"net"
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
	ForestID   string
	Size       string // small, medium, large
	Location   string
	Role       cloudinit.NodeRole
	ServerType string // Provider-specific server type
	Image      string // OS image to use
}

// Provision creates a new forest with the specified configuration
func (p *Provisioner) Provision(ctx context.Context, req ProvisionRequest) error {
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

	fmt.Printf("\n📦 Step 1/%d: Provisioning machines\n", 2+nodeCount)
	fmt.Printf("    Creating %d machine%s...\n", nodeCount, plural(nodeCount))

	// Provision nodes
	var provisionedServers []*provider.Server
	for i := 0; i < nodeCount; i++ {
		nodeName := fmt.Sprintf("%s-node-%d", req.ForestID, i+1)
		
		fmt.Printf("\n   Machine %d/%d: %s\n", i+1, nodeCount, nodeName)

		server, err := p.provisionNode(ctx, req, nodeName, i)
		if err != nil {
			// Rollback on failure
			fmt.Printf("\n❌ Provisioning failed: %s\n", err)
			fmt.Printf("🔄 Rolling back %d machine%s...\n", len(provisionedServers), plural(len(provisionedServers)))
			p.rollback(ctx, req.ForestID, provisionedServers)
			return fmt.Errorf("failed to provision node %s: %w", nodeName, err)
		}

		provisionedServers = append(provisionedServers, server)

		// Update the actual location used (may differ from requested if fallback occurred)
		forest.Location = server.Location

		// Register node in registry (IPv6 only)
		node := &Node{
			ID:       server.ID,
			ForestID: req.ForestID,
			Role:     string(req.Role),
			IP:       server.PublicIPv6,
			Location: server.Location,
			Status:   "active",
			Metadata: server.Labels,
		}

		if err := p.registry.RegisterNode(node); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to register node in registry: %s\n", err)
		}

		fmt.Printf("   ✅ Machine %d ready (IPv6: %s)\n", i+1, server.PublicIPv6)
	}

	// Update forest status and location
	fmt.Printf("\n📋 Step %d/%d: Finalizing registration\n", 2+nodeCount, 2+nodeCount)
	if err := p.registry.UpdateForest(forest); err != nil {
		fmt.Printf("   ⚠️  Warning: failed to update forest: %s\n", err)
	}
	if err := p.registry.UpdateForestStatus(req.ForestID, "active"); err != nil {
		fmt.Printf("   ⚠️  Warning: failed to update forest status: %s\n", err)
	}
	fmt.Printf("   ✅ Forest registered and ready\n")

	return nil
}

// provisionNode provisions a single node
func (p *Provisioner) provisionNode(ctx context.Context, req ProvisionRequest, nodeName string, index int) (*provider.Server, error) {
	// Generate cloud-init script
	fmt.Printf("      ⏳ Configuring cloud-init...\n")
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

	// Determine server type and image based on provisioning context
	serverType := req.ServerType
	if serverType == "" {
		// Fallback to legacy config if available
		if p.config.Infrastructure.Defaults != nil && p.config.Infrastructure.Defaults.ServerType != "" {
			serverType = p.config.Infrastructure.Defaults.ServerType
		} else {
			return nil, fmt.Errorf("server type not specified in request and no default configured")
		}
	}
	
	image := req.Image
	if image == "" {
		// Default to Ubuntu 24.04 if not specified
		image = "ubuntu-24.04"
		// Check legacy config
		if p.config.Infrastructure.Defaults != nil && p.config.Infrastructure.Defaults.Image != "" {
			image = p.config.Infrastructure.Defaults.Image
		}
	}

	// Create server
	fmt.Printf("      ⏳ Creating server on cloud provider...\n")
	createReq := provider.CreateServerRequest{
		Name:       nodeName,
		ServerType: serverType,
		Image:      image,
		Location:   req.Location,
		SSHKeys:    []string{p.config.GetSSHKeyName()},
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

	fmt.Printf("      ✓ Server created (ID: %s)\n", server.ID)
	fmt.Printf("      ⏳ Waiting for server to boot...\n")

	// Wait for server to be running
	if err := p.provider.WaitForServer(ctx, server.ID, provider.ServerStateRunning); err != nil {
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	// Fetch updated server info to get IP address
	server, err = p.provider.GetServer(ctx, server.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	fmt.Printf("      ✓ Server running\n")
	fmt.Printf("      ⏳ Verifying SSH connectivity...\n")
	
	// Wait for infrastructure to be ready (SSH accessible, cloud-init complete)
	if err := p.waitForInfrastructureReady(ctx, server); err != nil {
		return nil, fmt.Errorf("infrastructure readiness check failed: %w", err)
	}
	
	fmt.Printf("      ✓ SSH accessible\n")

	return server, nil
}

// waitForInfrastructureReady waits until the server's infrastructure is ready
// This checks SSH connectivity as an indicator that cloud-init has progressed
// far enough for the server to be usable
func (p *Provisioner) waitForInfrastructureReady(ctx context.Context, server *provider.Server) error {
	// IPv6-only
	if server.PublicIPv6 == "" {
		return fmt.Errorf("server has no IPv6 address")
	}

	timeout := p.config.Provisioning.GetReadinessTimeout()
	interval := p.config.Provisioning.GetReadinessInterval()
	sshPort := p.config.Provisioning.SSHPort

	// Format IPv6 addresses with brackets for SSH
	addr := fmt.Sprintf("[%s]:%d", server.PublicIPv6, sshPort)

	deadline := time.Now().Add(timeout)
	attempts := 0
	lastDotPrint := time.Now()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		attempts++

		// Check SSH port connectivity
		if err := p.checkSSHConnectivity(addr); err == nil {
			return nil
		}

		// Print progress dots every 5 seconds
		if time.Since(lastDotPrint) >= 5*time.Second {
			fmt.Printf(".")
			lastDotPrint = time.Now()
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return fmt.Errorf("timeout after %d attempts (max %s)", attempts, timeout)
}

// checkSSHConnectivity attempts a TCP connection to verify SSH is accepting connections
func (p *Provisioner) checkSSHConnectivity(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// Teardown removes a forest and all its resources
func (p *Provisioner) Teardown(ctx context.Context, forestID string) error {
	fmt.Printf("🗑️  Tearing down forest: %s\n\n", forestID)

	// Get all nodes for this forest
	nodes, err := p.registry.GetNodes(forestID)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	// Delete all servers
	if len(nodes) > 0 {
		fmt.Printf("Deleting %d machine%s...\n", len(nodes), plural(len(nodes)))
		for i, node := range nodes {
			fmt.Printf("   [%d/%d] Deleting %s...", i+1, len(nodes), node.ID)

			if err := p.provider.DeleteServer(ctx, node.ID); err != nil {
				fmt.Printf(" ⚠️  Warning: %s\n", err)
			} else {
				fmt.Printf(" ✅\n")
			}
		}
	}

	// Remove from registry
	fmt.Printf("\nCleaning up registry...")
	if err := p.registry.DeleteForest(forestID); err != nil {
		fmt.Printf(" ⚠️  Warning: %s\n", err)
	} else {
		fmt.Printf(" ✅\n")
	}

	return nil
}

// rollback removes all provisioned servers on failure
func (p *Provisioner) rollback(ctx context.Context, forestID string, servers []*provider.Server) {
	for i, server := range servers {
		fmt.Printf("   🗑️  Deleting machine %d/%d...\n", i+1, len(servers))
		if err := p.provider.DeleteServer(ctx, server.ID); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to delete server %s: %s\n", server.ID, err)
		} else {
			fmt.Printf("   ✅ Machine deleted\n")
		}
	}

	// Remove from registry
	p.registry.DeleteForest(forestID)
	fmt.Printf("   ✅ Rollback complete\n")
}

// getNodeCount returns the number of nodes for a given forest size
func getNodeCount(size string) int {
	switch size {
	case "small":
		return 1
	case "medium":
		return 3
	case "large":
		return 5
	default:
		return 1
	}
}

// plural returns "s" if count is not 1, empty string otherwise
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
