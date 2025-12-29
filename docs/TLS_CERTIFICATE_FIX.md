# TLS Certificate Error Fix

## Problem

Users on certain systems (especially Termux/Android, minimal Linux distributions, or containers) were experiencing certificate verification errors when trying to update morpheus:

```
Failed to check for updates: failed to check for updates: Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": tls: failed to verify certificate: x509: certificate signed by unknown authority
```

## Root Cause

The update checker was making HTTPS requests to GitHub's API without properly configuring TLS certificate verification. This caused failures on systems where:

1. CA (Certificate Authority) certificates are not installed
2. System certificate paths are non-standard
3. Certificate bundles are incomplete or missing

## Solution

The fix includes several improvements to the HTTP client configuration, plus an automatic curl fallback for maximum reliability:

### 1. Curl on Termux/Android (Primary Method)

**The most important improvement**: On Termux/Android, morpheus now uses `curl` as the **primary method** for HTTPS requests instead of the Go HTTP client. This is because curl handles TLS certificates much more reliably on Android systems.

**How it works**:
- Detects Termux/Android environment (via `$ANDROID_ROOT`, `$TERMUX_VERSION`, or `runtime.GOOS`)
- If curl is installed, uses it directly for all HTTPS requests
- Falls back to HTTP client only if curl is not available or fails
- On non-Android systems, continues to use the HTTP client as normal

**Benefits**:
- ✅ Works out of the box on Termux without certificate configuration
- ✅ Curl on Termux is properly configured for TLS by default
- ✅ No need to mess with CA certificates on Android
- ✅ Both update checking and binary downloads use curl
- ✅ Seamless - no user intervention required

**Requirements**:
- `curl` must be installed on Termux/Android
- Install with: `pkg install curl`
- Curl is usually already installed on most Termux setups

### 2. Proper TLS Configuration

- Added `crypto/tls` and `crypto/x509` imports
- Created `createTLSConfig()` function that:
  - Loads system CA certificate pool
  - Searches for CA certificates in common locations across different platforms
  - Supports custom certificate paths via `SSL_CERT_FILE` environment variable
  - Sets minimum TLS version to 1.2 for security

### 3. Multi-Platform Certificate Support

The updater now searches for CA certificates in these locations:

**Termux/Android** (uses `$PREFIX` environment variable):
- `$PREFIX/etc/tls/certs/ca-certificates.crt`
- `$PREFIX/etc/tls/cert.pem`
- `$PREFIX/etc/ssl/certs/ca-certificates.crt`
- `$PREFIX/etc/ssl/cert.pem`
- `/system/etc/security/cacerts` - Android system certificates

**Standard Linux/Unix**:
- `/etc/ssl/certs/ca-certificates.crt` - Debian/Ubuntu/Gentoo
- `/etc/pki/tls/certs/ca-bundle.crt` - Fedora/RHEL
- `/etc/ssl/ca-bundle.pem` - OpenSUSE
- `/etc/ssl/cert.pem` - OpenBSD
- `/usr/local/share/certs/ca-root-nss.crt` - FreeBSD
- `/etc/pki/tls/cacert.pem` - OpenELEC
- `/etc/certs/ca-certificates.crt` - Solaris 11.2+

**Custom**:
- Custom path from `$SSL_CERT_FILE` environment variable

### 4. Certificate Diagnostics Command

A new `diagnose-certs` command helps users troubleshoot certificate issues:

```bash
morpheus diagnose-certs
```

This command:
- Shows system information (OS, architecture, environment variables)
- Checks if the system certificate pool can be loaded
- Scans for certificate files in all known locations
- Tests TLS connectivity to GitHub API
- Provides specific recommendations for your system

Example output:
```
🔍 Morpheus TLS Certificate Diagnostics
========================================

OS: android
Arch: arm64
PREFIX: /data/data/com.termux/files/usr

System Certificate Pool:
  ❌ Failed to load: crypto/x509: system root pool is not available on Android

Certificate File Locations:
  ✓ /data/data/com.termux/files/usr/etc/tls/certs/ca-certificates.crt (214256 bytes)
  ✗ /system/etc/security/cacerts (not found)
  ...

Testing TLS Connection to GitHub:
  1. With system certificates...
     ✓ Success (status: 200 OK)

📋 Recommendations:
  ✓ Certificates appear to be installed
```

### 5. Debug Mode

Enable debug output to see exactly what's happening with certificate loading:

```bash
MORPHEUS_TLS_DEBUG=1 morpheus update
```

Debug output shows:
- Whether the system certificate pool loaded successfully
- Which certificate files were found and loaded
- Which certificate paths were skipped (and why)
- Total number of certificate bundles loaded
- Warnings if no certificates could be loaded

### 6. Better Error Messages

When a certificate error occurs, the user now receives helpful guidance:

```
⚠️  TLS Certificate Error Detected

This usually means CA certificates are not installed properly.

🔍 First, run diagnostics:
  morpheus diagnose-certs

💡 To fix this:
  • On Termux/Android: pkg install ca-certificates-java openssl
  • On Debian/Ubuntu: apt-get install ca-certificates
  • On Fedora/RHEL:   dnf install ca-certificates
  • On Alpine:        apk add ca-certificates

🐛 Debug mode:
  MORPHEUS_TLS_DEBUG=1 morpheus update

⚠️  Emergency bypass (NOT RECOMMENDED):
  MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

### 7. Emergency Bypass Option

For systems where installing CA certificates is not possible, users can bypass TLS verification (not recommended for security reasons):

```bash
MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

