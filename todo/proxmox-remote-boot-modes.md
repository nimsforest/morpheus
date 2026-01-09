# Feature: NimsForest VR (Proxmox)

## Status: ready

## Summary

`morpheus plant nimsforest vr` provisions a VR-capable monolith machine on Proxmox:

- **Single physical machine** with GPU passthrough
- **Two boot modes**: Linux (CachyOS + WiVRN) or Windows (SteamLink)
- **NimsForest runs inside** the active VM (not as parallel VMs)
- **Remote mode switching** via Proxmox API

This is for on-premise hardware with a dedicated GPU for VR streaming.

## Command

```bash
morpheus plant nimsforest vr
```

This creates:
1. A Proxmox VM with CachyOS, WiVRN, and NimsForest
2. A Proxmox VM with Windows and SteamLink (optional, can be added later)
3. GPU passthrough configured for both
4. Only ONE runs at a time

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          morpheus CLI                            │
│                                                                  │
│   morpheus plant nimsforest vr      # Initial setup              │
│   morpheus mode linux               # Switch to CachyOS          │
│   morpheus mode windows             # Switch to Windows          │
│   morpheus mode status              # Show current mode          │
│                                                                  │
└───────────────────────────┬──────────────────────────────────────┘
                            │ Proxmox API
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Proxmox VE Host                              │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                  Linux VM (CachyOS)                         │ │
│  │                                                             │ │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │ │
│  │   │   WiVRN     │  │ NimsForest  │  │    NATS     │        │ │
│  │   │  (VR out)   │  │  (compute)  │  │  (cluster)  │        │ │
│  │   └─────────────┘  └─────────────┘  └─────────────┘        │ │
│  │                                                             │ │
│  │   GPU: Full passthrough for VR + compute                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                            OR                                    │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                 Windows VM                                  │ │
│  │                                                             │ │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │ │
│  │   │  SteamLink  │  │ NimsForest  │  │    NATS     │        │ │
│  │   │  (VR out)   │  │  (compute)  │  │  (cluster)  │        │ │
│  │   └─────────────┘  └─────────────┘  └─────────────┘        │ │
│  │                                                             │ │
│  │   GPU: Full passthrough for VR + compute                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ⚠️  Only ONE VM runs at a time (GPU exclusive)                  │
└─────────────────────────────────────────────────────────────────┘
```

## What Each Mode Provides

### Linux Mode (CachyOS)
- **OS**: CachyOS (Arch-based, optimized for gaming/GPU)
- **VR**: WiVRN for wireless VR streaming
- **Compute**: NimsForest + NATS for distributed workloads
- **GPU**: Full NVIDIA/AMD support with latest drivers

### Windows Mode
- **OS**: Windows 10/11 Pro
- **VR**: SteamLink / Virtual Desktop for VR streaming
- **Compute**: NimsForest (Go binaries work on Windows)
- **GPU**: Native Windows drivers

## CLI Usage

```bash
# Provision a new VR-capable NimsForest node
morpheus plant nimsforest vr

# Check current mode
morpheus mode status
🎮 Current Mode: linux
   VM: nimsforest-vr-linux (VM 101)
   Status: running
   IP: 192.168.1.150
   GPU: NVIDIA RTX 4090
   Services:
     • WiVRN: active
     • NimsForest: active
     • NATS: active (cluster: forest-abc123)

# Switch to Windows
morpheus mode windows
Switching linux → windows...
  Stopping linux VM... ✓ (12s)
  Starting windows VM... ✓ (25s)
  Waiting for services... ✓

✅ Now in windows mode
   IP: 192.168.1.151
   Services: SteamLink, NimsForest, NATS

# Switch back to Linux
morpheus mode linux
Switching windows → linux...
  Stopping windows VM... ✓ (15s)
  Starting linux VM... ✓ (8s)

✅ Now in linux mode

