# IPv6 Configuration Modes

Morpheus supports three IPv6 modes to match your infrastructure needs.

## Mode Comparison

| Mode | IPv4 Used? | IPv6 Used? | Server Has Both? | Best For |
|------|-----------|-----------|------------------|----------|
| **Dual-stack (default)** | ✅ Primary | ✅ Available | ✅ Yes | Maximum compatibility |
| **Prefer IPv6** | ✅ Fallback | ✅ Primary | ✅ Yes | IPv6-first, IPv4 backup |
| **IPv6-only** | ❌ No | ✅ Only | ✅ Yes* | Pure IPv6 infrastructure |

*Server still gets both IPs from Hetzner, but Morpheus ignores IPv4

---

## Mode 1: Dual-Stack (Default) - Recommended

**Configuration:**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: false  # Default
    ipv6_only: false    # Default
```

**Behavior:**
- Morpheus uses IPv4 for SSH checks
- Server gets both IPv4 and IPv6 addresses
- Clients can connect via either protocol
- NATS listens on both IPv4 and IPv6

**Use this if:**
- ✅ You want maximum compatibility
- ✅ Your clients might be on IPv4-only networks
- ✅ You're not sure about IPv6 adoption
- ✅ You want "it just works" behavior

**Example:**
```bash
$ morpheus plant cloud wood
Waiting for infrastructure readiness (SSH on 95.217.0.1:22 via IPv4, timeout: 5m)...
✓ Node forest-123 provisioned (IPv4: 95.217.0.1, IPv6: 2001:db8::1)
```

Both addresses work for clients:
```bash
# Clients can use either
nats pub -s nats://95.217.0.1:4222 test "IPv4 message"
nats pub -s nats://[2001:db8::1]:4222 test "IPv6 message"
```

---

## Mode 2: Prefer IPv6 (IPv6-First)

**Configuration:**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: true   # Use IPv6 when available
    ipv6_only: false
```

**Behavior:**
- Morpheus uses IPv6 for SSH checks
- Falls back to IPv4 if IPv6 unavailable
- Server gets both IPv4 and IPv6 addresses
- Clients can connect via either protocol

**Use this if:**
- ✅ You have IPv6 connectivity on your network
- ✅ You want to use IPv6 but keep IPv4 as backup
- ✅ You're transitioning to IPv6
- ✅ You want to test IPv6 without commitment

**Example:**
```bash
$ morpheus plant cloud wood
Waiting for infrastructure readiness (SSH on [2001:db8::1]:22 via IPv6, timeout: 5m)...
✓ Node forest-123 provisioned (IPv4: 95.217.0.1, IPv6: 2001:db8::1)
```

Both addresses still work for clients.

**Requirements:**
- Your local network must have IPv6 connectivity
- Test with: `curl -6 ifconfig.co`

---

## Mode 3: IPv6-Only (Strict)

**Configuration:**
```yaml
infrastructure:
  defaults:
    prefer_ipv6: true   # Not strictly needed, but recommended
    ipv6_only: true     # Enforce IPv6-only
```

