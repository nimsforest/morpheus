package dns

import (
	"context"
)

// Provider defines the interface for DNS management
type Provider interface {
	// CreateRecord creates a DNS record
	CreateRecord(ctx context.Context, req CreateRecordRequest) (*Record, error)

	// DeleteRecord removes a DNS record
	DeleteRecord(ctx context.Context, domain, name, recordType string) error

	// ListRecords lists all DNS records for a domain
	ListRecords(ctx context.Context, domain string) ([]*Record, error)

	// GetRecord retrieves a specific DNS record
	GetRecord(ctx context.Context, domain, name, recordType string) (*Record, error)
}

// CreateRecordRequest contains parameters for creating a DNS record
type CreateRecordRequest struct {
	Domain string     // The zone/domain (e.g., "example.com")
	Name   string     // The record name (e.g., "forest-123" for forest-123.example.com)
	Type   RecordType // A, AAAA, CNAME, etc.
	Value  string     // IP address or target
	TTL    int        // Time-to-live in seconds (0 = use default)
}

// Record represents a DNS record
type Record struct {
	ID     string     // Provider-specific record ID
	Domain string     // The zone/domain
	Name   string     // The record name
	Type   RecordType // Record type
	Value  string     // IP address or target
	TTL    int        // Time-to-live in seconds
}

// RecordType represents the type of DNS record
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeTXT   RecordType = "TXT"
	RecordTypeSRV   RecordType = "SRV"
)

// ForestDNSConfig contains DNS settings for a forest
type ForestDNSConfig struct {
	// Domain is the base domain for DNS records (e.g., "morpheus.example.com")
	Domain string

	// ForestID is the unique identifier for the forest
	ForestID string

	// TTL is the time-to-live for DNS records
	TTL int
}

// GetForestHostname returns the hostname for a forest (e.g., "forest-123.morpheus.example.com")
func (c *ForestDNSConfig) GetForestHostname() string {
	return c.ForestID + "." + c.Domain
}

// GetNodeHostname returns the hostname for a specific node
func (c *ForestDNSConfig) GetNodeHostname(nodeIndex int) string {
	return c.ForestID + "-node-" + string(rune('0'+nodeIndex)) + "." + c.Domain
}

// GetNATSServiceHostname returns the hostname for the NATS service
func (c *ForestDNSConfig) GetNATSServiceHostname() string {
	return "nats." + c.ForestID + "." + c.Domain
}
