# IPv6 Support

Morpheus uses IPv6 by default. All Hetzner servers get both IPv4 and IPv6 addresses.

## Configuration

**Default (IPv6-first):**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: true  # Default - uses IPv6, falls back to IPv4
```

**IPv4-only (if your network doesn't support IPv6):**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: false  # Use IPv4 instead
```

## SSH with IPv6

```bash
# IPv4
ssh root@95.217.0.1

# IPv6
ssh root@2001:db8::1
```

## NATS with IPv6

```conf
# Listen on all interfaces (IPv4 + IPv6)
port: 4222

# Or explicitly IPv6
listen: "[::]:4222"

# Cluster with IPv6
cluster {
  name: "forest-123"
  listen: "[::]:6222"
  routes: [
    "nats://[2001:db8::1]:6222"
    "nats://[2001:db8::2]:6222"
  ]
}
```

## Client Connections

**Go:**
```go
nc, err := nats.Connect("nats://[2001:db8::1]:4222")
```

**CLI:**
```bash
nats pub -s nats://[2001:db8::1]:4222 test "Hello"
```

**Note:** IPv6 addresses require brackets in URLs.

## Troubleshooting

**Test your IPv6 connectivity:**
```bash
curl -6 ifconfig.co
ping6 2001:db8::1
ssh -6 root@2001:db8::1
```

**If provisioning fails (your network doesn't have IPv6):**
```yaml
# Use IPv4 instead
infrastructure:
  defaults:
    prefer_ipv6: false
```

**Test your IPv6:**
```bash
curl -6 ifconfig.co
# If this works, you have IPv6
```
