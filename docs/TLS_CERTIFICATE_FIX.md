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

The fix includes several improvements to the HTTP client configuration:

### 1. Proper TLS Configuration

- Added `crypto/tls` and `crypto/x509` imports
- Created `createTLSConfig()` function that:
  - Loads system CA certificate pool
  - Searches for CA certificates in common locations across different platforms
  - Supports custom certificate paths via `SSL_CERT_FILE` environment variable
  - Sets minimum TLS version to 1.2 for security

### 2. Multi-Platform Certificate Support

The updater now searches for CA certificates in these locations:

- `/system/etc/security/cacerts` - Android system certificates
- `/data/data/com.termux/files/usr/etc/tls/certs/ca-certificates.crt` - Termux
- `/etc/ssl/certs/ca-certificates.crt` - Debian/Ubuntu/Gentoo
- `/etc/pki/tls/certs/ca-bundle.crt` - Fedora/RHEL
- `/etc/ssl/ca-bundle.pem` - OpenSUSE
- `/etc/ssl/cert.pem` - OpenBSD
- `/usr/local/share/certs/ca-root-nss.crt` - FreeBSD
- `/etc/pki/tls/cacert.pem` - OpenELEC
- `/etc/certs/ca-certificates.crt` - Solaris 11.2+
- Custom path from `$SSL_CERT_FILE` environment variable

### 3. Better Error Messages

When a certificate error occurs, the user now receives helpful guidance:

```
⚠️  TLS Certificate Error Detected

This usually means CA certificates are not installed on your system.

To fix this:
  • On Termux/Android: pkg install ca-certificates
  • On Debian/Ubuntu: apt-get install ca-certificates
  • On Fedora/RHEL:   dnf install ca-certificates
  • On Alpine:        apk add ca-certificates

Alternatively (NOT RECOMMENDED), you can skip certificate verification:
  MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

### 4. Emergency Bypass Option

For systems where installing CA certificates is not possible, users can bypass TLS verification (not recommended for security reasons):

```bash
MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update
```

**Warning**: This disables certificate verification and should only be used as a last resort.

## How to Fix Certificate Errors

### Recommended Solution: Install CA Certificates

#### Termux/Android
```bash
pkg install ca-certificates
```

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
   - Added `crypto/tls` and `crypto/x509` imports
   - Created `createTLSConfig()` function
   - Updated `createHTTPClient()` to use the new TLS configuration
   - Added support for `MORPHEUS_SKIP_TLS_VERIFY` environment variable

2. **cmd/morpheus/main.go**:
   - Added `strings` import
   - Enhanced error handling in `handleUpdate()` to detect certificate errors
   - Added helpful guidance messages for certificate issues

3. **pkg/updater/updater_test.go** (new file):
   - Added tests for TLS configuration
   - Added tests for HTTP client creation
   - Added tests for the skip verification flag

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
