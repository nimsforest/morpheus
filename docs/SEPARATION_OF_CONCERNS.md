# Separation of Concerns: Morpheus vs NimsForest

## Overview

Morpheus and NimsForest have distinct, complementary responsibilities in the forest ecosystem.

## Responsibility Matrix

| Concern | Morpheus | NimsForest |
|---------|----------|------------|
| **Infrastructure** | ✅ Owns | Uses |
| Server provisioning | ✅ | |
| Network configuration | ✅ | |
| Firewall rules | ✅ | |
| OS setup | ✅ | |
| SSH keys | ✅ | |
| Storage (NFS/volumes) | ✅ | |
| **Application** | | ✅ Owns |
| NATS installation | | ✅ |
| NATS configuration | | ✅ |
| Clustering setup | | ✅ |
| Service orchestration | | ✅ |
| Application monitoring | | ✅ |
| Auto-scaling logic | | ✅ |

## Morpheus Responsibilities

### What Morpheus Does

**Infrastructure Layer**
- Provision cloud servers via provider APIs (Hetzner, AWS, etc.)
- Configure network interfaces and security groups
- Set up firewall rules (UFW) with required ports
- Install base OS packages
- Configure SSH access with keys
- Set up storage infrastructure (NFS, volumes)
- Prepare directories and permissions

**Output**
- Infrastructure-ready servers with:
  - OS updated and configured
  - Firewall configured with NATS ports open
  - Docker installed (if needed)
  - Storage configured (if needed)
  - Metadata available at `/etc/morpheus/node-info.json`

### What Morpheus Does NOT Do

- ❌ Install NATS
- ❌ Configure NATS clustering
- ❌ Manage application services
- ❌ Handle application-level scaling
- ❌ Application monitoring

## NimsForest Responsibilities

### What NimsForest Does

**Application Layer**
- Install NATS server binaries
- Configure NATS settings (ports, limits, accounts)
- Set up NATS clustering and routes
- Manage JetStream configuration
- Deploy and orchestrate services
- Handle application-level monitoring
- Implement auto-scaling logic
- Manage service discovery

**Input**
- Infrastructure-ready servers from Morpheus
- Node metadata from `/etc/morpheus/node-info.json`
- Callback notification when infrastructure is ready

### What NimsForest Does NOT Do

- ❌ Provision cloud servers
- ❌ Configure OS-level firewalls
- ❌ Manage cloud provider APIs
- ❌ Handle infrastructure teardown

## Integration Flow

### 1. Morpheus Provisions Infrastructure

```bash
morpheus plant cloud forest
```

**Morpheus:**
1. Calls Hetzner API to create servers
2. Waits for servers to be `running`
3. Cloud-init executes:
   - Updates OS
   - Installs base packages (docker, curl, etc.)
   - Configures firewall (ports 4222, 6222, 8222)
   - Sets up directories (`/opt/nimsforest`, `/var/lib/nimsforest`)
   - Writes node metadata to `/etc/morpheus/node-info.json`
   - Calls NimsForest callback API
4. Sets node status to `infrastructure_ready`

### 2. NimsForest Bootstraps Application

**NimsForest receives callback:**
```json
POST /api/v1/bootstrap
{
  "forest_id": "forest-123",
  "node_id": "server-456",
  "node_ip": "95.217.0.1",
  "role": "edge"
}
```

**NimsForest:**
1. SSH to node or use agent
2. Reads `/etc/morpheus/node-info.json`
3. Installs NATS server
4. Configures NATS with clustering
5. Starts NATS services
6. Updates forest registry to `active`

### 3. Alternative: Polling Mode

If callback fails, NimsForest can poll:

```bash
# On the node, NimsForest agent checks
if [ -f /etc/morpheus/node-info.json ]; then
  # Infrastructure is ready, proceed with bootstrap
  nimsforest-agent bootstrap
fi
```

## Configuration

### Morpheus Configuration

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1

integration:
  nimsforest_url: "https://nimsforest.example.com"  # Callback URL
  registry_url: ""  # Optional: Morpheus registry for infra state

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"
```

### NimsForest Configuration

```yaml
infrastructure:
  provisioner: morpheus  # or manual, ansible, terraform
  
nats:
  version: "2.10.7"
  cluster_name: "${FOREST_ID}"
  ports:
    client: 4222
    cluster: 6222
    monitor: 8222
  
  jetstream:
    enabled: true
    storage: /var/lib/nimsforest/jetstream
