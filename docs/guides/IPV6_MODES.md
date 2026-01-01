# IPv6 Configuration

## Default: IPv6-Only

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true
    ipv6_only: true    # Default
```

**Behavior:** Only uses IPv6. Fails if no IPv6 (by design).

**Why:** IPv4 costs extra on Hetzner. IPv6-only saves money.

## Requirements

- Your network must have IPv6
- Test: `curl -6 ifconfig.co`

## Cost Savings

| Setup | Monthly Cost (3-node forest) |
|-------|----------------------------|
| **IPv6-only (default)** | €8.97 |
| IPv4 + IPv6 | €8.97 + IPv4 fees |

**Hetzner IPv4 pricing:** Check current rates (charged per IPv4 address).

## If You Don't Have IPv6

You cannot use Morpheus without IPv6. Options:

1. **Enable IPv6 on your network** (best option)
2. **Use IPv6 tunnel:** Hurricane Electric (free)
3. **Use a VPS:** Provision from a VPS that has IPv6

## Note

Hetzner still may provide IPv4 by default, but Morpheus won't use it to avoid charges.
