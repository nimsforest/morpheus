package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/nimsforest/morpheus/pkg/guard"
)

// EnsureNetwork creates the full networking stack for a guard.
func (p *Provider) EnsureNetwork(ctx context.Context, req guard.NetworkRequest) (*guard.NetworkInfo, error) {
	names := newResourceNames(req.GuardID, req.ResourceGroup)
	tags := guardTags(req.GuardID, nil, req.WireGuardPort)

	// 1. Ensure resource group
	fmt.Printf("      Creating resource group %s...\n", names.ResourceGroup)
	_, err := p.rgClient.CreateOrUpdate(ctx, names.ResourceGroup, armresources.ResourceGroup{
		Location: to.Ptr(req.Location),
		Tags:     tags,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource group: %w", err)
	}

	// 2. Create NSG with SSH + WireGuard rules
	fmt.Printf("      Creating NSG %s...\n", names.NSG)
	nsgPoller, err := p.nsgClient.BeginCreateOrUpdate(ctx, names.ResourceGroup, names.NSG, armnetwork.SecurityGroup{
		Location: to.Ptr(req.Location),
		Tags:     tags,
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: to.Ptr("AllowSSH"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Priority:                 to.Ptr[int32](100),
						Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolTCP),
						Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
						Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
						SourceAddressPrefix:      to.Ptr("*"),
						SourcePortRange:          to.Ptr("*"),
						DestinationAddressPrefix: to.Ptr("*"),
						DestinationPortRange:     to.Ptr("22"),
					},
				},
				{
					Name: to.Ptr("AllowWireGuard"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Priority:                 to.Ptr[int32](110),
						Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolUDP),
						Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
						Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
						SourceAddressPrefix:      to.Ptr("*"),
						SourcePortRange:          to.Ptr("*"),
						DestinationAddressPrefix: to.Ptr("*"),
						DestinationPortRange:     to.Ptr(fmt.Sprintf("%d", req.WireGuardPort)),
					},
				},
			},
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin NSG creation: %w", err)
	}
	nsgResp, err := nsgPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSG: %w", err)
	}

	// 3. Create VNet + Subnet
	fmt.Printf("      Creating VNet %s (%s)...\n", names.VNet, req.VNetCIDR)
	vnetPoller, err := p.vnetClient.BeginCreateOrUpdate(ctx, names.ResourceGroup, names.VNet, armnetwork.VirtualNetwork{
		Location: to.Ptr(req.Location),
		Tags:     tags,
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{to.Ptr(req.VNetCIDR)},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: to.Ptr(names.Subnet),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: to.Ptr(req.SubnetCIDR),
						NetworkSecurityGroup: &armnetwork.SecurityGroup{
							ID: nsgResp.ID,
						},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin VNet creation: %w", err)
	}
	vnetResp, err := vnetPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VNet: %w", err)
	}

	// Get subnet ID from VNet response
	var subnetID string
	if vnetResp.Properties != nil && len(vnetResp.Properties.Subnets) > 0 {
		subnetID = *vnetResp.Properties.Subnets[0].ID
	}

	// 4. Create Public IP
	fmt.Printf("      Creating public IP %s...\n", names.PublicIP)
	pipPoller, err := p.pipClient.BeginCreateOrUpdate(ctx, names.ResourceGroup, names.PublicIP, armnetwork.PublicIPAddress{
		Location: to.Ptr(req.Location),
		Tags:     tags,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: to.Ptr(armnetwork.PublicIPAddressSKUNameStandard),
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin public IP creation: %w", err)
	}
	pipResp, err := pipPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP: %w", err)
	}

	// 5. Create NIC with IP forwarding enabled
	fmt.Printf("      Creating NIC %s (IP forwarding enabled)...\n", names.NIC)
	nicPoller, err := p.nicClient.BeginCreateOrUpdate(ctx, names.ResourceGroup, names.NIC, armnetwork.Interface{
		Location: to.Ptr(req.Location),
		Tags:     tags,
		Properties: &armnetwork.InterfacePropertiesFormat{
			EnableIPForwarding: to.Ptr(true),
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr("ipconfig1"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(subnetID),
						},
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: pipResp.ID,
						},
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
					},
				},
			},
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin NIC creation: %w", err)
	}
	nicResp, err := nicPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NIC: %w", err)
	}

	// Extract private IP
	var privateIP string
	if nicResp.Properties != nil && len(nicResp.Properties.IPConfigurations) > 0 {
		ipConfig := nicResp.Properties.IPConfigurations[0]
		if ipConfig.Properties != nil && ipConfig.Properties.PrivateIPAddress != nil {
			privateIP = *ipConfig.Properties.PrivateIPAddress
		}
	}

	var publicIP string
	if pipResp.Properties != nil && pipResp.Properties.IPAddress != nil {
		publicIP = *pipResp.Properties.IPAddress
	}

	return &guard.NetworkInfo{
		ResourceGroup: names.ResourceGroup,
		VNetID:        *vnetResp.ID,
		SubnetID:      subnetID,
		NSGID:         *nsgResp.ID,
		NICID:         *nicResp.ID,
		PublicIPID:    *pipResp.ID,
		PublicIP:      publicIP,
		PrivateIP:     privateIP,
	}, nil
}

