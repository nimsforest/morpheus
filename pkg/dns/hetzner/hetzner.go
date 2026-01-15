package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/dns"
)

const (
	// Hetzner Cloud API URL (DNS was migrated from dns.hetzner.com in late 2025)
	hetznerCloudAPIURL = "https://api.hetzner.cloud/v1"
)

// Provider implements the DNS Provider interface for Hetzner DNS
type Provider struct {
	apiToken string
	client   *http.Client
	// Cache zone IDs to avoid repeated lookups (zone name -> zone ID)
	zoneCache map[string]int64
}

// NewProvider creates a new Hetzner DNS provider
func NewProvider(apiToken string) (*Provider, error) {
	apiToken = strings.TrimSpace(apiToken)
	// Strip quotes that may be present from env var
	apiToken = strings.Trim(apiToken, "\"'")
	if apiToken == "" {
		return nil, fmt.Errorf("Hetzner DNS API token is required")
	}

	return &Provider{
		apiToken:  apiToken,
		client:    &http.Client{Timeout: 30 * time.Second},
		zoneCache: make(map[string]int64),
	}, nil
}

// CreateRecord creates a DNS record in Hetzner DNS using the Cloud API RRSets endpoint
func (p *Provider) CreateRecord(ctx context.Context, req dns.CreateRecordRequest) (*dns.Record, error) {
	// Get zone ID for the domain
	zoneID, err := p.getZoneID(ctx, req.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone: %w", err)
	}

	// Set default TTL if not specified
	ttl := req.TTL
	if ttl == 0 {
		ttl = 300 // 5 minutes default
	}

	// Cloud API uses RRSets - create or update an RRSet
	// TTL is at RRSet level, records don't have individual TTLs
	body := map[string]interface{}{
		"name": req.Name,
		"type": string(req.Type),
		"ttl":  ttl,
		"records": []map[string]interface{}{
			{
				"value": req.Value,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create RRSet via POST to /rrsets
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		hetznerCloudAPIURL+"/zones/"+zoneID+"/rrsets",
		bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create record: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return &dns.Record{
		ID:     fmt.Sprintf("%s-%s", req.Name, req.Type),
		Domain: req.Domain,
		Name:   req.Name,
		Type:   req.Type,
		Value:  req.Value,
		TTL:    ttl,
	}, nil
}

// CreateRRSet creates an RRSet with multiple records (e.g., multiple MX records)
func (p *Provider) CreateRRSet(ctx context.Context, domain, name, recordType string, ttl int, records []map[string]interface{}) error {
	// Get zone ID for the domain
	zoneID, err := p.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}

	// Cloud API uses RRSets - create an RRSet with multiple records
	body := map[string]interface{}{
		"name":    name,
		"type":    recordType,
		"ttl":     ttl,
		"records": records,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create RRSet via POST to /rrsets
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		hetznerCloudAPIURL+"/zones/"+zoneID+"/rrsets",
		bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to create rrset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create rrset: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DeleteRecord removes a DNS record from Hetzner DNS using the Cloud API
func (p *Provider) DeleteRecord(ctx context.Context, domain, name, recordType string) error {
	// Get zone ID for the domain
	zoneID, err := p.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}

	// Cloud API uses RRSet ID in format "{name}/{type}"
	rrsetID := fmt.Sprintf("%s/%s", name, recordType)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE",
		hetznerCloudAPIURL+"/zones/"+zoneID+"/rrsets/"+rrsetID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer resp.Body.Close()

	// 404 means already deleted - consider success
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete record: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ListRecords lists all DNS records for a domain
func (p *Provider) ListRecords(ctx context.Context, domain string) ([]*dns.Record, error) {
	// Get zone ID for the domain
	zoneID, err := p.getZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone: %w", err)
	}

	hRecords, err := p.listRecordsByZone(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	records := make([]*dns.Record, len(hRecords))
	for i, r := range hRecords {
		records[i] = &dns.Record{
			ID:     r.ID,
			Domain: domain,
			Name:   r.Name,
			Type:   dns.RecordType(r.Type),
			Value:  r.Value,
			TTL:    r.TTL,
		}
	}

	return records, nil
}

// GetRecord retrieves a specific DNS record
func (p *Provider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	records, err := p.ListRecords(ctx, domain)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.Name == name && string(r.Type) == recordType {
			return r, nil
		}
	}

	return nil, nil // Not found
}

// CreateZone creates a new DNS zone in Hetzner DNS
func (p *Provider) CreateZone(ctx context.Context, req dns.CreateZoneRequest) (*dns.Zone, error) {
	// Set default TTL if not specified
	ttl := req.TTL
	if ttl == 0 {
		ttl = 86400 // 24 hours default
	}

	body := map[string]interface{}{
		"name": req.Name,
		"ttl":  ttl,
		"mode": "primary", // Required by Cloud API - "primary" for zones we manage
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", hetznerCloudAPIURL+"/zones", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create zone: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Zone hetznerZone `json:"zone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the zone ID
	p.zoneCache[result.Zone.Name] = result.Zone.ID

	return &dns.Zone{
		ID:          fmt.Sprintf("%d", result.Zone.ID),
		Name:        result.Zone.Name,
		TTL:         result.Zone.TTL,
		Nameservers: result.Zone.AuthoritativeNameservers.Assigned,
	}, nil
}

// DeleteZone deletes a DNS zone from Hetzner DNS
func (p *Provider) DeleteZone(ctx context.Context, zoneName string) error {
	// Get the zone to find its ID
	zone, err := p.GetZone(ctx, zoneName)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}
	if zone == nil {
		// Zone doesn't exist - consider this success
		return nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", hetznerCloudAPIURL+"/zones/"+zone.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete zone: %w", err)
	}
	defer resp.Body.Close()

	// Cloud API returns 201 with async action, 200/204 for immediate success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete zone: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Remove from cache
	delete(p.zoneCache, zoneName)

	return nil
}

// GetZone retrieves a DNS zone by name from Hetzner DNS
func (p *Provider) GetZone(ctx context.Context, zoneName string) (*dns.Zone, error) {
	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.Name == zoneName {
			return zone, nil
		}
	}

	return nil, nil // Not found
}

// ListZones lists all DNS zones in Hetzner DNS
func (p *Provider) ListZones(ctx context.Context) ([]*dns.Zone, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", hetznerCloudAPIURL+"/zones", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list zones: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Zones []hetznerZone `json:"zones"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse zones response: %w", err)
	}

	zones := make([]*dns.Zone, len(result.Zones))
	for i, z := range result.Zones {
		// Cache zone IDs
		p.zoneCache[z.Name] = z.ID

		zones[i] = &dns.Zone{
			ID:          fmt.Sprintf("%d", z.ID),
			Name:        z.Name,
			TTL:         z.TTL,
			Nameservers: z.AuthoritativeNameservers.Assigned,
		}
	}

	return zones, nil
}

