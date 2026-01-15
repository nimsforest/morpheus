# Feature: DNS Delegation for Customer Subdomains

## Status: planning

## Summary

Extend Morpheus DNS capabilities to support multi-customer subdomain delegation via Hetzner DNS. Customers delegate subdomains (e.g., `experiencenet.customer.com`) to Hetzner nameservers, and Morpheus manages all records using per-customer API tokens.

## Architecture Reference

See [docs/architecture/DNS_DELEGATION.md](../docs/architecture/DNS_DELEGATION.md) for the complete architecture documentation.

## Tasks

### Phase 1: DNS Provider Extensions

- [ ] Task 1.1 - Add `CreateZone` method to DNS provider interface (~pkg/dns/interface.go)
- [ ] Task 1.2 - Add `DeleteZone` method to DNS provider interface (~pkg/dns/interface.go)
- [ ] Task 1.3 - Add `GetZone` method to DNS provider interface (~pkg/dns/interface.go)
- [ ] Task 1.4 - Implement zone management in Hetzner DNS provider (~pkg/dns/hetzner/hetzner.go)
- [ ] Task 1.5 - Add zone types to DNS package (~pkg/dns/interface.go)

### Phase 2: Multi-Customer Support

- [ ] Task 2.1 - Create customer configuration model (~pkg/customer/types.go)
- [ ] Task 2.2 - Add customer config loader with token resolution (~pkg/customer/config.go)
- [ ] Task 2.3 - Support multiple DNS provider instances per customer (~pkg/dns/factory.go)
- [ ] Task 2.4 - Add customer context to DNS operations (~pkg/dns/interface.go)

### Phase 3: Apex Domain & Floating IP Support

- [ ] Task 3.1 - Add Floating IP provider interface (~pkg/network/interface.go)
- [ ] Task 3.2 - Implement Hetzner Floating IP provider (~pkg/network/hetzner/floatingip.go)
- [ ] Task 3.3 - Add apex domain configuration model (~pkg/customer/apex.go)
- [ ] Task 3.4 - Integrate Floating IP provisioning with DNS record creation (~pkg/dns/apex.go)
- [ ] Task 3.5 - Add Floating IP health check and reassignment logic (~pkg/network/failover.go)
- [ ] Task 3.6 - Support full domain delegation (apex zone creation) (~pkg/dns/hetzner/hetzner.go)

### Phase 4: CLI Commands

- [ ] Task 4.1 - Add `morpheus dns zone create` command (~internal/commands/dns_zone.go)
- [ ] Task 4.2 - Add `morpheus dns zone list` command (~internal/commands/dns_zone.go)
- [ ] Task 4.3 - Add `morpheus dns zone delete` command (~internal/commands/dns_zone.go)
- [ ] Task 4.4 - Add `morpheus dns record create` command (~internal/commands/dns_record.go)
- [ ] Task 4.5 - Add `morpheus dns record list` command (~internal/commands/dns_record.go)
- [ ] Task 4.6 - Add `morpheus dns record delete` command (~internal/commands/dns_record.go)
- [ ] Task 4.7 - Add `--customer` flag support to DNS commands (~internal/commands/dns.go)
- [ ] Task 4.8 - Add `morpheus floating-ip create` command (~internal/commands/floatingip.go)
- [ ] Task 4.9 - Add `morpheus floating-ip assign` command (~internal/commands/floatingip.go)
- [ ] Task 4.10 - Add `morpheus dns apex setup` command for apex domain setup (~internal/commands/dns_apex.go)

### Phase 5: Customer Onboarding Automation

- [ ] Task 5.1 - Add `morpheus customer init` command for new customer setup (~internal/commands/customer.go)
- [ ] Task 5.2 - Generate NS record instructions for customer (~pkg/customer/onboarding.go)
- [ ] Task 5.3 - Add zone verification (check NS delegation) (~pkg/dns/verification.go)
- [ ] Task 5.4 - Add `morpheus customer verify` command (~internal/commands/customer.go)
- [ ] Task 5.5 - Generate apex setup instructions based on customer DNS provider (~pkg/customer/onboarding.go)

