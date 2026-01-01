# IPv6 Support in Morpheus

Morpheus fully supports IPv6! All Hetzner Cloud servers come with both IPv4 and IPv6 addresses.

## Quick Start

### Enable IPv6

Edit your `config.yaml`:

```yaml
infrastructure:
  defaults:
    prefer_ipv6: true  # Use IPv6 instead of IPv4
```

That's it! Morpheus will now use IPv6 for all connections.

## How It Works

### Automatic IP Assignment

When you provision a server, Hetzner automatically assigns:
- **IPv4**: e.g., `95.217.0.1`
- **IPv6**: e.g., `2001:db8:1234:5678::1`

Both addresses are active and work immediately.

### IP Preference

**With `prefer_ipv6: false` (default):**
- SSH checks use IPv4
- Registry stores IPv4
- Display shows both IPs

**With `prefer_ipv6: true`:**
- SSH checks use IPv6
- Registry stores IPv6
- Display shows both IPs

### Example Output

**IPv4 mode (default):**
```bash
$ morpheus plant cloud wood
Starting forest provisioning: forest-1735234567 (size: wood, location: fsn1)
Provisioning 1 node(s)...
Server 12345678 created, waiting for it to be ready...
Server running, verifying infrastructure readiness...
Waiting for infrastructure readiness (SSH on 95.217.0.1:22 via IPv4, timeout: 5m)...
✓ Infrastructure ready after 3 attempts (SSH accessible)
✓ Node forest-1735234567-node-1 provisioned successfully (IPv4: 95.217.0.1, IPv6: 2001:db8::1)
✓ Forest forest-1735234567 provisioned successfully!
```

**IPv6 mode:**
```bash
$ morpheus plant cloud wood
Starting forest provisioning: forest-1735234567 (size: wood, location: fsn1)
Provisioning 1 node(s)...
Server 12345678 created, waiting for it to be ready...
Server running, verifying infrastructure readiness...
Waiting for infrastructure readiness (SSH on [2001:db8::1]:22 via IPv6, timeout: 5m)...
✓ Infrastructure ready after 3 attempts (SSH accessible)
✓ Node forest-1735234567-node-1 provisioned successfully (IPv4: 95.217.0.1, IPv6: 2001:db8::1)
✓ Forest forest-1735234567 provisioned successfully!
```

## SSH with IPv6

### From Your Local Machine

**IPv4:**
```bash
ssh root@95.217.0.1
```

**IPv6 (requires brackets in some contexts):**
```bash
ssh root@2001:db8::1
# or with brackets:
ssh 'root@[2001:db8::1]'
```

### In Scripts

When using IPv6 in URLs or addresses, use brackets:

**Correct:**
```bash
curl http://[2001:db8::1]:8222/varz
```

**Wrong:**
```bash
curl http://2001:db8::1:8222/varz  # Ambiguous - too many colons!
```

## NATS with IPv6

NATS fully supports IPv6. When NimsForest configures NATS:

### NATS Configuration (IPv6)

```conf
# NATS listens on all interfaces by default (IPv4 + IPv6)
port: 4222

# Or explicitly bind to IPv6:
listen: "[::]:4222"  # IPv6 all interfaces
# listen: "0.0.0.0:4222"  # IPv4 all interfaces

# Cluster with IPv6
cluster {
  name: "forest-123"
  listen: "[::]:6222"
  routes: [
    "nats://[2001:db8::1]:6222"
    "nats://[2001:db8::2]:6222"
    "nats://[2001:db8::3]:6222"
  ]
}
```

### Client Connections (IPv6)

**Go client:**
```go
nc, err := nats.Connect("nats://[2001:db8::1]:4222")
```

**CLI:**
```bash
nats pub -s nats://[2001:db8::1]:4222 test "Hello IPv6"
```

**JavaScript:**
```javascript
const nc = await connect({ servers: ["nats://[2001:db8::1]:4222"] });
```

## Why Use IPv6?

### Advantages

1. **Future-proof** - IPv4 exhaustion is real
2. **Free** - Some providers charge for IPv4 (not Hetzner yet, but trend is growing)
3. **More addresses** - 340 undecillion vs 4 billion
4. **Better privacy** - Rotating addresses possible
5. **Modern** - Better for new infrastructure

### Disadvantages

1. **Not universal yet** - Some ISPs don't support IPv6
2. **Complexity** - Longer addresses, bracket syntax
3. **NAT workarounds** - May need IPv4 for legacy systems

### When to Use IPv6

**Use IPv6 if:**
- ✅ All your infrastructure supports it
- ✅ You want to be future-proof
- ✅ Your clients/users have IPv6 connectivity
- ✅ You're building new systems

**Stick with IPv4 if:**
- ⚠️ Legacy systems that don't support IPv6
- ⚠️ Clients primarily on IPv4-only networks
- ⚠️ Simplicity is more important than future-proofing

## Dual-Stack (IPv4 + IPv6)

The best approach is **dual-stack**: Both IPv4 and IPv6 enabled.

**Good news:** Morpheus servers are dual-stack by default!

### Access via Either Protocol

