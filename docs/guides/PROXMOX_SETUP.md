# Proxmox Remote Boot Modes Setup Guide

This guide walks you through setting up Proxmox VE for remote boot mode switching with Morpheus. This enables you to remotely switch a physical machine between different workloads:

- **linuxvrstreaming**: Linux VR streaming workstation (e.g., CachyOS + WiVRN) - exclusive GPU
- **windowsvrstreaming**: Windows VR streaming workstation - exclusive GPU
- **nimsforestnogpu**: NimsForest distributed compute without GPU
- **nimsforestsharedgpu**: NimsForest with GPU compute - cannot combine with VR streaming

## Prerequisites

- A machine with Proxmox VE 7.x or 8.x installed
- A dedicated GPU for passthrough (NVIDIA or AMD)
- Network access to the Proxmox API (local, VPN, or Tailscale)
- CPU with IOMMU support (Intel VT-d or AMD-Vi)

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                     Your Phone / Laptop                          │
│                                                                  │
│   $ morpheus mode switch windowsvrstreaming                      │
│                                                                  │
└───────────────────────────┬──────────────────────────────────────┘
                            │ HTTPS (Proxmox API)
                            │ via local network / VPN / Tailscale
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Proxmox VE Host                              │
│                   (always running)                               │
│                                                                  │
│  ┌────────────────┐  ┌──────────────────┐  ┌─────────────────┐  │
│  │ VM 101         │  │ VM 102           │  │ VM 103          │  │
│  │linuxvrstreaming│  │windowsvrstreaming│  │nimsforestnogpu  │  │
│  │ [GPU exclusive]│  │ [GPU exclusive]  │  │ [no GPU]        │  │
│  │    STOPPED     │  │    RUNNING       │  │    STOPPED      │  │
│  └────────────────┘  └──────────────────┘  └─────────────────┘  │
│                             ▲                                    │
│                             │                                    │
│               GPU: NVIDIA RTX (exclusive to Windows VR)          │
└─────────────────────────────────────────────────────────────────┘
```

**Key constraint**: Only ONE VM can have the GPU at a time. Switching modes takes ~10-30 seconds because the current VM must fully stop before another can claim the GPU.

## Step 1: Enable IOMMU

### For Intel CPUs

Edit `/etc/default/grub`:
```bash
GRUB_CMDLINE_LINUX_DEFAULT="quiet intel_iommu=on iommu=pt"
```

### For AMD CPUs

Edit `/etc/default/grub`:
```bash
GRUB_CMDLINE_LINUX_DEFAULT="quiet amd_iommu=on iommu=pt"
```

Apply changes:
```bash
update-grub
reboot
```

Verify IOMMU is enabled:
```bash
dmesg | grep -e DMAR -e IOMMU
```

## Step 2: Identify GPU for Passthrough

Find your GPU's PCI address:
```bash
lspci -nn | grep -i vga
# Example output: 01:00.0 VGA compatible controller [0300]: NVIDIA Corporation ... [10de:2684]
```

Find the IOMMU group:
```bash
#!/bin/bash
for d in /sys/kernel/iommu_groups/*/devices/*; do
    n=$(basename $(dirname $(dirname "$d")))
    echo "IOMMU Group $n: $(lspci -nns ${d##*/})"
done
```

Note: All devices in the same IOMMU group must be passed through together.

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

Verify GPU is using vfio-pci:
```bash
lspci -nnk -s 01:00
# Should show: Kernel driver in use: vfio-pci
```

## Step 4: Create VMs

### Linux VR Streaming VM (ID 101)

```bash
# Create VM
qm create 101 --name linuxvrstreaming --memory 32768 --cores 12 \
  --cpu host,hidden=1 --machine q35 --bios ovmf \
  --net0 virtio,bridge=vmbr0

# Add EFI disk
qm set 101 --efidisk0 local-lvm:1,efitype=4m,pre-enrolled-keys=0

