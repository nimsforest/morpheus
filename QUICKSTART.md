# Morpheus Quick Start Guide

Get your first Nims Forest up and running in 5 minutes!

## Prerequisites Checklist

- [ ] Go 1.24+ installed (`go version`)
- [ ] Hetzner Cloud account
- [ ] API token generated
- [ ] SSH key uploaded to Hetzner

## Step-by-Step Setup

### 1. Build Morpheus (2 minutes)

```bash
git clone https://github.com/yourusername/morpheus.git
cd morpheus
make build
```

### 2. Configure (1 minute)

Create `~/.morpheus/config.yaml`:

```bash
mkdir -p ~/.morpheus
cat > ~/.morpheus/config.yaml << EOF
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1
    - nbg1

secrets:
  hetzner_api_token: "YOUR_TOKEN_HERE"
EOF
```

Or use environment variable:

```bash
export HETZNER_API_TOKEN="your-token-here"
```

### 3. Plant Your Forest (5-10 minutes)

```bash
./bin/morpheus plant cloud wood
```

Expected output:

```
🌲 Planting Nims Forest...
Forest ID: forest-1735234567
Size: wood
Location: fsn1
Provider: hetzner

Provisioning 1 node(s)...
Server 12345678 created, waiting for it to be ready...
✓ Node forest-1735234567-node-1 provisioned successfully (IP: 95.217.123.45)

✓ Forest planted successfully!
```

### 4. Verify Your Forest

```bash
./bin/morpheus status forest-1735234567
```

### 5. Connect to Your Server

```bash
ssh root@95.217.123.45

# Check NATS is running
systemctl status nats-server

# View NATS info
curl http://localhost:8222/varz
```

### 6. Clean Up

```bash
./bin/morpheus teardown forest-1735234567
```

## What's Next?

- **Scale up**: Try `morpheus plant cloud forest` (3 nodes)
- **Monitor**: Check NATS at `http://<ip>:8222`
- **Customize**: Edit cloud-init templates in `pkg/cloudinit/`
- **Learn more**: Read [README.md](README.md) and [SETUP.md](SETUP.md)

## Common Issues

### Build fails

```bash
# Clean and retry
make clean
make deps
make build
```

### Can't find config

```bash
# Check config location
ls -la ~/.morpheus/config.yaml
# Or use current directory
cp config.example.yaml config.yaml
```

### SSH key not found

```bash
# List your keys in Hetzner
hcloud ssh-key list
# Update config to match exact name
```

## Cost Estimate

For testing with `cpx11` (smallest):
- ~€4.50/month = ~€0.006/hour
- 1 hour test: ~€0.01
- 8 hour workday: ~€0.05

For production with `cpx31` (recommended):
- wood (1 node): ~€18/month
- forest (3 nodes): ~€54/month
- jungle (5 nodes): ~€90/month

Remember: Hetzner bills by the minute, so teardown when not needed!

## Quick Commands Reference

```bash
# Build
make build

# Plant forests
morpheus plant cloud wood     # 1 node
morpheus plant cloud forest   # 3 nodes
morpheus plant cloud jungle   # 5 nodes

# Manage forests
morpheus list                 # List all forests
morpheus status <forest-id>   # Show details
morpheus teardown <forest-id> # Delete forest

# Help
morpheus help                 # Show help
morpheus version              # Show version
```

## Getting Help

- 📖 Full docs: [README.md](README.md)
- 🔧 Setup guide: [SETUP.md](SETUP.md)
- ❓ FAQ: [docs/FAQ.md](docs/FAQ.md)
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/morpheus/issues)

Happy planting! 🌲