### Phase 6: Venture Service Integration

- [ ] Task 6.1 - Define venture service record templates (~pkg/venture/templates.go)
- [ ] Task 6.2 - Add `morpheus venture enable` command (~internal/commands/venture.go)
- [ ] Task 6.3 - Auto-provision DNS records when venture is enabled (~pkg/venture/provisioner.go)
- [ ] Task 6.4 - Auto-cleanup DNS records when venture is disabled (~pkg/venture/provisioner.go)
- [ ] Task 6.5 - Handle apex records for ventures that need them (~pkg/venture/apex.go)

### Phase 7: Testing & Documentation

- [ ] Task 7.1 - Unit tests for zone management (~pkg/dns/hetzner/hetzner_test.go)
- [ ] Task 7.2 - Unit tests for customer config (~pkg/customer/config_test.go)
- [ ] Task 7.3 - Unit tests for Floating IP provider (~pkg/network/hetzner/floatingip_test.go)
- [ ] Task 7.4 - Integration tests with Hetzner DNS API (~pkg/dns/hetzner/integration_test.go)
- [ ] Task 7.5 - Integration tests with Hetzner Floating IP API (~pkg/network/hetzner/integration_test.go)
- [ ] Task 7.6 - Update main README with DNS delegation usage (~README.md)
- [ ] Task 7.7 - Add customer onboarding guide (~docs/guides/CUSTOMER_ONBOARDING.md)

## Parallelization

**Group A (Independent - can run in parallel):**
- Task 1.1, 1.2, 1.3, 1.5 (DNS interface changes)
- Task 3.1 (Floating IP interface - independent)

**Group B (Depends on A):**
- Task 1.4 (DNS implementation depends on interface)
- Task 3.2 (Floating IP implementation depends on interface)

**Group C (Independent - can run in parallel with B):**
- Task 2.1, 2.2 (customer config model)
- Task 3.3 (apex config model)

**Group D (Depends on B, C):**
- Task 2.3, 2.4 (multi-customer DNS integration)
- Task 3.4, 3.5, 3.6 (apex/floating IP integration)

**Group E (Depends on D):**
- Task 4.1-4.10 (CLI commands depend on core implementation)

**Group F (Depends on E):**
- Task 5.1-5.5 (onboarding commands)

**Group G (Depends on D, can parallel with F):**
- Task 6.1-6.5 (venture integration)

**Group H (Depends on all above):**
- Task 7.1-7.7 (testing & documentation)

## Files

### New Files
- `pkg/customer/types.go` - Customer configuration types
- `pkg/customer/config.go` - Customer config loader
- `pkg/customer/onboarding.go` - Onboarding utilities
- `pkg/customer/apex.go` - Apex domain configuration
- `pkg/dns/factory.go` - DNS provider factory with multi-customer support
- `pkg/dns/verification.go` - DNS delegation verification
- `pkg/dns/apex.go` - Apex domain DNS integration
- `pkg/network/interface.go` - Network provider interface (Floating IP)
- `pkg/network/hetzner/floatingip.go` - Hetzner Floating IP provider
- `pkg/network/hetzner/floatingip_test.go` - Floating IP tests
- `pkg/network/failover.go` - Floating IP health check and failover
- `pkg/venture/templates.go` - Venture service DNS templates
- `pkg/venture/provisioner.go` - Venture DNS provisioner
- `pkg/venture/apex.go` - Venture apex record handling
- `internal/commands/dns.go` - DNS command group
- `internal/commands/dns_zone.go` - Zone subcommands
- `internal/commands/dns_record.go` - Record subcommands
- `internal/commands/dns_apex.go` - Apex domain setup command
- `internal/commands/floatingip.go` - Floating IP commands
- `internal/commands/customer.go` - Customer commands
- `internal/commands/venture.go` - Venture commands
- `docs/guides/CUSTOMER_ONBOARDING.md` - Customer onboarding guide

