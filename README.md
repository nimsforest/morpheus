# Morpheus 🌲

[![Build Status](https://github.com/yourusername/morpheus/workflows/Build%20and%20Test/badge.svg)](https://github.com/yourusername/morpheus/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-66.4%25-yellow)](https://github.com/yourusername/morpheus/actions)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

**Infrastructure provisioning tool for Nims Forest** - Automatically provision cloud servers with the right configuration for NATS-based distributed systems.

## What Does Morpheus Do?

Morpheus handles **infrastructure only**:
- ✅ Provision cloud servers (Hetzner, AWS, GCP, etc.)
- ✅ Configure OS, networking, and firewalls
- ✅ Prepare directories and storage
- ✅ Hand off to NimsForest for application setup

**Morpheus does NOT install NATS** - that's [NimsForest's](https://github.com/yourusername/nimsforest) responsibility.

## Quick Start

```bash
# 1. Build
git clone https://github.com/yourusername/morpheus.git
cd morpheus
make build

# 2. Configure
export HETZNER_API_TOKEN="your-token"
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# 3. Plant a forest (1 node)
./bin/morpheus plant cloud wood

# 4. Check status
./bin/morpheus list
./bin/morpheus status forest-<id>

# 5. Teardown when done
./bin/morpheus teardown forest-<id>
```

## Installation

### Prerequisites
- Go 1.21+
- Hetzner Cloud account with API token
- SSH key uploaded to Hetzner Cloud

### Build from Source

```bash
git clone https://github.com/yourusername/morpheus.git
cd morpheus
make deps     # Download dependencies
make build    # Build binary
make install  # Install to /usr/local/bin (optional)
```

### Get Hetzner API Token

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Go to Security → API Tokens
3. Click "Generate API Token"
4. Set permissions to "Read & Write"
5. Copy the token

### Upload SSH Key

```bash
# Via CLI
hcloud ssh-key create --name main --public-key-from-file ~/.ssh/id_ed25519.pub

# Or via console: Security → SSH Keys → Add SSH Key
```

## Configuration

Create `~/.morpheus/config.yaml` or `./config.yaml`:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31       # 4 vCPU, 8 GB RAM
    image: ubuntu-24.04
    ssh_key: main            # Must match Hetzner SSH key name
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

## Usage

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

```bash
# List keys in Hetzner
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

- **GitHub**: https://github.com/yourusername/morpheus
- **Documentation**: [docs/](docs/)
- **Issues**: https://github.com/yourusername/morpheus/issues
- **NimsForest**: https://github.com/yourusername/nimsforest

---

**Status**: Production Ready ✅  
**Version**: 1.1.0  
**Last Updated**: December 26, 2025
