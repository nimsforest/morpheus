# Morpheus Forest Automation Roadmap

## Core Principle

**Morpheus is stateless.** All state lives in a remote registry (Hetzner StorageBox).

```
┌─────────────────┐
│    Morpheus     │  Stateless CLI
│   (any device)  │  No local state
└────────┬────────┘
         │
         ▼ reads/writes via WebDAV
┌─────────────────────────────────────┐
│       Hetzner StorageBox            │
│                                     │
│  /morpheus/registry.json            │
│  {                                  │
│    "forests": {...},                │
│    "nodes": {...}                   │
│  }                                  │
└─────────────────────────────────────┘
         ▲
         │ NATS cluster routes from registry
         │
┌────────┴────────┬─────────────────┐
│                 │                 │
▼                 ▼                 ▼
┌─────────┐  ┌─────────┐  ┌─────────┐
│  VM 1   │  │  VM 2   │  │  VM 3   │
│  NATS   │◄─┤  NATS   │◄─┤  NATS   │  Cluster
│ Forest  │  │ Forest  │  │ Forest  │
└─────────┘  └─────────┘  └─────────┘
```

---

## Task 1: StorageBox Registry

### Why StorageBox?

- **Simple**: Just a JSON file via WebDAV
- **Cheap**: ~€3/month for 100GB
- **No server**: No service to run/maintain
- **Hetzner native**: Same provider as VMs
- **Safe enough**: WebDAV supports ETags for optimistic locking

### Registry File Format

```json
{
  "version": 1,
  "updated_at": "2025-01-02T10:00:00Z",
  "forests": {
    "forest-1234567890": {
      "id": "forest-1234567890",
      "provider": "hetzner",
      "location": "fsn1",
      "size": "small",
      "status": "active",
      "created_at": "2025-01-02T10:00:00Z",
      "last_expansion": null
    }
  },
  "nodes": {
    "forest-1234567890": [
      {
        "id": "12345678",
        "ip": "2a01:4f8:c012:abc::1",
        "role": "edge",
        "status": "active",
        "created_at": "2025-01-02T10:00:05Z"
      }
    ]
  }
}
```

### Auto-Setup Flow

```
$ morpheus plant cloud small

🌲 No registry configured.

Morpheus needs a place to store forest state.
This allows running morpheus from any device.

Options:
  1. Create new Hetzner StorageBox (~€3/month)
  2. Use existing StorageBox
  3. Skip (local-only, single device)

Choice [1]: 1

Creating StorageBox... ✓
  Name: morpheus-registry
  URL:  https://u123456.your-storagebox.de

Initializing registry... ✓

Saving to config... ✓
  ~/.morpheus/config.yaml updated

Continuing with provisioning...
```

### Concurrency Safety

WebDAV supports ETags for optimistic locking:

```go
// Read
resp := GET /registry.json
etag := resp.Header("ETag")  // e.g., "abc123"
data := parse(resp.Body)

// Modify
data.Forests["new"] = forest

// Write with lock
req := PUT /registry.json
req.Header("If-Match", etag)  // Only if unchanged
resp := send(req)

if resp.Status == 412 {  // Precondition Failed
    // Someone else modified - retry from read
    retry()
}
```

### Files to Create

```
pkg/registry/
├── storagebox.go   # WebDAV client with ETag support
├── types.go        # RegistryData, Forest, Node structs
├── setup.go        # Auto-create StorageBox via Robot API
└── registry_test.go
```

---

## Task 2: NATS Server Installation

### Each VM Runs

1. **NATS Server** - Message broker
2. **NimsForest** - Business logic (connects to local NATS)

### Cluster Formation

```
Node 1 (first):
  - NATS starts as seed
  - No routes configured
  - Other nodes connect to it

Node 2+ (subsequent):
  - Gets Node 1's IP from registry
  - NATS config has route to Node 1
  - Joins cluster automatically
```

### NATS Config Template

```conf
# /etc/nats/nats.conf
port: 4222
http_port: 8222

jetstream {
  store_dir: /var/lib/nats/jetstream
  max_mem: 1G
  max_file: 10G
}

cluster {
  name: {{.ForestID}}
  port: 6222
  
  {{if not .IsFirstNode}}
  routes = [
    {{range .ClusterNodes}}
    nats-route://[{{.}}]:6222
    {{end}}
  ]
  {{end}}
}
```

### Service Dependencies

