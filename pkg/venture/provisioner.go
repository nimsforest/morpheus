package venture

import (
	"context"
	"fmt"
	"strings"

	"github.com/nimsforest/morpheus/pkg/dns"
)

// Provisioner handles DNS provisioning for ventures
type Provisioner struct {
	dnsProvider dns.Provider
}

// NewProvisioner creates a new venture provisioner
func NewProvisioner(provider dns.Provider) *Provisioner {
	return &Provisioner{
		dnsProvider: provider,
	}
}

// ProvisionResult contains the result of a provisioning operation
type ProvisionResult struct {
	Zone           *dns.Zone    // The created or existing zone
	Records        []*dns.Record // The created DNS records
	ZoneCreated    bool          // Whether a new zone was created
	Nameservers    []string      // NS records to configure at parent domain
}

// ProvisionRecords creates DNS records for a venture.
// domain is the full domain for the venture (e.g., "experiencenet.customer.com")
// vars contains values for placeholders (e.g., "ServerIP" -> "1.2.3.4")
func (p *Provisioner) ProvisionRecords(ctx context.Context, ventureName, domain string, vars map[string]string) (*ProvisionResult, error) {
	if p.dnsProvider == nil {
		return nil, fmt.Errorf("DNS provider is not configured")
	}

	// Get the venture template
	template, err := GetTemplate(ventureName)
	if err != nil {
		return nil, err
	}

	result := &ProvisionResult{
		Records: make([]*dns.Record, 0, len(template.Records)),
	}

	// Check if zone exists, create if needed
	zone, err := p.dnsProvider.GetZone(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to check zone existence: %w", err)
	}

	if zone == nil {
		// Create the zone
		zone, err = p.dnsProvider.CreateZone(ctx, dns.CreateZoneRequest{
			Name: domain,
			TTL:  86400, // 24 hours default
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create zone %s: %w", domain, err)
		}
		result.ZoneCreated = true
	}

	result.Zone = zone
	result.Nameservers = zone.Nameservers

	// Create DNS records from template
	for _, recordTemplate := range template.Records {
		value := expandPlaceholders(recordTemplate.Value, vars, domain)

		record, err := p.dnsProvider.CreateRecord(ctx, dns.CreateRecordRequest{
			Domain: domain,
			Name:   recordTemplate.Name,
			Type:   recordTemplate.Type,
			Value:  value,
			TTL:    recordTemplate.TTL,
		})
		if err != nil {
			// Log error but continue with other records
			fmt.Printf("Warning: failed to create record %s.%s: %v\n", recordTemplate.Name, domain, err)
			continue
		}

		result.Records = append(result.Records, record)
	}

	return result, nil
}

// CleanupRecords removes DNS records for a venture.
// This will delete all records defined in the venture template and optionally the zone.
func (p *Provisioner) CleanupRecords(ctx context.Context, ventureName, domain string, deleteZone bool) error {
	if p.dnsProvider == nil {
		return fmt.Errorf("DNS provider is not configured")
	}

	// Get the venture template
	template, err := GetTemplate(ventureName)
	if err != nil {
		return err
	}

	// Check if zone exists
	zone, err := p.dnsProvider.GetZone(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to check zone existence: %w", err)
	}

	if zone == nil {
		// Zone doesn't exist, nothing to cleanup
		return nil
	}

	// Delete DNS records from template
	for _, recordTemplate := range template.Records {
		err := p.dnsProvider.DeleteRecord(ctx, domain, recordTemplate.Name, string(recordTemplate.Type))
		if err != nil {
			// Log error but continue with other records
			fmt.Printf("Warning: failed to delete record %s.%s: %v\n", recordTemplate.Name, domain, err)
		}
	}

	// Delete the zone if requested
	if deleteZone {
		err = p.dnsProvider.DeleteZone(ctx, domain)
		if err != nil {
			return fmt.Errorf("failed to delete zone %s: %w", domain, err)
		}
	}

	return nil
}

// ListVentureRecords lists all DNS records for a venture domain
func (p *Provisioner) ListVentureRecords(ctx context.Context, domain string) ([]*dns.Record, error) {
	if p.dnsProvider == nil {
		return nil, fmt.Errorf("DNS provider is not configured")
	}

	// Check if zone exists
	zone, err := p.dnsProvider.GetZone(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to check zone existence: %w", err)
	}

	if zone == nil {
		return nil, fmt.Errorf("zone %s does not exist", domain)
	}

	return p.dnsProvider.ListRecords(ctx, domain)
}

// expandPlaceholders replaces placeholders in a template value with actual values.
// Supported placeholders:
//   - {{.ServerIP}} - replaced with vars["ServerIP"]
//   - @ - replaced with the domain name for CNAME records
func expandPlaceholders(value string, vars map[string]string, domain string) string {
	result := value

	// Replace {{.Key}} placeholders with values from vars
	for key, val := range vars {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, val)
	}

	// Handle @ reference for CNAME records (@ points to the zone apex)
	// In Hetzner DNS, we use the full domain name for CNAME targets
	if result == "@" {
		result = domain + "."
	}

	return result
}

// GetVentureDomain constructs the venture domain from customer domain and venture name
// e.g., GetVentureDomain("customer.com", "experiencenet") returns "experiencenet.customer.com"
func GetVentureDomain(customerDomain, ventureName string) string {
	return ventureName + "." + customerDomain
}
