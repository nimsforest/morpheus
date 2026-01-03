# Morpheus Forest Automation Roadmap

## Overview

Morpheus provisions "land" (VMs) that run:
1. **NATS Server** - Message broker with built-in clustering
2. **NimsForest** - Event-driven business logic that connects to NATS

NimsForest doesn't handle discovery or clustering - NATS does that natively.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         MORPHEUS (CLI)                           │
│                    Runs on: Termux / Laptop                      │
│                                                                  │
│   morpheus plant cloud small  → Creates VM + installs software   │
│   morpheus grow               → Checks health, adds nodes        │
│   morpheus teardown           → Removes VMs                      │
└──────────────────────────────────────────────────────────────────┘
                                   │
                                   │ provisions
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                        VM (Hetzner VPS)                          │
│                                                                  │
│  ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│  │    NATS Server      │    │         NimsForest              │  │
│  │                     │    │                                 │  │
│  │  • Port 4222 client │◄───│  Connects to localhost:4222     │  │
│  │  • Port 6222 cluster│    │  Runs Trees, Nims, etc.         │  │
│  │  • Routes to peers  │    │                                 │  │
│  └─────────────────────┘    └─────────────────────────────────┘  │
│                                                                  │
│  systemd: nats.service       systemd: nimsforest.service         │
└──────────────────────────────────────────────────────────────────┘
                                   │
                                   │ NATS cluster routes (port 6222)
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                        VM 2, VM 3, ...                           │
│                    (Same setup, clustered)                       │
└──────────────────────────────────────────────────────────────────┘
```

## Current State

- [x] `morpheus plant cloud small/medium/large` - provisions VMs
- [x] Cloud-init installs packages, configures firewall
- [x] NimsForest binary download from configured URL
- [x] NimsForest systemd service
- [ ] **NATS server installation** ← Missing
- [ ] **NATS cluster configuration** ← Missing
- [ ] `morpheus grow` command

---

## Task 1: NATS Server Installation

**Goal:** Each provisioned VM runs NATS server with clustering enabled.

### 1.1 Download NATS server in cloud-init

**File:** `pkg/cloudinit/templates.go`

Add to runcmd:
```yaml
# Download and install NATS server
- |
  NATS_VERSION="2.10.24"
  curl -fsSL "https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-amd64.tar.gz" | tar xz
  mv nats-server-v${NATS_VERSION}-linux-amd64/nats-server /usr/local/bin/
  chmod +x /usr/local/bin/nats-server
```

### 1.2 Create NATS systemd service

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
```

### 1.3 Generate NATS config with cluster routes

**File:** `pkg/cloudinit/templates.go`

Add to `TemplateData`:
```go
// NATS cluster configuration
ClusterNodes  []string // IPv6 addresses of other nodes
ClusterName   string   // Cluster name (forest ID)
IsFirstNode   bool     // First node is the seed
```

Generate `/etc/nats/nats.conf`:
```yaml
- |
  mkdir -p /etc/nats
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

### 1.4 Update provisioner to track and pass node IPs

**File:** `pkg/forest/provisioner.go`

```go
// When provisioning node N:
// 1. Get IPs of nodes 1..N-1 from registry
// 2. Pass them as ClusterNodes
// 3. First node has IsFirstNode=true
```

### 1.5 Update NimsForest service to wait for NATS

```yaml
- |
  cat > /etc/systemd/system/nimsforest.service << 'EOF'
  [Unit]
  Description=NimsForest
  After=nats.service
  Requires=nats.service

  [Service]
  Type=simple
  User=ubuntu
  Environment=NATS_URL=nats://localhost:4222
  Environment=FOREST_ID={{.ForestID}}
  ExecStart=/opt/nimsforest/bin/nimsforest
  Restart=always
  RestartSec=5

  [Install]
  WantedBy=multi-user.target
  EOF
```

### Acceptance Criteria
- [ ] NATS server installed on each VM
- [ ] NATS starts before NimsForest
- [ ] Multi-node forests form a NATS cluster
- [ ] `nats server list` shows all nodes
- [ ] NimsForest connects to local NATS successfully

---

## Task 2: `morpheus grow` Command

**Goal:** Check forest health and expand capacity.

### 2.1 Add `grow` command structure

**File:** `cmd/morpheus/main.go`

```go
case "grow":
    return runGrow(args[1:])