# Add main disk (adjust storage as needed)
qm set 101 --scsi0 local-lvm:100,ssd=1,discard=on

# Add GPU passthrough (exclusive)
qm set 101 --hostpci0 01:00,pcie=1,x-vga=1

# Add USB controller for peripherals (optional)
qm set 101 --hostpci1 00:14.0,pcie=1
```

Install CachyOS (or your preferred Linux distro) from ISO, then install WiVRN.

### Windows VR Streaming VM (ID 102)

```bash
# Create VM
qm create 102 --name windowsvrstreaming --memory 32768 --cores 12 \
  --cpu host,hidden=1 --machine q35 --bios ovmf \
  --net0 virtio,bridge=vmbr0

# Add EFI disk
qm set 102 --efidisk0 local-lvm:1,efitype=4m,pre-enrolled-keys=0

# Add main disk
qm set 102 --scsi0 local-lvm:200,ssd=1,discard=on

# Add GPU passthrough (exclusive - same GPU, different VM)
qm set 102 --hostpci0 01:00,pcie=1,x-vga=1

# Add virtio drivers ISO for Windows
qm set 102 --ide2 local:iso/virtio-win.iso,media=cdrom
```

Install Windows, then install:
1. VirtIO drivers from the ISO
2. NVIDIA/AMD GPU drivers
3. QEMU Guest Agent (for IP detection)
4. VR software (SteamVR, Virtual Desktop, etc.)

### NimsForest No GPU VM (ID 103)

```bash
# Create VM (no GPU passthrough)
qm create 103 --name nimsforestnogpu --memory 16384 --cores 8 \
  --cpu host --machine q35 \
  --net0 virtio,bridge=vmbr0

# Add main disk
qm set 103 --scsi0 local-lvm:50,ssd=1,discard=on
```

Install Ubuntu 24.04 for NimsForest distributed compute.

### NimsForest Shared GPU VM (ID 104) - Optional

```bash
# Create VM (with GPU for compute workloads)
qm create 104 --name nimsforestsharedgpu --memory 24576 --cores 10 \
  --cpu host --machine q35 \
  --net0 virtio,bridge=vmbr0

# Add main disk
qm set 104 --scsi0 local-lvm:80,ssd=1,discard=on

# Add GPU passthrough (shared mode - can't run with VR streaming)
qm set 104 --hostpci0 01:00,pcie=1
```

Install Ubuntu 24.04 with CUDA/ROCm for GPU compute workloads.

**Note:** `nimsforestsharedgpu` cannot run alongside `linuxvrstreaming` or `windowsvrstreaming` because they all need the GPU.

## Step 5: Create Proxmox API Token

```bash
# Create user (if not using root)
pveum user add morpheus@pam --comment "Morpheus automation"

# Grant VM admin permissions
pveum aclmod / -user morpheus@pam -role PVEVMAdmin

# Create API token (SAVE THE SECRET!)
pveum user token add morpheus@pam morpheus-token --privsep=0
```

Output will look like:
```
┌──────────────┬──────────────────────────────────────────────────┐
│ key          │ value                                            │
╞══════════════╪══════════════════════════════════════════════════╡
│ full-tokenid │ morpheus@pam!morpheus-token                      │
├──────────────┼──────────────────────────────────────────────────┤
│ info         │ {"privsep":"0"}                                  │
├──────────────┼──────────────────────────────────────────────────┤
│ value        │ xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx             │
└──────────────┴──────────────────────────────────────────────────┘
```

**Save the token value!** You won't be able to see it again.

## Step 6: Configure Morpheus

Add to your `~/.morpheus/config.yaml`:

```yaml
proxmox:
  host: "192.168.1.100"           # Your Proxmox IP
  port: 8006
  node: "pve"                      # Proxmox node name
  
  # API token (from Step 5)
  api_token_id: "morpheus@pam!morpheus-token"
  api_token_secret: "${PROXMOX_API_TOKEN}"  # Set via environment
  
  # Self-signed certs are common in home labs
  verify_ssl: false
  
  # Boot modes
  modes:
    linuxvrstreaming:
      vmid: 101
      description: "Linux VR streaming (CachyOS + WiVRN)"
      gpu_mode: exclusive
      
    windowsvrstreaming:
      vmid: 102
      description: "Windows VR streaming"
      gpu_mode: exclusive
      
    nimsforestnogpu:
      vmid: 103
      description: "NimsForest distributed compute (no GPU)"
      gpu_mode: none
      
    nimsforestsharedgpu:
      vmid: 104
      description: "NimsForest with GPU compute"
      gpu_mode: shared
      conflicts_with:
        - linuxvrstreaming
        - windowsvrstreaming
