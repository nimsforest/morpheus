# Morpheus TODO

## Architecture

```
┌─────────────────┐
│    Morpheus     │  Stateless CLI (phone/laptop)
└────────┬────────┘
         │ reads/writes
         ▼
┌─────────────────┐
│ Hetzner         │  Registry = JSON file
│ StorageBox      │  (auto-created if missing)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Hetzner VMs     │  Each runs NATS + NimsForest
│ (the forest)    │  Self-register on boot
└─────────────────┘
```

---

## Task 1: StorageBox Registry
**Status:** ⬜ Not Started  
**Priority:** Critical (blocks everything else)  
**Estimated:** 4-5 hours

Morpheus needs remote state storage. Use Hetzner StorageBox (WebDAV).

### 1.1 Create StorageBox client

**File:** `pkg/registry/storagebox.go` (new)

```go
package registry

type StorageBoxRegistry struct {
    URL      string
    Username string
    Password string
}

func NewStorageBoxRegistry(url, user, pass string) *StorageBoxRegistry

// Read registry from StorageBox
func (r *StorageBoxRegistry) Load() (*RegistryData, error)

// Write registry with optimistic locking (ETag)
func (r *StorageBoxRegistry) Save(data *RegistryData) error

// Atomic read-modify-write with retry
func (r *StorageBoxRegistry) Update(fn func(*RegistryData) error) error
```

### 1.2 Define registry data structure

**File:** `pkg/registry/types.go` (new)

```go
type RegistryData struct {
    Version   int                  `json:"version"`
    UpdatedAt time.Time            `json:"updated_at"`
    Forests   map[string]*Forest   `json:"forests"`
    Nodes     map[string][]*Node   `json:"nodes"`
}

type Forest struct {
    ID          string    `json:"id"`
    Provider    string    `json:"provider"`
    Location    string    `json:"location"`
    Size        string    `json:"size"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
    RegistryURL string    `json:"registry_url"`
}

