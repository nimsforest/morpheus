package azure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/nimsforest/morpheus/pkg/guard"
	"github.com/nimsforest/morpheus/pkg/machine"
)

// Provider implements guard.GuardProvider for Azure.
type Provider struct {
	subscriptionID string
	resourceGroup  string
	location       string
	vmSize         string
	image          string

	// Azure SDK clients
	rgClient      *armresources.ResourceGroupsClient
	vmClient      *armcompute.VirtualMachinesClient
	nsgClient     *armnetwork.SecurityGroupsClient
	secRuleClient *armnetwork.SecurityRulesClient
	vnetClient    *armnetwork.VirtualNetworksClient
	subnetClient  *armnetwork.SubnetsClient
	pipClient     *armnetwork.PublicIPAddressesClient
	nicClient     *armnetwork.InterfacesClient
	peeringClient *armnetwork.VirtualNetworkPeeringsClient
	rtClient      *armnetwork.RouteTablesClient
}

// Ensure Provider satisfies guard.GuardProvider
var _ guard.GuardProvider = (*Provider)(nil)

// NewProvider creates a new Azure guard provider.
func NewProvider(subscriptionID, tenantID, clientID, clientSecret, resourceGroup, location, vmSize, image string) (*Provider, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource groups client: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSG client: %w", err)
	}

	secRuleClient, err := armnetwork.NewSecurityRulesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create security rules client: %w", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VNet client: %w", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet client: %w", err)
	}

	pipClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP client: %w", err)
	}

	nicClient, err := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NIC client: %w", err)
	}

	peeringClient, err := armnetwork.NewVirtualNetworkPeeringsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create peering client: %w", err)
	}

	rtClient, err := armnetwork.NewRouteTablesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create route table client: %w", err)
	}

	return &Provider{
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		location:       location,
		vmSize:         vmSize,
		image:          image,
		rgClient:       rgClient,
		vmClient:       vmClient,
		nsgClient:      nsgClient,
		secRuleClient:  secRuleClient,
		vnetClient:     vnetClient,
		subnetClient:   subnetClient,
		pipClient:      pipClient,
		nicClient:      nicClient,
		peeringClient:  peeringClient,
		rtClient:       rtClient,
	}, nil
}

// CreateServer creates an Azure VM for the guard.
func (p *Provider) CreateServer(ctx context.Context, req machine.CreateServerRequest) (*machine.Server, error) {
	publisher, offer, sku, version, err := parseImageReference(p.image)
	if err != nil {
		return nil, err
	}

	// The NIC must already be created via EnsureNetwork.
	// req.Labels["nic-id"] should contain the NIC resource ID.
	nicID, ok := req.Labels["nic-id"]
	if !ok || nicID == "" {
		return nil, fmt.Errorf("nic-id label is required for Azure VM creation")
	}

	tags := make(map[string]*string)
	for k, v := range req.Labels {
		val := v
		tags[k] = &val
	}

	vmParams := armcompute.VirtualMachine{
		Location: to.Ptr(req.Location),
		Tags:     tags,
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(p.vmSize)),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					Publisher: to.Ptr(publisher),
					Offer:    to.Ptr(offer),
					SKU:      to.Ptr(sku),
					Version:  to.Ptr(version),
				},
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardLRS),
					},
				},
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  to.Ptr(req.Name),
				AdminUsername: to.Ptr("azureuser"),
				CustomData:    to.Ptr(req.UserData),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					DisablePasswordAuthentication: to.Ptr(true),
					SSH: &armcompute.SSHConfiguration{
						PublicKeys: sshKeysToAzure(req.SSHKeys),
					},
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr(nicID),
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: to.Ptr(true),
						},
					},
				},
			},
		},
	}

	rg := extractLabelOrDefault(req.Labels, "resource-group", p.resourceGroup)

	poller, err := p.vmClient.BeginCreateOrUpdate(ctx, rg, req.Name, vmParams, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin VM creation: %w", err)
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	serverID := ""
	if resp.ID != nil {
		serverID = *resp.ID
	}

	return &machine.Server{
		ID:       serverID,
		Name:     req.Name,
		Location: req.Location,
		State:    machine.ServerStateStarting,
		Labels:   req.Labels,
	}, nil
}

