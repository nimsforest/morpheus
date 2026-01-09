# Feature: Proxmox Remote Boot Modes

## Status: ready

## Summary

Add Proxmox VE support to Morpheus for managing on-premise hardware with multiple boot modes (OS configurations). This enables remotely switching a physical machine between different workloads:

- **CachyOS + WiVRN**: VR streaming workstation with full GPU passthrough
- **Windows Pro**: Gaming/productivity with full GPU access
- **NimsForest**: Distributed compute node (may or may not need GPU)

The physical host never powers down - only VMs are stopped/started.

## Technical Constraints

1. **GPU Passthrough**: Only ONE VM can have GPU access at a time
2. **VM Restart Required**: Cannot hot-swap GPU between VMs (~10-30s downtime)
3. **Host Stays Up**: Proxmox host remains running, only guests restart
4. **Network Prerequisite**: Need network access to Proxmox API (local or VPN/Tailscale)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Morpheus CLI                             │
│                                                                  │
│   morpheus mode list              # Show available modes         │
│   morpheus mode switch cachyos    # Switch to CachyOS mode       │
│   morpheus mode status            # Show current mode            │
│                                                                  │
└───────────────────────────┬──────────────────────────────────────┘
                            │ Proxmox API (HTTPS)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Proxmox VE Host                              │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ VM 101       │  │ VM 102       │  │ VM 103       │           │
│  │ cachyos      │  │ windows      │  │ nimsforest   │           │
│  │ [GPU:0000:01]│  │ [GPU:0000:01]│  │ [no GPU]     │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                  │
│  GPU: Passed to whichever VM is running                          │
└─────────────────────────────────────────────────────────────────┘
```

## Tasks

### Phase 1: Proxmox Provider Core
- [ ] Task 1.1 - Create `pkg/machine/proxmox/client.go` - Proxmox API client (~pkg/machine/proxmox/)
- [ ] Task 1.2 - Create `pkg/machine/proxmox/proxmox.go` - Provider implementation (~pkg/machine/proxmox/)
- [ ] Task 1.3 - Create `pkg/machine/proxmox/types.go` - VM, Node, Cluster types (~pkg/machine/proxmox/)
- [ ] Task 1.4 - Add Proxmox config section to config.go (~pkg/config/config.go)

### Phase 2: Boot Mode Management
- [ ] Task 2.1 - Create `pkg/bootmode/interface.go` - Boot mode abstraction (~pkg/bootmode/)
- [ ] Task 2.2 - Create `pkg/bootmode/proxmox.go` - Proxmox-specific mode switching (~pkg/bootmode/)
- [ ] Task 2.3 - Create `pkg/bootmode/types.go` - Mode definitions with GPU requirements (~pkg/bootmode/)

### Phase 3: CLI Commands
- [ ] Task 3.1 - Add `morpheus mode list` command (~cmd/morpheus/)
- [ ] Task 3.2 - Add `morpheus mode switch <name>` command (~cmd/morpheus/)
- [ ] Task 3.3 - Add `morpheus mode status` command (~cmd/morpheus/)

### Phase 4: Documentation & Testing
- [ ] Task 4.1 - Create Proxmox setup guide (~docs/guides/PROXMOX_SETUP.md)
- [ ] Task 4.2 - Add provider tests (~pkg/machine/proxmox/*_test.go)
- [ ] Task 4.3 - Add boot mode tests (~pkg/bootmode/*_test.go)

## Parallelization

Group A (can run in parallel):
- Task 1.1, 1.2, 1.3 (Proxmox provider core)
- Task 2.1, 2.2, 2.3 (Boot mode abstraction)

Group B (depends on A):
- Task 1.4 (config integration)
- Task 3.1, 3.2, 3.3 (CLI commands)

Group C (depends on B):
- Task 4.1, 4.2, 4.3 (docs and tests)

## Files

### New Files
- pkg/machine/proxmox/client.go
- pkg/machine/proxmox/proxmox.go
- pkg/machine/proxmox/types.go
- pkg/bootmode/interface.go
- pkg/bootmode/proxmox.go
- pkg/bootmode/types.go
- docs/guides/PROXMOX_SETUP.md

### Modified Files
- pkg/config/config.go (add proxmox section)
- cmd/morpheus/main.go (add mode commands)

## Config Example

```yaml
# ~/.morpheus/config.yaml

