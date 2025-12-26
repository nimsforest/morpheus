# Morpheus Architecture

This document describes the architecture and design decisions of Morpheus.

## Overview

Morpheus is a cloud infrastructure provisioning tool that automates the creation and management of NATS-based distributed systems. It follows a modular, provider-agnostic design.

## Core Components

### 1. Provider Layer (`pkg/provider/`)

**Purpose**: Abstract cloud provider APIs behind a common interface.

**Interface**:
```go
type Provider interface {
    CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error)
    GetServer(ctx context.Context, serverID string) (*Server, error)
    DeleteServer(ctx context.Context, serverID string) error
    WaitForServer(ctx context.Context, serverID string, state ServerState) error
    ListServers(ctx context.Context, filters map[string]string) ([]*Server, error)
}
```

**Implementations**:
- `hetzner/`: Hetzner Cloud provider using hcloud-go/v2

**Design Decisions**:
- Generic interface allows easy addition of new providers
- Context-based cancellation for long-running operations
- Typed server states for consistent status handling

### 2. Configuration Layer (`pkg/config/`)

**Purpose**: Manage application configuration and secrets.

**Features**:
- YAML-based configuration files
- Environment variable overrides
- Validation on load
- Separation of infrastructure config and secrets

**Configuration Structure**:
```yaml
infrastructure:
  provider: string
  defaults:
    server_type: string
    image: string
    ssh_key: string
  locations: []string

secrets:
  hetzner_api_token: string
```

### 3. Cloud-Init Layer (`pkg/cloudinit/`)

**Purpose**: Generate cloud-init scripts for server bootstrapping.

**Node Roles**:
- **Edge**: NATS server with JetStream and clustering
- **Compute**: Docker-based compute workers
- **Storage**: NFS-based distributed storage

**Template Features**:
- Go text/template based
- Role-specific customization
- Automatic package installation
- Service configuration
- Firewall rules

### 4. Forest Registry (`pkg/forest/registry.go`)

**Purpose**: Track forests and their nodes.

**Data Model**:

```go
type Forest struct {
    ID        string
    Size      string
    Location  string
    Provider  string
    Status    string
    CreatedAt time.Time
}

type Node struct {
    ID        string
    ForestID  string
    Role      string
    IP        string
    Location  string
    Status    string
    Metadata  map[string]string
    CreatedAt time.Time
}
```

**Storage**: JSON file at `~/.morpheus/registry.json`

**Operations**:
- RegisterForest()
- RegisterNode()
- GetForest()
- GetNodes()
- UpdateStatus()
- DeleteForest()

### 5. Provisioning Engine (`pkg/forest/provisioner.go`)

**Purpose**: Orchestrate the provisioning workflow.

**Workflow**:
1. Create forest entry in registry
2. Calculate node count based on size
3. For each node:
   - Generate cloud-init script
   - Create server via provider
   - Wait for server to be running
   - Wait for cloud-init to complete
   - Register node in registry
4. Update forest status to "active"

**Error Handling**:
- Automatic rollback on failure
- Delete partially provisioned servers
- Clean up registry entries
- Return detailed error messages

### 6. CLI Interface (`cmd/morpheus/`)

**Purpose**: User-facing command-line interface.

**Commands**:
- `plant`: Create new forest
- `teardown`: Delete forest and resources
- `list`: Show all forests
- `status`: Show forest details

**Features**:
- Clear error messages
- Progress indicators
- Colored output (emojis)
- Config file auto-discovery

## Data Flow

### Provisioning Flow

```
User Command
    ↓
CLI Parser
    ↓
Load Config
    ↓
Initialize Provider
    ↓
Initialize Registry
    ↓
Create Provisioner
    ↓
Generate Cloud-Init
    ↓
Provider.CreateServer() ──→ Hetzner API
    ↓
Provider.WaitForServer()
    ↓
Registry.RegisterNode()
    ↓
Update Status
    ↓
Return Success
```

### Teardown Flow

