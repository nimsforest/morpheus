# Hetzner Server Type Configuration

## Philosophy

Morpheus is **opinionated** about infrastructure choices:

- **Operating System**: Ubuntu 24.04 LTS (only)
- **Architecture**: x86-64 (only)
- **Network**: IPv6-only by default (IPv4 costs extra)

Server types are configured directly in your config file - no magic mappings.

## Why x86-only?

**Ubuntu 24.04 on Hetzner Cloud does not support ARM architecture.**

While Hetzner offers ARM-based servers (CAX series), the `ubuntu-24.04` image is only available for x86 architecture. Since Morpheus is opinionated about using Ubuntu, we only support x86 server types.

## Configuration

Server type is set directly in your config:

```yaml
machine:
  provider: hetzner
  hetzner:
    server_type: cx22           # Primary server type
    server_type_fallback:       # Fallbacks if primary unavailable
      - cpx11
      - cx32
    image: ubuntu-24.04
    location: fsn1
```

## Recommended Server Types

### For Testing/Development

**Primary**: `cx22` (~€3.29/mo)
- 2 vCPU (shared AMD)
- 4 GB RAM

**Fallbacks**:
- `cpx11` (~€4.49/mo) - Dedicated vCPU
- `cx32` (~€6.29/mo) - More resources

### For Production

**Primary**: `cpx21` (~€8.49/mo)
- 3 vCPU (dedicated AMD)
- 4 GB RAM

**Fallbacks**:
- `cx32` (~€6.29/mo) - Shared but more RAM
- `cpx31` (~€15.49/mo) - More powerful

## Server Type Selection Algorithm

```
1. Read server_type from config
2. Check if available in configured/preferred locations
3. If unavailable, try each fallback in order
4. Use first server type available in any location
```

## Hetzner Server Type Naming

- **CX** series: Shared vCPU (Intel/AMD), cheaper
- **CPX** series: Dedicated vCPU (AMD), better performance
- **CAX** series: ARM (Ampere Altra), **NOT SUPPORTED** (Ubuntu incompatible)

Number = size tier (11 < 21 < 31 < 41 < 51)

## Location Availability

Not all server types are available in all locations. The selection algorithm automatically:

1. Checks which locations support the server type
2. Filters to configured location or defaults
3. Falls back to next server type if no match

**Default location preference order**:
1. `hel1` - Helsinki, Finland
2. `nbg1` - Nuremberg, Germany
3. `fsn1` - Falkenstein, Germany
4. `ash` - Ashburn, VA, USA
5. `hil` - Hillsboro, OR, USA

## Cost Estimates (2024)

| Type | vCPU | RAM | Price/mo |
|------|------|-----|----------|
| cx22 | 2 (shared) | 4 GB | ~€3.29 |
| cx32 | 4 (shared) | 8 GB | ~€6.29 |
| cpx11 | 2 (dedicated) | 2 GB | ~€4.49 |
| cpx21 | 3 (dedicated) | 4 GB | ~€8.49 |
| cpx31 | 4 (dedicated) | 8 GB | ~€15.49 |
| cpx41 | 8 (dedicated) | 16 GB | ~€29.49 |

## Summary

**Opinionated Choices**:
- Ubuntu 24.04 LTS
- x86-64 architecture only
- IPv6-only by default
- Config-driven server type selection

**Benefits**:
- Simple, explicit configuration
- No magic mappings to understand
- Full control over server types and fallbacks
- Works everywhere Ubuntu 24.04 is supported

**Trade-offs**:
- Can't use ARM servers (Ubuntu incompatible)
- Locked to Ubuntu (by design)
- IPv6 required by default (by design)

This is **intentional** - Morpheus is opinionated to keep things simple.
