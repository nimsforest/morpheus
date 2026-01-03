# Morpheus TODO

## Current Sprint: Forest Automation

### Understanding

```
VM runs:
├── NATS Server (message broker, handles clustering)
│   └── Port 4222 (clients), 6222 (cluster), 8222 (monitoring)
│
└── NimsForest (business logic)
    └── Connects to localhost:4222
```

**NATS handles all clustering/discovery.** NimsForest just connects to local NATS.

---

### Task 1: NATS Server Installation
**Status:** ⬜ Not Started  
**Priority:** High  
**Estimated:** 3-4 hours

Each VM needs NATS server running before NimsForest can start.

#### Subtasks

- [ ] **1.1** Add NATS config to `pkg/config/config.go`
  ```go
  type IntegrationConfig struct {
      // ... existing fields ...
      NATSInstall  bool   `yaml:"nats_install"`
      NATSVersion  string `yaml:"nats_version"`  // e.g., "2.10.24"
  }
  ```

- [ ] **1.2** Add cluster fields to `TemplateData` in `pkg/cloudinit/templates.go`
  ```go
  // NATS cluster configuration
  NATSInstall   bool
  NATSVersion   string
  ClusterName   string   // Forest ID
  ClusterNodes  []string // IPv6 addresses of existing nodes
  IsFirstNode   bool     // First node has no routes
  ```

- [ ] **1.3** Add NATS download to cloud-init template
  ```yaml
  # Download NATS server
  - |
    NATS_VERSION="{{.NATSVersion}}"
    curl -fsSL "https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-amd64.tar.gz" | tar xz
    mv nats-server-*/nats-server /usr/local/bin/
    chmod +x /usr/local/bin/nats-server
  ```

- [ ] **1.4** Generate NATS config with cluster routes
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

- [ ] **1.5** Create NATS systemd service
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

- [ ] **1.6** Update NimsForest service to depend on NATS
  ```yaml
  [Unit]
  Description=NimsForest
  After=nats.service
  Requires=nats.service

  [Service]
  Environment=NATS_URL=nats://localhost:4222
  # ... rest of service ...
  ```

- [ ] **1.7** Update `pkg/forest/provisioner.go` to pass cluster info
  - First node: `IsFirstNode = true`, empty `ClusterNodes`
  - Subsequent nodes: Get existing node IPs from registry

- [ ] **1.8** Add `GetActiveNodeIPs(forestID) []string` to registry

- [ ] **1.9** Update `config.example.yaml` with NATS settings

- [ ] **1.10** Add tests for NATS config generation

---

### Task 2: `morpheus grow` Command
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 4-5 hours  
**Depends on:** Task 1

Interactive command to check forest health and expand.

#### Subtasks

- [ ] **2.1** Create `pkg/nats/monitor.go` - Query NATS monitoring API
  ```go
  // NATS exposes stats at http://[ip]:8222/varz
  func GetServerStats(nodeIP string) (*ServerStats, error)
  
  type ServerStats struct {
      CPU        float64
      Memory     int64
      Connections int
      InMsgs     int64
      OutMsgs    int64
  }
  ```

- [ ] **2.2** Add `grow` command to `cmd/morpheus/main.go`
  ```go
  case "grow":
      return runGrow(args[1:])
  ```

- [ ] **2.3** Implement forest selection
  - If one forest: use it
  - If multiple: prompt or require `morpheus grow <forest-id>`

- [ ] **2.4** Query all nodes in forest
  ```go
  nodes := registry.GetNodes(forestID)
  for _, node := range nodes {
      stats := nats.GetServerStats(node.IP)
      // aggregate stats
  }
  ```

- [ ] **2.5** Display with progress bars and warnings
  ```
  🌲 Forest: forest-1234567890

  Resource Usage:
    CPU:    72% ████████████████████░░░░░░░░
    Memory: 85% █████████████████████████░░░ ⚠️

  Nodes (2):
    ID        IP              CPU    MEM    STATUS
    node-1    2a01:4f8::1     65%    80%    healthy
    node-2    2a01:4f8::2     78%    90%    warning ⚠️

  ⚠️  Memory above 80% threshold

  Add 1 node? [y/N]:
  ```

- [ ] **2.6** On confirm, provision new node with existing IPs as ClusterNodes

- [ ] **2.7** Add tests

---

### Task 3: `morpheus grow --auto`
**Status:** ⬜ Not Started  
**Priority:** Medium  
**Estimated:** 2-3 hours  
**Depends on:** Task 2

Non-interactive mode for cron/automation.

#### Subtasks

- [ ] **3.1** Add flags to grow command
  ```
  --auto           Run without prompts
  --threshold N    Trigger at N% (default: 80)
  --output json    Machine-readable output
  ```

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

- [ ] **3.3** Track last expansion time in registry
  ```go
  type Forest struct {
      // ... existing ...
      LastExpansion time.Time `json:"last_expansion,omitempty"`
  }
  ```

- [ ] **3.4** Implement auto logic
  ```go
  if time.Since(forest.LastExpansion) < cooldown {
      return "cooldown active"
  }
  if maxPercent > threshold {
      provisionNode(forestID)
  }
  ```

- [ ] **3.5** JSON output mode for scripting

- [ ] **3.6** Update config.example.yaml

- [ ] **3.7** Add tests

---

## Quick Reference

**NATS Ports:**
- 4222 - Client connections (NimsForest connects here)
- 6222 - Cluster routes (nodes connect to each other)
- 8222 - HTTP monitoring API (morpheus queries this)

**Key Files:**
- `pkg/cloudinit/templates.go` - VM setup scripts
- `pkg/forest/provisioner.go` - Orchestrates provisioning
- `pkg/forest/registry.go` - Tracks forests/nodes
- `cmd/morpheus/main.go` - CLI commands

**Test:**
```bash
go test ./...
go build ./...
```

---

## Completed

- [x] Basic provisioning (`morpheus plant cloud small/medium/large`)
- [x] NimsForest binary auto-install
- [x] Configurable download URL
- [x] NimsForest systemd service

---

## How to Pick Up a Task

1. Read task description and subtasks
2. Check dependencies (e.g., Task 2 needs Task 1)
3. Work through subtasks in order
4. Run `go test ./...` after changes
5. Update this file when done
