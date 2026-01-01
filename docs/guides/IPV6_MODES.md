# IPv6 Configuration Modes

Three modes available:

## IPv6-First (Default)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true   # Default
    ipv6_only: false
```

**Behavior:** Uses IPv6, falls back to IPv4 if unavailable.

## IPv4-Only (Legacy)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: false
    ipv6_only: false
```

**Behavior:** Uses IPv4. For networks without IPv6 support.

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

**IPv6-first (default)** is recommended. Modern infrastructure.

Use **IPv4-only** only if your network doesn't support IPv6.

**Note:** Hetzner always assigns both IPv4 and IPv6. You cannot remove IPv4 from servers.
