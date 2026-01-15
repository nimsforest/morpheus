# Feature: DNS Management via Hetzner

## Status: done

## Summary

Extend Morpheus DNS capabilities to support zone creation and record management via Hetzner DNS API. Support both internal ventures (where we control the apex) and customer subdomain delegation.

## Architecture Reference

See [docs/architecture/DNS_DELEGATION.md](../docs/architecture/DNS_DELEGATION.md) for the complete architecture documentation.

## Tasks

### Phase 1: DNS Provider Extensions

- [x] Task 1.1 - Add `CreateZone` method to DNS provider interface (~pkg/dns/interface.go)
- [x] Task 1.2 - Add `DeleteZone` method to DNS provider interface (~pkg/dns/interface.go)
- [x] Task 1.3 - Add `GetZone` and `ListZones` methods to DNS provider interface (~pkg/dns/interface.go)
- [x] Task 1.4 - Add zone types to DNS package (~pkg/dns/interface.go)
- [x] Task 1.5 - Implement zone management in Hetzner DNS provider (~pkg/dns/hetzner/hetzner.go)

### Phase 2: Multi-Customer Support

- [x] Task 2.1 - Create customer configuration model (~pkg/customer/types.go)
- [x] Task 2.2 - Add customer config loader with token resolution (~pkg/customer/config.go)
- [x] Task 2.3 - Support multiple DNS provider instances per customer (simplified - direct in commands)

### Phase 3: CLI Commands

- [x] Task 3.1 - Add `morpheus dns zone create` command (~internal/commands/dns_zone.go)
- [x] Task 3.2 - Add `morpheus dns zone list` command (~internal/commands/dns_zone.go)
- [x] Task 3.3 - Add `morpheus dns zone delete` command (~internal/commands/dns_zone.go)
- [x] Task 3.4 - Add `morpheus dns record create` command (~internal/commands/dns_record.go)
- [x] Task 3.5 - Add `morpheus dns record list` command (~internal/commands/dns_record.go)
- [x] Task 3.6 - Add `morpheus dns record delete` command (~internal/commands/dns_record.go)
- [x] Task 3.7 - Add `--customer` flag support to DNS commands (~internal/commands/dns.go)

### Phase 4: Customer Onboarding Automation

- [x] Task 4.1 - Add `morpheus customer init` command for new customer setup (~internal/commands/customer.go)
- [x] Task 4.2 - Generate NS record instructions for customer (~pkg/customer/onboarding.go)
- [x] Task 4.3 - Add zone verification (check NS delegation) (~pkg/dns/verification.go)
- [x] Task 4.4 - Add `morpheus customer verify` command (~internal/commands/customer.go)

### Phase 5: Venture Service Integration

- [x] Task 5.1 - Define venture service record templates (~pkg/venture/templates.go)
- [x] Task 5.2 - Add `morpheus venture enable` command (~internal/commands/venture.go)
- [x] Task 5.3 - Auto-provision DNS records when venture is enabled (~pkg/venture/provisioner.go)
- [x] Task 5.4 - Auto-cleanup DNS records when venture is disabled (~pkg/venture/provisioner.go)

### Phase 6: Testing & Documentation

- [x] Task 6.1 - Unit tests for zone management (~pkg/dns/hetzner/hetzner_test.go)
- [x] Task 6.2 - Unit tests for customer config (~pkg/customer/config_test.go)
- [ ] Task 6.3 - Integration tests with Hetzner DNS API (~pkg/dns/hetzner/integration_test.go)
- [x] Task 6.4 - Update main README with DNS management usage (~README.md)
- [x] Task 6.5 - Add customer onboarding guide (~docs/guides/CUSTOMER_ONBOARDING.md)

## Parallelization

**Group A (Independent - can run in parallel):**
- Task 1.1, 1.2, 1.3, 1.4 (interface changes)

**Group B (Depends on A):**
- Task 1.5 (implementation depends on interface)

**Group C (Independent - can run in parallel with B):**
- Task 2.1, 2.2 (customer config model)