**Warning**: This disables certificate verification and should only be used as a last resort. A warning message is displayed when this option is used.

## How to Fix Certificate Errors

### Quick Fix for Termux/Android (Recommended)

The simplest solution on Termux/Android is to install curl:

```bash
pkg install curl
```

Morpheus will automatically use curl for all HTTPS requests on Android, which avoids certificate issues entirely.

### Step 1: Diagnose the Issue

First, run the built-in diagnostics tool:

```bash
morpheus diagnose-certs
```

This will check:
- Whether curl is available (recommended for Termux)
- System certificate pool status
- Available certificate file locations
- TLS connectivity to GitHub
- Provide specific recommendations for your system

### Step 2: Choose Your Fix

#### Option A: Install curl (Easiest - Termux/Android only)
```bash
pkg install curl
```

Morpheus will automatically detect and use curl. No certificate configuration needed!

#### Option B: Install CA Certificates (Alternative)
```bash
pkg update
pkg install ca-certificates-java openssl
```

**Note**: This is more complex and may not work as reliably as curl on Termux.

#### Debian/Ubuntu
```bash
sudo apt-get update
sudo apt-get install ca-certificates
```

#### Fedora/RHEL/CentOS
```bash
sudo dnf install ca-certificates
```

#### Alpine Linux
```bash
apk add ca-certificates
```

#### Arch Linux
```bash
sudo pacman -S ca-certificates
```

### Step 3: Debug Mode (if issues persist)

If you're still having issues after installing certificates, enable debug mode to see what's happening:

```bash
MORPHEUS_TLS_DEBUG=1 morpheus update
```

This will show:
- Which certificate bundles are being loaded
- Where the certificates are being found
- Any errors encountered while loading certificates

### Alternative: Use Environment Variable for Custom Certificates

If you have certificates in a custom location:

```bash
export SSL_CERT_FILE=/path/to/your/ca-certificates.crt
morpheus update
```

### Last Resort: Skip Verification (Not Recommended)

Only use this if you cannot install certificates and understand the security risks:

```bash
MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

## Testing

The fix includes comprehensive tests in `pkg/updater/updater_test.go`:

- Test normal TLS configuration creation
- Test TLS configuration with `MORPHEUS_SKIP_TLS_VERIFY` enabled
- Test HTTP client creation
- Test proper transport configuration

Run tests with:

```bash
go test ./pkg/updater/... -v
```

## Technical Details

### Code Changes

1. **pkg/updater/updater.go**:
   - **NEW**: Added `checkForUpdateCurl()` - automatic curl fallback for update checking
   - **NEW**: Added `downloadFileCurl()` - automatic curl fallback for binary downloads
   - **NEW**: Modified `CheckForUpdate()` to try HTTP first, then curl on certificate errors
   - **NEW**: Modified `downloadFile()` to try HTTP first, then curl on certificate errors
   - Added `crypto/tls` and `crypto/x509` imports
   - Created `createTLSConfig()` function with:
     - System certificate pool loading
     - Multi-platform certificate path search using `$PREFIX` for Termux
     - Debug mode support via `MORPHEUS_TLS_DEBUG` environment variable
     - Warning when no certificates are found
     - Support for `MORPHEUS_SKIP_TLS_VERIFY` environment variable
   - Updated `createHTTPClient()` to use the new TLS configuration
   - Both Android/Termux and standard platforms now use proper TLS
   - Refactored into `checkForUpdateHTTP()` and `parseReleaseInfo()` helper functions

2. **cmd/morpheus/main.go**:
   - Added `strings` import
   - Enhanced error handling in `handleUpdate()` to detect certificate errors
   - Updated error messages to suggest `diagnose-certs` command
   - Improved Termux-specific installation instructions (ca-certificates-java, openssl)
   - Added mention of debug mode
   - Added `diagnose-certs` command handler
   - Updated help text

3. **cmd/morpheus/diagnose-certs.go** (new file):
   - Comprehensive certificate diagnostics tool
   - Shows system information and environment variables
   - Checks system certificate pool status
   - Scans all known certificate locations
   - Tests TLS connectivity to GitHub API
   - Provides platform-specific recommendations

4. **pkg/updater/updater_test.go** (new file):
   - Added tests for TLS configuration
   - Added tests for HTTP client creation
   - Added tests for the skip verification flag

5. **docs/TLS_CERTIFICATE_FIX.md** (new file):
   - Comprehensive technical documentation
   - Troubleshooting guide
   - Platform-specific installation instructions

### Security Considerations

- TLS 1.2 is enforced as the minimum version
- System certificate pools are preferred
- Certificate verification is enabled by default
- The skip verification option prints a warning to stderr
- Manual download URL is always provided as a fallback

## Related Issues

This fix addresses certificate verification errors that were preventing users from using the `morpheus update` command on systems with missing or misconfigured CA certificates.

## Future Improvements

Potential enhancements for the future:

1. Add automatic certificate installation for Termux users
2. Implement certificate pinning for GitHub API endpoints
3. Add certificate expiry warnings
4. Support for custom CA certificate paths via config file
5. Better offline detection and error messaging
