# Termux Update Fix

If `morpheus update` fails on Termux, you need to install curl.

## Solution

```bash
pkg install curl
```

That's it! Morpheus uses curl for all HTTPS requests on Termux/Android.

## Why curl?

On Termux/Android, morpheus uses `curl` for downloading updates because it works reliably out of the box.

## Verification

After installing curl, verify it works:

```bash
morpheus update
```

You should see the update check succeed.

## Still Having Issues?

If curl is installed and you're still having problems, open an issue at:
https://github.com/nimsforest/morpheus/issues

Include the output of:
```bash
which curl
curl --version
morpheus version
```
