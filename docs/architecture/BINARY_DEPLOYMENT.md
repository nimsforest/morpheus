# Binary Deployment Architecture

**Philosophy:** Deploy Go compiled binaries directly, not Docker containers.

## Why Direct Binary Deployment?

Morpheus provisions cloud VMs that are already isolated. Docker adds unnecessary overhead:

- ❌ **Extra daemon** consuming resources
- ❌ **Container layer** slowing down startup
- ❌ **Additional attack surface**
- ❌ **Complexity** in logging and debugging
- ❌ **No real benefit** - you already have VM isolation

Instead, we deploy Go binaries directly:

- ✅ **Simple** - Download binary, run it
- ✅ **Fast** - No container overhead
- ✅ **Lightweight** - No Docker daemon
- ✅ **Native** - Direct systemd integration
- ✅ **Debuggable** - Standard Linux processes
- ✅ **Go philosophy** - "Compile once, run anywhere"

## Directory Structure

Morpheus creates this structure on each node:

```
/opt/nimsforest/
  bin/              # Go binaries (nats-server, etc.)
  
/var/lib/nimsforest/
  data/             # NATS data files
  jetstream/        # JetStream storage
  
/var/log/nimsforest/
  nats.log          # NATS logs
  
/etc/nimsforest/
  nats.conf         # NATS configuration
  cluster.conf      # Cluster configuration
```

## Deployment Pattern

NimsForest should follow this pattern:

### 1. Download Binary

```bash
# Download NATS server binary
NATS_VERSION="v2.10.7"
wget https://github.com/nats-io/nats-server/releases/download/${NATS_VERSION}/nats-server-${NATS_VERSION}-linux-amd64.tar.gz

# Extract to /opt/nimsforest/bin
tar -xzf nats-server-${NATS_VERSION}-linux-amd64.tar.gz
mv nats-server-${NATS_VERSION}-linux-amd64/nats-server /opt/nimsforest/bin/
chmod +x /opt/nimsforest/bin/nats-server
```

### 2. Create Configuration

```bash
cat > /etc/nimsforest/nats.conf << 'EOF'
# NATS Server Configuration
port: 4222
http_port: 8222

# Logging
log_file: "/var/log/nimsforest/nats.log"
logtime: true
debug: false
trace: false

# JetStream
jetstream {
  store_dir: "/var/lib/nimsforest/jetstream"
  max_memory_store: 1GB
  max_file_store: 10GB
}

# Cluster configuration (if multi-node)
cluster {
  name: "nimsforest"
  port: 6222
  routes: [
    nats://node1:6222
    nats://node2:6222
    nats://node3:6222
  ]
}

# Leafnode configuration (optional)
leafnodes {
  port: 7777
}
EOF
```

### 3. Create Systemd Service

```bash
cat > /etc/systemd/system/nats.service << 'EOF'
[Unit]
Description=NATS Server
After=network.target
Documentation=https://docs.nats.io

[Service]
Type=simple
User=ubuntu
Group=ubuntu

# Binary and config
ExecStart=/opt/nimsforest/bin/nats-server -c /etc/nimsforest/nats.conf

# Restart policy
Restart=always
RestartSec=5s

# Limits
LimitNOFILE=65536
LimitNPROC=4096

# Logging
StandardOutput=append:/var/log/nimsforest/nats.log
StandardError=append:/var/log/nimsforest/nats.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/nimsforest /var/log/nimsforest

[Install]
WantedBy=multi-user.target
EOF
```

### 4. Start Service

```bash
# Reload systemd
systemctl daemon-reload

# Enable and start
systemctl enable nats
systemctl start nats

# Check status
systemctl status nats
```

## NimsForest Bootstrap Script Example

Here's what NimsForest should do when Morpheus calls it:

