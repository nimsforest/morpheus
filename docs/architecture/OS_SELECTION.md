# OS Selection

## Recommendation

**Use Ubuntu 24.04 LTS for all nodes.**

## Configuration

```yaml
infrastructure:
  defaults:
    image: ubuntu-24.04
    server_type: cx23
```

## Why Ubuntu

- **Forest nodes (NATS):** Lightweight enough, well-supported
- **Nims nodes (GPU):** Required for NVIDIA drivers and CUDA
- **Consistency:** One OS to maintain

## GPU Setup (Nims Nodes)

```bash
# Install NVIDIA drivers
apt update
apt install -y ubuntu-drivers-common
ubuntu-drivers autoinstall

# Install CUDA (if needed)
apt install -y nvidia-cuda-toolkit

# Reboot
reboot

# Verify
nvidia-smi
```

## Debian Alternative

Debian 12 saves ~50MB RAM but:
- No benefit for NATS (identical performance)
- Doesn't support GPU workloads
- Not worth the complexity

**Conclusion:** Stick with Ubuntu 24.04.
