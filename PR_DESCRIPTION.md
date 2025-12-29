# Fix Termux/Android Update Failures - Use curl Instead

## Summary

Fixes update failures on Termux/Android by using `curl` for all HTTPS requests instead of Go's HTTP client. This completely eliminates certificate configuration issues.

**Before:**
```
morpheus version v1.2.4
🔍 Checking for updates...
Failed to check for updates: failed to check for updates: 
Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": 
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

**After:**
```bash
pkg install curl
morpheus update  # Works! ✅
```

## Problem

Users on Termux/Android were unable to use `morpheus update` due to TLS certificate verification failures. The Go HTTP client couldn't verify GitHub's SSL certificates on Android systems, even after installing CA certificates packages.

## Solution

**Use curl on Termux/Android** - curl handles TLS certificates properly out of the box on Termux, so we bypass Go's HTTP client entirely for Android systems.

### How It Works

1. **Automatic Detection**: Morpheus detects Termux/Android via `$TERMUX_VERSION`, `$ANDROID_ROOT`, or `runtime.GOOS == "android"`
2. **Curl First**: On Android, if curl is installed, use it for all HTTPS requests
3. **Clear Error**: If curl is not installed, show: `"curl is not installed. Install it with: pkg install curl"`
4. **Standard Fallback**: On non-Android systems, continue using Go's HTTP client

### Implementation

**`pkg/updater/updater.go`:**
- `CheckForUpdate()` - detects Android and uses curl
- `checkForUpdateCurl()` - fetches GitHub API via curl
- `downloadFile()` - detects Android and uses curl
- `downloadFileCurl()` - downloads binaries via curl

**`cmd/morpheus/diagnose.go`** (new):
- Simple diagnostics command to check if curl is installed
- Shows OS, architecture, and Termux detection status
- Provides clear instructions if curl is missing

**`cmd/morpheus/main.go`:**
- Added `diagnose` command
- Simplified error handling (removed TLS-specific messages)

## Changes

### Added
- `morpheus diagnose` - Check if curl is installed and system is ready for updates
- Curl-based HTTPS implementation for Termux/Android
- `TERMUX_UPDATE_FIX.md` - Simple user guide
- `CURL_SOLUTION.md` - Technical documentation

### Modified
- `pkg/updater/updater.go` - Use curl on Android, HTTP client elsewhere
- `cmd/morpheus/main.go` - Added diagnose command, simplified error messages
- `CHANGELOG.md` - Documented the fix

### Removed
- All TLS certificate configuration code (no longer needed for Android)
- Certificate path searching logic
- Complex certificate diagnostics
- TLS debug modes and bypass options

## User Experience

### For Termux Users

**Simple fix:**
```bash
pkg install curl
morpheus update
```

**Check if ready:**
```bash
morpheus diagnose
```

**Output:**
```
🔍 Morpheus Update Diagnostics
==============================

OS: android
Arch: arm64
Termux/Android detected: true

📋 Termux/Android Requirements:
  ✓ curl is installed at: /data/data/com.termux/files/usr/bin/curl
  ✓ curl version: curl 8.5.0 (aarch64-unknown-linux-android)

✅ Everything looks good!
   You can run: morpheus update
```

### For Non-Termux Users

No changes - continues to work as before using Go's HTTP client.

## Testing

✅ All tests pass:
```bash
go test ./...
```

✅ Builds successfully:
```bash
go build ./cmd/morpheus
```

✅ Commands work:
- `morpheus diagnose` - Shows system status
- `morpheus update` - Uses curl on Termux, HTTP client elsewhere
- `morpheus check-update` - Works on both platforms

## Benefits

- ✅ **Simple** - One command to fix: `pkg install curl`
- ✅ **Reliable** - curl works out of the box on Termux
- ✅ **No configuration** - No need to install or configure CA certificates
- ✅ **Clean code** - Removed complex TLS certificate handling for Android
- ✅ **Clear errors** - If curl is missing, tells user exactly what to install
- ✅ **Platform-aware** - Automatically detects and adapts to the environment

## Documentation

- `TERMUX_UPDATE_FIX.md` - Quick fix guide for Termux users
- `CURL_SOLUTION.md` - Technical details of the solution
- `CHANGELOG.md` - Updated with the fix

## Related Issues

Fixes certificate verification errors on Termux/Android that prevented users from using `morpheus update`.
