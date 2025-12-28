# Termux DNS Issue Fix

## The Problem

You're seeing this error in Termux:

```
Failed to check for updates: Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": 
dial tcp: lookup api.github.com on [::1]:53: read udp [::1]:55541->[::1]:53: read: connection refused
```

**This is a Termux-specific DNS configuration issue.** Android/Termux DNS works differently than regular Linux.

## Why This Happens on Termux

Termux relies on Android's network stack, but sometimes:
- Android's DNS isn't properly exposed to Termux
- VPN apps interfere with DNS resolution
- Private DNS settings block certain connections
- The system is configured to use `localhost` as DNS server

## Quick Fixes for Termux

### Fix 1: Check Your Network Connection

First, verify basic connectivity:

```bash
# Test basic network
ping -c 4 8.8.8.8

# If this fails, you have no internet at all
# If this works, it's a DNS issue
```

### Fix 2: Test DNS Resolution

```bash
# Install dnsutils if needed
pkg install dnsutils -y

# Test DNS
nslookup api.github.com

# If this fails with "connection refused", continue to Fix 3
```

### Fix 3: Disable Private DNS (Common Culprit)

Android's "Private DNS" feature can interfere with Termux:

1. Open **Android Settings**
2. Go to: **Network & Internet** → **Private DNS**
3. Change to: **Off** or **Automatic**
4. Test again in Termux:

```bash
nslookup api.github.com
morpheus update
```

### Fix 4: Restart Termux

Sometimes Termux just needs a fresh start:

```bash
# Exit Termux completely
exit

# Force stop Termux from Android Settings:
# Settings → Apps → Termux → Force Stop

# Reopen Termux and test
nslookup api.github.com
```

### Fix 5: Check VPN/DNS Apps

If you're using a VPN or DNS app (like NextDNS, Blokada, etc.):

1. **Temporarily disable** the VPN/DNS app
2. **Test** if morpheus update works
3. **If it works**, reconfigure your VPN to allow Termux traffic

For VPNs, you may need to:
- Whitelist Termux package: `com.termux`
- Allow DNS queries from Termux
- Disable "Always-on VPN" temporarily

### Fix 6: Use Alternative DNS with curl

As a workaround, you can force curl to use specific DNS:

```bash
# Test with Google DNS
curl --dns-servers 8.8.8.8 https://api.github.com/repos/nimsforest/morpheus/releases/latest

# If this works, the issue is DNS configuration
```

Unfortunately, Go's HTTP client (used by morpheus) doesn't support per-request DNS servers.

### Fix 7: Reinstall Termux (Last Resort)

If nothing else works:

1. **Backup your data:**
   ```bash
   # Backup config
   cp ~/.morpheus/config.yaml ~/storage/shared/morpheus-config-backup.yaml
   
   # Backup registry
   cp ~/.morpheus/registry.json ~/storage/shared/morpheus-registry-backup.json
   
   # Backup SSH keys
   cp -r ~/.ssh ~/storage/shared/ssh-backup/
   ```

2. **Uninstall Termux** from Android

3. **Reinstall from F-Droid**: https://f-droid.org/en/packages/com.termux/

4. **Restore data:**
   ```bash
   # Restore after reinstall
   mkdir -p ~/.morpheus ~/.ssh
   cp ~/storage/shared/morpheus-config-backup.yaml ~/.morpheus/config.yaml
   cp ~/storage/shared/morpheus-registry-backup.json ~/.morpheus/registry.json
   cp -r ~/storage/shared/ssh-backup/* ~/.ssh/
   ```

## The Enhanced Morpheus Updater

The good news: I've **enhanced the morpheus updater** to be more resilient to these issues:

### What's Improved

1. **Automatic Retry** (up to 3 attempts with backoff)
2. **Better DNS handling** (dual-stack IPv4/IPv6 support)
3. **Helpful error messages** (specific troubleshooting for each error type)

### Now You Get Better Errors

