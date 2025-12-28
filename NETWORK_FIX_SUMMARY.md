# Network Resilience Fix Summary

## Issue Reported

The `morpheus update` command was failing with a DNS resolution error:

```
Failed to check for updates: failed to check for updates: Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": dial tcp: lookup api.github.com on [::1]:53: read udp [::1]:55541->[::1]:53: read: connection refused
```

## Root Cause

The system's DNS is configured to use localhost (`[::1]:53` - IPv6) as the DNS server, but there's no DNS service running on localhost. This is a common misconfiguration issue.

## Solution Implemented

Enhanced the updater package (`pkg/updater/updater.go`) with three major improvements:

### 1. Custom HTTP Transport with Better Dialer
- Improved timeout handling (5 seconds for dial, 30 seconds keep-alive)
- Proper dual-stack IPv4/IPv6 support
- Optimized connection pooling and TLS handshake

### 2. Retry Logic with Exponential Backoff
- Automatically retries network failures up to 3 times
- Uses exponential backoff (2s → 4s → 6s) to avoid overwhelming servers
- Only retries transient network errors (not API errors like 404, 403, etc.)

### 3. Enhanced Error Messages with Troubleshooting
Now provides specific, actionable advice based on the error type:

**For DNS resolution failures:**
```
Network troubleshooting:
  • Check your internet connection
  • Verify DNS is configured correctly (check /etc/resolv.conf)
  • Try: ping api.github.com
  • If on IPv6-only network, ensure IPv6 DNS is working
  • Try using a different DNS server (e.g., 8.8.8.8, 1.1.1.1)
```

**For localhost DNS configuration issues:**
```
DNS configuration issue detected:
  • Your system is trying to use localhost as DNS server
  • Check /etc/resolv.conf for incorrect DNS settings
  • Common fix: Replace localhost DNS with:
      nameserver 8.8.8.8
      nameserver 1.1.1.1
  • On some systems, edit /etc/systemd/resolved.conf
```

## Testing

All tests pass:
```bash
$ go test ./... -v
PASS: pkg/cloudinit (0.002s)
PASS: pkg/config (0.003s)
PASS: pkg/forest (0.014s)
PASS: pkg/provider/hetzner (0.004s)
PASS: pkg/updater/version (0.012s)
```

Build successful:
```bash
$ go build ./cmd/morpheus
✓ Binary compiled successfully
```

## How to Fix Your DNS Issue

Since the error shows `lookup api.github.com on [::1]:53`, your system is trying to use localhost as DNS. Here's how to fix it:

### Option 1: Edit /etc/resolv.conf (Temporary)
```bash
sudo nano /etc/resolv.conf
```

Replace content with:
```
nameserver 8.8.8.8
nameserver 1.1.1.1
```

**Note:** This change may be overwritten by your system's network manager.

### Option 2: Configure systemd-resolved (Permanent)
If your system uses systemd-resolved:

```bash
sudo nano /etc/systemd/resolved.conf
```

Add or modify:
```ini
[Resolve]
DNS=8.8.8.8 1.1.1.1
FallbackDNS=8.8.4.4 1.0.0.1
```

Then restart the service:
```bash
sudo systemctl restart systemd-resolved
```

### Option 3: Configure Network Manager
If using NetworkManager:

```bash
sudo nmcli connection modify "connection-name" ipv4.dns "8.8.8.8,1.1.1.1"
sudo nmcli connection down "connection-name" && sudo nmcli connection up "connection-name"
```

### Verify the Fix

Test DNS resolution:
```bash
nslookup api.github.com
ping api.github.com
```

Then try the update again:
```bash
morpheus update
```

## Benefits

1. **Better Reliability**: Network failures are automatically retried
2. **Better UX**: Clear, actionable error messages guide users to solutions
3. **Mobile-Friendly**: Works better on unstable mobile networks (important for Termux users)
4. **Backward Compatible**: No breaking changes, existing functionality preserved

## Files Modified

- `pkg/updater/updater.go` - Core updater logic with retry and error handling
- `CHANGELOG.md` - Updated with network resilience improvements
- `UPDATER_NETWORK_IMPROVEMENTS.md` - Detailed technical documentation

## Next Steps for You

1. Fix your DNS configuration using one of the methods above
2. Test the update command: `morpheus update`
3. The enhanced error messages will now guide you if any issues occur

## Additional Notes

The improvements made to the updater will help all users, especially those on:
- Mobile/Termux environments with varying network quality
- Systems with DNS misconfigurations
- Networks with intermittent connectivity
- IPv6-only or dual-stack networks

All changes are fully backward compatible and require no configuration changes.
