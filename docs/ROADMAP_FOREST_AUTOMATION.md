# Morpheus Forest Automation Roadmap

## Overview

Morpheus provisions "land" (VMs/machines) that NimsForest grows on. This roadmap covers making machines auto-join forests and enabling forest self-expansion.

## Architecture

```
┌──────────────┐     provisions     ┌──────────────┐     runs      ┌──────────────┐
│   MORPHEUS   │ ─────────────────► │     LAND     │ ────────────► │  NIMSFOREST  │
│     (CLI)    │                    │   (VM/VPS)   │               │   (Runtime)  │
└──────────────┘                    └──────────────┘               └──────────────┘
       ▲                                                                  │
       │                         requests more land                       │
       └──────────────────────────────────────────────────────────────────┘
```

## Current State

- [x] `morpheus plant cloud small/medium/large` - provisions VMs on Hetzner
- [x] Cloud-init installs packages, configures firewall, creates directories
- [x] NimsForest binary download from configured URL
- [x] Systemd service created and started
- [ ] NATS cluster configuration (machines don't know about each other)
- [ ] Forest resource monitoring
- [ ] Auto-expansion

---

## Task 1: Post-Provision Forest Join

**Goal:** New machines automatically join the NATS cluster on boot.

### 1.1 Pass cluster info to cloud-init

**File:** `pkg/cloudinit/templates.go`

Add to `TemplateData`:
```go
// Cluster configuration
ClusterNodes []string // IPs of existing nodes to connect to
ClusterName  string   // Name of the NATS cluster
IsFirstNode  bool     // True if this is the first node (becomes seed)
```

**File:** `pkg/forest/provisioner.go`

When provisioning:
- First node: `IsFirstNode = true`, starts as seed
- Subsequent nodes: Get IPs of existing nodes from registry

### 1.2 Generate NATS config in cloud-init

**File:** `pkg/cloudinit/templates.go`

Add to runcmd section:
```yaml
# Generate NATS config
- |
  cat > /etc/nimsforest/nats.conf << EOF
  port: 4222
  cluster {
    name: {{.ClusterName}}
    port: 6222
    {{if .IsFirstNode}}
    # Seed node - others connect to us
    {{else}}
    routes = [
      {{range .ClusterNodes}}
      nats-route://{{.}}:6222
      {{end}}
    ]
    {{end}}
  }
  EOF
```

### 1.3 Update NimsForest systemd service

Pass the config file to nimsforest:
```ini
ExecStart=/opt/nimsforest/bin/nimsforest start --forest-id {{.ForestID}} --nats-config /etc/nimsforest/nats.conf
```

### 1.4 Registry tracks node IPs

**File:** `pkg/forest/registry.go`

Ensure registry stores node IPs so subsequent nodes can connect:
```go
func (r *Registry) GetActiveNodeIPs(forestID string) []string
```

### Acceptance Criteria
- [ ] First node starts as NATS seed
- [ ] Subsequent nodes connect to existing cluster
- [ ] `nats server list` shows all nodes
- [ ] Nodes reconnect automatically after restart

---

## Task 2: `morpheus grow` Command

**Goal:** Interactive command to check forest health and expand if needed.

### 2.1 Add NATS client to Morpheus

**File:** `pkg/nats/client.go` (new)

```go
package nats

type Client struct {
    conn *nats.Conn
}

func NewClient(url string) (*Client, error)
func (c *Client) GetClusterStats() (*ClusterStats, error)
func (c *Client) Close()

type ClusterStats struct {
    Nodes       []NodeStats
    TotalCPU    float64
    TotalMemory float64
    UsedCPU     float64
    UsedMemory  float64
}

type NodeStats struct {
    ID         string
    IP         string
    CPUPercent float64
    MemPercent float64
    Uptime     time.Duration
}
```

### 2.2 Implement `morpheus grow` command

**File:** `cmd/morpheus/main.go`

```go
case "grow":
    return runGrow(args[1:])

func runGrow(args []string) error {
    // 1. Connect to forest NATS cluster
    // 2. Query resource usage
    // 3. Display current state
    // 4. If above threshold, suggest expansion
    // 5. Prompt for confirmation
    // 6. Execute morpheus plant
}
```

### 2.3 Display format

```
🌲 Forest: forest-1234567890

Resource Usage:
  CPU:    72% ████████████████████░░░░░░░░ (3.6 / 5.0 cores)
  Memory: 85% █████████████████████████░░░ (6.8 / 8.0 GB) ⚠️

Nodes (2):
  ID          IP              CPU    MEM    STATUS
  node-1      2a01:4f8::1     65%    80%    active
  node-2      2a01:4f8::2     78%    90%    active ⚠️

⚠️  Memory usage above 80% threshold

Suggestion: Add 1 node to reduce load
Estimated cost: ~€3/month

Proceed? [y/N]: 
```

### 2.4 Connect to forest

Need to know which forest to query. Options:
- `morpheus grow forest-123` - explicit forest ID
- `morpheus grow` - if only one forest, use that
- `morpheus grow` - prompt to select if multiple

### Acceptance Criteria
- [ ] `morpheus grow` connects to NATS cluster
- [ ] Displays CPU/memory usage per node
- [ ] Flags resources above 80%
- [ ] Suggests expansion when needed
- [ ] On confirm, provisions new node that joins cluster

---

## Task 3: `morpheus grow --auto`

**Goal:** Non-interactive auto-expansion for automated/scheduled use.

### 3.1 Add --auto flag

**File:** `cmd/morpheus/main.go`

```go
func runGrow(args []string) error {
    auto := hasFlag(args, "--auto")
    threshold := getFlagValue(args, "--threshold", "80")
    
    if auto {
        return runGrowAuto(forestID, threshold)
    }
    return runGrowInteractive(forestID)
}

func runGrowAuto(forestID string, threshold int) error {
    stats := getClusterStats(forestID)
    
    if stats.UsedCPU > threshold || stats.UsedMemory > threshold {
        fmt.Println("⚠️  Threshold exceeded, provisioning new node...")
        return provisionNode(forestID)
    }
    
    fmt.Println("✅ Resource usage within limits")
    return nil
}
```

### 3.2 Logging for automation

When running with `--auto`, output should be:
- Machine-parseable (JSON option: `--output json`)
- Suitable for cron/systemd timer logs

```bash
# Cron example
*/15 * * * * morpheus grow forest-123 --auto --threshold 80
```

### 3.3 Expansion limits

Add safety limits:
```go
type GrowConfig struct {
    MaxNodes      int   // Max nodes to provision (default: 10)
    CooldownMins  int   // Minutes between expansions (default: 15)
    MaxExpandPer  int   // Max nodes to add per run (default: 1)
}
```

### Acceptance Criteria
- [ ] `morpheus grow --auto` runs without prompts
- [ ] Respects threshold flag
- [ ] Provisions node when threshold exceeded
- [ ] Does nothing when within limits
- [ ] Has cooldown to prevent rapid expansion
- [ ] Outputs JSON with `--output json`

---

## Task 4: Forest Self-Monitoring (NimsForest side)

**Note:** This is implemented in NimsForest, not Morpheus.

NimsForest should:
1. Monitor its own resource usage
2. Call Morpheus when expansion needed
3. Track idle machines for potential teardown

This could be via:
- NimsForest calling `morpheus grow --auto` periodically
- NimsForest calling a Morpheus HTTP API
- NimsForest publishing to NATS, Morpheus subscribes

---

## Implementation Order

```
Phase 1: Cluster Join (Task 1)
├── 1.1 Add cluster fields to TemplateData
├── 1.2 Generate NATS config in cloud-init
├── 1.3 Update systemd service
└── 1.4 Registry tracks node IPs

Phase 2: Interactive Grow (Task 2)
├── 2.1 Add NATS client package
├── 2.2 Implement grow command
├── 2.3 Display formatting
└── 2.4 Forest selection

Phase 3: Auto Grow (Task 3)
├── 3.1 Add --auto flag
├── 3.2 JSON output mode
└── 3.3 Safety limits

Phase 4: Integration (Task 4 - NimsForest)
└── Forest self-monitoring
```

---

## Config Changes

**File:** `config.example.yaml`

```yaml
integration:
  nimsforest_install: true
  nimsforest_download_url: "https://nimsforest.io/bin/nimsforest"

# NEW: Growth/expansion settings
growth:
  enabled: true
  threshold_cpu: 80      # Percentage
  threshold_memory: 80   # Percentage
  max_nodes: 10          # Maximum nodes per forest
  cooldown_minutes: 15   # Min time between expansions
```

---

## Dependencies

- `github.com/nats-io/nats.go` - NATS client for Go

---

## Files to Modify/Create

### Task 1
- [ ] `pkg/cloudinit/templates.go` - Add cluster config to templates
- [ ] `pkg/forest/provisioner.go` - Pass cluster info during provisioning
- [ ] `pkg/forest/registry.go` - Add GetActiveNodeIPs method

### Task 2
- [ ] `pkg/nats/client.go` (new) - NATS client wrapper
- [ ] `pkg/nats/stats.go` (new) - Cluster stats types
- [ ] `cmd/morpheus/main.go` - Add grow command

### Task 3
- [ ] `cmd/morpheus/main.go` - Add --auto flag handling
- [ ] `pkg/config/config.go` - Add growth config section

---

## Testing

### Task 1 Tests
- Unit: Cloud-init generates valid NATS config
- Integration: Two nodes can form cluster

### Task 2 Tests
- Unit: Stats display formatting
- Unit: Threshold detection
- Integration: grow command connects to real cluster

### Task 3 Tests
- Unit: Auto mode respects threshold
- Unit: Cooldown prevents rapid expansion
- Unit: JSON output is valid

---

## Notes

- All machines use IPv6 (Hetzner default)
- NATS cluster port: 6222
- NATS client port: 4222
- Forest ID format: `forest-{timestamp}`
