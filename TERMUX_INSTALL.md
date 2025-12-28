# 📱 Install Morpheus on Your Android Phone

## One Command - That's It!

Open Termux and paste this:

```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

---

## Before You Start

### 1. Install Termux
- **Get it from F-Droid** (NOT Google Play): https://f-droid.org/en/packages/com.termux/
- Google Play version is outdated and won't work

### 2. Get a Hetzner Account (Free)
- Sign up at: https://console.hetzner.cloud/
- You'll need this for the API token

---

## What the Installer Does

The script automatically:
1. ✅ Installs dependencies (Go, Git, Make, OpenSSH)
2. ✅ Clones the Morpheus repository
3. ✅ Builds the binary (~5 minutes)
4. ✅ Generates an SSH key
5. ✅ Helps you configure your API token

**Total time:** ~10 minutes

---

## During Installation

### You'll Need:

**1. Hetzner API Token**
- Go to: https://console.hetzner.cloud/
- Click: Security → API Tokens → Generate
- Permissions: **Read & Write**
- Copy the token when the installer asks

**2. Upload SSH Key**
- The installer will show you your public key
- Go to: https://console.hetzner.cloud/
- Click: Security → SSH Keys → Add SSH Key
- Paste your key
- Name it: `android`

---

## After Installation

Test it works:
```bash
morpheus version
```

Create your first infrastructure:
```bash
# Create 1 server (~€18/month)
morpheus plant cloud wood

# Wait ~5-10 minutes for provisioning

# Check status
morpheus list
morpheus status forest-<id>

# Clean up when done
morpheus teardown forest-<id>
```

---

## Quick Commands

```bash
# Create infrastructure
morpheus plant cloud wood      # 1 server
morpheus plant cloud forest    # 3 servers (cluster)
morpheus plant cloud jungle    # 5 servers (large cluster)

# Manage
morpheus list                  # List all forests
morpheus status forest-123     # Check details
morpheus teardown forest-123   # Delete forest
morpheus help                  # Show help
```

---

## Troubleshooting

### "curl: command not found"
Install curl first:
```bash
pkg install curl -y
```

### "Failed to load config"
Set your API token:
```bash
export HETZNER_API_TOKEN="your_token_here"
echo 'export HETZNER_API_TOKEN="your_token"' >> ~/.bashrc
```

### "SSH key not found: android"
Edit config to match your SSH key name in Hetzner:
```bash
nano ~/.morpheus/config.yaml
# Change: ssh_key: android
```

---

## More Help

- **Quick Start Guide**: [docs/TERMUX_QUICKSTART.md](https://github.com/nimsforest/morpheus/blob/main/docs/TERMUX_QUICKSTART.md)
- **Full Documentation**: [docs/ANDROID_TERMUX.md](https://github.com/nimsforest/morpheus/blob/main/docs/ANDROID_TERMUX.md)
- **Report Issues**: https://github.com/nimsforest/morpheus/issues

---

## Why Termux?

**Morpheus is a CLI tool. Termux is a terminal. This is the natural way.**

No need for a control server, no monthly costs, just direct CLI usage like on desktop.

Read more: [Mobile Philosophy](https://github.com/nimsforest/morpheus/blob/main/docs/MOBILE_PHILOSOPHY.md)

---

**Happy provisioning from your pocket!** 🌲📱
