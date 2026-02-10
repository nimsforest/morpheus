package azure

import (
	"fmt"
	"strings"
)

const (
	// TagManagedBy identifies resources managed by morpheus-azureguard
	TagManagedBy = "managed-by"
	// TagManagedByValue is the tag value for guard-managed resources
	TagManagedByValue = "morpheus-azureguard"
	// TagGuardID identifies the guard a resource belongs to
	TagGuardID = "guard-id"
	// TagMeshCIDRs stores the mesh CIDRs as a comma-separated string
	TagMeshCIDRs = "mesh-cidrs"
	// TagWGPort stores the WireGuard port
	TagWGPort = "wg-port"
)

// resourceNames generates consistent Azure resource names from a guard ID.
type resourceNames struct {
	GuardID       string
	ResourceGroup string
	VNet          string
	Subnet        string
	NSG           string
	NIC           string
	PublicIP      string
	VM            string
}

func newResourceNames(guardID, rgPrefix string) resourceNames {
	rg := rgPrefix
	if rg == "" {
		rg = "morpheus-guards"
	}
	return resourceNames{
		GuardID:       guardID,
		ResourceGroup: rg,
		VNet:          fmt.Sprintf("%s-vnet", guardID),
		Subnet:        fmt.Sprintf("%s-subnet", guardID),
		NSG:           fmt.Sprintf("%s-nsg", guardID),
		NIC:           fmt.Sprintf("%s-nic", guardID),
		PublicIP:      fmt.Sprintf("%s-pip", guardID),
		VM:            fmt.Sprintf("%s-vm", guardID),
	}
}

// guardTags returns the standard tags for a guard resource.
func guardTags(guardID string, meshCIDRs []string, wgPort int) map[string]*string {
	managed := TagManagedByValue
	gid := guardID
	cidrs := strings.Join(meshCIDRs, ",")
	port := fmt.Sprintf("%d", wgPort)
	return map[string]*string{
		TagManagedBy: &managed,
		TagGuardID:   &gid,
		TagMeshCIDRs: &cidrs,
		TagWGPort:    &port,
	}
}

// parseImageReference parses "Publisher:Offer:SKU:Version" into components.
func parseImageReference(image string) (publisher, offer, sku, version string, err error) {
	parts := strings.Split(image, ":")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid image format %q, expected Publisher:Offer:SKU:Version", image)
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}
