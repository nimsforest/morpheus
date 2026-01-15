package forest

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/cloudinit"
	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/dns"
	"github.com/nimsforest/morpheus/pkg/machine"
	"github.com/nimsforest/morpheus/pkg/sshutil"
	"github.com/nimsforest/morpheus/pkg/storage"
)

// Provisioner handles forest provisioning
type Provisioner struct {
	machine machine.Provider
	storage storage.Registry
	dns     dns.Provider
	config  *config.Config
}

// NewProvisioner creates a new forest provisioner
// Accepts any machine provider that implements the machine.Provider interface
// and any storage that implements the storage.Registry interface
func NewProvisioner(m machine.Provider, s storage.Registry, cfg *config.Config) *Provisioner {
	return &Provisioner{
		machine: m,
		storage: s,
		config:  cfg,
	}
}

// NewProvisionerWithDNS creates a new forest provisioner with DNS support
func NewProvisionerWithDNS(m machine.Provider, s storage.Registry, d dns.Provider, cfg *config.Config) *Provisioner {
	return &Provisioner{
		machine: m,
		storage: s,
		dns:     d,
		config:  cfg,
	}
}

// ProvisionRequest contains parameters for provisioning a forest
type ProvisionRequest struct {
	ForestID   string
	NodeCount  int // Number of nodes to provision
	Location   string
	ServerType string // Provider-specific server type
	Image      string // OS image to use
}

// Provision creates a new forest with the specified configuration
func (p *Provisioner) Provision(ctx context.Context, req ProvisionRequest) error {
	// Validate node count
	nodeCount := req.NodeCount
	if nodeCount <= 0 {
		nodeCount = 1 // Default to single node
	}

	// Register forest
	forest := &storage.Forest{
		ID:        req.ForestID,
		NodeCount: nodeCount,
		Location:  req.Location,
		Provider:  p.config.GetMachineProvider(),
		Status:    "provisioning",
	}

	if err := p.storage.RegisterForest(forest); err != nil {
		return fmt.Errorf("failed to register forest: %w", err)
	}

	fmt.Printf("\n📦 Step 1/%d: Provisioning machines\n", 2+nodeCount)
	fmt.Printf("    Creating %d machine%s...\n", nodeCount, plural(nodeCount))

	// Provision nodes
	var provisionedServers []*machine.Server
	for i := 0; i < nodeCount; i++ {
		nodeName := fmt.Sprintf("%s-node-%d", req.ForestID, i+1)

		fmt.Printf("\n   Machine %d/%d: %s\n", i+1, nodeCount, nodeName)

		server, err := p.provisionNode(ctx, req, nodeName, i, nodeCount, func(s *machine.Server) {
			// Register node immediately after server creation (before SSH verification)
			// This ensures teardown can find and delete it even if interrupted
			// Store both IPv4 and IPv6 addresses for flexible connectivity
			node := &storage.Node{
				ID:       s.ID,
				ForestID: req.ForestID,
				IP:       s.GetPreferredIP(), // Primary IP (IPv6 preferred)
				IPv6:     s.PublicIPv6,
				IPv4:     s.PublicIPv4,
				Location: s.Location,
				Status:   "provisioning", // Will be updated to "active" after SSH verification
				Metadata: s.Labels,
			}
			if err := p.storage.RegisterNode(node); err != nil {
				fmt.Printf("   ⚠️  Warning: failed to register node in storage: %s\n", err)
			}
		})
		if err != nil {
			// Rollback on failure - nodes are already registered, so teardown will find them
			fmt.Printf("\n❌ Provisioning failed: %s\n", err)
			fmt.Printf("🔄 Rolling back %d machine%s...\n", len(provisionedServers)+1, plural(len(provisionedServers)+1))
			p.rollback(ctx, req.ForestID, provisionedServers)
			return fmt.Errorf("failed to provision node %s: %w", nodeName, err)
		}

		provisionedServers = append(provisionedServers, server)

		// Update the actual location used (may differ from requested if fallback occurred)
		forest.Location = server.Location

		// Update node status to active now that SSH verification passed
		if err := p.storage.UpdateNodeStatus(req.ForestID, server.ID, "active"); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to update node status: %s\n", err)
		}

		// Display IP address info
		if server.PublicIPv6 != "" && server.PublicIPv4 != "" {
			fmt.Printf("   ✅ Machine %d ready (IPv6: %s, IPv4: %s)\n", i+1, server.PublicIPv6, server.PublicIPv4)
		} else if server.PublicIPv6 != "" {
			fmt.Printf("   ✅ Machine %d ready (IPv6: %s)\n", i+1, server.PublicIPv6)
		} else {
			fmt.Printf("   ✅ Machine %d ready (IPv4: %s)\n", i+1, server.PublicIPv4)
		}

		// Create DNS records if DNS provider is configured
		if p.dns != nil && p.config.DNS.Domain != "" {
			p.createDNSRecords(ctx, req.ForestID, server, i)
		}
	}

	// Update forest status and location
	fmt.Printf("\n📋 Step %d/%d: Finalizing registration\n", 2+nodeCount, 2+nodeCount)
	if err := p.storage.UpdateForest(forest); err != nil {
		fmt.Printf("   ⚠️  Warning: failed to update forest: %s\n", err)
	}
	if err := p.storage.UpdateForestStatus(req.ForestID, "active"); err != nil {
		fmt.Printf("   ⚠️  Warning: failed to update forest status: %s\n", err)
	}
	fmt.Printf("   ✅ Forest registered and ready\n")

	return nil
}