```

## Node Metadata

Morpheus writes node information that NimsForest can use:

```json
{
  "forest_id": "forest-1735234567",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z",
  "registry_url": "https://registry.example.com",
  "callback_url": "https://nimsforest.example.com"
}
```

## Firewall Ports

Morpheus opens these ports for NimsForest:

| Port | Protocol | Purpose | Service |
|------|----------|---------|---------|
| 22 | TCP | SSH | Management |
| 4222 | TCP | NATS client | NATS |
| 6222 | TCP | NATS cluster | NATS |
| 8222 | TCP | NATS monitoring | NATS |
| 7777 | TCP | NATS leafnode | NATS |
| 2049 | TCP | NFS | Storage nodes |

## Directory Structure

Morpheus creates directories for NimsForest:

```
/opt/nimsforest/          # Application binaries
/var/lib/nimsforest/      # Data storage (JetStream, etc.)
/var/log/nimsforest/      # Application logs
/etc/morpheus/            # Morpheus metadata
/mnt/nimsforest-storage/  # NFS exports (storage nodes)
```

## Error Handling

### Morpheus Failures

If Morpheus fails to provision:
- Automatic rollback
- Delete partially created servers
- Clean up registry entries
- Report detailed error

### NimsForest Failures

If NimsForest bootstrap fails:
- Infrastructure remains (for debugging)
- Retry mechanism in NimsForest
- Manual intervention possible via SSH

## Benefits of Separation

### 1. **Single Responsibility**
- Each tool has one clear purpose
- Easier to maintain and debug
- Simpler codebases

### 2. **Flexibility**
- Use Morpheus with any application
- Use NimsForest with manual infrastructure
- Swap infrastructure providers easily

### 3. **Testing**
- Test infrastructure separately from application
- Mock/stub boundaries clearly defined
- Integration tests at clear interfaces

### 4. **Deployment Options**

**Option A: Morpheus + NimsForest (Automated)**
```bash
morpheus plant cloud forest
# Morpheus calls NimsForest callback
# NimsForest bootstraps automatically
```

**Option B: Morpheus → Manual → NimsForest**
```bash
morpheus plant cloud forest
# Verify infrastructure
ssh root@server
# Bootstrap NimsForest manually
nimsforest bootstrap --forest-id forest-123
```

**Option C: Manual Infrastructure + NimsForest**
```bash
# Create servers manually
# Install OS, configure firewall
# Run NimsForest bootstrap
nimsforest bootstrap --forest-id forest-123
```

## API Contract

### Morpheus → NimsForest

**Callback on infrastructure ready:**

```http
POST /api/v1/bootstrap
Content-Type: application/json

{
  "forest_id": "forest-123",
  "node_id": "server-456",
  "node_ip": "95.217.0.1",
  "role": "edge",
  "metadata": {
    "location": "fsn1",
    "provider": "hetzner"
  }
}
```

**Response:**
```json
{
  "status": "accepted",
  "message": "Bootstrap queued",
  "estimated_time": "5m"
}
```

### NimsForest → Morpheus

**Update node status (optional):**

```http
POST /api/v1/nodes/:node_id/status
Content-Type: application/json

{
  "status": "active",
  "nats_version": "2.10.7",
  "cluster_name": "forest-123"
}
```

## Migration Path

### Current State (v1.0.0)
- Morpheus installs NATS (wrong!)
- Mixed responsibilities

### Target State (v1.1.0)
- Morpheus provisions infrastructure only
- NimsForest handles application
- Clear separation

### Migration Steps

1. **Deploy v1.1.0 Morpheus**
   - New cloud-init templates
   - Callback integration

2. **Deploy NimsForest Bootstrap**
   - Accept callbacks
   - Handle NATS installation

3. **Verify Integration**
   - Test end-to-end flow
   - Monitor callback success

4. **Cutover**
   - Use new flow for new forests
   - Migrate existing forests

## Future Enhancements

### Morpheus
- [ ] Multi-cloud support (AWS, GCP, Azure)
- [ ] Network topology management
- [ ] VPC/subnet configuration
- [ ] Load balancer provisioning

### NimsForest
- [ ] Auto-scaling triggers
- [ ] Health checks and self-healing
- [ ] Zero-downtime upgrades
- [ ] Disaster recovery

### Integration
- [ ] WebSocket notifications
- [ ] Bidirectional status sync
- [ ] Grafana/Prometheus integration
- [ ] Audit logging

## Summary

| Aspect | Morpheus | NimsForest |
|--------|----------|------------|
| **Layer** | Infrastructure | Application |
| **Scope** | Servers, networks, storage | NATS, services, clustering |
| **Lifecycle** | Provision → Teardown | Install → Upgrade → Scale |
| **Language** | Go | (Any - Go, Python, etc.) |
| **State** | Cloud provider + Registry | Forest topology |
| **Expertise** | Cloud/DevOps | NATS/Distributed systems |

**Golden Rule**: If it touches cloud APIs or OS-level config → Morpheus. If it touches NATS or application services → NimsForest.