```

### 2.2 Connect to NATS cluster

To check cluster health, Morpheus needs to connect to NATS monitoring endpoint.

```go
// Connect via HTTP to NATS monitoring port (8222)
func getClusterStats(nodeIP string) (*ClusterStats, error) {
    resp, err := http.Get(fmt.Sprintf("http://[%s]:8222/varz", nodeIP))
    // Parse JSON response for CPU, memory, connections
}
```

### 2.3 Display format

```
🌲 Forest: forest-1234567890

NATS Cluster:
  Nodes: 2/2 healthy
  Connections: 45 active
  Messages: 1.2M in, 3.4M out

Resource Usage:
  CPU:    72% ████████████████████░░░░░░░░
  Memory: 85% █████████████████████████░░░ ⚠️

Nodes:
  ID              IP                  CPU    MEM    CONNS
  node-1          2a01:4f8::1         65%    80%    23
  node-2          2a01:4f8::2         78%    90%    22     ⚠️

⚠️  Memory above 80% on node-2

Suggestion: Add 1 node
Proceed? [y/N]: 
```

### 2.4 Expansion flow

1. User confirms
2. Morpheus provisions new node
3. New node gets existing node IPs in `ClusterNodes`
4. NATS automatically forms cluster
5. NimsForest starts and connects

### Acceptance Criteria
- [ ] `morpheus grow` shows cluster health
- [ ] Flags nodes above threshold
- [ ] On confirm, provisions new node
- [ ] New node joins cluster automatically

---

## Task 3: `morpheus grow --auto`

**Goal:** Non-interactive expansion for automation.

### 3.1 Add flags

```go
--auto           No prompts
--threshold N    Expansion threshold (default: 80)
--output json    Machine-readable output
```

### 3.2 Auto-expansion logic

```go
func runGrowAuto(forestID string, threshold int) error {
    stats := getClusterStats(forestID)
    
    if stats.MaxMemoryPercent > threshold || stats.MaxCPUPercent > threshold {
        log.Println("Threshold exceeded, provisioning...")
        return provisionNode(forestID)
    }
    
    log.Println("Within limits, no action needed")
    return nil
}
```

### 3.3 Safety limits

```go
type GrowthConfig struct {
    MaxNodes        int  // Don't exceed this many nodes
    CooldownMinutes int  // Wait between expansions
}
```

### Acceptance Criteria
- [ ] `morpheus grow --auto` works without prompts
- [ ] Respects threshold
- [ ] Has cooldown between expansions
- [ ] JSON output available

---

## Implementation Order

```
Phase 1: NATS Installation (Task 1)
├── 1.1 Download NATS in cloud-init
├── 1.2 Create NATS systemd service  
├── 1.3 Generate cluster config
├── 1.4 Pass node IPs from provisioner
└── 1.5 NimsForest depends on NATS

Phase 2: Grow Command (Task 2)
├── 2.1 Add grow command
├── 2.2 Query NATS monitoring API
├── 2.3 Display formatting
└── 2.4 Expansion flow

Phase 3: Auto-Grow (Task 3)
├── 3.1 Add flags
├── 3.2 Auto logic
└── 3.3 Safety limits
```

---

## Config Changes

**File:** `config.example.yaml`

```yaml
integration:
  nimsforest_install: true
  nimsforest_download_url: "https://nimsforest.io/bin/nimsforest"
  
  # NATS configuration
  nats_install: true
  nats_version: "2.10.24"

# Growth settings  
growth:
  enabled: true
  threshold_cpu: 80
  threshold_memory: 80
  max_nodes: 10
  cooldown_minutes: 15
```

---

## Files to Modify

### Task 1 (NATS Installation)
- [ ] `pkg/cloudinit/templates.go` - Add NATS download, config generation
- [ ] `pkg/config/config.go` - Add NATS config options
- [ ] `pkg/forest/provisioner.go` - Pass cluster node IPs
- [ ] `pkg/forest/registry.go` - Track node IPs for clustering

### Task 2 (Grow Command)
- [ ] `cmd/morpheus/main.go` - Add grow command
- [ ] `pkg/nats/monitor.go` (new) - Query NATS monitoring API

### Task 3 (Auto-Grow)
- [ ] `cmd/morpheus/main.go` - Add flags
- [ ] `pkg/config/config.go` - Add growth config

---

## Key Insight: NATS Handles Clustering

NimsForest just does:
```go
nc, err := nats.Connect(os.Getenv("NATS_URL"))
```

All clustering is handled by NATS server. NimsForest instances on different VMs automatically share messages through the NATS cluster.

**Morpheus just needs to:**
1. Install NATS server
2. Configure cluster routes between nodes
3. Install NimsForest
4. Point NimsForest at local NATS

**NimsForest doesn't need to know about clustering at all.**
