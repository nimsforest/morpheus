# IPv6 Configuration

## Default: IPv6-Only

```yaml
infrastructure:
  provider: hetzner
  # IPv6-only by default - no IPv4 allocated (saves costs)
```

**Behavior:** Only uses IPv6. If your network lacks IPv6, enable IPv4 fallback.

**Why:** IPv4 costs extra on Hetzner. IPv6-only saves money.

## Requirements

- Your network should have IPv6 (preferred)
- Test: `morpheus check` or `morpheus check network`
- Alternative: `curl -6 ifconfig.co` (may not work on Termux)

## IPv4 Fallback

If your network doesn't have IPv6 connectivity, you can enable IPv4 fallback:

```yaml
infrastructure:
  provider: hetzner
  enable_ipv4_fallback: true  # Enable IPv4 (costs extra)
```

**Behavior with IPv4 fallback enabled:**
- Servers are provisioned with both IPv4 and IPv6 addresses
- Morpheus tries IPv6 first, falls back to IPv4 if unreachable
- You can connect to servers via either protocol

## Cost Comparison

| Setup | Monthly Cost (3-node forest) |
|-------|----------------------------|
| **IPv6-only (default)** | €8.97 |
| IPv4 + IPv6 (fallback enabled) | €8.97 + IPv4 fees |

**Hetzner IPv4 pricing:** Check current rates (charged per IPv4 address).

## Network Check

Run `morpheus check network` to see your connectivity options:

```bash
$ morpheus check network
📡 Network Connectivity

   Checking IPv6...
   ✅ IPv6 is available
      Your IPv6 address: 2001:db8::1

   Checking IPv4...
   ✅ IPv4 is available
      Your IPv4 address: 203.0.113.1

   ✅ Both IPv6 and IPv4 are available
      Morpheus will use IPv6 by default (recommended, saves costs)
```

## If You Only Have IPv4

If you only have IPv4 connectivity:

1. Enable IPv4 fallback in your config:
   ```yaml
   infrastructure:
     enable_ipv4_fallback: true
   ```

2. Note that IPv4 costs extra on Hetzner

3. Consider getting IPv6:
   - **Enable IPv6 on your network** (best option)
   - **Use IPv6 tunnel:** Hurricane Electric (free)
   - **Use a VPS:** Provision from a VPS that has IPv6

## Note

With IPv4 fallback disabled (default), Morpheus provisions servers with IPv6 only - no IPv4 address is allocated, saving costs.
