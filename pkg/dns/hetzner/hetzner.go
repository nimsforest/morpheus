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
	hetznerDNSAPIURL = "https://dns.hetzner.com/api/v1"
)

// Provider implements the DNS Provider interface for Hetzner DNS
type Provider struct {
	apiToken string
	client   *http.Client
	// Cache zone IDs to avoid repeated lookups
	zoneCache map[string]string
}

// NewProvider creates a new Hetzner DNS provider
func NewProvider(apiToken string) (*Provider, error) {
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return nil, fmt.Errorf("Hetzner DNS API token is required")
	}

	return &Provider{
		apiToken:  apiToken,
		client:    &http.Client{Timeout: 30 * time.Second},
		zoneCache: make(map[string]string),
	}, nil
}

// CreateRecord creates a DNS record in Hetzner DNS
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

	// Create the record
	body := map[string]interface{}{
		"zone_id": zoneID,
		"name":    req.Name,
		"type":    string(req.Type),
		"value":   req.Value,
		"ttl":     ttl,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", hetznerDNSAPIURL+"/records", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Auth-API-Token", p.apiToken)
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

	var result struct {
		Record hetznerRecord `json:"record"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &dns.Record{
		ID:     result.Record.ID,
		Domain: req.Domain,
		Name:   result.Record.Name,
		Type:   dns.RecordType(result.Record.Type),
		Value:  result.Record.Value,
		TTL:    result.Record.TTL,
	}, nil
}

// DeleteRecord removes a DNS record from Hetzner DNS
func (p *Provider) DeleteRecord(ctx context.Context, domain, name, recordType string) error {
	// Get zone ID for the domain
	zoneID, err := p.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}

	// Find the record
	records, err := p.listRecordsByZone(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to list records: %w", err)
	}

	var recordID string
	for _, r := range records {
		if r.Name == name && r.Type == recordType {
			recordID = r.ID
			break
		}
	}

	if recordID == "" {
		// Record doesn't exist - consider this success
		return nil
	}

	// Delete the record
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", hetznerDNSAPIURL+"/records/"+recordID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Auth-API-Token", p.apiToken)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer resp.Body.Close()

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

// getZoneID returns the zone ID for a domain, using cache if available
func (p *Provider) getZoneID(ctx context.Context, domain string) (string, error) {
	// Check cache first
	if zoneID, ok := p.zoneCache[domain]; ok {
		return zoneID, nil
	}

	// List all zones and find the matching one
	httpReq, err := http.NewRequestWithContext(ctx, "GET", hetznerDNSAPIURL+"/zones", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Auth-API-Token", p.apiToken)

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

	if bestMatch.ID == "" {
		return "", fmt.Errorf("no zone found for domain: %s", domain)
	}

	// Cache the zone ID
	p.zoneCache[domain] = bestMatch.ID

	return bestMatch.ID, nil
}

// listRecordsByZone lists all records in a zone
func (p *Provider) listRecordsByZone(ctx context.Context, zoneID string) ([]hetznerRecord, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", hetznerDNSAPIURL+"/records?zone_id="+zoneID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Auth-API-Token", p.apiToken)

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
		Records []hetznerRecord `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}

	return result.Records, nil
}

// hetznerZone represents a DNS zone in Hetzner's API
type hetznerZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// hetznerRecord represents a DNS record in Hetzner's API
type hetznerRecord struct {
	ID     string `json:"id"`
	ZoneID string `json:"zone_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	TTL    int    `json:"ttl"`
}