// createDNSRecords creates DNS records for a provisioned server
func (p *Provisioner) createDNSRecords(ctx context.Context, forestID string, server *machine.Server, nodeIndex int) {
	domain := p.config.DNS.Domain
	ttl := p.config.DNS.TTL

	// Create A record if IPv4 is available
	if server.PublicIPv4 != "" {
		recordName := fmt.Sprintf("%s-node-%d", forestID, nodeIndex+1)
		_, err := p.dns.CreateRecord(ctx, dns.CreateRecordRequest{
			Domain: domain,
			Name:   recordName,
			Type:   dns.RecordTypeA,
			Value:  server.PublicIPv4,
			TTL:    ttl,
		})
		if err != nil {
			fmt.Printf("   ⚠️  Warning: failed to create A record: %s\n", err)
		} else {
			fmt.Printf("   🌐 DNS: %s.%s -> %s\n", recordName, domain, server.PublicIPv4)
		}
	}

	// Create AAAA record if IPv6 is available
	if server.PublicIPv6 != "" {
		recordName := fmt.Sprintf("%s-node-%d", forestID, nodeIndex+1)
		_, err := p.dns.CreateRecord(ctx, dns.CreateRecordRequest{
			Domain: domain,
			Name:   recordName,
			Type:   dns.RecordTypeAAAA,
			Value:  server.PublicIPv6,
			TTL:    ttl,
		})
		if err != nil {
			fmt.Printf("   ⚠️  Warning: failed to create AAAA record: %s\n", err)
		} else {
			fmt.Printf("   🌐 DNS: %s.%s -> %s\n", recordName, domain, server.PublicIPv6)
		}
	}
}

