# Morpheus 🌲

[![Build Status](https://github.com/nimsforest/morpheus/workflows/Build%20and%20Test/badge.svg)](https://github.com/nimsforest/morpheus/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-66.4%25-yellow)](https://github.com/nimsforest/morpheus/actions)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

**Infrastructure provisioning tool for Nims Forest** - Automatically provision cloud servers with the right configuration for NATS-based distributed systems.

## Getting Started

### 📱 Run Morpheus from Your Android Phone

**Step 1: Install F-Droid (if you don't have it yet)**

F-Droid is an open-source app store for Android. Get it from:
- **Website:** https://f-droid.org/
- Tap "Download F-Droid" and install the APK

**Step 2: Install Termux from F-Droid**

⚠️ **Important:** Do NOT use Google Play Store (it's outdated)

1. Open F-Droid app
2. Search for "Termux"
3. Install Termux

Or direct link: https://f-droid.org/en/packages/com.termux/

**Step 3: Get Your Hetzner API Token**

Get your token from [Hetzner Cloud Console](https://console.hetzner.cloud/). See [Get Hetzner API Token](#get-hetzner-api-token) below for detailed instructions.

**Step 4: Install Morpheus (One Command!)**

Open Termux and paste this (optionally with your token):

```bash
# Option 1: With token (recommended)
export HETZNER_API_TOKEN="your_token_here"
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# Option 2: Set token later
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

**Step 5: Use Morpheus**

```bash
morpheus plant cloud wood    # Create infrastructure
morpheus list                # View forests
morpheus status forest-123   # Check status
morpheus teardown forest-123 # Clean up
```

📖 **Full guide:** [Termux Quick Start](docs/TERMUX_QUICKSTART.md)

---

### 💻 Desktop/Laptop

```bash
# Clone and build
git clone https://github.com/nimsforest/morpheus.git
cd morpheus
make build

# Configure
export HETZNER_API_TOKEN="your-token"
cp config.example.yaml config.yaml

# Use it
./bin/morpheus plant cloud wood
./bin/morpheus list
```

---

## What Does Morpheus Do?

Morpheus handles **infrastructure only**:
- ✅ Provision cloud servers (Hetzner, AWS, GCP, etc.)
- ✅ Configure OS, networking, and firewalls
- ✅ Prepare directories and storage
- ✅ Hand off to NimsForest for application setup

**Morpheus does NOT install NATS** - that's [NimsForest's](https://github.com/yourusername/nimsforest) responsibility.

## Installation

### Prerequisites

**For Termux:**
- Android phone with [Termux from F-Droid](https://f-droid.org/en/packages/com.termux/)
- Hetzner Cloud account (free to create)
- *Go is automatically installed by the installer*

**For Desktop:**
- Go 1.25+ (or whatever version you have)
- Hetzner Cloud account with API token
- SSH key (Morpheus will automatically upload it to Hetzner if not already there)

### Build from Source (Desktop)

```bash
git clone https://github.com/nimsforest/morpheus.git
cd morpheus
make build    # Build binary
make install  # Install to /usr/local/bin (optional)
```

**Note:** Termux users should use the automated installer instead.

### Mobile Usage

**Primary: Run directly on Android/Termux**

Morpheus is a CLI tool. Termux is a terminal. Running it directly is the natural way:

```bash
# Quick install (non-interactive)
export HETZNER_API_TOKEN="your_token"
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

- ✅ Free (no server costs)
- ✅ Native CLI experience
- ✅ Full functionality from your pocket
- ✅ Works offline for local commands
- ✅ Fully automated installation

**Alternative: Control Server** (only for specific needs)

Use a control server only if you need:
- 24/7 persistent environment (CI/CD, automation)
- Team collaboration (shared instance)
- Long-running operations (phone can't stay on)

See: [Control Server Guide](docs/CONTROL_SERVER_SETUP.md)

### Get Hetzner API Token

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Go to Security → API Tokens
3. Click "Generate API Token"
4. Set permissions to "Read & Write"
5. Copy the token

### SSH Key Setup

**Automatic Upload (Recommended):**

Morpheus automatically uploads your SSH key to Hetzner Cloud if it doesn't exist:

1. Make sure you have an SSH key at `~/.ssh/id_ed25519.pub` (or other common locations)
2. Configure the key name in `config.yaml` (e.g., `ssh_key: main`)
3. When you provision your first server, Morpheus will automatically upload the key if needed

**Manual Upload (Optional):**

If you prefer to upload manually:

```bash
# Via CLI
hcloud ssh-key create --name main --public-key-from-file ~/.ssh/id_ed25519.pub

# Or via console: Security → SSH Keys → Add SSH Key
```

**Custom Key Path:**

You can specify a custom SSH key path in your config:

```yaml
infrastructure:
  defaults:
    ssh_key: main
    ssh_key_path: "~/.ssh/custom_key.pub"  # Optional
```

## Configuration

Create `~/.morpheus/config.yaml` or `./config.yaml`:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31       # 4 vCPU, 8 GB RAM
    image: ubuntu-24.04
    ssh_key: main            # SSH key name (auto-uploaded if not found)
    ssh_key_path: ""         # Optional: custom path to local SSH public key
  locations:
    - fsn1  # Falkenstein, Germany
    - nbg1  # Nuremberg, Germany
    - hel1  # Helsinki, Finland

integration:
  nimsforest_url: "https://nimsforest.example.com"  # Optional: NimsForest callback
  registry_url: ""  # Optional: Morpheus registry

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"  # Or set directly
```

**Server Types:**
- `cpx11`: 2 vCPU, 2 GB RAM (~€4.50/mo) - Testing
- `cpx21`: 3 vCPU, 4 GB RAM (~€9/mo) - Small production
- `cpx31`: 4 vCPU, 8 GB RAM (~€18/mo) - **Recommended**
- `cpx41`: 8 vCPU, 16 GB RAM (~€36/mo) - High load
- `cpx51`: 16 vCPU, 32 GB RAM (~€72/mo) - Enterprise

## Commands

All commands work the same on Desktop and Termux!

**Quick Reference:**
- `plant cloud <size>` - Create infrastructure
- `list` - View all forests
- `status <id>` - Check forest details
- `teardown <id>` - Delete forest
- `update` - Update to latest version
- `version` - Show current version
- `help` - Show help

### Plant a Forest

```bash
morpheus plant cloud <size>
```

**Sizes:**
- `wood` - 1 node (single server)
- `forest` - 3 nodes (cluster)
- `jungle` - 5 nodes (large cluster)

**Examples:**

```bash
morpheus plant cloud wood     # 1 node, ~5-10 min
morpheus plant cloud forest   # 3 nodes, ~15-30 min
morpheus plant cloud jungle   # 5 nodes, ~25-50 min
```

**What happens:**
1. Creates Hetzner servers
2. Configures OS (Ubuntu 24.04)
3. Sets up firewall (ports 22, 4222, 6222, 8222, 7777)
4. Installs Docker
5. Creates directories (`/opt/nimsforest`, `/var/lib/nimsforest`)
6. Writes metadata to `/etc/morpheus/node-info.json`
7. Calls NimsForest (if configured)
8. Status: `infrastructure_ready`

### List Forests

```bash
morpheus list
```

Output:
```
FOREST ID            SIZE    LOCATION  STATUS       CREATED
----------------------------------------------------------------------------
forest-1735234567    forest  fsn1      active       2025-12-26 10:30:00
forest-1735234890    wood    nbg1      active       2025-12-26 11:15:00
```

### Check Status

```bash
morpheus status forest-<id>
```

Output:
```
🌲 Forest: forest-1735234567
Size: forest
Location: fsn1
Provider: hetzner
Status: active
Created: 2025-12-26 10:30:00

Nodes (3):
ID        ROLE   IP             LOCATION  STATUS
-----------------------------------------------------------
12345678  edge   95.217.123.45  fsn1      active
12345679  edge   95.217.123.46  fsn1      active
12345680  edge   95.217.123.47  fsn1      active
```

### Teardown

```bash
morpheus teardown forest-<id>
```

Deletes all servers and cleans up resources.

### Update Morpheus

**Automatic update (recommended):**

```bash
morpheus update        # Check for updates and install if available
morpheus check-update  # Just check for updates without installing
```

The update command will:
1. Check GitHub for the latest release
2. Show release notes
3. Ask for confirmation
4. Download and install the pre-built binary for your platform
5. Back up your current version to `<path>.backup`

**Manual update with pre-built binaries:**

If automatic update fails, you can download binaries directly:

```bash
# Example: Download latest Linux ARM64 binary
gh release download --pattern 'morpheus-linux-arm64'
chmod +x morpheus-linux-arm64
sudo mv morpheus-linux-arm64 /usr/local/bin/morpheus

# Or with curl
curl -LO https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm64
chmod +x morpheus-linux-arm64
sudo mv morpheus-linux-arm64 /usr/local/bin/morpheus
```

**Manual update by building from source (if needed):**

```bash
# Desktop/Laptop
cd morpheus
git pull
make build
sudo make install

# Termux
cd ~/morpheus
git pull
make build
cp bin/morpheus $PREFIX/bin/
```

**Available pre-built binaries:**

Every release includes binaries for:
- Linux (amd64, arm64, arm)
- macOS (amd64, arm64)

Download from: https://github.com/nimsforest/morpheus/releases

The `morpheus update` command automatically downloads the correct binary for your platform!

### Other Commands

```bash
morpheus version  # Show version
morpheus help     # Show help
```

## Architecture

### Separation of Concerns

| Concern | Morpheus | NimsForest |
|---------|----------|------------|
| Server provisioning | ✅ | |
| OS & network setup | ✅ | |
| Firewall config | ✅ | |
| NATS installation | | ✅ |
| NATS clustering | | ✅ |
| Service orchestration | | ✅ |

**Morpheus** = Infrastructure as Code  
**NimsForest** = Application Orchestration

See [docs/SEPARATION_OF_CONCERNS.md](docs/SEPARATION_OF_CONCERNS.md) for details.

### Integration Flow

```
1. morpheus plant cloud forest
   ↓
2. Provision servers (Hetzner API)
   ↓
3. Cloud-init: OS setup, firewall, directories
   ↓
4. Status: infrastructure_ready
   ↓
5. Callback to NimsForest (optional)
   ↓
6. NimsForest installs NATS
   ↓
7. Status: active
```

### Node Metadata

Morpheus writes `/etc/morpheus/node-info.json`:

```json
{
  "forest_id": "forest-1735234567",
  "role": "edge",
  "provisioner": "morpheus",
  "provisioned_at": "2025-12-26T10:30:00Z",
  "registry_url": "",
  "callback_url": ""
}
```

NimsForest reads this file to bootstrap the application.

## Testing

```bash
make test               # Run all tests
make test-cover         # Show coverage
make test-coverage      # Generate HTML report
```

**Coverage: 66.4%**
- pkg/config: 100%
- pkg/cloudinit: 86.7%
- pkg/forest: 50.7%
- pkg/provider/hetzner: 28.0%

## Development

```bash
make build    # Build binary
make fmt      # Format code
make vet      # Run linters
make clean    # Clean artifacts
```

## Troubleshooting

### "SSH key not found: main"

**This should rarely happen now** - Morpheus automatically uploads SSH keys!

If you still see this error:

```bash
# Check if local SSH key exists
ls -la ~/.ssh/*.pub

# Generate one if missing
ssh-keygen -t ed25519 -C "your_email@example.com"

# List keys in Hetzner (if you want to verify)
hcloud ssh-key list

# Update config.yaml to match exact key name
```

### "Failed to load config"

```bash
# Create config
mkdir -p ~/.morpheus
cp config.example.yaml ~/.morpheus/config.yaml
# Edit with your token
```

### "timeout waiting for server"

Check:
- Hetzner Cloud console for server status
- Resource limits on your account
- Network connectivity

### Cloud-init not completing

```bash
# SSH to server
ssh root@<server-ip>

# Check cloud-init status
cloud-init status

# View logs
tail -f /var/log/cloud-init-output.log
```

## FAQ

**Q: Does Morpheus install NATS?**  
A: No. Morpheus only provisions infrastructure. NimsForest installs NATS.

**Q: Can I use Morpheus without NimsForest?**  
A: Yes! Set `integration.nimsforest_url: ""` and handle application setup manually.

**Q: What cloud providers are supported?**  
A: Currently Hetzner Cloud. AWS, GCP, Azure coming in future releases.

**Q: How much does it cost?**  
A: Hetzner charges by the minute. Example with cpx31:
- wood (1 node): ~€18/month
- forest (3 nodes): ~€54/month
- jungle (5 nodes): ~€90/month

**Q: Can I change forest size after creation?**  
A: Not yet. You need to teardown and recreate. Auto-scaling is planned.

**Q: Is my API token secure?**  
A: Use environment variables (`HETZNER_API_TOKEN`) and never commit config files with tokens.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Quick tips:
- Follow Go best practices
- Add tests for new features
- Update documentation
- Run `make lint` before committing

## Roadmap

- [ ] Multi-cloud support (AWS, GCP, Azure)
- [ ] Auto-scaling
- [ ] Built-in monitoring
- [ ] Private networks
- [ ] Load balancer integration
- [ ] Backup/restore

## License

MIT License - see [LICENSE](LICENSE)

## Links

- **GitHub**: https://github.com/nimsforest/morpheus
- **Documentation**: [docs/](docs/)
  - [Termux Quick Start](docs/TERMUX_QUICKSTART.md) - 10-minute setup with one command
  - [Android/Termux Guide](docs/ANDROID_TERMUX.md) - Complete guide (primary mobile approach)
  - [Mobile Philosophy](docs/MOBILE_PHILOSOPHY.md) - Why Termux is the natural way for CLI tools
  - [Control Server Setup](docs/CONTROL_SERVER_SETUP.md) - Alternative for 24/7, teams, CI/CD
  - [Separation of Concerns](docs/SEPARATION_OF_CONCERNS.md) - Architecture details
- **Issues**: https://github.com/nimsforest/morpheus/issues
- **NimsForest**: https://github.com/yourusername/nimsforest

---

**Status**: Production Ready ✅  
**Version**: 1.1.0  
**Last Updated**: December 26, 2025