// GetServer retrieves server information by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*machine.Server, error) {
	rg := extractResourceGroup(serverID)
	vmName := extractResourceName(serverID)
	if rg == "" || vmName == "" {
		return nil, fmt.Errorf("invalid server ID format: %s", serverID)
	}

	resp, err := p.vmClient.Get(ctx, rg, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: to.Ptr(armcompute.InstanceViewTypesInstanceView),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	state := machine.ServerStateUnknown
	if resp.Properties != nil && resp.Properties.InstanceView != nil {
		for _, status := range resp.Properties.InstanceView.Statuses {
			if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
				switch *status.Code {
				case "PowerState/running":
					state = machine.ServerStateRunning
				case "PowerState/starting":
					state = machine.ServerStateStarting
				case "PowerState/stopped", "PowerState/deallocated":
					state = machine.ServerStateStopped
				}
			}
		}
	}

	tags := make(map[string]string)
	if resp.Tags != nil {
		for k, v := range resp.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}

	location := ""
	if resp.Location != nil {
		location = *resp.Location
	}

	return &machine.Server{
		ID:       serverID,
		Name:     vmName,
		Location: location,
		State:    state,
		Labels:   tags,
	}, nil
}

// DeleteServer removes a VM.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	rg := extractResourceGroup(serverID)
	vmName := extractResourceName(serverID)
	if rg == "" || vmName == "" {
		return fmt.Errorf("invalid server ID format: %s", serverID)
	}

	poller, err := p.vmClient.BeginDelete(ctx, rg, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin VM deletion: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}
	return nil
}

// WaitForServer waits until the server is in the specified state.
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state machine.ServerState) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		server, err := p.GetServer(ctx, serverID)
		if err != nil {
			return err
		}

		if server.State == state {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

// ListServers lists all VMs with optional filters.
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*machine.Server, error) {
	var servers []*machine.Server

	pager := p.vmClient.NewListPager(p.resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VMs: %w", err)
		}
		for _, vm := range page.Value {
			if vm.ID == nil {
				continue
			}
			tags := make(map[string]string)
			if vm.Tags != nil {
				for k, v := range vm.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
			}

			// Apply filters
			match := true
			for k, v := range filters {
				if tags[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}

			name := ""
			if vm.Name != nil {
				name = *vm.Name
			}
			location := ""
			if vm.Location != nil {
				location = *vm.Location
			}

			servers = append(servers, &machine.Server{
				ID:       *vm.ID,
				Name:     name,
				Location: location,
				Labels:   tags,
			})
		}
	}

	return servers, nil
}

// GetGuard reconstructs guard info from Azure resources by guard ID.
func (p *Provider) GetGuard(ctx context.Context, guardID string) (*guard.Guard, error) {
	names := newResourceNames(guardID, p.resourceGroup)

	// Check resource group exists and has our tags
	rgResp, err := p.rgClient.Get(ctx, names.ResourceGroup, nil)
	if err != nil {
		return nil, fmt.Errorf("guard not found: %w", err)
	}

	// Verify it's managed by us
	if rgResp.Tags == nil || rgResp.Tags[TagManagedBy] == nil || *rgResp.Tags[TagManagedBy] != TagManagedByValue {
		return nil, fmt.Errorf("resource group %s is not managed by morpheus-azureguard", names.ResourceGroup)
	}

	g := &guard.Guard{
		ID:            guardID,
		Provider:      "azure",
		ResourceGroup: names.ResourceGroup,
	}

	if rgResp.Location != nil {
		g.Location = *rgResp.Location
	}
	if rgResp.Tags[TagGuardID] != nil {
		g.ID = *rgResp.Tags[TagGuardID]
	}
	if rgResp.Tags[TagMeshCIDRs] != nil && *rgResp.Tags[TagMeshCIDRs] != "" {
		g.MeshCIDRs = strings.Split(*rgResp.Tags[TagMeshCIDRs], ",")
	}
	if rgResp.Tags[TagWGPort] != nil {
		if port, err := strconv.Atoi(*rgResp.Tags[TagWGPort]); err == nil {
			g.WireGuardPort = port
		}
	}

	// Get VM info
	vmResp, err := p.vmClient.Get(ctx, names.ResourceGroup, names.VM, &armcompute.VirtualMachinesClientGetOptions{
		Expand: to.Ptr(armcompute.InstanceViewTypesInstanceView),
	})
	if err == nil {
		if vmResp.ID != nil {
			g.ServerID = *vmResp.ID
		}
		g.Status = "running"
		if vmResp.Properties != nil && vmResp.Properties.InstanceView != nil {
			for _, status := range vmResp.Properties.InstanceView.Statuses {
				if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
					g.Status = strings.TrimPrefix(*status.Code, "PowerState/")
				}
			}
		}
	}

	// Get public IP
	pipResp, err := p.pipClient.Get(ctx, names.ResourceGroup, names.PublicIP, nil)
	if err == nil && pipResp.Properties != nil && pipResp.Properties.IPAddress != nil {
		g.PublicIP = *pipResp.Properties.IPAddress
		g.PublicIPID = *pipResp.ID
	}

	// Get NIC for private IP
	nicResp, err := p.nicClient.Get(ctx, names.ResourceGroup, names.NIC, nil)
	if err == nil {
		if nicResp.ID != nil {
			g.NICID = *nicResp.ID
		}
		if nicResp.Properties != nil && len(nicResp.Properties.IPConfigurations) > 0 {
			ipCfg := nicResp.Properties.IPConfigurations[0]
			if ipCfg.Properties != nil && ipCfg.Properties.PrivateIPAddress != nil {
				g.PrivateIP = *ipCfg.Properties.PrivateIPAddress
			}
		}
	}

	// Get VNet
	vnetResp, err := p.vnetClient.Get(ctx, names.ResourceGroup, names.VNet, nil)
	if err == nil && vnetResp.ID != nil {
		g.VNetID = *vnetResp.ID

		// Check peerings
		if vnetResp.Properties != nil && vnetResp.Properties.VirtualNetworkPeerings != nil {
			for _, peering := range vnetResp.Properties.VirtualNetworkPeerings {
				if peering.Name != nil && peering.Properties != nil && peering.Properties.RemoteVirtualNetwork != nil {
					pi := guard.PeeringInfo{
						Name: *peering.Name,
					}
					if peering.Properties.RemoteVirtualNetwork.ID != nil {
						pi.RemoteVNetID = *peering.Properties.RemoteVirtualNetwork.ID
					}
					g.Peerings = append(g.Peerings, pi)
				}
			}
		}
	}

	// Get NSG
	nsgResp, err := p.nsgClient.Get(ctx, names.ResourceGroup, names.NSG, nil)
	if err == nil && nsgResp.ID != nil {
		g.NSGID = *nsgResp.ID
	}

	return g, nil
}

