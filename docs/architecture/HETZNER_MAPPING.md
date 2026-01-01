# Hetzner Machine Type Mapping

## Philosophy

Morpheus is **opinionated** about infrastructure choices:

- **Operating System**: Ubuntu 24.04 LTS (only)
- **Architecture**: x86-64 (only)
- **Network**: IPv6-only (IPv4 costs extra)

This document explains how abstract machine profiles map to Hetzner-specific server types.

## Why x86-only?

**Ubuntu 24.04 on Hetzner Cloud does not support ARM architecture.**

While Hetzner offers ARM-based servers (CAX series), the `ubuntu-24.04` image is only available for x86 architecture. Since Morpheus is opinionated about using Ubuntu, we only map to x86 server types.

## Machine Profile Mapping

### ProfileSmall → 1 Machine

**Primary**: `cx22` (€3.29/mo)
- 2 vCPU (shared AMD)
- 4 GB RAM
- Good for: Testing, development, small edge nodes

**Fallbacks** (if cx22 unavailable):
1. `cpx11` (€4.49/mo) - Dedicated vCPU, better performance
2. `cx21` (€3.29/mo) - Older generation Intel

**NOT included**: `cax11` (ARM, incompatible with ubuntu-24.04)

---

### ProfileMedium → 3 Machines

**Primary**: `cpx21` (€8.49/mo)
- 3 vCPU (dedicated AMD)
- 4 GB RAM
- Good for: Small production clusters, consistent performance

**Fallbacks** (if cpx21 unavailable):
1. `cx32` (€6.29/mo) - More vCPU and RAM but shared (cheaper)
2. `cpx31` (€15.49/mo) - More powerful if needed

**NOT included**: `cax21` (ARM, incompatible with ubuntu-24.04)

---

### ProfileLarge → 5 Machines

**Primary**: `cpx41` (€29.49/mo)
- 8 vCPU (dedicated AMD)
- 16 GB RAM
- Good for: Production workloads, consistent performance

**Fallbacks** (if cpx41 unavailable):
1. `cpx51` (€57.49/mo) - 16 vCPU, 32 GB RAM (more powerful)
2. `cx52` (€24.29/mo) - Same specs but shared vCPU (cheaper)

**NOT included**: `cax41` (ARM, incompatible with ubuntu-24.04)

## Server Type Selection Algorithm

```
1. Get machine profile (small, medium, large)
2. Look up Hetzner mapping → primary + fallbacks (ALL x86)
3. For each server type (primary first, then fallbacks):
   a. Get available locations for this type
   b. If locations match user's preferences → USE IT
   c. If not, try next fallback
4. If all fail → error (no suitable server type)
```

## Hetzner Server Type Naming

- **CX** series: Shared vCPU (Intel/AMD), cheaper
- **CPX** series: Dedicated vCPU (AMD), better performance
- **CAX** series: ARM (Ampere Altra), **NOT USED** (Ubuntu incompatible)

Number = size tier (11 < 21 < 31 < 41 < 51)

## Future Considerations

### If ARM Support is Needed

To support ARM in the future:

1. **Option A**: Add ARM-compatible image
   - Use Ubuntu ARM builds
   - Update mapping to include CAX series
   - Add architecture detection

2. **Option B**: Add config option
   ```yaml
   infrastructure:
     architecture: arm  # or 'x86' (default)
   ```

3. **Option C**: Auto-detect
   - Query Hetzner for image architectures
   - Filter server types by image compatibility
   - More complex but fully automatic

### Current Decision: Keep x86-only

**Rationale**:
- Ubuntu 24.04 is x86-only on Hetzner
- Morpheus is opinionated about Ubuntu
- Therefore: x86-only is the right choice
- Simpler code, no architecture complexity
- Works for 99% of use cases

## Cost Optimization

The mapping prioritizes:

1. **Small**: Lowest cost (cx22)
2. **Medium**: Balance of performance and cost (cpx21)
3. **Large**: Performance for production (cpx41)

Fallbacks consider:
- Availability in user's preferred locations
- Price (cheaper alternatives listed)
- Performance (better alternatives listed)

## Location Availability

Not all server types are available in all locations. The selection algorithm automatically:

1. Checks which locations support the server type
2. Filters to user's preferred locations
3. Falls back to next server type if no match

**Default location preference order**:
1. `fsn1` - Falkenstein, Germany
2. `nbg1` - Nuremberg, Germany
3. `hel1` - Helsinki, Finland
4. `ash` - Ashburn, VA, USA
5. `hil` - Hillsboro, OR, USA

## Summary

**Opinionated Choices**:
- ✅ Ubuntu 24.04 LTS
- ✅ x86-64 architecture
- ✅ IPv6-only
- ❌ No ARM (incompatible)
- ❌ No other OS images
- ❌ No IPv4 (costs extra)

**Benefits**:
- Simple, predictable
- Works everywhere Ubuntu 24.04 is supported
- No architecture confusion
- Lower cost (IPv6-only, shared vCPU options)

**Trade-offs**:
- Can't use ARM servers (even though they're cheaper)
- Locked to Ubuntu (by design)
- IPv6 required (by design)

This is **intentional** - Morpheus is opinionated to keep things simple.
