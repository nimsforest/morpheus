# Install Morpheus on Termux

## One Command Install

```bash
wget -qO- https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

Or with curl:

```bash
curl -sSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

## Prerequisites

1. **Termux** from F-Droid: https://f-droid.org/en/packages/com.termux/
2. **Hetzner account** (free): https://console.hetzner.cloud/

## That's It!

The installer will guide you through:
- Getting your Hetzner API token
- Uploading your SSH key
- Building Morpheus

Takes ~10 minutes total.

## After Install

```bash
morpheus plant cloud wood      # Create infrastructure
morpheus list                  # View forests
morpheus status forest-123     # Check details
morpheus teardown forest-123   # Clean up
```

## More Info

- **Quick Start**: [docs/TERMUX_QUICKSTART.md](docs/TERMUX_QUICKSTART.md)
- **Full Guide**: [docs/ANDROID_TERMUX.md](docs/ANDROID_TERMUX.md)
- **Help**: https://github.com/nimsforest/morpheus/issues
