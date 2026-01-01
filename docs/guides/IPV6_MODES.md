# IPv6 Configuration Modes

Three modes available:

## Dual-Stack (Default)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: false
    ipv6_only: false
```

**Behavior:** Uses IPv4. Server has both IPs. Maximum compatibility.

## IPv6-First

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true
    ipv6_only: false
```

**Behavior:** Uses IPv6, falls back to IPv4 if unavailable.

## IPv6-Only (Strict)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true
    ipv6_only: true
```

**Behavior:** Only uses IPv6. Fails if IPv6 unavailable.

**Requirements:**
- Your network must have IPv6
- All clients must have IPv6
- Test first: `curl -6 ifconfig.co`

## Recommendation

Use **dual-stack** (default) for maximum compatibility.

**Note:** Hetzner always assigns both IPv4 and IPv6. You cannot remove IPv4 from servers.
