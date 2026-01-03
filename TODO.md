# Morpheus TODO

## Architecture (Updated for Embedded NATS)

```
┌─────────────────┐
│    Morpheus     │  Stateless CLI (phone/laptop)
└────────┬────────┘
         │ reads/writes
         ▼
┌─────────────────┐
│ Hetzner         │  Registry = JSON file at /mnt/forest/registry.json
│ StorageBox      │  Mounted via CIFS on each node
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Hetzner VMs     │  Each runs NimsForest with EMBEDDED NATS
│ (the forest)    │  Self-discovers peers via shared registry
└─────────────────┘
```

**Key Change:** NimsForest now embeds NATS server directly. No separate NATS installation needed!

---

## MVP Plan: Integration with Embedded NATS

### How It Works Now

```
Node-1 starts:
  → Reads /mnt/forest/registry.json → no other nodes yet
  → Starts nimsforest with embedded NATS as cluster of 1

Node-2 starts:
  → Reads registry → sees node-1
  → Connects to node-1 via NATS gossip
  → Cluster is now 2 nodes

Node-N added:
  → Reads registry → finds at least one peer
  → NATS gossip propagates membership automatically
```

**A new node only needs ONE existing peer.** NATS gossip handles the rest.

---

## Task 1: Update Cloud-Init for Embedded NATS
**Status:** ⬜ Not Started  
**Priority:** Critical  
**Estimated:** 3-4 hours

Simplify cloud-init to work with nimsforest's embedded NATS.

### 1.1 Remove separate NATS installation

**File:** `pkg/cloudinit/templates.go`

Remove all the NATS download/systemd sections. NimsForest binary now handles everything.

### 1.2 Add StorageBox CIFS mount

```yaml
# Mount StorageBox for shared registry
- |
  mkdir -p /mnt/forest
  echo "//{{.StorageBoxHost}}/backup /mnt/forest cifs user={{.StorageBoxUser}},pass={{.StorageBoxPassword}},uid=root,gid=root 0 0" >> /etc/fstab
  mount /mnt/forest
```

### 1.3 Write node-info.json

```yaml
# Write node info for nimsforest
- |
  mkdir -p /etc/morpheus
  cat > /etc/morpheus/node-info.json << 'EOF'
  {
    "forest_id": "{{.ForestID}}",
    "node_id": "{{.NodeID}}"
  }
  EOF
```

### 1.4 Register node in shared registry

```yaml
# Add this node to registry (atomic read-modify-write)
- |
  REGISTRY=/mnt/forest/registry.json
  NODE_INFO='{"id":"{{.NodeID}}","ip":"{{.NodeIP}}","forest_id":"{{.ForestID}}"}'
  
  # Create registry if missing
  [ -f "$REGISTRY" ] || echo '{"nodes":{}}' > "$REGISTRY"
  
  # Use flock for atomic update
  flock "$REGISTRY" python3 << 'PYEOF'
  import json
  with open('/mnt/forest/registry.json', 'r+') as f:
      reg = json.load(f)
      if '{{.ForestID}}' not in reg['nodes']:
          reg['nodes']['{{.ForestID}}'] = []
      # Add node if not exists
      if not any(n['id'] == '{{.NodeID}}' for n in reg['nodes']['{{.ForestID}}']):
          reg['nodes']['{{.ForestID}}'].append({"id": "{{.NodeID}}", "ip": "{{.NodeIP}}", "forest_id": "{{.ForestID}}"})
      f.seek(0)
      f.truncate()
      json.dump(reg, f, indent=2)
  PYEOF
```

### 1.5 Create JetStream directory

```yaml
- mkdir -p /var/lib/nimsforest/jetstream
```

### 1.6 Simplified NimsForest service

```yaml
- |
  cat > /etc/systemd/system/nimsforest.service << 'EOF'
  [Unit]
  Description=NimsForest (with embedded NATS)
  After=network-online.target mnt-forest.mount
  
  [Service]
  ExecStart=/opt/nimsforest/bin/forest
  Restart=always
  RestartSec=5
  WorkingDirectory=/var/lib/nimsforest
  
  [Install]
  WantedBy=multi-user.target
  EOF
  
  systemctl daemon-reload
  systemctl enable nimsforest
  systemctl start nimsforest
```