// ListGuards discovers all guards from Azure resource groups
// tagged with managed-by=morpheus-azureguard.
func (p *Provider) ListGuards(ctx context.Context) ([]*guard.Guard, error) {
	var guards []*guard.Guard

	pager := p.rgClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource groups: %w", err)
		}
		for _, rg := range page.Value {
			if rg.Tags == nil || rg.Tags[TagManagedBy] == nil || *rg.Tags[TagManagedBy] != TagManagedByValue {
				continue
			}
			if rg.Tags[TagGuardID] == nil {
				continue
			}

			guardID := *rg.Tags[TagGuardID]

			g := &guard.Guard{
				ID:            guardID,
				Provider:      "azure",
				ResourceGroup: *rg.Name,
			}
			if rg.Location != nil {
				g.Location = *rg.Location
			}
			if rg.Tags[TagMeshCIDRs] != nil && *rg.Tags[TagMeshCIDRs] != "" {
				g.MeshCIDRs = strings.Split(*rg.Tags[TagMeshCIDRs], ",")
			}
			if rg.Tags[TagWGPort] != nil {
				if port, err := strconv.Atoi(*rg.Tags[TagWGPort]); err == nil {
					g.WireGuardPort = port
				}
			}

			// Quick VM status check
			vmName := fmt.Sprintf("%s-vm", guardID)
			vmResp, err := p.vmClient.Get(ctx, *rg.Name, vmName, &armcompute.VirtualMachinesClientGetOptions{
				Expand: to.Ptr(armcompute.InstanceViewTypesInstanceView),
			})
			if err == nil {
				g.Status = "running"
				if vmResp.Properties != nil && vmResp.Properties.InstanceView != nil {
					for _, status := range vmResp.Properties.InstanceView.Statuses {
						if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
							g.Status = strings.TrimPrefix(*status.Code, "PowerState/")
						}
					}
				}
			} else {
				g.Status = "unknown"
			}

			// Quick public IP check
			pipName := fmt.Sprintf("%s-pip", guardID)
			pipResp, err := p.pipClient.Get(ctx, *rg.Name, pipName, nil)
			if err == nil && pipResp.Properties != nil && pipResp.Properties.IPAddress != nil {
				g.PublicIP = *pipResp.Properties.IPAddress
			}

			guards = append(guards, g)
		}
	}

	return guards, nil
}

// sshKeysToAzure converts SSH key strings to Azure SSH public key objects.
func sshKeysToAzure(keys []string) []*armcompute.SSHPublicKey {
	var result []*armcompute.SSHPublicKey
	for _, key := range keys {
		result = append(result, &armcompute.SSHPublicKey{
			Path:    to.Ptr("/home/azureuser/.ssh/authorized_keys"),
			KeyData: to.Ptr(key),
		})
	}
	return result
}

// extractLabelOrDefault gets a label value or returns a default.
func extractLabelOrDefault(labels map[string]string, key, defaultVal string) string {
	if v, ok := labels[key]; ok && v != "" {
		return v
	}
	return defaultVal
}