Instead of just a cryptic error, morpheus now tells you:

```
DNS configuration issue detected:
  • Your system is trying to use localhost as DNS server
  • Check your network connection
  • Disable Private DNS in Android Settings
  • Restart Termux
  • Try disabling VPN/DNS apps temporarily
```

## Testing the Fix

After trying any fix above, test with:

```bash
# Test DNS resolution
nslookup api.github.com

# Should show something like:
# Server:         8.8.8.8
# Address:        8.8.8.8#53
# Name:   api.github.com
# Address: 140.82.121.6

# Then test morpheus
morpheus update
```

## Common Termux DNS Issues

### Issue: "connection refused to [::1]:53"

**Cause:** System is trying to use localhost as DNS server  
**Fix:** Try Fix 3 (Disable Private DNS) or Fix 4 (Restart Termux)

### Issue: "no such host"

**Cause:** DNS resolution completely failing  
**Fix:** Check internet connection, try Fix 3 (Private DNS)

### Issue: "timeout"

**Cause:** Network slow or VPN blocking  
**Fix:** Check WiFi/mobile data, disable VPN temporarily

### Issue: Works in browser, fails in Termux

**Cause:** Android isolating Termux network access  
**Fix:** Try Fix 3 (Private DNS) and Fix 5 (VPN settings)

## Alternative Workaround: Manual Update

If DNS issues persist, you can manually update morpheus:

```bash
# Download latest binary directly
cd /tmp

# For ARM64 (most modern Android phones)
curl -L -o morpheus https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm64

# For ARM (older phones)
# curl -L -o morpheus https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm

# Make executable
chmod +x morpheus

# Verify it works
./morpheus version

# Install (replace existing)
mv morpheus $PREFIX/bin/morpheus

# Or if $PREFIX is not set:
# mv morpheus /data/data/com.termux/files/usr/bin/morpheus

# Verify installation
morpheus version
```

## Report the Issue

If none of these fixes work, please report the issue with details:

```bash
# Collect debug info
echo "=== Termux DNS Debug Info ===" > ~/dns-debug.txt
echo "" >> ~/dns-debug.txt

echo "Android Version:" >> ~/dns-debug.txt
getprop ro.build.version.release >> ~/dns-debug.txt
echo "" >> ~/dns-debug.txt

echo "Architecture:" >> ~/dns-debug.txt
uname -m >> ~/dns-debug.txt
echo "" >> ~/dns-debug.txt

echo "Network Test:" >> ~/dns-debug.txt
ping -c 2 8.8.8.8 >> ~/dns-debug.txt 2>&1
echo "" >> ~/dns-debug.txt

echo "DNS Test:" >> ~/dns-debug.txt
nslookup api.github.com >> ~/dns-debug.txt 2>&1
echo "" >> ~/dns-debug.txt

echo "Termux Version:" >> ~/dns-debug.txt
pkg show termux | grep Version >> ~/dns-debug.txt
echo "" >> ~/dns-debug.txt

cat ~/dns-debug.txt
```

Then share at: https://github.com/nimsforest/morpheus/issues

## Additional Termux Resources

- **Termux Wiki:** https://wiki.termux.com/wiki/Main_Page
- **Termux Networking:** https://wiki.termux.com/wiki/Networking
- **Termux Issues:** https://github.com/termux/termux-app/issues

## Summary

**Most Common Fix:** Disable "Private DNS" in Android Settings (Fix 3)

**Quick Checklist:**
1. ✅ Check internet connection (ping 8.8.8.8)
2. ✅ Test DNS (nslookup api.github.com)
3. ✅ Disable Private DNS in Android Settings
4. ✅ Restart Termux
5. ✅ Disable VPN temporarily
6. ✅ Try manual update as workaround

**Good news:** The enhanced morpheus updater will now help you troubleshoot with better error messages!

---

**Need more help?** Open an issue: https://github.com/nimsforest/morpheus/issues
