package guard

import (
	"context"
	"time"

	"github.com/nimsforest/morpheus/pkg/machine"
)

// GuardProvider extends machine.Provider with networking and discovery
// operations needed for WireGuard gateway VMs.
// All state is stored in Azure (resource groups, tags) — no local registry.
type GuardProvider interface {
	machine.Provider

	// EnsureNetwork creates or verifies the guard network infrastructure
	// (resource group, NSG, VNet, subnet, public IP, NIC).
	EnsureNetwork(ctx context.Context, req NetworkRequest) (*NetworkInfo, error)

	// CleanupNetwork removes all network resources for a guard
	// by deleting the resource group.
	CleanupNetwork(ctx context.Context, guardID string) error

	// ConfigureNICForwarding enables IP forwarding on a NIC.
	ConfigureNICForwarding(ctx context.Context, nicID string) error

	// EnsureNSGRule creates or updates an NSG rule.
	EnsureNSGRule(ctx context.Context, req NSGRuleRequest) error

	// PeerNetwork creates bidirectional VNet peering and route tables.
	PeerNetwork(ctx context.Context, req PeerRequest) error

	// UnpeerNetwork removes VNet peering.
	UnpeerNetwork(ctx context.Context, guardID, peeringName string) error

	// Discovery — state lives in Azure, not locally.

	// GetGuard reconstructs guard info from Azure resources by guard ID.
	GetGuard(ctx context.Context, guardID string) (*Guard, error)

	// ListGuards discovers all guards from Azure resource groups tagged
	// with managed-by=morpheus-azureguard.
	ListGuards(ctx context.Context) ([]*Guard, error)
}

// Guard represents a provisioned WireGuard gateway VM.
// Reconstructed from Azure resource tags and properties — not persisted locally.
type Guard struct {
	ID            string            `json:"id"`
	Provider      string            `json:"provider"`
	Location      string            `json:"location"`
	Status        string            `json:"status"`
	PublicIP      string            `json:"public_ip"`
	PrivateIP     string            `json:"private_ip"`
	ServerID      string            `json:"server_id"`
	VNetID        string            `json:"vnet_id,omitempty"`
	SubnetID      string            `json:"subnet_id,omitempty"`
	NSGID         string            `json:"nsg_id,omitempty"`
	NICID         string            `json:"nic_id,omitempty"`
	PublicIPID    string            `json:"public_ip_id,omitempty"`
	ResourceGroup string           `json:"resource_group,omitempty"`
	MeshCIDRs     []string          `json:"mesh_cidrs,omitempty"`
	WireGuardPort int               `json:"wireguard_port"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Peerings      []PeeringInfo     `json:"peerings,omitempty"`
}

// PeeringInfo tracks a VNet peering created by this guard.
type PeeringInfo struct {
	Name         string `json:"name"`
	RemoteVNetID string `json:"remote_vnet_id"`
	RouteTableID string `json:"route_table_id,omitempty"`
}

// NetworkRequest contains parameters for creating guard network infrastructure.
type NetworkRequest struct {
	GuardID       string
	Location      string
	ResourceGroup string
	VNetCIDR      string
	SubnetCIDR    string
	WireGuardPort int
}

// NetworkInfo contains the created network resource IDs.
type NetworkInfo struct {
	ResourceGroup string
	VNetID        string
	SubnetID      string
	NSGID         string
	NICID         string
	PublicIPID    string
	PublicIP      string
	PrivateIP     string
}

// NSGRuleRequest defines a network security group rule.
type NSGRuleRequest struct {
	GuardID       string
	ResourceGroup string
	NSGName       string
	RuleName      string
	Priority      int
	Protocol      string // "Tcp", "Udp", "*"
	DestPort      string // e.g. "51820"
	Direction     string // "Inbound", "Outbound"
}

// PeerRequest contains parameters for VNet peering.
type PeerRequest struct {
	GuardID        string
	GuardVNetID    string
	RemoteVNetID   string
	PeeringName    string
	GuardPrivateIP string
	MeshCIDRs      []string
	SubnetID       string // Remote subnet to attach route table
}

// CreateGuardRequest contains parameters for creating a guard VM.
type CreateGuardRequest struct {
	Location      string
	WireGuardConf string // Contents of wg0.conf
	MeshCIDRs     []string
}

// GuardStatus represents the current state of a guard.
type GuardStatus struct {
	Guard   *Guard
	VMState machine.ServerState
}
