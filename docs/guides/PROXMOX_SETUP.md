# NimsForest VR - Proxmox Setup Guide

This guide sets up a **VR-capable NimsForest node** on Proxmox with:

- **Linux mode**: CachyOS + WiVRN for wireless VR streaming
- **Windows mode**: Windows + SteamLink for SteamVR ecosystem
- **NimsForest** runs inside both modes (monolith architecture)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          morpheus CLI                            │
│                                                                  │
│   morpheus plant nimsforest vr      # Initial setup              │
│   morpheus mode linux               # Switch to CachyOS          │
│   morpheus mode windows             # Switch to Windows          │
│                                                                  │
└───────────────────────────┬──────────────────────────────────────┘
                            │ Proxmox API
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Proxmox VE Host                              │
│                                                                  │
│  ┌──────────────────────┐  OR  ┌──────────────────────┐         │
│  │  Linux VM (CachyOS)  │      │   Windows VM         │         │
│  │                      │      │                      │         │
│  │  • WiVRN (VR)        │      │  • SteamLink (VR)    │         │
│  │  • NimsForest        │      │  • NimsForest        │         │
│  │  • NATS cluster      │      │  • NATS cluster      │         │
│  │  • GPU passthrough   │      │  • GPU passthrough   │         │
│  └──────────────────────┘      └──────────────────────┘         │
│                                                                  │
│  ⚠️  Only ONE VM runs at a time (GPU exclusive)                  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Proxmox VE 7.x or 8.x installed
- A dedicated GPU for passthrough (NVIDIA or AMD)
- CPU with IOMMU support (Intel VT-d or AMD-Vi)
- Network access to Proxmox API

## Step 1: Enable IOMMU

### Intel CPUs

Edit `/etc/default/grub`:
```bash
GRUB_CMDLINE_LINUX_DEFAULT="quiet intel_iommu=on iommu=pt"
```

### AMD CPUs

```bash
GRUB_CMDLINE_LINUX_DEFAULT="quiet amd_iommu=on iommu=pt"
```

Apply and reboot:
```bash
update-grub
reboot
```

Verify:
```bash
dmesg | grep -e DMAR -e IOMMU
```

## Step 2: Identify GPU for Passthrough

Find your GPU:
```bash
lspci -nn | grep -i vga
# Example: 01:00.0 VGA compatible controller [0300]: NVIDIA Corporation ... [10de:2684]
```

Find IOMMU group:
```bash
#!/bin/bash
for d in /sys/kernel/iommu_groups/*/devices/*; do
    n=$(basename $(dirname $(dirname "$d")))
    echo "IOMMU Group $n: $(lspci -nns ${d##*/})"
done
```

## Step 3: Blacklist GPU Driver on Host

Create `/etc/modprobe.d/blacklist-gpu.conf`:
```bash
# For NVIDIA
blacklist nouveau
blacklist nvidia
blacklist nvidia_drm
blacklist nvidia_modeset

# For AMD
blacklist radeon
blacklist amdgpu
```

Create `/etc/modprobe.d/vfio.conf`:
```bash
# Replace with your GPU's vendor:device IDs
options vfio-pci ids=10de:2684,10de:22ba
```

Update initramfs:
```bash
update-initramfs -u -k all
reboot
```

Verify:
```bash
lspci -nnk -s 01:00
# Should show: Kernel driver in use: vfio-pci
```

## Step 4: Create Linux VM (CachyOS)

```bash
# Create VM
qm create 101 --name nimsforest-vr-linux --memory 32768 --cores 12 \
  --cpu host,hidden=1 --machine q35 --bios ovmf \
  --net0 virtio,bridge=vmbr0

# Add EFI disk
qm set 101 --efidisk0 local-lvm:1,efitype=4m,pre-enrolled-keys=0

# Add main disk
qm set 101 --scsi0 local-lvm:100,ssd=1,discard=on

# Add GPU passthrough
qm set 101 --hostpci0 01:00,pcie=1,x-vga=1

# Optional: USB controller for VR headset
qm set 101 --hostpci1 00:14.0,pcie=1
```

### Install CachyOS

1. Download CachyOS ISO from https://cachyos.org/
2. Boot VM from ISO
3. Install with Desktop environment (KDE recommended for VR)
4. Reboot and install GPU drivers

### Install WiVRN

```bash
# On CachyOS (Arch-based)
yay -S wivrn-git

# Enable and start
systemctl --user enable wivrn
systemctl --user start wivrn
```

### Install NimsForest

```bash
# Download latest release
curl -LO https://github.com/nimsforest/nimsforest/releases/latest/download/nimsforest-linux-amd64
chmod +x nimsforest-linux-amd64
sudo mv nimsforest-linux-amd64 /usr/local/bin/nimsforest

# Create systemd service
sudo tee /etc/systemd/system/nimsforest.service << 'EOF'
[Unit]
Description=NimsForest Node
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nimsforest run
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable --now nimsforest
```

## Step 5: Create Windows VM