**Group D (Depends on B, C):**
- Task 2.3 (multi-customer DNS integration)

**Group E (Depends on D):**
- Task 3.1-3.7 (CLI commands depend on core implementation)

**Group F (Depends on E):**
- Task 4.1-4.4 (onboarding commands)

**Group G (Depends on D, can parallel with F):**
- Task 5.1-5.4 (venture integration)

**Group H (Depends on all above):**
- Task 6.1-6.5 (testing & documentation)

## Files

### New Files
- `pkg/customer/types.go` - Customer configuration types
- `pkg/customer/config.go` - Customer config loader
- `pkg/customer/onboarding.go` - Onboarding utilities
- `pkg/dns/factory.go` - DNS provider factory with multi-customer support
- `pkg/dns/verification.go` - DNS delegation verification
- `pkg/venture/templates.go` - Venture service DNS templates
- `pkg/venture/provisioner.go` - Venture DNS provisioner
- `internal/commands/dns.go` - DNS command group
- `internal/commands/dns_zone.go` - Zone subcommands
- `internal/commands/dns_record.go` - Record subcommands
- `internal/commands/customer.go` - Customer commands
- `internal/commands/venture.go` - Venture commands
- `docs/guides/CUSTOMER_ONBOARDING.md` - Customer onboarding guide

### Modified Files
- `pkg/dns/interface.go` - Add zone management interface
- `pkg/dns/hetzner/hetzner.go` - Implement zone management
- `pkg/dns/hetzner/hetzner_test.go` - Add zone tests
- `internal/cli/root.go` - Register new command groups
- `README.md` - Add DNS management documentation
- `docs/README.md` - Add links to new docs

## API Design

### Extended DNS Provider Interface

```go
type Provider interface {
    // Existing methods
    CreateRecord(ctx context.Context, req CreateRecordRequest) (*Record, error)
    DeleteRecord(ctx context.Context, domain, name, recordType string) error
    ListRecords(ctx context.Context, domain string) ([]*Record, error)
    GetRecord(ctx context.Context, domain, name, recordType string) (*Record, error)

    // New zone management methods
    CreateZone(ctx context.Context, req CreateZoneRequest) (*Zone, error)
    DeleteZone(ctx context.Context, zoneName string) error
    GetZone(ctx context.Context, zoneName string) (*Zone, error)
    ListZones(ctx context.Context) ([]*Zone, error)
}

type Zone struct {
    ID          string
    Name        string
    TTL         int
    Nameservers []string
}

type CreateZoneRequest struct {
    Name string
    TTL  int
}
```

### Customer Configuration

```go
type Customer struct {
    ID       string
    Name     string
    Domain   string   // Root domain (e.g., "customer.com")
    Ventures []string // Enabled ventures (e.g., ["experiencenet", "nimsforest"])
    Hetzner  HetznerConfig
}

type HetznerConfig struct {
    ProjectID string
    APIToken  string // From env or Bitwarden reference
}
```

### CLI Usage Examples

```bash
# Zone management
morpheus dns zone create experiencenet.customer.com --customer acme
morpheus dns zone list --customer acme
morpheus dns zone delete experiencenet.customer.com --customer acme

# Record management
morpheus dns record create www.experiencenet.customer.com A 1.2.3.4 --customer acme
morpheus dns record list experiencenet.customer.com --customer acme
morpheus dns record delete www.experiencenet.customer.com A --customer acme

# Customer onboarding
morpheus customer init acme --domain customer.com
morpheus customer verify acme  # Check NS delegation

# Venture management
morpheus venture enable acme experiencenet
morpheus venture disable acme experiencenet
```

## Dependencies

- Hetzner DNS API access
- Customer Hetzner projects (manual setup)
- Bitwarden CLI (optional, for token retrieval)

## Notes

- Zone creation requires the token to have appropriate permissions in the Hetzner project
- NS delegation verification may take time due to DNS propagation
- Consider rate limiting for bulk operations
- Token rotation should be handled separately (post-MVP)