// CleanupNetwork removes all guard resources by deleting the resource group.
func (p *Provider) CleanupNetwork(ctx context.Context, guardID string) error {
	names := newResourceNames(guardID, p.resourceGroup)

	fmt.Printf("   Deleting resource group %s...\n", names.ResourceGroup)
	poller, err := p.rgClient.BeginDelete(ctx, names.ResourceGroup, nil)
	if err != nil {
		return fmt.Errorf("failed to begin resource group deletion: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete resource group: %w", err)
	}
	return nil
}

// ConfigureNICForwarding enables IP forwarding on a NIC.
func (p *Provider) ConfigureNICForwarding(ctx context.Context, nicID string) error {
	// IP forwarding is set at NIC creation time in EnsureNetwork,
	// this method exists for post-creation updates if needed.
	return nil
}

// EnsureNSGRule creates or updates an NSG rule.
func (p *Provider) EnsureNSGRule(ctx context.Context, req guard.NSGRuleRequest) error {
	protocol := armnetwork.SecurityRuleProtocolTCP
	switch req.Protocol {
	case "Udp":
		protocol = armnetwork.SecurityRuleProtocolUDP
	case "*":
		protocol = armnetwork.SecurityRuleProtocolAsterisk
	}

	direction := armnetwork.SecurityRuleDirectionInbound
	if req.Direction == "Outbound" {
		direction = armnetwork.SecurityRuleDirectionOutbound
	}

	poller, err := p.secRuleClient.BeginCreateOrUpdate(ctx, req.ResourceGroup, req.NSGName, req.RuleName, armnetwork.SecurityRule{
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Priority:                 to.Ptr(int32(req.Priority)),
			Protocol:                 to.Ptr(protocol),
			Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
			Direction:                to.Ptr(direction),
			SourceAddressPrefix:      to.Ptr("*"),
			SourcePortRange:          to.Ptr("*"),
			DestinationAddressPrefix: to.Ptr("*"),
			DestinationPortRange:     to.Ptr(req.DestPort),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin NSG rule creation: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create NSG rule: %w", err)
	}
	return nil
}

// PeerNetwork creates bidirectional VNet peering and a route table.
func (p *Provider) PeerNetwork(ctx context.Context, req guard.PeerRequest) error {
	names := newResourceNames(req.GuardID, p.resourceGroup)

	// Extract VNet names from resource IDs
	guardVNetName := names.VNet
	remoteVNetName := extractResourceName(req.RemoteVNetID)
	remoteRG := extractResourceGroup(req.RemoteVNetID)

	// 1. Guard VNet -> Remote VNet peering
	fmt.Printf("   Creating peering: guard -> remote...\n")
	fwdName := fmt.Sprintf("%s-to-%s", guardVNetName, remoteVNetName)
	fwdPoller, err := p.peeringClient.BeginCreateOrUpdate(ctx, names.ResourceGroup, guardVNetName, fwdName, armnetwork.VirtualNetworkPeering{
		Properties: &armnetwork.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: to.Ptr(true),
			AllowForwardedTraffic:     to.Ptr(true),
			RemoteVirtualNetwork: &armnetwork.SubResource{
				ID: to.Ptr(req.RemoteVNetID),
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin forward peering: %w", err)
	}
	_, err = fwdPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create forward peering: %w", err)
	}

	// 2. Remote VNet -> Guard VNet peering
	fmt.Printf("   Creating peering: remote -> guard...\n")
	revName := fmt.Sprintf("%s-to-%s", remoteVNetName, guardVNetName)
	revPoller, err := p.peeringClient.BeginCreateOrUpdate(ctx, remoteRG, remoteVNetName, revName, armnetwork.VirtualNetworkPeering{
		Properties: &armnetwork.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: to.Ptr(true),
			AllowForwardedTraffic:     to.Ptr(true),
			UseRemoteGateways:         to.Ptr(false),
			RemoteVirtualNetwork: &armnetwork.SubResource{
				ID: to.Ptr(req.GuardVNetID),
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin reverse peering: %w", err)
	}
	_, err = revPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create reverse peering: %w", err)
	}

	// 3. Create route table on remote subnet for mesh CIDRs
	if len(req.MeshCIDRs) > 0 && req.SubnetID != "" {
		fmt.Printf("   Creating route table for mesh CIDRs...\n")
		rtName := fmt.Sprintf("%s-routes", req.PeeringName)
		var routes []*armnetwork.Route
		for i, cidr := range req.MeshCIDRs {
			routes = append(routes, &armnetwork.Route{
				Name: to.Ptr(fmt.Sprintf("mesh-route-%d", i)),
				Properties: &armnetwork.RoutePropertiesFormat{
					AddressPrefix:    to.Ptr(cidr),
					NextHopType:      to.Ptr(armnetwork.RouteNextHopTypeVirtualAppliance),
					NextHopIPAddress: to.Ptr(req.GuardPrivateIP),
				},
			})
		}

		rtPoller, err := p.rtClient.BeginCreateOrUpdate(ctx, remoteRG, rtName, armnetwork.RouteTable{
			Location: to.Ptr(p.location),
			Properties: &armnetwork.RouteTablePropertiesFormat{
				Routes: routes,
			},
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to begin route table creation: %w", err)
		}
		rtResp, err := rtPoller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create route table: %w", err)
		}

		// Associate route table with remote subnet
		subnetName := extractResourceName(req.SubnetID)
		remoteVNetForSubnet := extractParentResourceName(req.SubnetID)
		subnetRG := extractResourceGroup(req.SubnetID)

		// Get current subnet config
		subnetResp, err := p.subnetClient.Get(ctx, subnetRG, remoteVNetForSubnet, subnetName, nil)
		if err != nil {
			return fmt.Errorf("failed to get remote subnet: %w", err)
		}

		subnetResp.Properties.RouteTable = &armnetwork.RouteTable{
			ID: rtResp.ID,
		}

		subPoller, err := p.subnetClient.BeginCreateOrUpdate(ctx, subnetRG, remoteVNetForSubnet, subnetName, subnetResp.Subnet, nil)
		if err != nil {
			return fmt.Errorf("failed to begin subnet update: %w", err)
		}
		_, err = subPoller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to update subnet with route table: %w", err)
		}
	}

	return nil
}

// UnpeerNetwork removes VNet peering.
func (p *Provider) UnpeerNetwork(ctx context.Context, guardID, peeringName string) error {
	names := newResourceNames(guardID, p.resourceGroup)

	poller, err := p.peeringClient.BeginDelete(ctx, names.ResourceGroup, names.VNet, peeringName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin peering deletion: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete peering: %w", err)
	}
	return nil
}

// extractResourceName extracts the last segment from an Azure resource ID.
func extractResourceName(resourceID string) string {
	parts := splitResourceID(resourceID)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// extractResourceGroup extracts the resource group from an Azure resource ID.
func extractResourceGroup(resourceID string) string {
	parts := splitResourceID(resourceID)
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractParentResourceName extracts the parent resource name (e.g., VNet name from a subnet ID).
func extractParentResourceName(resourceID string) string {
	parts := splitResourceID(resourceID)
	// For subnet: .../virtualNetworks/vnetName/subnets/subnetName
	// Parent is vnetName, which is 2 positions before the end
	if len(parts) >= 3 {
		return parts[len(parts)-3]
	}
	return ""
}

// splitResourceID splits an Azure resource ID into path segments.
func splitResourceID(id string) []string {
	var parts []string
	for _, s := range splitSlash(id) {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

func splitSlash(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '/' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