Your servers are reachable via both:

```bash
# IPv4
ssh root@95.217.0.1
curl http://95.217.0.1:8222/varz

# IPv6
ssh root@2001:db8::1
curl http://[2001:db8::1]:8222/varz
```

Both work simultaneously. Clients choose based on their connectivity.

## Troubleshooting

### "Server has no public IP address"

**Problem:** Server provisioning fails with no IP error.

**Solution:** This shouldn't happen on Hetzner. Check:
```bash
# Verify server has IPs
hcloud server describe <server-id>
```

### "SSH timeout on IPv6"

**Problem:** SSH check hangs when using IPv6.

**Causes:**
1. **Your network doesn't support IPv6** - Try IPv4 mode
2. **Firewall blocking IPv6** - Check Hetzner firewall rules
3. **Server IPv6 not activated** - Rare, contact Hetzner

**Solution:**
```yaml
# Switch back to IPv4
infrastructure:
  defaults:
    prefer_ipv6: false
```

### Test IPv6 Connectivity

**From your local machine:**
```bash
# Test if you have IPv6
curl -6 ifconfig.co

# Ping an IPv6 address
ping6 2001:db8::1

# Check if server accepts IPv6 SSH
ssh -6 root@2001:db8::1
```

### NATS Not Binding to IPv6

**Problem:** NATS only listens on IPv4.

**Solution:** Update NATS config to bind to IPv6:
```conf
listen: "[::]:4222"  # All interfaces (IPv6 + IPv4)
```

Or explicitly both:
```conf
listen: ["0.0.0.0:4222", "[::]:4222"]
```

## Cloud-Init and IPv6

Morpheus cloud-init scripts automatically work with both IPv4 and IPv6:

```bash
# In cloud-init, this works for both:
INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
INSTANCE_IPV6=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv6)
```

Hetzner's metadata service provides both addresses.

## Configuration Examples

### IPv4 Only (Default)

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cx23
    image: ubuntu-24.04
    prefer_ipv6: false  # Use IPv4
```

### IPv6 Only

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cx23
    image: ubuntu-24.04
    prefer_ipv6: true  # Use IPv6
```

### Per-Role Configuration

```yaml
infrastructure:
  defaults:
    prefer_ipv6: false
  
  roles:
    edge:
      prefer_ipv6: true   # Forest nodes use IPv6
    compute:
      prefer_ipv6: false  # Nims nodes use IPv4
```

**Note:** Per-role IP preference not yet implemented. Coming soon!

## Technical Details

### IPv6 Address Format

**Full notation:**
```
2001:0db8:85a3:0000:0000:8a2e:0370:7334
```

**Compressed (zero omission):**
```
2001:db8:85a3::8a2e:370:7334
```

**Loopback:**
```
::1  (equivalent to 127.0.0.1 in IPv4)
```

**All interfaces:**
```
::  (equivalent to 0.0.0.0 in IPv4)
```

### SSH Address Formatting

Morpheus automatically formats addresses correctly:

| IP Type | Format | Example |
|---------|--------|---------|
| **IPv4** | `ip:port` | `95.217.0.1:22` |
| **IPv6** | `[ip]:port` | `[2001:db8::1]:22` |

The brackets are **required** for IPv6 to avoid ambiguity with colons.

### Registry Storage

The forest registry stores your preferred IP:

```json
{
  "id": "12345678",
  "forest_id": "forest-1735234567",
  "role": "edge",
  "ip": "2001:db8::1",  // IPv6 if prefer_ipv6: true
  "location": "fsn1",
  "status": "active"
}
```

## FAQ

**Q: Do I need to configure anything for IPv6?**  
A: No! All servers get IPv6 by default. Set `prefer_ipv6: true` if you want to use it.

**Q: Can I use both IPv4 and IPv6?**  
A: Yes! Servers are dual-stack. Clients can connect via either.

**Q: Does IPv6 cost extra?**  
A: No. Hetzner provides IPv6 for free with all servers.

**Q: Will NATS work with IPv6?**  
A: Yes! NATS fully supports IPv6. Just use bracket notation in URLs.

**Q: What if my ISP doesn't support IPv6?**  
A: Keep `prefer_ipv6: false` (default). Use IPv4 for now.

**Q: Can I mix IPv4 and IPv6 nodes in a cluster?**  
A: Not recommended. NATS clustering works best when all nodes use the same IP version.

**Q: How do I test if IPv6 works?**  
A: Set `prefer_ipv6: true`, provision a node, SSH via IPv6 address.

## Summary

- ✅ All Morpheus servers get IPv4 + IPv6 (dual-stack)
- ✅ Set `prefer_ipv6: true` to use IPv6 for connections
- ✅ Both protocols work simultaneously
- ✅ NATS fully supports IPv6
- ✅ No extra cost or configuration needed

**Recommendation:** Start with IPv4 (default), switch to IPv6 when ready.

---

**Related Documentation:**
- [Morpheus Configuration](../README.md)
- [NATS Documentation](https://docs.nats.io/)
- [Hetzner Cloud IPv6](https://docs.hetzner.com/cloud/servers/ipv6/)
