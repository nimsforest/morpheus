# Morpheus TODO

## Architecture (Embedded NATS)

```
┌─────────────────┐
│    Morpheus     │  Stateless CLI (phone/laptop)
└────────┬────────┘
         │ reads/writes via WebDAV
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

NimsForest embeds NATS server directly. No separate NATS installation needed.

---

## How It Works

```
Node-1 starts:
  → Mounts /mnt/forest (StorageBox via CIFS)
  → Reads /mnt/forest/registry.json → no other nodes yet
  → Starts nimsforest with embedded NATS as cluster of 1

Node-2 starts:
  → Mounts /mnt/forest (StorageBox via CIFS)
  → Reads registry → sees node-1
  → Connects to node-1 via NATS gossip
  → Cluster is now 2 nodes

Node-N added:
  → Reads registry → finds at least one peer
  → NATS gossip propagates membership automatically
```

A new node only needs ONE existing peer. NATS gossip handles the rest.

---

## Completed

- [x] Basic `morpheus plant cloud small/medium/large`
- [x] NimsForest binary download
- [x] NimsForest systemd service
- [x] Configurable download URL
- [x] StorageBox Registry - WebDAV client with optimistic locking
- [x] Registry config with StorageBoxHost for CIFS mount
- [x] NATS monitoring code (`pkg/nats/monitor.go`)
- [x] `morpheus grow` command - cluster health monitoring
- [x] `morpheus grow --auto` - non-interactive mode with JSON output
- [x] Firewall rules for NATS ports
- [x] Cloud-init mounts StorageBox at /mnt/forest
- [x] Cloud-init registers node in shared registry (using jq)
- [x] NimsForest service depends on mnt-forest.mount
- [x] Environment variables: FOREST_ID, NODE_ID, NODE_ROLE, NODE_IP, REGISTRY_PATH

---

## Quick Reference

**File Locations on Each Node:**
- `/etc/morpheus/node-info.json` - Node identity (forest_id, node_id)
- `/mnt/forest/registry.json` - Shared registry (peer discovery)
- `/var/lib/nimsforest/` - NimsForest data (including embedded JetStream)
- `/opt/nimsforest/bin/nimsforest` - NimsForest binary (with embedded NATS)

**What NimsForest Does on Startup:**
1. Reads `FOREST_ID` and `NODE_ID` from environment
2. Reads `REGISTRY_PATH` (/mnt/forest/registry.json) for peer IPs
3. Starts embedded NATS with auto-configured cluster settings
4. Connects to any discovered peer via NATS gossip
5. NATS gossip propagates cluster membership automatically

**Ports:**
- 6222 - NATS cluster (required for node-to-node)
- 4222 - NATS client (localhost only by default)
- 8222 - NATS monitoring (for `morpheus grow`)

**Config Example:**
```yaml
registry:
  type: storagebox
  url: https://uXXXXX.your-storagebox.de/morpheus/registry.json  # WebDAV for CLI
  storagebox_host: uXXXXX.your-storagebox.de                      # CIFS for nodes
  username: uXXXXX
  password: ${STORAGEBOX_PASSWORD}
```

---

## Future Improvements

### True Local Mode (No Docker)
Now simpler with embedded NATS - just run the NimsForest binary directly.

### Health Checks
Add health endpoint that Morpheus can query:
- Is NimsForest running?
- Is embedded NATS healthy?
- Are peers reachable?
