# IPv6 Setup

Morpheus uses IPv6-only by default. Hetzner charges extra for IPv4.

## Requirements

**Check your network connectivity:**
```bash
# Test both IPv6 and IPv4 connectivity
morpheus check network

# Or test IPv6 specifically
morpheus check ipv6

# Or test IPv4 specifically
morpheus check ipv4
```

## Configuration

**Default (IPv6-only):**
```yaml
infrastructure:
  provider: hetzner
  # IPv6-only by default (no IPv4 allocated)
```

**With IPv4 Fallback:**
```yaml
infrastructure:
  provider: hetzner
  enable_ipv4_fallback: true  # Enable IPv4 addresses (costs extra)
```

When IPv4 fallback is enabled:
- Servers get both IPv4 and IPv6 addresses
- Morpheus tries IPv6 first, falls back to IPv4 if unreachable
- Additional cost for IPv4 addresses on Hetzner

## SSH Connections

**IPv6:**
```bash
ssh root@2001:db8::1
```

**IPv4:**
```bash
ssh root@203.0.113.1
```

Morpheus will show you the appropriate IP addresses after provisioning.

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

**Go (IPv6):**
```go
nc, err := nats.Connect("nats://[2001:db8::1]:4222")
```

**Go (IPv4):**
```go
nc, err := nats.Connect("nats://203.0.113.1:4222")
```

**CLI:**
```bash
nats pub -s nats://[2001:db8::1]:4222 test "Hello"
```

**Note:** IPv6 addresses require brackets in URLs.

## Troubleshooting

**If you don't have IPv6:**

Option 1: Enable IPv4 fallback (costs extra)
```yaml
infrastructure:
  enable_ipv4_fallback: true
```

Option 2: Get IPv6 connectivity
1. Enable IPv6 on your ISP/network
2. Use IPv6 tunnel (e.g., Hurricane Electric - free)
3. Use a VPS with IPv6 to run Morpheus

**Test connectivity to Hetzner server:**
```bash
# IPv6
ping6 2001:db8::1
ssh -6 root@2001:db8::1

# IPv4 (if fallback enabled)
ping 203.0.113.1
ssh root@203.0.113.1
```
