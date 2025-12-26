# Frequently Asked Questions

## General Questions

### What is Morpheus?

Morpheus is an automated infrastructure provisioning tool that helps you create and manage NATS-based distributed systems (called "forests") on cloud providers like Hetzner Cloud.

### Why would I use Morpheus?

Morpheus eliminates the manual steps of:
- Creating servers via console or CLI
- SSHing into each server to configure
- Installing and configuring NATS
- Setting up clustering
- Managing firewall rules

With Morpheus, all of this happens automatically with a single command.

### What does "planting a forest" mean?

In Morpheus terminology, a "forest" is a cluster of NATS servers. "Planting" means creating and provisioning the infrastructure for this cluster.

## Installation & Setup

### What are the prerequisites?

- Go 1.24 or higher
- A Hetzner Cloud account
- An API token with read/write permissions
- At least one SSH key uploaded to Hetzner Cloud

### How do I get a Hetzner API token?

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Go to your project
3. Navigate to Security → API Tokens
4. Click "Generate API Token"
5. Give it a name and set permissions to "Read & Write"
6. Copy the token (you won't see it again!)

### Where should I store my API token?

You have two options:
1. In the config file at `~/.morpheus/config.yaml` (remember to exclude from git)
2. As an environment variable: `export HETZNER_API_TOKEN="your-token"`

We recommend using environment variables for better security.

### How do I upload my SSH key to Hetzner?

Via CLI:
```bash
hcloud ssh-key create --name main --public-key-from-file ~/.ssh/id_ed25519.pub
```

Or via the web console: Security → SSH Keys → Add SSH Key

## Usage

### What forest sizes are available?

- **wood**: 1 node (single NATS server)
- **forest**: 3 nodes (NATS cluster)
- **jungle**: 5 nodes (large cluster)

### How much does it cost?

It depends on the server type and number of nodes. For example, with `cpx31` (€18/month):
- wood: ~€18/month (1 server)
- forest: ~€54/month (3 servers)
- jungle: ~€90/month (5 servers)

Prices are prorated to the minute, so you only pay for what you use.

### How long does provisioning take?

- Server creation: 1-2 minutes per node
- Cloud-init bootstrap: 2-5 minutes per node
- **Total**: 5-15 minutes depending on forest size

### Can I provision multiple forests?

Yes! Each forest gets a unique ID and is tracked independently in the registry.

### How do I connect to my NATS servers?

After provisioning, use `morpheus status <forest-id>` to get the IP addresses, then:

```bash
# SSH into a node
ssh root@<node-ip>

# Check NATS status
systemctl status nats-server

# Access NATS HTTP monitoring
curl http://localhost:8222/varz
```

### Can I choose the data center location?

Yes! Edit your `config.yaml` and specify locations:

```yaml
infrastructure:
  locations:
    - fsn1  # Falkenstein, Germany
    - nbg1  # Nuremberg, Germany
    - hel1  # Helsinki, Finland
```

The first location in the list will be used.

### How do I delete a forest?

```bash
morpheus teardown <forest-id>
```

This will delete all servers and clean up resources.

### What happens if provisioning fails?

Morpheus automatically rolls back:
- Deletes any partially provisioned servers
- Cleans up registry entries
- Reports the error for debugging

You won't be left with orphaned resources.

## Troubleshooting

### "Failed to load config: no such file or directory"

Create a config file:
```bash
mkdir -p ~/.morpheus
cp config.example.yaml ~/.morpheus/config.yaml
# Edit with your settings
```

### "SSH key not found: main"

The SSH key name in your config doesn't match what's in Hetzner. Check with:
```bash
hcloud ssh-key list
```

Update your config to match the exact name.

### "Failed to create server: authentication failed"

Your API token is invalid or expired. Generate a new one and update your config or environment variable.

### "timeout waiting for server to reach state: running"

This usually means:
- Server creation failed on Hetzner's side
- Network issues
- Resource limits reached

Check the Hetzner Cloud console for details.

### Cloud-init didn't complete

SSH into the server and check:
```bash
ssh root@<server-ip>
cloud-init status          # Check status
tail -f /var/log/cloud-init-output.log  # View logs
```

### How do I debug provisioning issues?

1. Check Morpheus output for error messages
2. Verify your config is correct
3. Check Hetzner Cloud console for server status
4. SSH into the server and check cloud-init logs
5. Verify firewall rules aren't blocking access

## Configuration

### What server types are recommended?

For production:
- **cpx31**: 4 vCPU, 8 GB RAM (recommended)
- **cpx41**: 8 vCPU, 16 GB RAM (high load)

For testing:
- **cpx11**: 2 vCPU, 2 GB RAM (economical)
- **cpx21**: 3 vCPU, 4 GB RAM (small production)

### Can I use a different OS image?

Currently, Morpheus is optimized for Ubuntu 24.04. Other Ubuntu versions may work but are untested.

### Can I customize the cloud-init script?

Yes! Edit the templates in `pkg/cloudinit/templates.go` and rebuild Morpheus.

### Can I use private networks?

Not yet, but it's on the roadmap! Currently, all communication uses public IPs.

## Advanced Usage

### Can I provision nodes in multiple locations?

Not in a single forest currently. Each forest is provisioned in one location. You can create multiple forests in different locations.

### Can I add nodes to an existing forest?

Not yet. You need to teardown and recreate with a larger size. Auto-scaling is planned for a future release.

### Can I use Morpheus with other cloud providers?

Currently only Hetzner Cloud is supported. AWS, GCP, Azure, and OVH support is planned.

### Can I backup my forests?

Currently, you need to manually backup:
- NATS data directories
- Configuration files
- Application data

Automated backup is planned for a future release.

### Can I monitor my forests?

NATS exposes monitoring at `http://<node-ip>:8222`:
- `/varz`: Server info
- `/connz`: Connections
- `/routez`: Routes
- `/subsz`: Subscriptions

Built-in monitoring is planned for a future release.

## Development

### How can I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed guidelines.

### How do I build from source?

```bash
git clone https://github.com/yourusername/morpheus.git
cd morpheus
make build
```

### How do I run tests?

```bash
make test
```

### Can I add support for another cloud provider?

Yes! See the [Architecture docs](ARCHITECTURE.md) for details on implementing the Provider interface.

## Security

### Is it safe to store API tokens in config files?

It's not recommended. Use environment variables instead and never commit config files with tokens to version control.

### Are the servers secure?

Cloud-init configures basic security:
- UFW firewall enabled
- Only necessary ports open (22, 4222, 6222, 8222)
- Root SSH access (you should disable this after setup)

Always follow security best practices:
- Change default passwords
- Enable fail2ban
- Use private networks when available
- Keep systems updated

### Does Morpheus support TLS/SSL?

Not yet. NATS TLS configuration is planned for a future release.

## Pricing & Billing

### How am I billed?

Hetzner bills by the hour (prorated to the minute). You're charged from server creation until deletion.

### Can I use spot/preemptible instances?

Not yet. Support for cost-optimized instances is planned.

### What about data transfer costs?

Hetzner includes 20 TB of traffic per month. Additional traffic costs €1.19/TB.

## Support

### Where can I get help?

1. Read this FAQ and [SETUP.md](../SETUP.md)
2. Check [GitHub Issues](https://github.com/yourusername/morpheus/issues)
3. Ask in [GitHub Discussions](https://github.com/yourusername/morpheus/discussions)
4. Email: support@example.com

### How do I report a bug?

Open an issue with:
- Morpheus version (`morpheus version`)
- Full error message
- Steps to reproduce
- Your environment (OS, Go version)

### Is there a community?

Check out our [GitHub Discussions](https://github.com/yourusername/morpheus/discussions) to connect with other users!

## Roadmap

### What features are planned?

See [CHANGELOG.md](../CHANGELOG.md) for planned features:
- Multi-cloud support
- Auto-scaling
- Built-in monitoring
- Web UI
- Backup/restore
- Private networks
- TLS support

### When will feature X be released?

We don't have fixed release dates. Follow the project on GitHub for updates!

### Can I request a feature?

Yes! Open a discussion or issue on GitHub with your suggestion.