// getZoneID returns the zone ID for a domain, using cache if available
func (p *Provider) getZoneID(ctx context.Context, domain string) (string, error) {
	// Check cache first
	if zoneID, ok := p.zoneCache[domain]; ok {
		return fmt.Sprintf("%d", zoneID), nil
	}

	// List all zones and find the matching one
	httpReq, err := http.NewRequestWithContext(ctx, "GET", hetznerCloudAPIURL+"/zones", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to list zones: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Zones []hetznerZone `json:"zones"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse zones response: %w", err)
	}

	// Find the zone that matches the domain
	// The domain might be a subdomain, so we need to find the best match
	var bestMatch hetznerZone
	for _, zone := range result.Zones {
		if domain == zone.Name || strings.HasSuffix(domain, "."+zone.Name) {
			if bestMatch.Name == "" || len(zone.Name) > len(bestMatch.Name) {
				bestMatch = zone
			}
		}
	}

	if bestMatch.ID == 0 {
		return "", fmt.Errorf("no zone found for domain: %s", domain)
	}

	// Cache the zone ID
	p.zoneCache[domain] = bestMatch.ID

	return fmt.Sprintf("%d", bestMatch.ID), nil
}

// listRecordsByZone lists all records in a zone using the new Cloud API RRSets endpoint
func (p *Provider) listRecordsByZone(ctx context.Context, zoneID string) ([]hetznerRecord, error) {
	// New Cloud API uses /zones/{id}/rrsets for record management
	httpReq, err := http.NewRequestWithContext(ctx, "GET", hetznerCloudAPIURL+"/zones/"+zoneID+"/rrsets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list records: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		RRSets []hetznerRRSet `json:"rrsets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}

	// Convert RRSets to flat record list for compatibility
	var records []hetznerRecord
	for _, rrset := range result.RRSets {
		for _, rec := range rrset.Records {
			records = append(records, hetznerRecord{
				ID:     fmt.Sprintf("%s-%s", rrset.Name, rrset.Type),
				ZoneID: zoneID,
				Name:   rrset.Name,
				Type:   rrset.Type,
				Value:  rec.Value,
				TTL:    rec.TTL,
			})
		}
	}

	return records, nil
}

// hetznerZone represents a DNS zone in Hetzner's Cloud API
type hetznerZone struct {
	ID                       int64                    `json:"id"`
	Name                     string                   `json:"name"`
	TTL                      int                      `json:"ttl"`
	Mode                     string                   `json:"mode"`
	Status                   string                   `json:"status"`
	AuthoritativeNameservers authoritativeNameservers `json:"authoritative_nameservers"`
}

// authoritativeNameservers holds nameserver info from Cloud API
type authoritativeNameservers struct {
	Assigned []string `json:"assigned"`
}

// hetznerRRSet represents a DNS record set in Hetzner's Cloud API
type hetznerRRSet struct {
	Name    string           `json:"name"`
	Type    string           `json:"type"`
	Records []hetznerRRValue `json:"records"`
}

// hetznerRRValue represents a single record value in an RRSet
type hetznerRRValue struct {
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// hetznerRecord represents a flattened DNS record for internal use
type hetznerRecord struct {
	ID     string `json:"id"`
	ZoneID string `json:"zone_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	TTL    int    `json:"ttl"`
}
