# Morpheus TODO

## Architecture

```
┌─────────────────┐
│    Morpheus     │  CLI (phone/laptop)
└────────┬────────┘
         │ WebDAV
         ▼
┌─────────────────┐
│ Hetzner         │  Shared registry at /mnt/forest/registry.json
│ StorageBox      │  Mounted via CIFS on each node
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Forest Nodes    │  Each runs NimsForest with embedded NATS
│ (Hetzner VMs)   │  Peers discover each other via shared registry
└─────────────────┘
```

All nodes are identical - they run NimsForest with embedded NATS. StorageBox provides shared storage.

---

## How It Works

```
Node-1 starts:
  → Mounts /mnt/forest (StorageBox via CIFS)
  → Registers in /mnt/forest/registry.json
  → Starts NimsForest with embedded NATS (cluster of 1)

Node-2 starts:
  → Mounts /mnt/forest
  → Registers in registry, sees node-1
  → Starts NimsForest, joins cluster via NATS gossip

Node-N:
  → Same pattern - NATS gossip handles membership
```

---

## Completed

- [x] `morpheus plant cloud small/medium/large`
- [x] NimsForest binary download and systemd service
- [x] StorageBox Registry (WebDAV + CIFS mount)
- [x] NATS monitoring (`pkg/nats/monitor.go`)
- [x] `morpheus grow` - cluster health monitoring
- [x] Cloud-init mounts StorageBox, registers node (using jq)
- [x] Environment variables: FOREST_ID, NODE_ID, NODE_IP, REGISTRY_PATH

---

## Quick Reference

**Node Files:**
- `/etc/morpheus/node-info.json` - Node identity
- `/mnt/forest/registry.json` - Shared peer registry
- `/opt/nimsforest/bin/nimsforest` - NimsForest binary

**NimsForest Startup:**
1. Reads FOREST_ID, NODE_ID from environment
2. Reads REGISTRY_PATH for peer IPs
3. Starts embedded NATS, joins cluster via gossip

**Ports:**
- 4222 - NATS client
- 6222 - NATS cluster
- 8222 - NATS monitoring

**Config:**
```yaml
registry:
  type: storagebox
  url: https://uXXXXX.your-storagebox.de/morpheus/registry.json
  storagebox_host: uXXXXX.your-storagebox.de
  username: uXXXXX
  password: ${STORAGEBOX_PASSWORD}
```

---

## Future

- Local mode (run NimsForest directly without cloud)
- Health checks endpoint
