# Architecture Update: Separation of Concerns

## Overview

Morpheus v1.1.0 introduces a clear separation of responsibilities between infrastructure provisioning (Morpheus) and application orchestration (NimsForest).

## Key Changes

### Before (v1.0.0) ❌

Morpheus was doing too much:
- Provisioning servers ✅
- Installing NATS ❌ (Application concern)
- Configuring NATS clustering ❌ (Application concern)
- Managing services ❌ (Application concern)

### After (v1.1.0) ✅

Clear separation:

**Morpheus** (Infrastructure Layer):
- ✅ Provision cloud servers
- ✅ Configure OS and networking
- ✅ Set up firewalls with required ports
- ✅ Prepare directories and storage
- ✅ Signal readiness to NimsForest

**NimsForest** (Application Layer):
- ✅ Install NATS binaries
- ✅ Configure NATS clustering
- ✅ Manage application services
- ✅ Handle scaling and monitoring

## Implementation Details

### Cloud-Init Templates

Refactored to remove application-level concerns:

**Removed:**
- NATS server installation
- NATS configuration files
- Service startup scripts
- Clustering logic

**Kept:**
- OS package installation (curl, docker, jq)
- Firewall configuration (UFW with NATS ports)
- Directory creation (`/opt/nimsforest`, `/var/lib/nimsforest`)
- Metadata file (`/etc/morpheus/node-info.json`)
- NimsForest callback integration

### Integration Points

#### 1. Node Metadata File

Morpheus writes infrastructure details:

```json
{
  "forest_id": "forest-123",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z",
  "registry_url": "https://registry.example.com",
  "callback_url": "https://nimsforest.example.com"
}
```

#### 2. Callback API

When infrastructure is ready, Morpheus calls:

```http
POST /api/v1/bootstrap
Content-Type: application/json

{
  "forest_id": "forest-123",
  "node_id": "server-456",
  "node_ip": "95.217.0.1",
  "role": "edge"
}
```

#### 3. Firewall Configuration

Morpheus opens ports for NimsForest:

```bash
ufw allow 22/tcp    # SSH
ufw allow 4222/tcp  # NATS client
ufw allow 6222/tcp  # NATS cluster
ufw allow 8222/tcp  # NATS monitoring
ufw allow 7777/tcp  # NATS leafnode
```

### Configuration Changes

New `integration` section in config:

```yaml
infrastructure:
  provider: hetzner
  # ... existing config ...

integration:
  nimsforest_url: "https://nimsforest.example.com"  # NEW
  registry_url: "https://registry.example.com"      # NEW

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"
```

### Code Changes

#### `pkg/cloudinit/templates.go`

```go
// OLD - NATS installation in template ❌
runcmd:
  - curl -L https://github.com/nats-io/nats-server/...
  - tar -xzf /tmp/nats-server.tar.gz
  - systemctl start nats-server

// NEW - Infrastructure only ✅
runcmd:
  - ufw allow 4222/tcp comment 'NATS client port'
  - mkdir -p /opt/nimsforest
  - curl -X POST {{.CallbackURL}}/api/v1/bootstrap
```

#### `pkg/config/config.go`

```go
// NEW: Integration configuration
type IntegrationConfig struct {
    NimsForestURL string `yaml:"nimsforest_url"`
    RegistryURL   string `yaml:"registry_url"`
}

type Config struct {
    Infrastructure InfrastructureConfig
    Integration    IntegrationConfig  // NEW
    Secrets        SecretsConfig
}
```

#### `pkg/cloudinit/templates.go`

```go
// OLD - Had NATSServers for clustering ❌
type TemplateData struct {
    NodeRole     NodeRole
    ForestID     string
    NATSServers  []string  // REMOVED
    RegistryURL  string
}

// NEW - Callback-based integration ✅
type TemplateData struct {
    NodeRole    NodeRole
    ForestID    string
    RegistryURL string
    CallbackURL string  // NEW
    SSHKeys     []string
}
```

## Deployment Flow

### Phase 1: Infrastructure (Morpheus)

```bash
$ morpheus plant cloud forest

🌲 Planting Nims Forest...
Forest ID: forest-1735234567
Size: forest
Location: fsn1
Provider: hetzner

Provisioning 3 node(s)...
Server 12345678 created, waiting for it to be ready...
✓ Infrastructure ready (IP: 95.217.0.1)
✓ Callback sent to NimsForest
✓ Node forest-1735234567-node-1 infrastructure provisioned

Status: infrastructure_ready (waiting for NimsForest)
```

### Phase 2: Application (NimsForest)

NimsForest receives callback and bootstraps:

```bash
$ nimsforest-agent bootstrap

Received bootstrap request for forest-1735234567
Reading node metadata from /etc/morpheus/node-info.json
Installing NATS v2.10.7...
Configuring cluster routes...
Starting NATS server...
✓ NATS server active
Status: active
```

