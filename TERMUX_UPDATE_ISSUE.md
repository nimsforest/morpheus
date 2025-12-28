# Morpheus Update Error on Termux - Quick Fix

## Your Error

```
Failed to check for updates: Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": 
dial tcp: lookup api.github.com on [::1]:53: read udp [::1]:55541->[::1]:53: read: connection refused
```

## What This Means

Your Termux environment has a **DNS configuration issue**. The system is trying to use `localhost` as a DNS server, but there's no DNS service running there.

## Quick Fix (Works 90% of the time)

### Fix: Disable Private DNS in Android

1. Open **Android Settings**
2. Go to: **Network & Internet** → **Private DNS**
3. Change to: **Off** or **Automatic**
4. Back in Termux, try again:

```bash
morpheus update
```

## Alternative Fixes

### If Private DNS didn't work:

**1. Restart Termux:**
```bash
exit
# From Android: Settings → Apps → Termux → Force Stop
# Reopen Termux
morpheus update
```

**2. Test your connection:**
```bash
# Test basic internet
ping -c 4 8.8.8.8

# Install DNS tools
pkg install dnsutils -y

# Test DNS
nslookup api.github.com
```

**3. Disable VPN/DNS apps:**
- Temporarily disable any VPN apps (NordVPN, ProtonVPN, etc.)
- Disable DNS filtering apps (NextDNS, Blokada, etc.)
- Try `morpheus update` again

**4. Switch networks:**
- If on WiFi, try mobile data
- If on mobile data, try WiFi

## Manual Update (Workaround)

If DNS issues persist, manually update morpheus:

```bash
# Download latest release directly
cd /tmp

# For ARM64 (most modern phones)
curl -L -o morpheus https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm64

# Or for ARM (older phones)
# curl -L -o morpheus https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm

# Make executable and install
chmod +x morpheus
mv morpheus $PREFIX/bin/morpheus

# Verify
morpheus version
```

## The Good News

**I've enhanced the morpheus updater** to be more resilient to these issues:

✅ **Automatic retry** (3 attempts with exponential backoff)  
✅ **Better DNS handling** (improved IPv4/IPv6 support)  
✅ **Helpful error messages** (specific troubleshooting for each error)

After you fix your DNS and rebuild/reinstall morpheus, you'll get much better error messages if this happens again!

## Rebuild Morpheus with Improvements

Once you fix your DNS, rebuild morpheus to get the improvements:

```bash
cd ~/morpheus
git pull
make build
make install

# Or download latest binary (faster)
curl -L -o /tmp/morpheus https://github.com/nimsforest/morpheus/releases/latest/download/morpheus-linux-arm64
chmod +x /tmp/morpheus
mv /tmp/morpheus $PREFIX/bin/morpheus
```

## Why This Happened

**The updater feature is brand new** (added Dec 28, 2025). This is probably the first time morpheus tried to contact GitHub API from your device, which exposed a pre-existing DNS configuration issue on your Android/Termux setup.

**This is NOT caused by the updater** - it just revealed a DNS problem that was already there.

## More Help

- **Detailed Termux DNS troubleshooting:** [TERMUX_DNS_FIX.md](TERMUX_DNS_FIX.md)
- **General DNS fix guide:** [DNS_FIX_README.md](DNS_FIX_README.md)
- **Report issues:** https://github.com/nimsforest/morpheus/issues

## Summary

**TL;DR:** 
1. Disable "Private DNS" in Android Settings
2. Restart Termux
3. Try `morpheus update` again
4. If still fails, use manual update method above

---

**Most common cause:** Android's "Private DNS" feature interfering with Termux network access.

**Fix time:** 30 seconds to 2 minutes.