```

Set the API token:
```bash
export PROXMOX_API_TOKEN="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

## Step 7: Use Morpheus

```bash
# List available modes
morpheus mode list

# Check current status
morpheus mode status

# Switch to Windows VR streaming
morpheus mode switch windowsvrstreaming

# Switch to Linux VR streaming
morpheus mode switch linuxvrstreaming

# Switch to NimsForest (no GPU)
morpheus mode switch nimsforestnogpu

# Switch to NimsForest with GPU (only if VR streaming is not running)
morpheus mode switch nimsforestsharedgpu
```

## Remote Access Options

### Option A: Tailscale (Recommended)

Install Tailscale on your Proxmox host:
```bash
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up
```

Now you can access Proxmox from anywhere via Tailscale IP.

### Option B: WireGuard VPN

Set up WireGuard on your router or a dedicated server.

### Option C: Port Forwarding (Not Recommended)

If you must expose Proxmox to the internet:
1. Use a strong password and API token
2. Enable 2FA on Proxmox
3. Consider using Cloudflare Tunnel instead

## Troubleshooting

### GPU Not Detected in VM

1. Verify IOMMU is enabled: `dmesg | grep -e DMAR -e IOMMU`
2. Check GPU is using vfio-pci: `lspci -nnk -s 01:00`
3. Ensure `cpu: host,hidden=1` is set in VM config

### VM Won't Start - GPU In Use

Only ONE VM can use the GPU. Stop any running GPU VM first:
```bash
morpheus mode switch <other-mode>
```

### Connection Refused

1. Check Proxmox is running: `systemctl status pveproxy`
2. Verify firewall allows port 8006
3. Test API access: `curl -k https://192.168.1.100:8006/api2/json`

### QEMU Guest Agent Not Working

Install in each VM:

**Linux:**
```bash
apt install qemu-guest-agent
systemctl enable --now qemu-guest-agent
```

**Windows:**
Install from the virtio-win ISO: `guest-agent/qemu-ga-x86_64.msi`

## Performance Tips

### For VR (CachyOS + WiVRN)

1. Pin CPU cores to the VM for consistent performance
2. Use hugepages for memory
3. Disable CPU power saving in BIOS
4. Use a wired network connection for low latency

### For Gaming (Windows)

1. Install GPU drivers in Safe Mode first
2. Use Looking Glass for local display
3. Consider CPU pinning for consistent frame times

### VM Config Optimizations

```conf
# /etc/pve/qemu-server/101.conf
cpu: host,hidden=1,flags=+pcid
args: -cpu host,kvm=off,hv_vendor_id=proxmox
machine: q35
balloon: 0
```

## Security Notes

1. **Never expose Proxmox directly to the internet** - use VPN or Tailscale
2. **Use API tokens instead of passwords** - tokens can be revoked individually
3. **Enable 2FA on Proxmox web UI** - even if using API tokens
4. **Keep Proxmox updated** - security patches are important

## Next Steps

- Set up scheduled mode switching (e.g., Windows at night for game updates)
- Integrate with Home Assistant for voice control
- Configure Wake-on-LAN for remote host power control