## Benefits

### 1. Single Responsibility Principle
- Each tool does one thing well
- Easier to understand and maintain
- Clear ownership

### 2. Flexibility
- Use Morpheus without NimsForest
- Use NimsForest with manual infrastructure
- Easy to swap components

### 3. Testability
- Test infrastructure independently
- Mock application layer easily
- Clear integration points

### 4. Scalability
- Infrastructure can scale independently
- Application can evolve separately
- Different teams can own layers

## Migration Guide

### For Existing Deployments

**Option 1: Fresh Deploy (Recommended)**
```bash
# Teardown old forest
morpheus teardown forest-old

# Deploy with new version
morpheus plant cloud forest
```

**Option 2: In-Place Update**
```bash
# Keep infrastructure
# Manually bootstrap NimsForest on existing nodes
ssh root@node1
nimsforest bootstrap --forest-id forest-old
```

### For New Deployments

1. Deploy Morpheus v1.1.0
2. Configure `integration.nimsforest_url` 
3. Ensure NimsForest is ready to accept callbacks
4. Run `morpheus plant cloud forest`
5. NimsForest automatically bootstraps

## Testing

### Unit Tests Updated

```bash
$ go test ./pkg/cloudinit -v

✓ TestGenerateEdgeNode - Verifies no NATS installation
✓ TestGenerateComputeNode - Checks callback integration
✓ TestGenerateStorageNode - Validates firewall config
✓ TestGenerateWithoutCallbacks - Handles missing URLs
```

### Integration Test Plan

```bash
# 1. Provision infrastructure
morpheus plant cloud wood

# 2. Verify infrastructure-only
ssh root@node
ls /opt/nimsforest  # Should exist
which nats-server   # Should NOT exist

# 3. Verify callback
cat /var/log/cloud-init-output.log | grep callback

# 4. Trigger NimsForest
# (Manual or via callback)

# 5. Verify application
which nats-server   # Should NOW exist
systemctl status nats-server
```

## Documentation Updates

### New Documents
- ✅ `docs/SEPARATION_OF_CONCERNS.md` - Detailed explanation
- ✅ `ARCHITECTURE_UPDATE.md` - This document

### Updated Documents  
- ✅ `config.example.yaml` - Added integration section
- ✅ `README.md` - Updated responsibilities
- ✅ Cloud-init templates - Removed NATS installation

### Test Updates
- ✅ `pkg/cloudinit/templates_test.go` - Verify no NATS
- ✅ `pkg/config/config_test.go` - Test integration config

## Breaking Changes

### Configuration

**Required Changes:**
```yaml
# ADD THIS to config.yaml
integration:
  nimsforest_url: "https://your-nimsforest-url"  # Required if using callbacks
  registry_url: ""  # Optional
```

### Cloud-Init Output

**Before:** Servers came up with NATS running
**After:** Servers come up infrastructure-ready, waiting for NimsForest

### Node Status

**New Status Values:**
- `infrastructure_ready` - Morpheus finished, waiting for NimsForest
- `active` - NimsForest bootstrap complete (set by NimsForest)

## Rollback Plan

If issues arise:

```bash
# Use v1.0.0 templates temporarily
git checkout v1.0.0 pkg/cloudinit/templates.go
make build
```

## Timeline

- **v1.0.0** (Current) - Mixed responsibilities
- **v1.1.0** (This release) - Clean separation
- **v1.2.0** (Future) - Enhanced integration, monitoring

## Success Criteria

- ✅ All tests pass
- ✅ No NATS installation in cloud-init
- ✅ Callback integration working
- ✅ Documentation updated
- ✅ Backward compatibility (can still work without callbacks)

## Questions & Answers

**Q: What if NimsForest is not available?**
A: Morpheus completes successfully. Infrastructure is ready. NimsForest can bootstrap later via polling or manual trigger.

**Q: Can I still use Morpheus standalone?**
A: Yes! Set `integration.nimsforest_url: ""` and infrastructure will be provisioned without callbacks.

**Q: Do I need to change my deployment process?**
A: Only if you want automatic bootstrap. Otherwise, manually trigger NimsForest after Morpheus.

**Q: What about monitoring?**
A: Infrastructure monitoring stays with Morpheus (server health). Application monitoring is NimsForest (NATS metrics).

## Conclusion

This separation creates a cleaner, more maintainable architecture where:
- **Morpheus** = Infrastructure as Code
- **NimsForest** = Application Orchestration

Each tool excels at its responsibility, making the system more flexible, testable, and scalable.

---

**Status:** ✅ Complete
**Version:** v1.1.0
**Date:** December 26, 2025