### 1.7 Update TemplateData struct

```go
type TemplateData struct {
    // Existing fields...
    
    // Node identification
    ForestID string
    NodeID   string
    NodeIP   string
    
    // StorageBox mount (for shared registry)
    StorageBoxHost     string  // uXXXXX.your-storagebox.de
    StorageBoxUser     string  // uXXXXX
    StorageBoxPassword string
    
    // Removed:
    // - NATSInstall, NATSVersion (not needed anymore)
    // - ClusterNodes, IsFirstNode (NATS gossip handles this)
}
```

### Acceptance Criteria
- [ ] Cloud-init templates no longer install separate NATS
- [ ] StorageBox mounted at /mnt/forest/ on each node
- [ ] /etc/morpheus/node-info.json written correctly
- [ ] Node registered in /mnt/forest/registry.json
- [ ] NimsForest starts and forms cluster automatically

---

## Task 2: Update Provisioner for Simplified Flow
**Status:** ⬜ Not Started  
**Priority:** High  
**Estimated:** 2-3 hours

### 2.1 Update provisioner to pass StorageBox creds

**File:** `pkg/forest/provisioner.go`

```go
func (p *Provisioner) ProvisionNode(ctx context.Context, req ProvisionRequest) (*Node, error) {
    // Generate node ID
    nodeID := fmt.Sprintf("node-%s", generateID())
    
    // Get node's primary IPv6
    nodeIP := req.PrimaryIPv6
    
    // Build cloud-init data
    cloudInitData := cloudinit.TemplateData{
        ForestID:           req.ForestID,
        NodeID:             nodeID,
        NodeIP:             nodeIP,
        StorageBoxHost:     p.config.Registry.StorageBoxHost,
        StorageBoxUser:     p.config.Registry.Username,
        StorageBoxPassword: p.config.Registry.Password,
        // NimsForest binary
        NimsforestURL:      req.NimsforestURL,
    }
    
    // No need to query existing cluster nodes anymore
    // NATS gossip handles peer discovery from registry
    
    return p.provider.CreateServer(ctx, createReq)
}
```

### 2.2 Remove NATS-specific cluster logic

The provisioner no longer needs to:
- Track which node is "first"
- Pass list of existing cluster IPs
- Configure NATS routes

NimsForest handles all this internally by reading `/mnt/forest/registry.json`.

### Acceptance Criteria
- [ ] Provisioner passes StorageBox mount info
- [ ] No NATS-specific configuration in provisioner
- [ ] Each node gets unique NodeID

---

## Task 3: Update Registry Config
**Status:** ⬜ Not Started  
**Priority:** High  
**Estimated:** 1-2 hours

### 3.1 Add StorageBox mount config

**File:** `pkg/config/config.go`

```go
type RegistryConfig struct {
    Type            string `yaml:"type"`             // "storagebox"
    URL             string `yaml:"url"`              // WebDAV URL (for Morpheus access)
    Username        string `yaml:"username"`         // uXXXXX
    Password        string `yaml:"password"`
    StorageBoxHost  string `yaml:"storagebox_host"`  // uXXXXX.your-storagebox.de (for CIFS mount)
}
```

### 3.2 Example config.yaml

```yaml
registry:
  type: storagebox
  url: https://uXXXXX.your-storagebox.de/morpheus/registry.json  # WebDAV for CLI
  storagebox_host: uXXXXX.your-storagebox.de                      # CIFS for nodes
  username: uXXXXX
  password: ${STORAGEBOX_PASSWORD}
```

### Acceptance Criteria
- [ ] Config supports both WebDAV URL and CIFS host
- [ ] Morpheus CLI uses WebDAV
- [ ] Nodes use CIFS mount

---

## Task 4: Update `morpheus grow` for Embedded NATS
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 2-3 hours

### 4.1 Query NATS monitoring from embedded server

NimsForest exposes NATS monitoring on port 8222. The existing monitoring code should still work.

```go
// pkg/nats/monitor.go - no changes needed
// NATS HTTP monitoring at :8222 still available
```

### 4.2 Update grow to read from shared registry

