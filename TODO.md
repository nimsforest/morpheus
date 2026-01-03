# Morpheus TODO

## Current Sprint: Forest Automation

### Task 1: Post-Provision Cluster Join
**Status:** 🟡 In Progress  
**Priority:** High  
**Estimated:** 2-3 hours

Machines provisioned by `morpheus plant` should auto-join the NATS cluster.

#### Subtasks

- [ ] **1.1** Add cluster fields to `TemplateData` in `pkg/cloudinit/templates.go`
  ```go
  ClusterNodes []string // IPs of existing nodes
  ClusterName  string   // NATS cluster name  
  IsFirstNode  bool     // True for seed node
  ```

- [ ] **1.2** Update `pkg/forest/provisioner.go` to pass cluster info
  - First node: `IsFirstNode = true`
  - Later nodes: Get existing node IPs from registry

- [ ] **1.3** Add `GetActiveNodeIPs(forestID string) []string` to `pkg/forest/registry.go`

- [ ] **1.4** Generate NATS config in cloud-init template
  - Create `/etc/nimsforest/nats.conf` with cluster routes
  - Seed node has no routes, others connect to seeds

- [ ] **1.5** Update systemd service to pass `--nats-config` flag

- [ ] **1.6** Add tests for cluster config generation

---

### Task 2: `morpheus grow` Command
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 4-5 hours  
**Depends on:** Task 1

Interactive command to check forest health and expand.

#### Subtasks

- [ ] **2.1** Add NATS Go client dependency
  ```bash
  go get github.com/nats-io/nats.go
  ```

- [ ] **2.2** Create `pkg/nats/client.go` - NATS connection wrapper
  ```go
  func NewClient(url string) (*Client, error)
  func (c *Client) GetClusterStats() (*ClusterStats, error)
  ```

- [ ] **2.3** Create `pkg/nats/stats.go` - Stats types
  ```go
  type ClusterStats struct {
      Nodes       []NodeStats
      TotalCPU    float64
      UsedCPU     float64
      TotalMemory float64
      UsedMemory  float64
  }
  ```

- [ ] **2.4** Add `grow` command to `cmd/morpheus/main.go`
  - Parse forest ID from args or select if only one
  - Connect to NATS cluster
  - Query stats
  - Display with progress bars
  - Flag >80% usage
  - Prompt for expansion

- [ ] **2.5** Implement expansion logic
  - On confirm, call existing provisioning code
  - New node should auto-join cluster (Task 1)

- [ ] **2.6** Add tests for grow command

---

### Task 3: `morpheus grow --auto`
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 2-3 hours  
**Depends on:** Task 2

Non-interactive auto-expansion for cron/automation.

#### Subtasks

- [ ] **3.1** Add flags to grow command
  - `--auto` - No prompts, auto-expand if needed
  - `--threshold N` - Custom threshold (default 80)
  - `--output json` - Machine-readable output

- [ ] **3.2** Add growth config to `pkg/config/config.go`
  ```go
  type GrowthConfig struct {
      Enabled         bool `yaml:"enabled"`
      ThresholdCPU    int  `yaml:"threshold_cpu"`
      ThresholdMemory int  `yaml:"threshold_memory"`
      MaxNodes        int  `yaml:"max_nodes"`
      CooldownMinutes int  `yaml:"cooldown_minutes"`
  }
  ```

- [ ] **3.3** Implement cooldown tracking
  - Store last expansion time in registry
  - Prevent expansion if within cooldown

- [ ] **3.4** Add JSON output mode

- [ ] **3.5** Update `config.example.yaml` with growth settings

- [ ] **3.6** Add tests for auto mode

---

## Backlog

### `morpheus shrink`
Tear down idle nodes. NimsForest reports idle nodes, Morpheus removes them.

### `morpheus status`
Enhanced status showing cluster health, not just node list.

### Central Registry (Optional)
For connected (non-airgapped) forests, sync to central registry.

---

## Completed

- [x] Basic provisioning (`morpheus plant cloud small/medium/large`)
- [x] NimsForest auto-install via cloud-init
- [x] Configurable download URL for NimsForest binary
- [x] Systemd service creation

---

## How to Pick Up a Task

1. Check task status and dependencies
2. Read the detailed roadmap: `docs/ROADMAP_FOREST_AUTOMATION.md`
3. Mark task as 🟡 In Progress
4. Complete subtasks in order
5. Run tests: `go test ./...`
6. Update this file with completion status

---

## Quick Reference

**Key Files:**
- `pkg/cloudinit/templates.go` - Cloud-init scripts
- `pkg/forest/provisioner.go` - Provisioning logic
- `pkg/forest/registry.go` - Forest/node tracking
- `cmd/morpheus/main.go` - CLI commands
- `pkg/config/config.go` - Configuration

**Test Commands:**
```bash
go test ./...           # All tests
go test ./pkg/cloudinit # Cloud-init tests
go build ./...          # Build check
```

**NATS Ports:**
- 4222 - Client connections
- 6222 - Cluster routing
- 8222 - Monitoring