```bash
#!/bin/bash
# NimsForest bootstrap script
set -e

# Read node info from Morpheus
NODE_INFO="/etc/morpheus/node-info.json"
FOREST_ID=$(jq -r '.forest_id' $NODE_INFO)
ROLE=$(jq -r '.role' $NODE_INFO)

echo "Bootstrapping NimsForest node: $FOREST_ID (role: $ROLE)"

# 1. Download NATS binary
echo "Downloading NATS server..."
NATS_VERSION="v2.10.7"
wget -q https://github.com/nats-io/nats-server/releases/download/${NATS_VERSION}/nats-server-${NATS_VERSION}-linux-amd64.tar.gz
tar -xzf nats-server-${NATS_VERSION}-linux-amd64.tar.gz
mv nats-server-${NATS_VERSION}-linux-amd64/nats-server /opt/nimsforest/bin/
chmod +x /opt/nimsforest/bin/nats-server
rm -rf nats-server-*.tar.gz nats-server-*

# 2. Generate NATS configuration
echo "Configuring NATS..."
cat > /etc/nimsforest/nats.conf << EOF
port: 4222
http_port: 8222
log_file: "/var/log/nimsforest/nats.log"

jetstream {
  store_dir: "/var/lib/nimsforest/jetstream"
}

cluster {
  name: "$FOREST_ID"
  port: 6222
}
EOF

# 3. Create systemd service
echo "Creating systemd service..."
cat > /etc/systemd/system/nats.service << 'SVCEOF'
[Unit]
Description=NATS Server
After=network.target

[Service]
Type=simple
User=ubuntu
ExecStart=/opt/nimsforest/bin/nats-server -c /etc/nimsforest/nats.conf
Restart=always
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SVCEOF

# 4. Start NATS
echo "Starting NATS..."
systemctl daemon-reload
systemctl enable nats
systemctl start nats

# 5. Verify
sleep 2
systemctl status nats

echo "✓ NimsForest bootstrap complete!"
```

## Advantages Over Docker

### Performance

```
Docker:
  VM boot → Docker daemon → Pull image → Create container → Start NATS
  Time: 2-5 minutes
  Memory: ~200MB (daemon) + ~100MB (container) = 300MB overhead

Direct Binary:
  VM boot → Download binary → Start NATS
  Time: 30-60 seconds
  Memory: ~10MB (just the binary)
```

### Simplicity

```bash
# Docker approach (complex)
docker pull nats:latest
docker run -d --name nats \
  -p 4222:4222 \
  -p 6222:6222 \
  -p 8222:8222 \
  -v /data:/data \
  -v /config:/config \
  nats:latest -c /config/nats.conf

# Direct binary (simple)
/opt/nimsforest/bin/nats-server -c /etc/nimsforest/nats.conf
```

### Debugging

```bash
# Docker (multi-step)
docker ps                    # Find container
docker logs nats             # View logs
docker exec -it nats sh      # Enter container
docker inspect nats          # Check config

# Direct binary (standard Linux)
systemctl status nats        # Check status
journalctl -u nats -f        # View logs
ps aux | grep nats           # See process
```

## Security

Both approaches can be secure, but direct binaries are simpler:

### Docker Security
- Container escape vulnerabilities
- Docker daemon vulnerabilities
- Image supply chain attacks
- Complex networking (iptables, bridge, overlay)

### Direct Binary Security
- Standard Linux security (AppArmor, SELinux)
- Systemd sandboxing (ProtectSystem, PrivateTmp)
- Standard firewall (ufw)
- Simpler attack surface

## Resource Limits

Systemd provides resource controls without Docker:

```ini
[Service]
# CPU limit
CPUQuota=200%

# Memory limit
MemoryLimit=2G
MemoryHigh=1.5G

# File descriptors
LimitNOFILE=65536

# Processes
LimitNPROC=4096
```

## Monitoring

Standard Linux tools work perfectly:

```bash
# Process monitoring
systemctl status nats
ps aux | grep nats

# Resource usage
top -p $(pgrep nats-server)
htop

# Logs
journalctl -u nats -f
tail -f /var/log/nimsforest/nats.log

# Network
netstat -tlnp | grep nats
ss -tlnp | grep nats
```

## When Docker WOULD Make Sense

Docker would be valuable if you needed:

- 🐳 **Multiple applications** with conflicting dependencies
- 🐳 **Language runtimes** (Node.js, Python apps with many dependencies)
- 🐳 **Complex stacks** (app + database + cache + queue)
- 🐳 **Developer environments** (local development with docker-compose)
- 🐳 **Multi-tenancy** (running untrusted code)

But for NATS (single Go binary, no dependencies), Docker is overkill.

## Conclusion

For Morpheus + NimsForest:

- ✅ **Morpheus** provisions infrastructure (VMs, firewall, directories)
- ✅ **NimsForest** deploys binaries (download, configure, systemd)
- ✅ **No Docker** - simpler, faster, lighter

This aligns with:
- Go's compile-once philosophy
- Linux's native process management
- Morpheus's clean separation of concerns
- Cloud-native simplicity

---

**Next Steps:**

1. Morpheus: Remove Docker from cloud-init ✅ (Done)
2. NimsForest: Implement binary deployment script
3. Test: Verify NATS cluster formation
4. Document: Update NimsForest bootstrap guide
