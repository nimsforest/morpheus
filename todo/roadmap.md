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
morpheus grow <forest-id>       # Check health
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
```

---

## Implementation

### Phase 1: Refactor Structure ✅
- [x] Rename `pkg/provider` → `pkg/machine`
- [x] Rename `pkg/registry` → `pkg/storage`
- [x] Create `pkg/dns`
- [x] Update config: `machine`, `dns`, `storage` sections
- [x] Remove `Size` from Forest model (use NodeCount instead)

### Phase 2: CLI Simplification ✅
- [x] `morpheus plant` (no args, defaults to 1 node)
- [x] `morpheus plant --nodes N`
- [x] `morpheus grow <forest-id>` (check health)
- [x] `morpheus grow <forest-id> --nodes N` (add nodes)

### Phase 3: DNS Provider ✅
- [x] `pkg/dns/interface.go`
- [x] `pkg/dns/hetzner/` - Hetzner DNS API
- [x] `pkg/dns/none/` - No-op
- [x] Create A/AAAA records on provision
- [x] Delete records on teardown
- [ ] Service record: `nats.<forest>.<domain>` (future)

### Phase 4: Additional Providers (Future)
- [ ] `pkg/dns/cloudflare/`
- [ ] `pkg/dns/hosts/`
- [ ] `pkg/storage/s3/`
- [ ] `pkg/machine/aws/`
- [ ] `pkg/machine/gcp/`

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
type Registry interface {
    RegisterForest(forest) error
    RegisterNode(node) error
    GetForest(forestID) (*Forest, error)
    GetNodes(forestID) ([]*Node, error)
    UpdateForest(updated) error
    UpdateForestStatus(forestID, status) error
    UpdateNodeStatus(forestID, nodeID, status) error
    DeleteForest(forestID) error
    ListForests() []*Forest
}
```

---

## Directory Structure

```
pkg/
├── machine/
│   ├── interface.go
│   ├── profile.go
│   ├── hetzner/
│   │   ├── hetzner.go
│   │   └── profiles.go
│   ├── local/
│   │   └── local.go
│   └── none/
│       └── none.go
├── dns/
│   ├── interface.go
│   ├── hetzner/
│   │   └── hetzner.go
│   └── none/
│       └── none.go
├── storage/
│   ├── interface.go
│   ├── types.go
│   ├── local.go
│   └── storagebox.go
├── forest/
│   └── provisioner.go
└── config/
    └── config.go
```

---

## Notes

### Backward Compatibility
The old config format (using `infrastructure`, `registry`, etc.) is still supported
through automatic migration in the config loader. Users can continue using their
existing config files.

### Legacy Package Preservation
The old `pkg/provider` and `pkg/registry` packages are preserved for backward
compatibility with any external code that might depend on them. New code should
use `pkg/machine` and `pkg/storage` respectively.