**Behavior:**
- Morpheus **only** uses IPv6 for SSH checks
- **Fails** if server has no IPv6 address
- Ignores IPv4 address completely
- Server still gets both IPs (but Morpheus doesn't use IPv4)
- Registry stores only IPv6 address

**Use this if:**
- ✅ You're building a pure IPv6 infrastructure
- ✅ All your clients have IPv6 connectivity
- ✅ You want to force IPv6 adoption
- ✅ You're running in a modern cloud environment
- ✅ You're experimenting with IPv6-only architecture

**Example:**
```bash
$ morpheus plant cloud wood
Waiting for infrastructure readiness (SSH on [2001:db8::1]:22 via IPv6 [IPv6-only mode], timeout: 5m)...
✓ Node forest-123 provisioned (IPv4: 95.217.0.1, IPv6: 2001:db8::1)
```

**Important:** Even in IPv6-only mode, the server still has both addresses. Morpheus just doesn't use IPv4.

**Requirements:**
- ⚠️ **Your local network MUST have IPv6** (provisioning fails without it)
- ⚠️ **All NATS clients MUST have IPv6** (they can't connect otherwise)
- ⚠️ **You MUST configure NATS to listen on IPv6**

Test first:
```bash
# Check if you have IPv6
curl -6 ifconfig.co

# If this times out, DO NOT use ipv6_only mode
```

---

## Can You Actually Remove IPv4 from Servers?

**Short answer: No, not with Hetzner Cloud.**

Hetzner Cloud **always** assigns both IPv4 and IPv6 to every server. You cannot opt out of IPv4.

**What `ipv6_only: true` does:**
- Morpheus ignores the IPv4 address
- Registry doesn't store the IPv4 address
- SSH checks use IPv6 only

**What it DOESN'T do:**
- Remove IPv4 from the server (impossible)
- Prevent clients from connecting via IPv4 (they still can)
- Disable IPv4 at OS level (server still responds on IPv4)

### To Truly Disable IPv4

If you really want IPv4 disabled, you'd need to:

1. **Firewall block IPv4** (in cloud-init):
```yaml
# Add to cloud-init template
runcmd:
  - ufw deny from any to any proto tcp port 4222 # Block NATS on IPv4
  - ufw allow from any to [your-ipv6-subnet] proto tcp port 4222 # Allow only IPv6
```

2. **Configure NATS to only bind IPv6**:
```conf
# nats.conf
listen: "[::]:4222"  # IPv6 only, NOT 0.0.0.0
```

3. **Disable IPv4 at OS level** (extreme):
```bash
# In cloud-init
sysctl -w net.ipv6.conf.all.disable_ipv4=1
```

But this is **not recommended** because:
- Breaks many standard tools (DNS, package managers)
- Makes troubleshooting harder
- Provides minimal benefit
- You're still paying for the IPv4 (Hetzner includes it)

---

## Recommended Setup

### For Most Users (Default)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: false  # Use IPv4
    ipv6_only: false
```

**Why:** Maximum compatibility, works everywhere.

### For Modern Infrastructure (IPv6-First)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true   # Use IPv6
    ipv6_only: false    # But fall back to IPv4 if needed
```

**Why:** Use IPv6 when possible, graceful fallback.

### For Pure IPv6 Environments (Advanced)

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true
    ipv6_only: true     # Strict IPv6-only
```

**Why:** Force IPv6 adoption, fail fast if IPv6 unavailable.

---

## Why NOT Remove IPv4 Entirely?

### 1. **Client Compatibility**

| Network Type | IPv4 Support | IPv6 Support |
|--------------|-------------|--------------|
| Home users | ✅ 100% | ⚠️ 30-50% |
| Mobile carriers | ✅ 100% | ⚠️ 60-70% |
| Corporate networks | ✅ 100% | ⚠️ 20-40% |
| Cloud providers | ✅ 100% | ✅ 95%+ |

If you remove IPv4, 40-70% of potential clients **cannot connect**.

### 2. **Your Local Network**

Most home/office ISPs still don't provide IPv6:
- You couldn't SSH to your servers
- You couldn't run `morpheus status`
- Provisioning would fail

### 3. **It's Free**

Hetzner gives you both IPs at no cost. Why limit yourself?

### 4. **Transition Period**

IPv4 → IPv6 migration takes years. Dual-stack lets you:
- Support both client types
- Gradually migrate clients
- No service disruption

---

## Testing Your IPv6 Connectivity

**Before enabling IPv6 modes, test:**

```bash
# Do you have IPv6?
curl -6 ifconfig.co
# Should return your IPv6 address
# If it times out, you DON'T have IPv6

# Can you reach IPv6 hosts?
ping6 google.com
# Should get responses
# If it fails, your IPv6 is broken

# Can you SSH via IPv6?
ssh -6 root@2001:db8::1
# Should connect
# If it times out, firewall or routing issue
```

**If any of these fail, DON'T use `ipv6_only: true`**

---

## FAQ

**Q: Should I remove IPv4?**  
A: No. Keep dual-stack for maximum compatibility.

**Q: When should I use `ipv6_only: true`?**  
A: Only if ALL these are true:
- Your network has IPv6
- All clients have IPv6
- You want to force IPv6 adoption
- You understand the limitations

**Q: Does `ipv6_only` save money?**  
A: No. Hetzner provides both IPs for free.

**Q: Can clients still use IPv4 in `ipv6_only` mode?**  
A: Yes! The server still has IPv4. Only Morpheus ignores it.

**Q: How do I truly disable IPv4?**  
A: Configure OS firewall and NATS to reject IPv4 connections. (Not recommended)

**Q: What's the recommended mode?**  
A: Dual-stack (default) for most users. IPv6-first if you have good IPv6 connectivity.

---

## Summary

| Mode | Config | Use Case |
|------|--------|----------|
| **Dual-stack** | `prefer_ipv6: false` | Default, maximum compatibility |
| **IPv6-first** | `prefer_ipv6: true` | Modern infrastructure, graceful fallback |
| **IPv6-only** | `ipv6_only: true` | Strict IPv6, experimental, advanced |

**Recommendation:** Keep dual-stack. Don't remove IPv4 unless you have a specific reason and understand the limitations.

The internet is still transitioning to IPv6. Dual-stack is the safe, practical choice. 🌍