proxmox:
  host: "192.168.1.100"           # Proxmox host IP
  port: 8006                       # Proxmox API port
  node: "pve"                      # Proxmox node name
  
  # Authentication (use API token, not password)
  api_token_id: "morpheus@pam!morpheus-token"
  api_token_secret: "${PROXMOX_API_TOKEN}"
  
  # TLS (self-signed certs common in home labs)
  verify_ssl: false
  
  # Boot modes - map friendly names to VM IDs
  modes:
    cachyos:
      vmid: 101
      description: "CachyOS + WiVRN (VR streaming)"
      gpu_passthrough: true
      
    windows:
      vmid: 102  
      description: "Windows 11 Pro"
      gpu_passthrough: true
      
    nimsforest:
      vmid: 103
      description: "NimsForest distributed compute"
      gpu_passthrough: false
```

## API Reference

### Proxmox VE API

Base URL: `https://{host}:8006/api2/json`

**Authentication:**
```
Authorization: PVEAPIToken={tokenid}={secret}
```

**Key Endpoints:**
```
GET  /nodes/{node}/qemu           # List all VMs
GET  /nodes/{node}/qemu/{vmid}/status/current  # VM status
POST /nodes/{node}/qemu/{vmid}/status/start    # Start VM
POST /nodes/{node}/qemu/{vmid}/status/stop     # Stop VM (graceful)
POST /nodes/{node}/qemu/{vmid}/status/shutdown # Shutdown VM (ACPI)
```

## CLI Examples

```bash
# List available modes
$ morpheus mode list
MODE         VMID    STATUS     GPU    DESCRIPTION
---------------------------------------------------------------
cachyos      101     running    yes    CachyOS + WiVRN (VR streaming)
windows      102     stopped    yes    Windows 11 Pro
nimsforest   103     stopped    no     NimsForest distributed compute

Current mode: cachyos

# Check current status
$ morpheus mode status
🎮 Current Mode: cachyos (VM 101)

Status: running
Uptime: 2h 34m
GPU:    NVIDIA RTX 4090 (passed through)
IP:     192.168.1.150

# Switch to Windows
$ morpheus mode switch windows

Switching from cachyos → windows...
  Shutting down cachyos (VM 101)... ✓ (8s)
  Starting windows (VM 102)... ✓ (15s)
  Waiting for network... ✓

✅ Now in windows mode
   IP: 192.168.1.151
   GPU: NVIDIA RTX 4090 (passed through)

# Switch to NimsForest (no GPU needed)
$ morpheus mode switch nimsforest

Switching from windows → nimsforest...
  Shutting down windows (VM 102)... ✓ (12s)
  Starting nimsforest (VM 103)... ✓ (5s)
  Waiting for network... ✓

✅ Now in nimsforest mode
   IP: 192.168.1.152
   GPU: none (not required)
```

## Proxmox VM Setup Prerequisites

Before using this feature, you need to set up the VMs in Proxmox:

### 1. Create VMs for Each Mode

```bash
# CachyOS VM (ID 101)
qm create 101 --name cachyos --memory 32768 --cores 12 \
  --cpu host --machine q35 --bios ovmf

# Windows VM (ID 102)  
qm create 102 --name windows --memory 32768 --cores 12 \
  --cpu host --machine q35 --bios ovmf

# NimsForest VM (ID 103)
qm create 103 --name nimsforest --memory 16384 --cores 8 \
  --cpu host
```

### 2. Configure GPU Passthrough

Edit VM config (`/etc/pve/qemu-server/{vmid}.conf`):

```conf
# For GPU passthrough VMs
cpu: host,hidden=1
hostpci0: 0000:01:00,pcie=1,x-vga=1
```

### 3. Create API Token

```bash
# In Proxmox shell
pveum user add morpheus@pam
pveum aclmod / -user morpheus@pam -role PVEVMAdmin
pveum user token add morpheus@pam morpheus-token --privsep=0
# Save the token secret!
```

## Safety Features

1. **Graceful Shutdown**: Uses ACPI shutdown, waits for clean shutdown
2. **Timeout Protection**: 60s timeout for shutdown, falls back to stop
3. **State Verification**: Confirms VM stopped before starting another
4. **GPU Conflict Prevention**: Won't start GPU VM if another GPU VM is running
5. **Dry Run Mode**: `morpheus mode switch windows --dry-run`

## Future Enhancements

- [ ] Wake-on-LAN support (wake host remotely)
- [ ] Scheduled mode switching (cron-style)
- [ ] Mode aliases in config
- [ ] Integration with Home Assistant
- [ ] Mobile notifications when switch completes
- [ ] Auto-switch based on time of day