```bash
# Create VM
qm create 102 --name nimsforest-vr-windows --memory 32768 --cores 12 \
  --cpu host,hidden=1 --machine q35 --bios ovmf \
  --net0 virtio,bridge=vmbr0

# Add EFI disk
qm set 102 --efidisk0 local-lvm:1,efitype=4m,pre-enrolled-keys=0

# Add main disk (Windows needs more space)
qm set 102 --scsi0 local-lvm:200,ssd=1,discard=on

# Add GPU passthrough (same GPU)
qm set 102 --hostpci0 01:00,pcie=1,x-vga=1

# Add VirtIO drivers ISO
qm set 102 --ide2 local:iso/virtio-win.iso,media=cdrom
```

### Install Windows

1. Download Windows 10/11 ISO
2. Download VirtIO drivers ISO from https://fedorapeople.org/groups/virt/virtio-win/
3. Boot and install Windows
4. Install VirtIO drivers from ISO
5. Install QEMU Guest Agent
6. Install GPU drivers
7. Install Steam and SteamLink

### Install NimsForest on Windows

```powershell
# Download from GitHub releases
Invoke-WebRequest -Uri "https://github.com/nimsforest/nimsforest/releases/latest/download/nimsforest-windows-amd64.exe" -OutFile "nimsforest.exe"

# Run as service (use NSSM or similar)
```

## Step 6: Create Proxmox API Token

```bash
pveum user add morpheus@pam --comment "Morpheus automation"
pveum aclmod / -user morpheus@pam -role PVEVMAdmin
pveum user token add morpheus@pam morpheus-token --privsep=0
```

**Save the token value!**

## Step 7: Configure Morpheus

Add to `~/.morpheus/config.yaml`:

```yaml
proxmox:
  host: "192.168.1.100"
  port: 8006
  node: "pve"
  api_token_id: "morpheus@pam!morpheus-token"
  api_token_secret: "${PROXMOX_API_TOKEN}"
  verify_ssl: false

vr:
  linux:
    vmid: 101
    name: "nimsforest-vr-linux"
    memory: 32768
    cores: 12
    disk_size: 100
    
  windows:
    vmid: 102
    name: "nimsforest-vr-windows"
    memory: 32768
    cores: 12
    disk_size: 200
    
  gpu_pci: "0000:01:00"

nimsforest:
  cluster_id: "forest-abc123"
```

Set the token:
```bash
export PROXMOX_API_TOKEN="your-token-value"
```

## Step 8: Use Morpheus

```bash
# Check current mode
morpheus mode status

# Switch to Linux for WiVRN VR
morpheus mode linux

# Switch to Windows for SteamVR
morpheus mode windows

# List modes
morpheus mode list
```

## Usage Examples

### Switch to Linux for Wireless VR

```bash
$ morpheus mode linux

Switching windows → linux...
  Stopping windows VM... ✓ (15s)
  Starting linux VM... ✓ (8s)
  Waiting for network... ✓

✅ Now in linux mode
   IP: 192.168.1.150
   Services: WiVRN, NimsForest, NATS
```

### Check Status

```bash
$ morpheus mode status

🎮 Current Mode: linux
   VM: nimsforest-vr-linux (101)
   Status: running
   Uptime: 2h 34m
   IP: 192.168.1.150
   GPU: NVIDIA RTX 4090
   
   Services:
     • wivrn: active
     • nimsforest: active  
     • nats: active (cluster: forest-abc123)
```

### List All Modes

```bash
$ morpheus mode list

MODE      VMID   STATUS    OS        VR SOFTWARE
linux     101    running   CachyOS   WiVRN
windows   102    stopped   Win11     SteamLink
```

## Remote Access

### Option A: Tailscale (Recommended)

```bash
# On Proxmox host
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up
```

Now access from anywhere via Tailscale IP.

### Option B: WireGuard VPN

Set up WireGuard on your router.

## Troubleshooting

### GPU Not Detected

1. Verify IOMMU: `dmesg | grep -e DMAR -e IOMMU`
2. Check vfio-pci binding: `lspci -nnk -s 01:00`
3. Ensure `cpu: host,hidden=1` in VM config

### VM Won't Start

Only ONE VM can have the GPU at a time. Stop the other VM first:
```bash
morpheus mode linux  # This stops windows first
```

### WiVRN Not Working

1. Check service: `systemctl --user status wivrn`
2. Ensure VR headset is connected
3. Check firewall allows UDP ports

### NimsForest Not Connecting

1. Check service: `systemctl status nimsforest`
2. Verify network connectivity
3. Check NATS cluster configuration

## What Each Mode Provides

### Linux Mode (CachyOS)
- **VR**: WiVRN for wireless Quest/Pico streaming
- **Desktop**: KDE Plasma with GPU acceleration
- **Compute**: Full CUDA/ROCm support for NimsForest workloads
- **Drivers**: Latest NVIDIA/AMD from Arch repos

### Windows Mode
- **VR**: SteamLink, Virtual Desktop, native SteamVR
- **Gaming**: Native Windows game support
- **Compute**: CUDA support for NimsForest workloads
- **Compatibility**: Windows-only VR apps

## Why This Architecture?

1. **Monolith = Simple**: One machine, two boot options, no parallel VM complexity
2. **Full GPU**: Each mode gets exclusive GPU (required for VR)
3. **NimsForest Always**: Both modes participate in the distributed cluster
4. **Best of Both**: Open-source VR (WiVRN) + SteamVR ecosystem
5. **Remote Control**: Switch modes from your phone
