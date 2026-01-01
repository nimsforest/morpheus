# Binary Deployment

**Deploy Go binaries directly, not Docker containers.**

## Why

- Faster startup
- Lower memory usage
- Simpler debugging
- Cloud VMs already provide isolation

## Directory Structure

```
/opt/nimsforest/bin/         # Go binaries (nats-server)
/var/lib/nimsforest/         # Data storage
/var/log/nimsforest/         # Logs
/etc/nimsforest/             # Configuration
```

## Deployment Pattern

### 1. Download Binary

```bash
NATS_VERSION="v2.10.7"
wget https://github.com/nats-io/nats-server/releases/download/${NATS_VERSION}/nats-server-${NATS_VERSION}-linux-amd64.tar.gz
tar -xzf nats-server-*.tar.gz
mv nats-server-*/nats-server /opt/nimsforest/bin/
chmod +x /opt/nimsforest/bin/nats-server
```

### 2. Create Configuration

```bash
cat > /etc/nimsforest/nats.conf << 'EOF'
port: 4222
http_port: 8222
log_file: "/var/log/nimsforest/nats.log"

jetstream {
  store_dir: "/var/lib/nimsforest/jetstream"
}

cluster {
  name: "forest-123"
  port: 6222
}
EOF
```

### 3. Create Systemd Service

```bash
cat > /etc/systemd/system/nats.service << 'EOF'
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
EOF
```

### 4. Start Service

```bash
systemctl daemon-reload
systemctl enable nats
systemctl start nats
systemctl status nats
```

## Monitoring

```bash
# Process status
systemctl status nats

# Logs
journalctl -u nats -f

# Resource usage
ps aux | grep nats-server
```

## Docker Alternative

**Local mode only:** `morpheus plant local` uses Docker for isolated testing.

**Cloud mode:** Direct binaries (production).
