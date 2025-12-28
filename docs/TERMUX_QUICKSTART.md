# Morpheus on Termux - Quick Start

**Get Morpheus running on your Android phone in 10 minutes.**

## What You Need

- Android phone
- Internet connection (WiFi recommended for first install)
- Hetzner Cloud account (free to create)

## Installation (One Command!)

### Step 1: Install Termux

**Download from F-Droid** (NOT Google Play):
- https://f-droid.org/en/packages/com.termux/

### Step 2: Run Installer

Open Termux and paste this command:

```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

That's it! The installer will:
1. Install dependencies (Go, Git, Make, OpenSSH)
2. Clone Morpheus
3. Build the binary (~5 minutes)
4. Generate SSH key
5. Help you configure API token

## Configuration

**What gets installed:**
- Go (whatever version Termux provides - typically recent)
- Git, Make, OpenSSH
- Morpheus binary

The installer will ask you for:

### 1. Hetzner API Token

Get it here: https://console.hetzner.cloud/

1. Go to **Security** → **API Tokens**
2. Click **"Generate API Token"**
3. Name: `morpheus`
4. Permissions: **Read & Write**
5. Copy the token

Paste it when the installer asks.

### 2. SSH Key Upload

The installer will show you your SSH public key.

Upload it here: https://console.hetzner.cloud/

1. Go to **Security** → **SSH Keys**
2. Click **"Add SSH Key"**
3. Paste your public key
4. Name: `android`
5. Save

## First Test

After installation completes:

```bash
# Check version
morpheus version

# Create a test forest (1 server)
morpheus plant cloud wood

# Wait ~5-10 minutes for provisioning

# Check status
morpheus list

# Get details
morpheus status forest-<id>

# Clean up
morpheus teardown forest-<id>
```

## Daily Usage

```bash
# Create infrastructure
morpheus plant cloud wood      # 1 server
morpheus plant cloud forest    # 3 servers
morpheus plant cloud jungle    # 5 servers

# Manage
morpheus list                  # List all
morpheus status forest-123     # Details
morpheus teardown forest-123   # Delete

# Help
morpheus help
```

## Troubleshooting

### "curl: command not found"

Install curl first:
```bash
pkg install curl -y
```

Then run the installer again.

### "Permission denied"

The installer handles this, but if you installed manually:
```bash
chmod +x ~/morpheus/bin/morpheus
```

### "Failed to load config"

```bash
# Check token is set
echo $HETZNER_API_TOKEN

# If empty, set it:
export HETZNER_API_TOKEN="your_token_here"
echo 'export HETZNER_API_TOKEN="your_token"' >> ~/.bashrc
source ~/.bashrc
```

### "SSH key not found"

```bash
# Edit config to match your SSH key name in Hetzner
nano ~/.morpheus/config.yaml

# Change this line:
ssh_key: android  # Must match name in Hetzner console
```

### Build fails or takes forever

```bash
# Make sure you have enough storage
df -h

# Termux needs at least 500MB free
# Clear some space and try again
```

## What's Next?

Once Morpheus is working:

1. **Provision infrastructure** for your projects
2. **Integrate with NimsForest** for NATS clustering
3. **Manage on the go** from your phone
4. **Save money** vs running a control server (€54/year)

## More Info

- **Full Guide:** [ANDROID_TERMUX.md](ANDROID_TERMUX.md)
- **Philosophy:** [MOBILE_PHILOSOPHY.md](MOBILE_PHILOSOPHY.md)
- **Troubleshooting:** [ANDROID_TERMUX.md#troubleshooting](ANDROID_TERMUX.md#troubleshooting)

## Need Help?

Open an issue: https://github.com/nimsforest/morpheus/issues

Include:
- Command you ran
- Error message
- Output of: `go version` and `uname -a`

---

**Happy provisioning from your pocket!** 🌲📱