// provisionNode provisions a single node
// The onCreated callback is called immediately after the server is created (before SSH verification)
// to allow early registration for cleanup purposes
func (p *Provisioner) provisionNode(ctx context.Context, req ProvisionRequest, nodeName string, index int, nodeCount int, onCreated func(*machine.Server)) (*machine.Server, error) {
	// Generate unique node ID for this node
	nodeID := nodeName // e.g., "myforest-node-1"

	// Generate cloud-init script
	fmt.Printf("      ⏳ Configuring cloud-init...\n")
	cloudInitData := cloudinit.TemplateData{
		ForestID:              req.ForestID,
		RegistryURL:           p.config.Integration.RegistryURL,
		CallbackURL:           p.config.Integration.NimsForestURL,
		NimsForestInstall:     p.config.Integration.NimsForestInstall,
		NimsForestDownloadURL: p.config.Integration.NimsForestDownloadURL,

		// Node identification (for embedded NATS peer discovery)
		NodeID:    nodeID,
		NodeIndex: index,
		NodeCount: nodeCount,

		// StorageBox mount for shared registry (enables NATS peer discovery)
		StorageBoxHost:     p.config.Storage.StorageBox.Host,
		StorageBoxUser:     p.config.Storage.StorageBox.Username,
		StorageBoxPassword: p.config.Storage.StorageBox.Password,
	}

	// Fall back to legacy config if new config is empty
	if cloudInitData.StorageBoxHost == "" {
		cloudInitData.StorageBoxHost = p.config.Registry.StorageBoxHost
	}
	if cloudInitData.StorageBoxUser == "" {
		cloudInitData.StorageBoxUser = p.config.Registry.Username
	}
	if cloudInitData.StorageBoxPassword == "" {
		cloudInitData.StorageBoxPassword = p.config.Registry.Password
	}

	userData, err := cloudinit.Generate(cloudInitData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	// Determine server type and image
	serverType := req.ServerType
	if serverType == "" {
		serverType = p.config.GetServerType()
	}

	image := req.Image
	if image == "" {
		image = p.config.GetImage()
	}

	// Create server
	sshKeyName := p.config.GetSSHKeyName()
	fmt.Printf("      ⏳ Creating server on cloud provider...\n")
	fmt.Printf("      SSH key: %s\n", sshKeyName)
	createReq := machine.CreateServerRequest{
		Name:       nodeName,
		ServerType: serverType,
		Image:      image,
		Location:   req.Location,
		SSHKeys:    []string{sshKeyName},
		UserData:   userData,
		Labels: map[string]string{
			"managed-by": "morpheus",
			"forest-id":  req.ForestID,
		},
		EnableIPv4: p.config.IsIPv4Enabled(),
	}

	server, err := p.machine.CreateServer(ctx, createReq)
	if err != nil {
		return nil, err
	}

	fmt.Printf("      ✓ Server created (ID: %s)\n", server.ID)

	// Store the location immediately
	server.Location = req.Location

	// Register node immediately so teardown can find it even if interrupted
	if onCreated != nil {
		onCreated(server)
	}

	fmt.Printf("      ⏳ Waiting for server to boot...\n")

	// Wait for server to be running
	if err := p.machine.WaitForServer(ctx, server.ID, machine.ServerStateRunning); err != nil {
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	// Fetch updated server info to get IP address
	server, err = p.machine.GetServer(ctx, server.ID)
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
func (p *Provisioner) waitForInfrastructureReady(ctx context.Context, server *machine.Server) error {
	// Check that we have at least one IP address
	if server.PublicIPv6 == "" && server.PublicIPv4 == "" {
		return fmt.Errorf("server has no IP address")
	}

	timeout := p.config.Provisioning.GetReadinessTimeout()
	interval := p.config.Provisioning.GetReadinessInterval()
	sshPort := p.config.Provisioning.SSHPort

	// Try IPv6 first if available, fall back to IPv4 if configured
	primaryIP := server.PublicIPv6
	fallbackIP := ""

	if primaryIP == "" {
		// No IPv6, use IPv4 as primary
		primaryIP = server.PublicIPv4
	} else if p.config.IsIPv4Enabled() && server.PublicIPv4 != "" {
		// IPv6 available, but IPv4 fallback is enabled
		fallbackIP = server.PublicIPv4
	}

	// Format address for TCP connection (IPv6 needs brackets with port)
	primaryAddr := sshutil.FormatSSHAddress(primaryIP, sshPort)
	fallbackAddr := ""
	if fallbackIP != "" {
		fallbackAddr = sshutil.FormatSSHAddress(fallbackIP, sshPort)
	}

	deadline := time.Now().Add(timeout)
	attempts := 0
	lastStatus := ""
	usingFallback := false

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		attempts++

		// Check SSH port connectivity on primary address
		addr := primaryAddr
		if usingFallback && fallbackAddr != "" {
			addr = fallbackAddr
		}

		status, err := p.checkSSHConnectivityWithStatus(addr)
		if err == nil {
			fmt.Printf("\n")
			if usingFallback {
				fmt.Printf("      ⚠️  Connected via IPv4 fallback\n")
			}
			return nil
		}

		// If primary (IPv6) fails with network unreachable/no route and we have fallback, try IPv4
		if !usingFallback && fallbackAddr != "" {
			if status == "network unreachable" || status == "no route" || status == "timeout" {
				// Quick check if IPv4 is reachable
				fallbackStatus, fallbackErr := p.checkSSHConnectivityWithStatus(fallbackAddr)
				if fallbackErr == nil {
					fmt.Printf("\n")
					fmt.Printf("      ⚠️  IPv6 unreachable, using IPv4 fallback\n")
					return nil
				}
				// If IPv4 seems more promising (port closed = server exists), switch to it
				if fallbackStatus == "port closed" || fallbackStatus == "connecting" {
					fmt.Printf("      ⚠️  IPv6 %s, trying IPv4 fallback...\n", status)
					usingFallback = true
				}
			}
		}

		// Only print status when it changes, or every 5 attempts to show progress
		if status != lastStatus || attempts%5 == 0 {
			ipLabel := "IPv6"
			if usingFallback {
				ipLabel = "IPv4"
			}
			fmt.Printf("      SSH check attempt %d (%s): %s\n", attempts, ipLabel, status)
			lastStatus = status
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

// checkSSHConnectivityWithStatus attempts a TCP connection to verify SSH is accepting connections
// Returns a human-readable status and any error
func (p *Provisioner) checkSSHConnectivityWithStatus(addr string) (string, error) {
	// Use a shorter timeout (3s) since we retry frequently anyway
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		status := classifySSHError(err)
		return status, err
	}
	conn.Close()
	return "connected", nil
}

// classifySSHError returns a human-readable status for SSH connection errors
func classifySSHError(err error) string {
	if err == nil {
		return "connected"
	}

	errStr := strings.ToLower(err.Error())

	// Check for common error patterns
	switch {
	case strings.Contains(errStr, "connection refused"):
		return "port closed"
	case strings.Contains(errStr, "no route to host"):
		return "no route"
	case strings.Contains(errStr, "network is unreachable"):
		return "network unreachable"
	case strings.Contains(errStr, "i/o timeout"), strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection reset"):
		return "connection reset"
	case strings.Contains(errStr, "host is down"):
		return "host down"
	default:
		return "connecting"
	}
}

// Teardown removes a forest and all its resources
func (p *Provisioner) Teardown(ctx context.Context, forestID string) error {
	fmt.Printf("🗑️  Tearing down forest: %s\n\n", forestID)

	// Get all nodes for this forest
	nodes, err := p.storage.GetNodes(forestID)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	// Delete DNS records if DNS provider is configured
	if p.dns != nil && p.config.DNS.Domain != "" {
		fmt.Printf("Deleting DNS records...\n")
		for i, node := range nodes {
			recordName := fmt.Sprintf("%s-node-%d", forestID, i+1)

			// Delete A record
			if node.IPv4 != "" {
				if err := p.dns.DeleteRecord(ctx, p.config.DNS.Domain, recordName, string(dns.RecordTypeA)); err != nil {
					fmt.Printf("   ⚠️  Warning: failed to delete A record: %s\n", err)
				}
			}

			// Delete AAAA record
			if node.IPv6 != "" {
				if err := p.dns.DeleteRecord(ctx, p.config.DNS.Domain, recordName, string(dns.RecordTypeAAAA)); err != nil {
					fmt.Printf("   ⚠️  Warning: failed to delete AAAA record: %s\n", err)
				}
			}
		}
	}

	// Delete all servers
	if len(nodes) > 0 {
		fmt.Printf("Deleting %d machine%s...\n", len(nodes), plural(len(nodes)))
		for i, node := range nodes {
			fmt.Printf("   [%d/%d] Deleting %s...", i+1, len(nodes), node.ID)

			if err := p.machine.DeleteServer(ctx, node.ID); err != nil {
				fmt.Printf(" ⚠️  Warning: %s\n", err)
			} else {
				fmt.Printf(" ✅\n")
			}
		}
	}

	// Remove from storage
	fmt.Printf("\nCleaning up storage...")
	if err := p.storage.DeleteForest(forestID); err != nil {
		fmt.Printf(" ⚠️  Warning: %s\n", err)
	} else {
		fmt.Printf(" ✅\n")
	}

	return nil
}

// rollback removes all provisioned servers on failure
func (p *Provisioner) rollback(ctx context.Context, forestID string, _ []*machine.Server) {
	// Get all registered nodes from storage (includes nodes registered before SSH verification)
	nodes, err := p.storage.GetNodes(forestID)
	if err != nil {
		fmt.Printf("   ⚠️  Warning: failed to get nodes from storage: %s\n", err)
	}

	// Delete all servers that were registered
	for i, node := range nodes {
		fmt.Printf("   🗑️  Deleting machine %d/%d (%s)...\n", i+1, len(nodes), node.ID)
		if err := p.machine.DeleteServer(ctx, node.ID); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to delete server %s: %s\n", node.ID, err)
		} else {
			fmt.Printf("   ✅ Machine deleted\n")
		}
	}

	// Remove from storage
	p.storage.DeleteForest(forestID)
	fmt.Printf("   ✅ Rollback complete\n")
}

// plural returns "s" if count is not 1, empty string otherwise
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