type Node struct {
    ID        string    `json:"id"`
    ForestID  string    `json:"forest_id"`
    IP        string    `json:"ip"`
    Role      string    `json:"role"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 1.3 Auto-create StorageBox via Hetzner Robot API

**File:** `pkg/registry/setup.go` (new)

```go
// Check if StorageBox exists, create if not
func EnsureStorageBox(hetznerCredentials) (*StorageBoxConfig, error) {
    // 1. List existing StorageBoxes via Robot API
    // 2. If none with "morpheus" label, create one
    // 3. Create /morpheus/registry.json with empty registry
    // 4. Return connection details
}
```

**Note:** Hetzner Robot API (not Cloud API) manages StorageBox.

### 1.4 Add registry config

**File:** `pkg/config/config.go`

```go
type RegistryConfig struct {
    Type     string `yaml:"type"`     // "storagebox", "s3", "none"
    URL      string `yaml:"url"`      // WebDAV URL
    Username string `yaml:"username"`
    Password string `yaml:"password"` // Or ${STORAGEBOX_PASSWORD}
}
```

### 1.5 Interactive setup on first `plant`

**File:** `cmd/morpheus/main.go`

```go
func ensureRegistry(cfg *config.Config) error {
    if cfg.Registry.URL != "" {
        return nil // Already configured
    }
    
    fmt.Println("🌲 No registry configured.")
    fmt.Println()
    fmt.Println("Options:")
    fmt.Println("  1. Create new StorageBox (recommended)")
    fmt.Println("  2. Enter existing StorageBox URL")
    fmt.Println("  3. Continue without registry (single-device only)")
    fmt.Println()
    
    choice := prompt("Choice [1]: ")
    // Handle choice...
}
```

### 1.6 Update all commands to use remote registry

- `morpheus plant` - Write forest/node to registry
- `morpheus list` - Read from registry
- `morpheus status` - Read from registry
- `morpheus teardown` - Read then delete from registry
- `morpheus grow` - Read for node IPs

### 1.7 Node self-registration in cloud-init

**File:** `pkg/cloudinit/templates.go`

```yaml
# Register this node in the registry
- |
  curl -X PUT "{{.RegistryURL}}" \
    -u "{{.RegistryUsername}}:{{.RegistryPassword}}" \
    -H "Content-Type: application/json" \
    --data-binary @- << 'EOF'
  ... (read-modify-write logic)
  EOF
```

Or simpler: Morpheus registers the node after SSH is confirmed (current approach), nodes just report health.

### 1.8 Safety: Optimistic locking

```go
func (r *StorageBoxRegistry) Save(data *RegistryData) error {
    jsonData, _ := json.MarshalIndent(data, "", "  ")
    
    req, _ := http.NewRequest("PUT", r.URL, bytes.NewReader(jsonData))
    req.SetBasicAuth(r.Username, r.Password)
    
    if r.lastETag != "" {
        req.Header.Set("If-Match", r.lastETag)
    }
    
    resp, err := http.DefaultClient.Do(req)
    if resp.StatusCode == 412 { // Precondition Failed
        return ErrConcurrentModification // Caller should retry
    }
    return err
}
```

### Acceptance Criteria
- [ ] `morpheus plant` works without local state file
- [ ] `morpheus list` from different device shows same forests
- [ ] Concurrent writes don't corrupt registry
- [ ] StorageBox auto-created if missing

---

## Task 2: NATS Server Installation
**Status:** ⬜ Not Started  
**Priority:** High  
**Estimated:** 3-4 hours  
**Depends on:** Task 1 (needs registry for node IPs)

Each VM runs NATS server. Multi-node forests form NATS cluster.

### 2.1 Add NATS config options

**File:** `pkg/config/config.go`

```go
type IntegrationConfig struct {
    // ... existing ...
    NATSInstall bool   `yaml:"nats_install"`
    NATSVersion string `yaml:"nats_version"` // e.g., "2.10.24"
}
```

### 2.2 Add cluster fields to cloud-init TemplateData

**File:** `pkg/cloudinit/templates.go`

```go
type TemplateData struct {
    // ... existing ...
    
    // NATS
    NATSInstall  bool
    NATSVersion  string
    ClusterName  string   // Forest ID
    ClusterNodes []string // IPs of existing nodes (from registry)
    IsFirstNode  bool
}
```

### 2.3 Download NATS in cloud-init

```yaml
- |
  NATS_VERSION="{{.NATSVersion}}"
  curl -fsSL "https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-amd64.tar.gz" | tar xz
  mv nats-server-*/nats-server /usr/local/bin/
  chmod +x /usr/local/bin/nats-server
```

### 2.4 Generate NATS cluster config

```yaml
- |
  mkdir -p /etc/nats /var/lib/nats/jetstream
  cat > /etc/nats/nats.conf << 'EOF'
  port: 4222
  http_port: 8222
  
  jetstream {
    store_dir: /var/lib/nats/jetstream
    max_mem: 1G
    max_file: 10G
  }
  
  cluster {
    name: {{.ClusterName}}
    port: 6222
    {{if not .IsFirstNode}}
    routes = [
      {{range .ClusterNodes}}
      nats-route://[{{.}}]:6222
      {{end}}
    ]
    {{end}}
  }
  EOF
```

### 2.5 NATS systemd service

```yaml
- |
  cat > /etc/systemd/system/nats.service << 'EOF'
  [Unit]
  Description=NATS Server
  After=network.target

  [Service]
  Type=simple
  ExecStart=/usr/local/bin/nats-server -c /etc/nats/nats.conf
  Restart=always
  RestartSec=5

  [Install]
  WantedBy=multi-user.target
  EOF
  
  systemctl daemon-reload
  systemctl enable nats
  systemctl start nats
```

### 2.6 NimsForest depends on NATS

```yaml
[Unit]
Description=NimsForest
After=nats.service
Requires=nats.service

[Service]
Environment=NATS_URL=nats://localhost:4222
ExecStart=/opt/nimsforest/bin/nimsforest
...
```

### 2.7 Provisioner passes cluster info

**File:** `pkg/forest/provisioner.go`

```go
// Get existing node IPs from registry
existingNodes, _ := registry.GetNodes(forestID)
var clusterIPs []string
for _, n := range existingNodes {
    clusterIPs = append(clusterIPs, n.IP)
}

cloudInitData := cloudinit.TemplateData{
    // ...
    ClusterName:  forestID,
    ClusterNodes: clusterIPs,
    IsFirstNode:  len(clusterIPs) == 0,
}
```

### Acceptance Criteria
- [ ] NATS server running on each node
- [ ] Multi-node forest forms NATS cluster
- [ ] NimsForest connects to local NATS
- [ ] `nats server list` shows all nodes

---

## Task 3: `morpheus grow` Command
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 4-5 hours  
**Depends on:** Task 1, Task 2

Check forest health, expand if needed.

### 3.1 Query NATS monitoring API

**File:** `pkg/nats/monitor.go` (new)

```go
// NATS exposes stats at http://[ip]:8222/varz
func GetServerStats(nodeIP string) (*ServerStats, error) {
    url := fmt.Sprintf("http://[%s]:8222/varz", nodeIP)
    resp, err := http.Get(url)
    // Parse JSON response
}

type ServerStats struct {
    CPU         float64 `json:"cpu"`
    Mem         int64   `json:"mem"`
    Connections int     `json:"connections"`
    InMsgs      int64   `json:"in_msgs"`
    OutMsgs     int64   `json:"out_msgs"`
}
```

### 3.2 Add grow command

**File:** `cmd/morpheus/main.go`

```go
case "grow":
    return runGrow(args[1:])
```

### 3.3 Display format

```
🌲 Forest: forest-1234567890

NATS Cluster: 2 nodes, 45 connections

Resource Usage:
  CPU:    72% ████████████████░░░░░░░░
  Memory: 85% █████████████████████░░░ ⚠️

Nodes:
  NODE          IP                  CPU    MEM    CONNS
  node-1        2a01:4f8::1         65%    80%    23
  node-2        2a01:4f8::2         78%    90%    22     ⚠️

⚠️  Memory above 80%

Add 1 node? [y/N]: 
```

### 3.4 Provision on confirm

```go
if confirm {
    // Uses same provisioning logic as `plant`
    // New node gets existing IPs from registry
    provisionNode(forestID, registry)
}
```

### Acceptance Criteria
- [ ] `morpheus grow` shows cluster stats
- [ ] Flags resources above threshold
- [ ] On confirm, adds node to cluster

---

## Task 4: `morpheus grow --auto`
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 2-3 hours  
**Depends on:** Task 3

Non-interactive for cron/automation.

### 4.1 Add flags

```
--auto           No prompts
--threshold N    Trigger at N% (default: 80)  
--output json    Machine-readable
```

### 4.2 Safety limits

```go
type GrowthConfig struct {
    MaxNodes        int `yaml:"max_nodes"`        // default: 10
    CooldownMinutes int `yaml:"cooldown_minutes"` // default: 15
}
```

### 4.3 Track last expansion

Store in registry:
```json
{
  "forests": {
    "forest-123": {
      "last_expansion": "2025-01-02T10:00:00Z"
    }
  }
}
```

### Acceptance Criteria
- [ ] `morpheus grow --auto` works unattended
- [ ] Respects cooldown
- [ ] JSON output for scripting

---

## Quick Reference

**Hetzner APIs:**
- Cloud API (`api.hetzner.cloud`) - VMs, firewalls, SSH keys
- Robot API (`robot-ws.your-server.de`) - StorageBox, dedicated servers

**NATS Ports:**
- 4222 - Client (NimsForest connects here)
- 6222 - Cluster (nodes connect to each other)
- 8222 - HTTP monitoring (morpheus grow queries this)

**Files:**
- `pkg/registry/` - StorageBox client (new)
- `pkg/cloudinit/templates.go` - VM setup
- `pkg/forest/provisioner.go` - Orchestration
- `cmd/morpheus/main.go` - CLI

---

## Completed

- [x] Basic `morpheus plant cloud small/medium/large`
- [x] NimsForest binary download
- [x] NimsForest systemd service
- [x] Configurable download URL
- [x] StorageBox Registry (Task 1) - WebDAV client with optimistic locking
- [x] Registry config in config.go
- [x] NATS Server Installation (Task 2) - cloud-init templates for NATS
- [x] `morpheus grow` command (Task 3) - cluster health monitoring
- [x] `morpheus grow --auto` (Task 4) - non-interactive mode with JSON output

---

## Future Improvements

### Task 5: True Local Mode (No Docker)
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 3-4 hours

The current "local" provider uses Docker containers to simulate cloud VMs. This should be revisited to support true local mode that runs NATS and NimsForest binaries directly on the local machine without requiring Docker.

#### Goals:
- Download and run NATS binary directly (no Docker)
- Download and run NimsForest binary directly
- Store state in local registry file
- Support `morpheus plant local` without Docker dependency
- Useful for development, testing, and single-machine deployments

#### Implementation Ideas:
```go
// pkg/provider/native/native.go
type NativeProvider struct {
    binDir    string  // ~/.morpheus/bin/
    dataDir   string  // ~/.morpheus/data/
    processes map[string]*os.Process
}

func (p *NativeProvider) CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error) {
    // 1. Download NATS binary if not present
    // 2. Download NimsForest binary if not present  
    // 3. Start NATS as background process
    // 4. Start NimsForest as background process
    // 5. Return "server" representing local processes
}
```

#### Acceptance Criteria:
- [ ] `morpheus plant local small` works without Docker
- [ ] NATS runs as local process, accessible on localhost:4222
- [ ] `morpheus grow` can monitor local NATS instance
- [ ] `morpheus teardown` stops local processes cleanly
