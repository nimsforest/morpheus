package none

import (
	"context"
	"errors"

	"github.com/nimsforest/morpheus/pkg/dns"
)

// ErrZoneManagementNotSupported is returned when zone management operations are attempted
var ErrZoneManagementNotSupported = errors.New("zone management not supported by none provider")

// Provider is a no-op DNS provider that doesn't manage any DNS records
// Use this when you want to manage DNS records manually or don't need DNS
type Provider struct{}

// NewProvider creates a new no-op DNS provider
func NewProvider() (*Provider, error) {
	return &Provider{}, nil
}

// CreateRecord is a no-op that returns a dummy record
func (p *Provider) CreateRecord(ctx context.Context, req dns.CreateRecordRequest) (*dns.Record, error) {
	// Return a dummy record - no actual DNS changes are made
	return &dns.Record{
		ID:     "none",
		Domain: req.Domain,
		Name:   req.Name,
		Type:   req.Type,
		Value:  req.Value,
		TTL:    req.TTL,
	}, nil
}

// DeleteRecord is a no-op that always succeeds
func (p *Provider) DeleteRecord(ctx context.Context, domain, name, recordType string) error {
	return nil // No-op - always succeeds
}

// ListRecords is a no-op that returns an empty list
func (p *Provider) ListRecords(ctx context.Context, domain string) ([]*dns.Record, error) {
	return []*dns.Record{}, nil
}

// GetRecord is a no-op that returns nil (record not found)
func (p *Provider) GetRecord(ctx context.Context, domain, name, recordType string) (*dns.Record, error) {
	return nil, nil // Record not found
}

// CreateZone returns an error as zone management is not supported
func (p *Provider) CreateZone(ctx context.Context, req dns.CreateZoneRequest) (*dns.Zone, error) {
	return nil, ErrZoneManagementNotSupported
}

// DeleteZone returns an error as zone management is not supported
func (p *Provider) DeleteZone(ctx context.Context, zoneName string) error {
	return ErrZoneManagementNotSupported
}

// GetZone returns an error as zone management is not supported
func (p *Provider) GetZone(ctx context.Context, zoneName string) (*dns.Zone, error) {
	return nil, ErrZoneManagementNotSupported
}

// ListZones returns an error as zone management is not supported
func (p *Provider) ListZones(ctx context.Context) ([]*dns.Zone, error) {
	return nil, ErrZoneManagementNotSupported
}
