# Termux TLS Certificate Fix Guide

If `morpheus update` fails with a certificate error on Termux, this guide will help you fix it.

## Quick Fix (Recommended)

The **easiest and most reliable** solution is to install curl:

```bash
pkg install curl
```

Then try updating again:

```bash
morpheus update
```

**That's it!** Morpheus will automatically detect Termux and use curl for all HTTPS requests, completely bypassing certificate configuration issues.

## Why curl?

On Termux/Android, curl is better at handling TLS certificates than Go's built-in HTTP client because:
- ✅ Curl comes pre-configured with proper certificate handling on Termux
- ✅ No need to manually configure CA certificates
- ✅ Works out of the box after installation
- ✅ More reliable for HTTPS requests on Android

## Alternative: Install CA Certificates

If you prefer not to use curl, you can install CA certificates (but this is more complex):

```bash
pkg update
pkg install ca-certificates-java openssl
```

**Note**: This may not work as reliably as curl on Termux.

## Troubleshooting Steps

### Step 1: Run Diagnostics

First, check what's actually happening:

```bash
morpheus diagnose-certs
```

This will show you:
- ✓ Which certificate files are found
- ✗ Which certificate locations are missing
- Whether TLS connections work
- Specific recommendations for your setup

### Step 2: Check Certificate Installation

Verify the packages are installed:

```bash
pkg list-installed | grep -E 'ca-certificates|openssl'
```

You should see both packages. If not, install them:

```bash
pkg install ca-certificates-java openssl
```

### Step 3: Verify Certificate Files

Check if certificate files exist:

```bash
ls -lh $PREFIX/etc/tls/certs/ca-certificates.crt
ls -lh $PREFIX/etc/ssl/certs/ca-certificates.crt
```

At least one should exist and be non-empty (200KB+).

### Step 4: Enable Debug Mode

If it still doesn't work, enable debug mode to see exactly what's happening:

```bash
MORPHEUS_TLS_DEBUG=1 morpheus update
```

This will show:
- Which certificate bundles are being loaded
- Where they're being loaded from
- Any errors encountered

### Step 5: Check Environment Variables

Verify your Termux environment:

```bash
echo "PREFIX: $PREFIX"
echo "TERMUX_VERSION: $TERMUX_VERSION"
```

The PREFIX should be `/data/data/com.termux/files/usr`.

## Common Issues

### Issue 1: Only Installed `ca-certificates`

**Symptom**: Still getting certificate errors after installing `ca-certificates`

**Solution**: Install `ca-certificates-java` as well:
```bash
pkg install ca-certificates-java
```

### Issue 2: Old Package Cache

**Symptom**: Packages install but certificates still don't work

**Solution**: Update package cache and reinstall:
```bash
pkg update
pkg reinstall ca-certificates-java openssl
```

### Issue 3: Corrupted Certificates

**Symptom**: Certificate files exist but are empty or corrupted

**Solution**: Remove and reinstall:
```bash
pkg uninstall ca-certificates-java
pkg install ca-certificates-java
```

### Issue 4: System DNS Issues

**Symptom**: Gets past certificate check but fails on DNS

**Solution**: Morpheus automatically handles Termux DNS issues, but you can verify with:
```bash
MORPHEUS_TLS_DEBUG=1 morpheus diagnose-certs
```

## Emergency Workaround

If you absolutely cannot get certificates working (NOT RECOMMENDED for security reasons):

```bash
MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

**⚠️  WARNING**: This disables certificate verification and should only be used temporarily.

## Verification

After fixing the issue, verify it works:

```bash
# Should show detailed certificate loading
MORPHEUS_TLS_DEBUG=1 morpheus diagnose-certs

# Should work without errors
morpheus check-update

# Should successfully check and update if available
morpheus update
```

You should see output like:
```
✓ Loaded certificates from: /data/data/com.termux/files/usr/etc/tls/certs/ca-certificates.crt
📊 Total certificate bundles loaded: 1
```

## Still Having Issues?

If you're still having problems after following this guide:

1. Share the output of:
   ```bash
   morpheus diagnose-certs > diagnosis.txt
   MORPHEUS_TLS_DEBUG=1 morpheus update 2>&1 | head -50 >> diagnosis.txt
   pkg list-installed | grep -E 'ca-certificates|openssl' >> diagnosis.txt
   echo "PREFIX: $PREFIX" >> diagnosis.txt
   cat diagnosis.txt
   ```

2. Open an issue at: https://github.com/nimsforest/morpheus/issues

## Technical Details

Morpheus now:
- Automatically detects Termux via `$PREFIX`, `$TERMUX_VERSION`, or `$ANDROID_ROOT` environment variables
- Searches for certificates in `$PREFIX/etc/tls/certs/` and `$PREFIX/etc/ssl/certs/`
- Falls back to multiple certificate locations
- Provides debug output when `MORPHEUS_TLS_DEBUG=1` is set
- Warns when no certificates can be loaded

The fix ensures proper TLS certificate verification while maintaining security and providing helpful diagnostics when issues occur.
