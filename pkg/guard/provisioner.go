package guard

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/cloudinit"
	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/machine"
)

// Provisioner orchestrates guard VM creation.
type Provisioner struct {
	provider GuardProvider
	config   *config.Config
}

// NewProvisioner creates a new guard provisioner.
func NewProvisioner(p GuardProvider, cfg *config.Config) *Provisioner {
	return &Provisioner{
		provider: p,
		config:   cfg,
	}
}

// Provision creates a new guard VM with the full networking stack.
func (p *Provisioner) Provision(ctx context.Context, req CreateGuardRequest) (*Guard, error) {
	guardID := fmt.Sprintf("guard-%d", time.Now().Unix())
	guardCfg := p.config.Guard
	azureCfg := p.config.Machine.Azure

	location := req.Location
	if location == "" {
		location = azureCfg.Location
	}

	fmt.Printf("\n🛡️  Creating guard: %s\n", guardID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Guard ID:    %s\n", guardID)
	fmt.Printf("   Location:    %s\n", location)
	fmt.Printf("   VM Size:     %s\n", azureCfg.VMSize)
	fmt.Printf("   VNet CIDR:   %s\n", guardCfg.VNetCIDR)
	fmt.Printf("   Subnet CIDR: %s\n", guardCfg.SubnetCIDR)
	fmt.Printf("   WG Port:     %d\n", guardCfg.WGPort)
	if len(req.MeshCIDRs) > 0 {
		fmt.Printf("   Mesh CIDRs:  %s\n", strings.Join(req.MeshCIDRs, ", "))
	}
	fmt.Println()

	// Step 1: Create network infrastructure
	fmt.Printf("📦 Step 1/4: Creating network infrastructure\n")
	netInfo, err := p.provider.EnsureNetwork(ctx, NetworkRequest{
		GuardID:       guardID,
		Location:      location,
		ResourceGroup: azureCfg.ResourceGroup,
		VNetCIDR:      guardCfg.VNetCIDR,
		SubnetCIDR:    guardCfg.SubnetCIDR,
		WireGuardPort: guardCfg.WGPort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	fmt.Printf("   ✅ Network ready (Public IP: %s)\n\n", netInfo.PublicIP)

	// Step 2: Generate cloud-init
	fmt.Printf("📦 Step 2/4: Generating cloud-init\n")
	userData, err := cloudinit.GenerateGuard(cloudinit.GuardTemplateData{
		WireGuardConf: req.WireGuardConf,
		WireGuardPort: guardCfg.WGPort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	// Azure requires base64-encoded custom data
	userDataB64 := base64.StdEncoding.EncodeToString([]byte(userData))
	fmt.Printf("   ✅ Cloud-init generated\n\n")

	// Step 3: Create VM
	fmt.Printf("📦 Step 3/4: Creating VM\n")
	vmName := fmt.Sprintf("%s-vm", guardID)

	// Read SSH public key for Azure
	sshKeys, err := readSSHPublicKeys(p.config)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH keys: %w", err)
	}

	server, err := p.provider.CreateServer(ctx, machine.CreateServerRequest{
		Name:       vmName,
		ServerType: azureCfg.VMSize,
		Image:      azureCfg.Image,
		Location:   location,
		SSHKeys:    sshKeys,
		UserData:   userDataB64,
		Labels: map[string]string{
			"managed-by":     "morpheus-azureguard",
			"guard-id":       guardID,
			"mesh-cidrs":     strings.Join(req.MeshCIDRs, ","),
			"wg-port":        fmt.Sprintf("%d", guardCfg.WGPort),
			"nic-id":         netInfo.NICID,
			"resource-group": netInfo.ResourceGroup,
		},
		EnableIPv4: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	fmt.Printf("   ✅ VM created\n\n")

	// Step 4: Wait for VM to be running
	fmt.Printf("📦 Step 4/4: Waiting for VM to boot\n")
	if err := p.provider.WaitForServer(ctx, server.ID, machine.ServerStateRunning); err != nil {
		return nil, fmt.Errorf("VM failed to start: %w", err)
	}
	fmt.Printf("   ✅ VM running\n\n")

	guard := &Guard{
		ID:            guardID,
		Provider:      "azure",
		Location:      location,
		Status:        "active",
		PublicIP:      netInfo.PublicIP,
		PrivateIP:     netInfo.PrivateIP,
		ServerID:      server.ID,
		VNetID:        netInfo.VNetID,
		SubnetID:      netInfo.SubnetID,
		NSGID:         netInfo.NSGID,
		NICID:         netInfo.NICID,
		PublicIPID:    netInfo.PublicIPID,
		ResourceGroup: netInfo.ResourceGroup,
		MeshCIDRs:     req.MeshCIDRs,
		WireGuardPort: guardCfg.WGPort,
		CreatedAt:     time.Now(),
	}

	return guard, nil
}

// Teardown removes a guard and all its Azure resources.
func (p *Provisioner) Teardown(ctx context.Context, guardID string) error {
	fmt.Printf("\n🗑️  Tearing down guard: %s\n", guardID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Get guard info from Azure
	g, err := p.provider.GetGuard(ctx, guardID)
	if err != nil {
		return fmt.Errorf("guard not found: %w", err)
	}

	fmt.Printf("   Location: %s\n", g.Location)
	fmt.Printf("   VM:       %s\n", g.ServerID)
	fmt.Println()

	// Delete the resource group — this removes everything
	fmt.Printf("   Deleting all Azure resources...\n")
	if err := p.provider.CleanupNetwork(ctx, guardID); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	fmt.Printf("   ✅ All resources deleted\n")
	return nil
}

// readSSHPublicKeys reads SSH public keys from config paths.
func readSSHPublicKeys(cfg *config.Config) ([]string, error) {
	keyPath := cfg.GetSSHKeyPath()
	if keyPath == "" {
		// Try default locations
		defaultPaths := []string{
			homeDir() + "/.ssh/id_ed25519.pub",
			homeDir() + "/.ssh/id_rsa.pub",
		}
		for _, path := range defaultPaths {
			if data, err := readFile(path); err == nil {
				return []string{strings.TrimSpace(string(data))}, nil
			}
		}
		return nil, fmt.Errorf("no SSH public key found; set machine.ssh.key_path in config")
	}

	data, err := readFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key %s: %w", keyPath, err)
	}
	return []string{strings.TrimSpace(string(data))}, nil
}