```
User Command
    ↓
CLI Parser
    ↓
Load Registry
    ↓
Get Forest Nodes
    ↓
For Each Node:
    ├─→ Provider.DeleteServer() ──→ Hetzner API
    └─→ Wait for deletion
    ↓
Registry.DeleteForest()
    ↓
Return Success
```

## Design Patterns

### 1. Strategy Pattern

The Provider interface uses the Strategy pattern to allow different cloud provider implementations.

### 2. Registry Pattern

The Forest Registry maintains a centralized record of all deployments.

### 3. Template Pattern

Cloud-init generation uses Go's text/template package for flexible configuration.

### 4. Factory Pattern

Provider initialization uses a factory-like pattern based on configuration.

## Security Considerations

### 1. Secrets Management

- API tokens stored in config or environment variables
- Config files should be excluded from version control
- No secrets in logs or output

### 2. SSH Access

- Uses pre-uploaded SSH keys
- No password authentication
- Root access for initial bootstrap

### 3. Network Security

- Cloud-init configures UFW firewall
- Only necessary ports opened
- Consider using private networks (future)

## Scalability

### Current Limitations

- Single-machine orchestration
- JSON file-based registry (not distributed)
- Sequential node provisioning
- No built-in monitoring

### Future Improvements

- Distributed registry (etcd/consul)
- Parallel node provisioning
- Built-in health checks
- Auto-scaling
- Load balancing

## Testing Strategy

### Unit Tests

- Provider implementations
- Configuration parsing
- Cloud-init generation
- Registry operations

### Integration Tests

- Full provisioning workflow
- Error handling and rollback
- Provider API interactions

### Manual Testing

- Real cloud provider testing
- Cloud-init script validation
- Network connectivity tests

## Extension Points

### Adding a New Provider

1. Create `pkg/provider/<provider>/`
2. Implement Provider interface
3. Add provider-specific config
4. Update CLI to support provider
5. Add documentation and examples

### Adding a New Node Role

1. Add role constant in `pkg/cloudinit/templates.go`
2. Create cloud-init template
3. Update provisioner to support role
4. Add CLI option for role selection

### Adding New Commands

1. Add command handler in `cmd/morpheus/main.go`
2. Implement command logic
3. Update help text
4. Add documentation

## Dependencies

### Core Dependencies

- `github.com/hetznercloud/hcloud-go/v2`: Official Hetzner Cloud client
- `gopkg.in/yaml.v3`: YAML parsing

### Transitive Dependencies

- Prometheus client (via hcloud-go)
- Protocol buffers (via hcloud-go)

## Performance Considerations

### Provisioning Time

- Server creation: 1-2 minutes
- Cloud-init execution: 2-5 minutes
- Total per node: 3-7 minutes

### Optimization Opportunities

- Parallel server creation
- Cached server images with pre-installed software
- Connection pooling
- Incremental status updates

## Monitoring and Observability

### Current State

- Console output with progress indicators
- Registry tracks all resources
- Cloud-init logs on each server

### Future Additions

- Structured logging
- Metrics export (Prometheus)
- Distributed tracing
- Health check endpoints

## Error Handling

### Failure Scenarios

1. **API Errors**: Network issues, rate limits, authentication
2. **Timeout Errors**: Server provisioning, cloud-init
3. **Configuration Errors**: Invalid config, missing SSH keys
4. **Resource Errors**: Quota limits, out of capacity

### Recovery Strategies

- Automatic rollback on provisioning failure
- Cleanup of orphaned resources
- Detailed error messages with troubleshooting hints
- State preservation in registry

## Future Architecture

### Multi-Cloud Support

```
Provider Interface
    ├─→ Hetzner
    ├─→ AWS
    ├─→ GCP
    ├─→ Azure
    └─→ OVH
```

### Microservices Architecture

- API Server
- Provisioning Service
- Monitoring Service
- Registry Service
- Web UI

### Event-Driven Architecture

- Event bus for state changes
- Webhooks for integrations
- Async job processing

## Conclusion

Morpheus follows a modular, extensible architecture that makes it easy to add new providers, node types, and features while maintaining clean separation of concerns.
