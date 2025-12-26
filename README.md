# Morpheus 🌲

Morpheus is an automated infrastructure provisioning tool designed to help users and organizations plant and grow their Nims Forest. It seamlessly integrates with cloud providers to automatically provision, configure, and manage NATS-based distributed systems.

## Features

- **🚀 Automated Provisioning**: Automatically create and configure cloud servers with a single command
- **☁️ Hetzner Cloud Integration**: Native support for Hetzner Cloud with the official `hcloud-go/v2` client
- **🔧 Cloud-Init Bootstrap**: Automatic server configuration with NATS, Docker, and system dependencies
- **📊 Forest Registry**: Track and manage all your forests and nodes in a centralized registry
- **🗺️ Multi-Location Support**: Deploy across multiple data centers (fsn1, nbg1, hel1)
- **💾 Graceful Rollback**: Automatic cleanup on provisioning failures
- **🔥 Easy Teardown**: Remove entire forests and all associated resources with one command

## Installation

### Prerequisites

- Go 1.21 or higher
- Hetzner Cloud account with API token
- SSH key uploaded to Hetzner Cloud

### Build from Source

```bash
git clone https://github.com/yourusername/morpheus.git
cd morpheus
make build
```

This will create the `morpheus` binary in the `bin/` directory.

### Install Globally

```bash
make install
```

This will install `morpheus` to `/usr/local/bin/`.

## Configuration

### Setup

Create a configuration file at `~/.morpheus/config.yaml` or in your project directory as `config.yaml`:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main
  locations:
    - fsn1
    - nbg1
    - hel1

secrets:
  hetzner_api_token: "your-hetzner-api-token-here"
```

Alternatively, set the API token via environment variable:

```bash
export HETZNER_API_TOKEN="your-token-here"
```

### Available Server Types

Common Hetzner server types:
- `cpx11`: 2 vCPU, 2 GB RAM (economical)
- `cpx21`: 3 vCPU, 4 GB RAM
- `cpx31`: 4 vCPU, 8 GB RAM (recommended)
- `cpx41`: 8 vCPU, 16 GB RAM
- `cpx51`: 16 vCPU, 32 GB RAM

### Locations

Available Hetzner locations:
- `fsn1`: Falkenstein, Germany
- `nbg1`: Nuremberg, Germany
- `hel1`: Helsinki, Finland

## Usage

### Plant a Forest

Create a new Nims Forest in the cloud:

```bash
morpheus plant cloud <size>
```

**Size options:**
- `wood`: Single NATS server (1 node)
- `forest`: NATS cluster (3 nodes)
- `jungle`: Large NATS cluster (5 nodes)

**Examples:**

```bash
# Plant a small forest with 1 node
morpheus plant cloud wood

# Plant a medium forest with 3 nodes
morpheus plant cloud forest

# Plant a large forest with 5 nodes
morpheus plant cloud jungle
```

### List Forests

View all your provisioned forests:

```bash
morpheus list
```

Output:
```
FOREST ID            SIZE       LOCATION        STATUS       CREATED             
--------------------------------------------------------------------------------
forest-1735234567    forest     fsn1            active       2025-12-26 10:30:00
forest-1735234890    wood       nbg1            active       2025-12-26 11:15:00
```

### Check Forest Status

Get detailed information about a specific forest:

```bash
morpheus status <forest-id>
```

Example:
```bash
morpheus status forest-1735234567
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
ID              ROLE       IP                   LOCATION        STATUS      
------------------------------------------------------------------------
12345678        edge       95.217.123.45        fsn1            active
12345679        edge       95.217.123.46        fsn1            active
12345680        edge       95.217.123.47        fsn1            active
```

### Teardown a Forest

Remove a forest and all its resources:

```bash
morpheus teardown <forest-id>
```

Example:
```bash
morpheus teardown forest-1735234567
```

This will:
- Delete all servers associated with the forest
- Remove entries from the registry
- Clean up all Hetzner Cloud resources

### Version Information

```bash
morpheus version
```

### Help

```bash
morpheus help
```

## How It Works

### Provisioning Flow

1. **Forest Creation Request**: You run `morpheus plant cloud forest`
2. **API Communication**: Morpheus calls the Hetzner Cloud API to create servers
3. **Server Configuration**: Each server is created with:
   - Predefined server type (from config)
   - Ubuntu 24.04 base image
   - Your SSH key pre-installed
   - Cloud-init script for bootstrap
4. **Polling**: Morpheus polls until server status is `running`
5. **Cloud-Init Bootstrap**: The cloud-init script automatically:
   - Updates packages
   - Installs NATS server and dependencies
   - Configures NATS clustering
   - Sets up firewall rules
   - Registers node in the forest registry
6. **Registration**: Node is registered with IP, capacity, and location metadata
7. **Completion**: Forest is marked as `active` and ready to use

### Cloud-Init Templates

Morpheus includes cloud-init templates for different node roles:

- **Edge Nodes**: NATS server with JetStream and clustering
- **Compute Nodes**: Docker-based compute workers
- **Storage Nodes**: NFS-based distributed storage

### Rollback on Failure

If provisioning fails at any step, Morpheus automatically:
- Deletes all partially provisioned servers
- Cleans up the registry
- Reports the error for debugging

## Architecture

```
morpheus/
├── cmd/
│   └── morpheus/          # Main CLI application
│       └── main.go
├── pkg/
│   ├── config/            # Configuration management
│   │   └── config.go
│   ├── cloudinit/         # Cloud-init templates
│   │   └── templates.go
│   ├── forest/            # Forest registry and provisioning
│   │   ├── registry.go
│   │   └── provisioner.go
│   └── provider/          # Cloud provider abstractions
│       ├── interface.go
│       └── hetzner/
│           └── hetzner.go
├── config.example.yaml    # Example configuration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Lint

```bash
make lint
```

### Clean

```bash
make clean
```

## Security Best Practices

1. **Never commit API tokens**: Use environment variables or keep `config.yaml` out of version control
2. **Restrict SSH access**: Only allow SSH from trusted IPs
3. **Use strong passwords**: For NATS admin accounts (configurable in cloud-init)
4. **Enable firewall**: Cloud-init automatically configures UFW
5. **Keep systems updated**: Regularly update server packages

## Troubleshooting

### "Failed to create server: SSH key not found"

Make sure you've uploaded your SSH key to Hetzner Cloud and the name in `config.yaml` matches exactly.

```bash
# List your SSH keys
hcloud ssh-key list
```

### "Failed to load config: no such file or directory"

Create a config file at `~/.morpheus/config.yaml` or `./config.yaml` using the example:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your settings
```

### "timeout waiting for server to reach state: running"

This usually means:
- The server creation failed on Hetzner's side
- Network issues preventing status checks
- Resource limits reached on your account

Check the Hetzner Cloud console for more details.

### Cloud-init didn't complete

SSH into the server and check cloud-init logs:

```bash
ssh root@<server-ip>
tail -f /var/log/cloud-init-output.log
```

## Roadmap

- [ ] Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- [ ] Spot/preemptible instances for cost optimization
- [ ] Auto-scaling based on forest load
- [ ] Built-in monitoring and health checks
- [ ] Web UI for forest management
- [ ] Backup and disaster recovery
- [ ] Support for on-premises deployments

## Contributing

We welcome contributions! To contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Support

- 🐛 Report bugs via [GitHub Issues](https://github.com/yourusername/morpheus/issues)
- 💬 Join discussions in [GitHub Discussions](https://github.com/yourusername/morpheus/discussions)
- 📧 Email: support@example.com

---

**Note**: Morpheus is under active development. APIs and features may change.