```
nats.service        (starts first)
       │
       ▼
nimsforest.service  (After=nats.service)
       │
       └─► NATS_URL=nats://localhost:4222
```

---

## Task 3: `morpheus grow`

### What It Does

1. Reads forest info from registry
2. Queries each node's NATS monitoring endpoint
3. Displays resource usage
4. Suggests expansion if above threshold
5. On confirm, provisions new node

### NATS Monitoring API

NATS exposes stats at `http://[ip]:8222/varz`:

```json
{
  "cpu": 12.5,
  "mem": 536870912,
  "connections": 45,
  "in_msgs": 1234567,
  "out_msgs": 2345678,
  "slow_consumers": 0
}
```

### Display

```
🌲 Forest: forest-1234567890
   Provider: hetzner (fsn1)
   Created: 2 days ago

NATS Cluster:
   Nodes: 2 connected
   Messages: 1.2M in / 3.4M out

Resources:
   CPU:    45% ████████████░░░░░░░░░░░░░
   Memory: 82% ████████████████████░░░░░ ⚠️

   NODE        IP                CPU    MEM    STATUS
   node-1      2a01:4f8::1       40%    78%    healthy
   node-2      2a01:4f8::2       50%    86%    warning

⚠️  Memory usage above 80% threshold

Recommendation: Add 1 node (~€3/month)

Proceed? [y/N]: y

Provisioning node-3...
  Creating server... ✓
  Waiting for boot... ✓
  Joining cluster... ✓

✅ Node added. Cluster now has 3 nodes.
```

---

## Task 4: `morpheus grow --auto`

### Usage

```bash
# One-time check
morpheus grow forest-123 --auto

# Cron job (every 15 min)
*/15 * * * * morpheus grow --auto --threshold 80
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--auto` | No prompts, auto-expand | false |
| `--threshold N` | Expand when above N% | 80 |
| `--output json` | Machine-readable | false |
| `--dry-run` | Show what would happen | false |

### Safety

```yaml
# config.yaml
growth:
  max_nodes: 10           # Never exceed this
  cooldown_minutes: 15    # Wait between expansions
  max_per_run: 1          # Add at most 1 node per run
```

### JSON Output

```json
{
  "forest_id": "forest-123",
  "status": "expanded",
  "reason": "memory_above_threshold",
  "metrics": {
    "cpu_percent": 45,
    "memory_percent": 82
  },
  "action": {
    "type": "provision_node",
    "node_id": "node-3"
  }
}
```

---

## Implementation Order

```
Week 1: Registry (Task 1)
├── Day 1-2: StorageBox client (WebDAV + ETag)
├── Day 3: Auto-create via Robot API
├── Day 4: Interactive setup flow
└── Day 5: Migrate all commands to use registry

Week 2: NATS + Clustering (Task 2)
├── Day 1-2: Cloud-init templates for NATS
├── Day 3: Cluster config generation
└── Day 4-5: Testing multi-node clusters

Week 3: Grow Command (Task 3 + 4)
├── Day 1-2: NATS monitoring client
├── Day 3: Interactive grow command
├── Day 4: Auto mode + flags
└── Day 5: Testing + documentation
```

---

## Config Reference

```yaml
# ~/.morpheus/config.yaml

infrastructure:
  provider: hetzner

registry:
  type: storagebox                    # storagebox | s3 | none
  url: "https://u123456.your-storagebox.de/morpheus/registry.json"
  username: "u123456"
  password: "${STORAGEBOX_PASSWORD}"  # From environment

integration:
  nimsforest_install: true
  nimsforest_download_url: "https://nimsforest.io/bin/nimsforest"
  nats_install: true
  nats_version: "2.10.24"

growth:
  enabled: true
  threshold_cpu: 80
  threshold_memory: 80
  max_nodes: 10
  cooldown_minutes: 15

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"
```

---

## API Reference

### Hetzner Cloud API
- Base: `https://api.hetzner.cloud/v1`
- Used for: VMs, firewalls, SSH keys
- Auth: Bearer token

### Hetzner Robot API
- Base: `https://robot-ws.your-server.de`
- Used for: StorageBox management
- Auth: Basic auth (separate credentials)

### StorageBox WebDAV
- URL: `https://u{userid}.your-storagebox.de/`
- Auth: Basic auth
- Supports: PUT, GET, DELETE, PROPFIND
- ETags: Yes (for optimistic locking)

### NATS Monitoring
- URL: `http://[node-ip]:8222/varz`
- Auth: None (internal network only)
- Returns: JSON with server stats
