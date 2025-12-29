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

Check if everything is set up correctly:

```bash
morpheus diagnose
```

This will tell you if curl is installed and if everything is ready.

After installing curl, run:

```bash
morpheus update
```

You should see the update check succeed.

## Still Having Issues?

If curl is installed and you're still having problems, run:

```bash
morpheus diagnose
```

And open an issue at https://github.com/nimsforest/morpheus/issues with the output.