```go
func runGrow(forestID string) error {
    // Read registry via WebDAV (for CLI)
    reg, _ := registry.Load()
    nodes := reg.Nodes[forestID]
    
    // Query each node's NATS stats
    for _, node := range nodes {
        stats, _ := nats.GetServerStats(node.IP)
        // Display stats...
    }
    
    // Prompt for expansion if needed
    if shouldExpand(stats) {
        // Provision new node - it will auto-join via registry
        provisionNode(forestID)
    }
}
```

### Acceptance Criteria
- [ ] `morpheus grow` works with embedded NATS
- [ ] Stats queried from :8222 on each node
- [ ] New nodes auto-join cluster via registry

---

## Task 5: Firewall Rules
**Status:** ⬜ Not Started  
**Priority:** High  
**Estimated:** 1 hour

### 5.1 Update firewall for embedded NATS ports

| Port | Purpose | Required |
|------|---------|----------|
| 6222 | NATS cluster (between nodes) | Yes |
| 4222 | NATS client (internal use) | Optional |
| 8222 | NATS monitoring (for grow) | Optional |
| 445  | CIFS (StorageBox mount) | Outbound only |

### 5.2 Cloud-init firewall rules

```yaml
- |
  # Allow NATS cluster traffic between forest nodes
  ufw allow 6222/tcp comment 'NATS cluster'
  
  # Optional: allow monitoring queries from Morpheus
  ufw allow 8222/tcp comment 'NATS monitoring'
```

### Acceptance Criteria
- [ ] Nodes can communicate on port 6222
- [ ] StorageBox mount works (outbound 445)
- [ ] Monitoring accessible on 8222

---

## Quick Reference

**File Locations on Each Node:**
- `/etc/morpheus/node-info.json` - Node identity (forest_id, node_id)
- `/mnt/forest/registry.json` - Shared registry (peer discovery)
- `/var/lib/nimsforest/jetstream/` - JetStream data
- `/opt/nimsforest/bin/forest` - NimsForest binary

**What NimsForest Does on Startup:**
1. Reads `/etc/morpheus/node-info.json` for forest_id
2. Reads `/mnt/forest/registry.json` for peer IPs
3. Starts embedded NATS with cluster config
4. Connects to any discovered peer
5. NATS gossip propagates cluster membership

**Ports:**
- 6222 - NATS cluster (required for node-to-node)
- 4222 - NATS client (localhost only by default)
- 8222 - NATS monitoring (for `morpheus grow`)

---

## Completed

- [x] Basic `morpheus plant cloud small/medium/large`
- [x] NimsForest binary download
- [x] NimsForest systemd service
- [x] Configurable download URL
- [x] StorageBox Registry (Task 1 from old plan) - WebDAV client with optimistic locking
- [x] Registry config in config.go
- [x] NATS monitoring code (`pkg/nats/monitor.go`)
- [x] `morpheus grow` command - cluster health monitoring
- [x] `morpheus grow --auto` - non-interactive mode with JSON output

---

## Migration Notes

### What Changed

**Before (Separate NATS):**
```
Morpheus provisions:
  1. NATS binary download
  2. NATS config generation
  3. NATS systemd service
  4. NimsForest config pointing to localhost:4222
  5. NimsForest systemd service (depends on nats.service)
```

**After (Embedded NATS):**
```
Morpheus provisions:
  1. StorageBox CIFS mount
  2. Write /etc/morpheus/node-info.json
  3. Register in /mnt/forest/registry.json
  4. NimsForest binary download
  5. NimsForest systemd service (handles everything)
```

### Simplified Architecture Benefits

1. **One binary** - NimsForest contains everything
2. **No NATS config management** - Embedded server auto-configures
3. **Simpler cloud-init** - Fewer steps, less can go wrong
4. **Automatic peer discovery** - Registry just needs to exist
5. **NATS gossip handles clustering** - No manual route configuration

---

## Future Improvements

### Task 6: True Local Mode (No Docker)
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 2-3 hours

Now simpler with embedded NATS - just run the NimsForest binary directly.

```go
func (p *NativeProvider) CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error) {
    // 1. Download NimsForest binary (includes NATS)
    // 2. Write local node-info.json
    // 3. Create local registry.json
    // 4. Start NimsForest as background process
}
```

### Task 7: Health Checks
**Status:** ⬜ Not Started  
**Priority:** Low  

Add health endpoint that Morpheus can query:
- Is NimsForest running?
- Is embedded NATS healthy?
- Are peers reachable?
