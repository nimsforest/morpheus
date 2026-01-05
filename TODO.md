# Morpheus TODO

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Morpheus                               │
│                                                                 │
│        morpheus plant          morpheus grow <forest-id>        │
│                                                                 │
└───────────┬───────────────────┬───────────────────┬─────────────┘
            │                   │                   │
            ▼                   ▼                   ▼
   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
   │ Machine Provider│ │  DNS Provider   │ │Storage Provider │
   │                 │ │                 │ │                 │
   │  - hetzner      │ │  - hetzner      │ │  - storagebox   │
   │  - local        │ │  - cloudflare   │ │  - s3           │
   │  - aws          │ │  - hosts        │ │  - local        │
   │  - gcp          │ │  - none         │ │  - none         │
   └─────────────────┘ └─────────────────┘ └─────────────────┘
```

Three independent providers. Mix and match.

---

## CLI

```bash
morpheus plant                  # Create forest (1 node)
morpheus plant --nodes 3        # Create forest (3 nodes)
morpheus grow <forest-id>       # Add 1 node
morpheus grow <forest-id> -n 2  # Add 2 nodes
morpheus list                   # List forests
morpheus status <forest-id>     # Show details
morpheus teardown <forest-id>   # Delete forest
```

No `cloud`/`local`. No `small`/`medium`/`large`. Just plant and grow.

---

## Config

```yaml
machine:
  provider: hetzner
  hetzner:
    server_type: cx22
    image: ubuntu-24.04
    location: fsn1
  ssh:
    key_name: morpheus

dns:
  provider: hetzner
  domain: morpheus.example.com
  ttl: 300

storage:
  provider: storagebox
  storagebox:
    host: uXXXXX.your-storagebox.de
    username: uXXXXX
    password: ${STORAGEBOX_PASSWORD}

secrets:
  hetzner_api_token: ${HETZNER_API_TOKEN}
  hetzner_dns_token: ${HETZNER_DNS_TOKEN}
```

---

## Implementation

### Phase 1: Refactor Structure
- [ ] Rename `pkg/provider` → `pkg/machine`
- [ ] Rename `pkg/registry` → `pkg/storage`
- [ ] Create `pkg/dns`
- [ ] Update config: `machine`, `dns`, `storage` sections
- [ ] Remove `Size` from Forest model

### Phase 2: CLI Simplification
- [ ] `morpheus plant` (no args)
- [ ] `morpheus plant --nodes N`
- [ ] `morpheus grow <forest-id>`
- [ ] `morpheus grow <forest-id> --nodes N`

### Phase 3: DNS Provider
- [ ] `pkg/dns/interface.go`
- [ ] `pkg/dns/hetzner/` - Hetzner DNS API
- [ ] `pkg/dns/none/` - No-op
- [ ] Create A/AAAA records on provision
- [ ] Delete records on teardown
- [ ] Service record: `nats.<forest>.<domain>`

### Phase 4: Additional Providers
- [ ] `pkg/dns/cloudflare/`
- [ ] `pkg/dns/hosts/`
- [ ] `pkg/storage/s3/`

---

## Provider Interfaces

```go
// pkg/machine/interface.go
type Provider interface {
    CreateServer(ctx, req) (*Server, error)
    GetServer(ctx, id) (*Server, error)
    DeleteServer(ctx, id) error
    WaitForServer(ctx, id, state) error
    ListServers(ctx, filters) ([]*Server, error)
}

// pkg/dns/interface.go
type Provider interface {
    CreateRecord(ctx, req) (*Record, error)
    DeleteRecord(ctx, domain, name, type) error
    ListRecords(ctx, domain) ([]*Record, error)
}

// pkg/storage/interface.go
type Provider interface {
    GetRegistry(ctx) (*RegistryData, error)
    SaveRegistry(ctx, data) error
    GetMountConfig() *MountConfig
}
```

---

## Directory Structure

```
pkg/
├── machine/
│   ├── interface.go
│   ├── hetzner/
│   ├── local/
│   └── none/
├── dns/
│   ├── interface.go
│   ├── hetzner/
│   ├── cloudflare/
│   ├── hosts/
│   └── none/
├── storage/
│   ├── interface.go
│   ├── storagebox/
│   ├── s3/
│   ├── local/
│   └── none/
├── forest/
│   └── provisioner.go
└── config/
    └── config.go
```
