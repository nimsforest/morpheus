# IPv6 Setup

Morpheus uses IPv6-only by default. Hetzner charges extra for IPv4.

## Requirements

**Your local network must have IPv6:**
```bash
# Test IPv6 connectivity using Morpheus (recommended, works on all platforms including Termux)
morpheus check-ipv6

# Alternative: using curl (may not work on Termux due to certificate issues)
curl -6 ifconfig.co

# Should return your IPv6 address
# If it times out, you need to enable IPv6 on your network
```

## Configuration

**Default (IPv6-only):**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: true
    ipv6_only: true    # Default - no IPv4
```

## SSH with IPv6

```bash
ssh root@2001:db8::1
```

## NATS with IPv6

```conf
# Listen on IPv6
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

**If you don't have IPv6:**

You cannot use Morpheus without IPv6. Options:
1. Enable IPv6 on your ISP/network
2. Use IPv6 tunnel (e.g., Hurricane Electric)
3. Use a VPS with IPv6 to run Morpheus

**Test Hetzner server IPv6:**
```bash
ping6 2001:db8::1
ssh -6 root@2001:db8::1
```
