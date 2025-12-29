# Curl Solution for Termux Updates

## The Problem

Users on Termux/Android were getting HTTPS errors when trying to run `morpheus update`.

## The Solution

Morpheus now uses **curl** for all HTTPS requests on Termux/Android.

## For Users

Simply install curl:

```bash
pkg install curl
```

Then morpheus update will work:

```bash
morpheus update
```

## How It Works

1. Morpheus detects if it's running on Termux/Android
2. If curl is installed, it uses curl for HTTPS requests
3. If curl is not installed, it shows an error with installation instructions

## Benefits

- ✅ Simple - just one package to install
- ✅ Reliable - curl works out of the box on Termux
- ✅ No configuration needed
- ✅ Clear error messages if curl is missing

## Diagnostics

Check if everything is set up correctly:

```bash
morpheus diagnose-certs
```

This will tell you:
- If you're on Termux/Android
- If curl is installed
- What to do if something is wrong

## Technical Details

- On Termux/Android: Uses `curl` command directly
- On other systems: Uses Go's built-in HTTP client
- Automatic detection via environment variables: `$TERMUX_VERSION`, `$ANDROID_ROOT`, or `runtime.GOOS == "android"`
- Both update checking and binary downloads use curl on Termux

## Code Changes

- `pkg/updater/updater.go`: Added `checkForUpdateCurl()` and `downloadFileCurl()` functions
- `CheckForUpdate()` and `downloadFile()` detect Android and use curl
- `cmd/morpheus/diagnose-certs.go`: Simplified to check curl on Termux
- Removed all TLS certificate configuration code for Android

## That's It!

No need to configure certificates, no complex setup. Just install curl and morpheus works.
