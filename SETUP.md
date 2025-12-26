# Morpheus Setup Guide

This guide will walk you through setting up Morpheus and provisioning your first Nims Forest.

## Prerequisites

Before you begin, ensure you have:

1. **Go 1.21 or higher** installed
   ```bash
   go version
   ```

2. **A Hetzner Cloud account** with:
   - An active project
   - API token with read/write permissions
   - At least one SSH key uploaded

3. **Git** for cloning the repository

## Step 1: Clone and Build

```bash
# Clone the repository
git clone https://github.com/yourusername/morpheus.git
cd morpheus

# Download dependencies
make deps

# Build the binary
make build

# Optionally, install globally
make install
```

## Step 2: Get Your Hetzner API Token

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Select your project
3. Go to "Security" → "API Tokens"
4. Click "Generate API Token"
5. Give it a name (e.g., "morpheus")
6. Set permissions to "Read & Write"
7. Copy the token (you won't see it again!)

## Step 3: Upload Your SSH Key

If you haven't already uploaded your SSH key to Hetzner:

```bash
# Generate a new SSH key (if needed)
ssh-keygen -t ed25519 -C "your_email@example.com"

# Upload to Hetzner (using hcloud CLI)
hcloud ssh-key create --name main --public-key-from-file ~/.ssh/id_ed25519.pub
```

Or upload via the web console:
1. Go to "Security" → "SSH Keys"
2. Click "Add SSH Key"
3. Paste your public key (`~/.ssh/id_ed25519.pub`)
4. Name it "main" (or update config.yaml to match)

## Step 4: Configure Morpheus

Create a configuration file:

```bash
# Create config directory
mkdir -p ~/.morpheus

# Copy example config
cp config.example.yaml ~/.morpheus/config.yaml

# Edit with your settings
nano ~/.morpheus/config.yaml
```

Update the configuration:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main  # Must match your SSH key name in Hetzner
  locations:
    - fsn1
    - nbg1
    - hel1

secrets:
  hetzner_api_token: "YOUR_API_TOKEN_HERE"
```

**Important**: Never commit your `config.yaml` with the API token to version control!

### Alternative: Environment Variable

Instead of storing the token in the config file, you can use an environment variable:

```bash
export HETZNER_API_TOKEN="your-token-here"

# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export HETZNER_API_TOKEN="your-token-here"' >> ~/.bashrc
```

## Step 5: Verify Installation

Check that Morpheus is properly installed:

```bash
morpheus version
```

You should see:
```
Morpheus v1.0.0
```

Test the help command:

```bash
morpheus help
```

## Step 6: Plant Your First Forest

Now you're ready to provision your first forest!

### Small Forest (1 node)

```bash
morpheus plant cloud wood
```

This will:
- Create 1 server in Hetzner Cloud
- Install and configure NATS
- Register it in your local registry

Expected output:
```
🌲 Planting Nims Forest...
Forest ID: forest-1735234567
Size: wood
Location: fsn1
Provider: hetzner

Provisioning 1 node(s)...
Server 12345678 created, waiting for it to be ready...
Server running, waiting for cloud-init to complete...
✓ Node forest-1735234567-node-1 provisioned successfully (IP: 95.217.123.45)

✓ Forest planted successfully!

To check status: morpheus status forest-1735234567
To teardown: morpheus teardown forest-1735234567
```

### Medium Forest (3 nodes)

```bash
morpheus plant cloud forest
```

### Large Forest (5 nodes)

```bash
morpheus plant cloud jungle
```

## Step 7: Verify Your Forest

Check that your forest is running:

```bash
# List all forests
morpheus list

# Get detailed status
morpheus status forest-1735234567
```

## Step 8: Connect to Your Nodes

SSH into any node using the IP from the status command:

```bash
ssh root@95.217.123.45
```

Check NATS status:

```bash
# Check if NATS is running
systemctl status nats-server

# View NATS logs
journalctl -u nats-server -f

# Test NATS connection
curl http://localhost:8222/varz
```

## Step 9: Teardown (Optional)

When you're done testing, clean up:

```bash
morpheus teardown forest-1735234567
```

This will delete all servers and clean up resources.

## Common Server Types

Choose based on your needs:

| Type | vCPU | RAM | Price* | Use Case |
|------|------|-----|--------|----------|
| cpx11 | 2 | 2 GB | ~€4.50/mo | Testing |
| cpx21 | 3 | 4 GB | ~€9/mo | Small production |
| cpx31 | 4 | 8 GB | ~€18/mo | **Recommended** |
| cpx41 | 8 | 16 GB | ~€36/mo | High load |
| cpx51 | 16 | 32 GB | ~€72/mo | Enterprise |

*Prices are approximate and may vary

## Locations and Latency

Choose the location closest to your users:

- **fsn1** (Falkenstein, Germany): Western Europe, Central Europe
- **nbg1** (Nuremberg, Germany): Central Europe, Eastern Europe
- **hel1** (Helsinki, Finland): Northern Europe, Russia

## Troubleshooting

### Can't find config file

```bash
# Morpheus looks for config in these locations (in order):
# 1. ./config.yaml
# 2. ~/.morpheus/config.yaml

# Create the directory if it doesn't exist
mkdir -p ~/.morpheus
cp config.example.yaml ~/.morpheus/config.yaml
```

### API token not working

```bash
# Verify your token has read/write permissions
# Test with hcloud CLI
hcloud server list
```

### SSH key not found

```bash
# List keys in Hetzner
hcloud ssh-key list

# Make sure the name in config.yaml matches exactly
```

### Server creation fails

Check your Hetzner Cloud limits:
- Maximum servers per project
- Resource quotas
- Payment status

### Cloud-init not completing

SSH into the server and check logs:

```bash
ssh root@<server-ip>

# Check cloud-init status
cloud-init status

# View full logs
tail -f /var/log/cloud-init-output.log

# Check NATS installation
which nats-server
systemctl status nats-server
```

## Next Steps

- Read the [README.md](README.md) for full documentation
- Explore the [examples](examples/) directory
- Join our community discussions
- Contribute to the project

## Getting Help

If you run into issues:

1. Check this guide and the main README
2. Search existing GitHub Issues
3. Ask in GitHub Discussions
4. Open a new issue with:
   - Your Morpheus version (`morpheus version`)
   - Error messages (full output)
   - Steps to reproduce

Happy forest planting! 🌲