### Modified Files
- `pkg/dns/interface.go` - Add zone management interface
- `pkg/dns/hetzner/hetzner.go` - Implement zone management + apex support
- `pkg/dns/hetzner/hetzner_test.go` - Add zone tests
- `internal/cli/root.go` - Register new command groups
- `README.md` - Add DNS delegation documentation
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

### Floating IP Provider Interface

```go
// pkg/network/interface.go
type FloatingIPProvider interface {
    // Create provisions a new floating IP
    Create(ctx context.Context, req CreateFloatingIPRequest) (*FloatingIP, error)

    // Delete removes a floating IP
    Delete(ctx context.Context, id string) error

    // Assign attaches a floating IP to a server
    Assign(ctx context.Context, floatingIPID, serverID string) error

    // Unassign detaches a floating IP from its current server
    Unassign(ctx context.Context, floatingIPID string) error

    // Get retrieves floating IP details
    Get(ctx context.Context, id string) (*FloatingIP, error)

    // List returns all floating IPs
    List(ctx context.Context) ([]*FloatingIP, error)
}

type FloatingIP struct {
    ID         string
    IP         string
    Type       string // "ipv4" or "ipv6"
    ServerID   string // Empty if unassigned
    Location   string
    Name       string
    Labels     map[string]string
}

type CreateFloatingIPRequest struct {
    Type     string // "ipv4" or "ipv6"
    Location string
    Name     string
    Labels   map[string]string
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
    Apex     *ApexConfig // Optional apex domain configuration
}

type HetznerConfig struct {
    ProjectID   string
    DNSToken    string // From env or Bitwarden reference
    CloudToken  string // For Floating IP management
}

type ApexConfig struct {
    Mode         string // "subdomain", "full-delegation", "floating-ip"
    FloatingIPID string // For floating-ip mode
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

# Apex domain setup
morpheus dns apex setup acme --mode floating-ip      # Provision Floating IP for apex
morpheus dns apex setup acme --mode full-delegation  # Full domain delegation
morpheus dns apex status acme                        # Check apex configuration

# Floating IP management
morpheus floating-ip create --customer acme --name apex-ip --location fsn1
morpheus floating-ip list --customer acme
morpheus floating-ip assign <ip-id> <server-id> --customer acme
morpheus floating-ip delete <ip-id> --customer acme

# Customer onboarding
morpheus customer init acme --domain customer.com
morpheus customer init acme --domain customer.com --apex-mode floating-ip  # With apex
morpheus customer verify acme  # Check NS delegation

# Venture management
morpheus venture enable acme experiencenet
morpheus venture enable acme nimsforest --with-apex  # Venture needs apex A record
morpheus venture disable acme experiencenet
```

## Dependencies

- Hetzner DNS API access (for zone/record management)
- Hetzner Cloud API access (for Floating IP management)
- Customer Hetzner projects (manual setup)
- Bitwarden CLI (optional, for token retrieval)

## Notes

- Zone creation requires the token to have appropriate permissions in the Hetzner project
- Floating IP requires Hetzner Cloud token (separate from DNS token)
- NS delegation verification may take time due to DNS propagation
- Floating IPs incur monthly cost even when unassigned
- Consider rate limiting for bulk operations
- Token rotation should be handled separately (post-MVP)

## Apex Domain Decision Tree

```
Customer needs apex domain hosting?
│
├─ No → Use subdomain delegation only
│
└─ Yes → Does customer's DNS provider support ALIAS/ANAME?
         │
         ├─ Yes → Customer configures ALIAS at apex (no Morpheus action)
         │
         └─ No → Customer willing to do full domain delegation?
                 │
                 ├─ Yes → Full delegation mode
                 │        - Customer changes nameservers at registrar
                 │        - Morpheus manages entire zone including apex
                 │
                 └─ No → Floating IP mode
                         - Morpheus provisions Hetzner Floating IP
                         - Customer sets apex A record to Floating IP
                         - Morpheus manages Floating IP assignment
```