# List all modes
morpheus mode list
MODE      VM ID   STATUS    OS        VR SOFTWARE
linux     101     running   CachyOS   WiVRN
windows   102     stopped   Win11     SteamLink
```

## Tasks

### Phase 1: Core Proxmox Provider (Done ✅)
- [x] `pkg/machine/proxmox/client.go` - Proxmox API client
- [x] `pkg/machine/proxmox/proxmox.go` - Provider implementation
- [x] `pkg/machine/proxmox/types.go` - VM types

### Phase 2: Simplified Boot Mode
- [ ] Refactor `pkg/bootmode/` to use linux/windows modes
- [ ] Remove GPU mode complexity (always exclusive)
- [ ] Add mode config to main config.go

### Phase 3: Plant Command Integration
- [ ] Add `plant nimsforest vr` subcommand
- [ ] Create Proxmox VM templates for CachyOS
- [ ] Create Proxmox VM templates for Windows
- [ ] Auto-configure GPU passthrough

### Phase 4: Cloud-init / Setup Scripts
- [ ] CachyOS cloud-init with WiVRN + NimsForest
- [ ] Windows unattended setup with SteamLink + NimsForest
- [ ] NATS cluster configuration

### Phase 5: Documentation
- [ ] Update PROXMOX_SETUP.md guide
- [ ] Add VR-specific troubleshooting

## Config

```yaml
# ~/.morpheus/config.yaml

proxmox:
  host: "192.168.1.100"
  port: 8006
  node: "pve"
  api_token_id: "morpheus@pam!token"
  api_token_secret: "${PROXMOX_API_TOKEN}"
  verify_ssl: false

# VR node configuration
vr:
  # Linux VM settings
  linux:
    vmid: 101
    name: "nimsforest-vr-linux"
    memory: 32768          # 32 GB
    cores: 12
    disk_size: 100         # GB
    gpu_pci: "0000:01:00"  # GPU PCI address
    
  # Windows VM settings  
  windows:
    vmid: 102
    name: "nimsforest-vr-windows"
    memory: 32768
    cores: 12
    disk_size: 200         # GB (Windows needs more)
    gpu_pci: "0000:01:00"  # Same GPU

# NimsForest settings (applies to both modes)
nimsforest:
  cluster_id: "forest-abc123"
  nats_port: 4222
  registry_url: "https://registry.example.com"
```

## Provisioning Flow

```
morpheus plant nimsforest vr
         │
         ▼
┌─────────────────────────────────┐
│ 1. Connect to Proxmox API       │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 2. Detect GPU for passthrough   │
│    - Find IOMMU groups          │
│    - Verify vfio-pci binding    │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 3. Create Linux VM (CachyOS)    │
│    - Download/use CachyOS ISO   │
│    - Configure GPU passthrough  │
│    - Run cloud-init setup       │
│    - Install WiVRN + NimsForest │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 4. Create Windows VM (optional) │
│    - Use Windows ISO            │
│    - Configure GPU passthrough  │
│    - Run unattended setup       │
│    - Install SteamLink + NimsF  │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 5. Register with NimsForest     │
│    - Join NATS cluster          │
│    - Report to registry         │
└─────────────────────────────────┘
         │
         ▼
✅ VR node ready!
   Use: morpheus mode linux/windows
```

## Mode Switching Flow

```
morpheus mode windows
         │
         ▼
┌─────────────────────────────────┐
│ 1. Check current mode           │
│    - If already windows: done   │
│    - If linux: continue         │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 2. Graceful shutdown            │
│    - Notify NATS cluster        │
│    - Stop NimsForest services   │
│    - ACPI shutdown Linux VM     │
│    - Wait for stopped state     │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 3. Start target VM              │
│    - Start Windows VM           │
│    - Wait for running state     │
│    - Wait for network (IP)      │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ 4. Verify services              │
│    - Check NimsForest running   │
│    - Check NATS connected       │
│    - Check VR software ready    │
└─────────────────────────────────┘
         │
         ▼
✅ Switched to windows mode
```

## Why This Architecture?

1. **Monolith = Simpler**: One machine, two boot options, no parallel VM complexity
2. **Full GPU**: Each mode gets exclusive GPU access (required for VR)
3. **NimsForest Always**: Both modes can participate in the distributed cluster
4. **Best of Both**: Linux for open-source VR (WiVRN), Windows for SteamVR ecosystem
5. **Remote Control**: Switch modes from your phone via Proxmox API
