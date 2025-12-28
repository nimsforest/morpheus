# Running Morpheus CLI Directly on Android with Termux

**TL;DR:** Yes! You can run Morpheus CLI natively on your Android phone using Termux.

## Why Run Morpheus on Android?

**Because Morpheus is a CLI tool, and Termux is a terminal!** This is the natural way to use Morpheus on mobile:

- ✅ Native CLI experience - use Morpheus directly, not over SSH
- ✅ Run Morpheus anywhere - on the go, from your phone
- ✅ Free - no control server costs (~€4.50/month savings)
- ✅ Simple - no SSH, no remote server to maintain
- ✅ Works offline (for local commands like `list`, `status`)
- ✅ Full functionality - all commands work exactly as on desktop

## Requirements

- Android phone (ARM64 recommended, ARM32 supported)
- Termux app installed
- At least 500MB free storage
- Internet connection for provisioning

## Installation Guide

### Quick Install (Recommended)

**One command install!** The script runs non-interactively - no prompts or questions.

**Step 1: Get your Hetzner API token (optional but recommended)**

Get your token from the [Hetzner Cloud Console](https://console.hetzner.cloud/). See the [main README](../README.md#get-hetzner-api-token) for step-by-step instructions.

**Step 2: Run the installer**

```bash
# Recommended: Set token first
export HETZNER_API_TOKEN="your_token_here"
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# Or: Set token later manually
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

This automatically:
- ✅ Installs all dependencies (Go, Git, Make, OpenSSH)
- ✅ Clones the Morpheus repository
- ✅ Builds the binary (~5 minutes)
- ✅ Sets up configuration
- ✅ Generates SSH key if needed
- ✅ Saves API token to `~/.bashrc` (if set)
- ✅ Installs to PATH

**Environment variables (optional):**
- `HETZNER_API_TOKEN` - Your API token
- `MORPHEUS_FORCE_CLONE=1` - Force re-clone if directory exists
- `MORPHEUS_SKIP_INSTALL=1` - Skip installing to PATH

**Time:** 10 minutes total

---

### Manual Installation (If you prefer to do it step-by-step)

### Step 1: Install Termux

**Download from F-Droid** (NOT Google Play - it's outdated):
- Open: https://f-droid.org/en/packages/com.termux/
- Install Termux

### Step 2: Setup Termux Environment

Open Termux and run:

```bash
# Update package repositories
pkg update && pkg upgrade -y

# Install required packages
pkg install git golang make openssh -y

# Verify Go installation
go version
# Will show whatever Go version Termux provides (typically recent)
```

**Optional - Check compatibility first:**

```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/check-termux.sh | bash
```

**Note:** Termux automatically installs whatever Go version is in their repository (usually recent). Morpheus requires Go 1.25 to build.

### Step 3: Clone and Build Morpheus

```bash
# Clone the repository
git clone https://github.com/nimsforest/morpheus.git
cd morpheus

# Build for Android
make build

# Install (optional - adds to PATH)
make install

# Or run directly from bin/
./bin/morpheus version
```

### Step 4: Generate SSH Key

Morpheus needs an SSH key to access provisioned servers:

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "morpheus-android"
# Press Enter for all prompts (use defaults)

# Display your public key
cat ~/.ssh/id_ed25519.pub
# Copy this entire output - you'll upload it to Hetzner
```

### Step 5: Upload SSH Key to Hetzner

Using your phone browser:

1. Go to: https://console.hetzner.cloud/
2. Navigate to: **Security → SSH Keys**
3. Click **"Add SSH Key"**
4. Paste your public key from Step 4
5. Name it: `android` (or whatever you prefer)
6. Save

### Step 6: Get Hetzner API Token

Get your token from the [Hetzner Cloud Console](https://console.hetzner.cloud/). See the [main README](../README.md#get-hetzner-api-token) for detailed instructions.

### Step 7: Configure Morpheus

```bash
# Set API token as environment variable
echo 'export HETZNER_API_TOKEN="your_token_here"' >> ~/.bashrc
source ~/.bashrc

# Create config directory
mkdir -p ~/.morpheus

# Copy example config
cp config.example.yaml ~/.morpheus/config.yaml

# Edit the config
nano ~/.morpheus/config.yaml
```

In the config, make sure to set:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: android  # Must match your SSH key name in Hetzner!
  locations:
    - fsn1  # Falkenstein, Germany
    - nbg1  # Nuremberg, Germany

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"
```

Save and exit: `Ctrl+X`, then `Y`, then `Enter`

### Step 8: Test It!

```bash
# Check version
morpheus version

# Create a small test forest
morpheus plant cloud wood

# List forests
morpheus list

# Check status
morpheus status forest-<id>

# Clean up
morpheus teardown forest-<id>
```

## Daily Usage

### Quick Commands

```bash
# Create forests
morpheus plant cloud wood      # 1 node
morpheus plant cloud forest    # 3 nodes
morpheus plant cloud jungle    # 5 nodes

# Manage forests
morpheus list                  # List all
morpheus status forest-123     # Details
morpheus teardown forest-123   # Delete
```

### Termux Shortcuts

Add these to `~/.bashrc` for convenience:

```bash
# Quick aliases
alias mp='morpheus plant cloud'
alias ml='morpheus list'
alias ms='morpheus status'
alias mt='morpheus teardown'

# Quick CD to morpheus
alias cdm='cd ~/morpheus'
```

Reload: `source ~/.bashrc`

Now you can use:
```bash
mp wood           # Instead of: morpheus plant cloud wood
ml                # Instead of: morpheus list
ms forest-123     # Instead of: morpheus status forest-123
```

## Performance Considerations

### Build Time
- **First build:** 2-5 minutes (downloads dependencies)
- **Subsequent builds:** 30-60 seconds

### Provisioning Time
- **1 node (wood):** ~5-10 minutes
- **3 nodes (forest):** ~15-30 minutes
- **5 nodes (jungle):** ~25-50 minutes

### Battery Usage
- Morpheus is CPU-light (mostly API calls)
- Battery drain is minimal
- Use on WiFi to save mobile data

## Troubleshooting

### "Go version too old"

```bash
# Update Termux packages
pkg update && pkg upgrade -y
pkg install golang -y
go version
```

If Go is still too old, you may need to compile Go from source or use a newer Termux version.

### "Permission denied" when running morpheus

```bash
# Make binary executable
chmod +x ./bin/morpheus

# Or reinstall
make install
```

### "Failed to load config"

```bash
# Check config exists
ls -la ~/.morpheus/config.yaml

# Check token is set
echo $HETZNER_API_TOKEN

# If empty, set it again
export HETZNER_API_TOKEN="your_token"
echo 'export HETZNER_API_TOKEN="your_token"' >> ~/.bashrc
```

### "SSH key not found: android"

```bash
# List keys in Hetzner via hcloud CLI
pkg install hcloud -y
hcloud ssh-key list

# Or check via browser: https://console.hetzner.cloud/
# Update config.yaml to match exact key name
```

### Build fails with "cannot find package"

```bash
# Download dependencies first
make deps

# Then build
make build
```

### Termux keeps closing/connection lost

Termux may kill background processes. If provisioning takes long:

```bash
# Install termux-wake-lock
pkg install termux-api -y

# Keep screen on during provisioning
termux-wake-lock

# After done, release lock
termux-wake-unlock
```

## Architecture Notes

### Cross-Compilation

Morpheus is written in Go, which compiles natively for Android ARM/ARM64. The binary runs directly on your phone without emulation.

Termux provides:
- `linux/arm64` on 64-bit Android phones
- `linux/arm` on 32-bit Android phones

Both are fully supported by Go and Morpheus.

### Limitations

**What works:**
- ✅ All Morpheus commands (`plant`, `list`, `status`, `teardown`)
- ✅ API calls to Hetzner
- ✅ SSH key management
- ✅ Config file management
- ✅ Local registry (JSON storage)

**What might not work:**
- ❌ Some terminal formatting might look different
- ❌ Interactive prompts (use `yes | morpheus teardown` if needed)
- ❌ Large-scale operations (100+ nodes) might be slow

### Storage

Morpheus stores data in:
- **Config:** `~/.morpheus/config.yaml`
- **Registry:** `~/.morpheus/registry.json`
- **Binary:** `/data/data/com.termux/files/usr/bin/morpheus`

Total storage: ~50MB including Go toolchain and dependencies.

## Comparison: Termux vs Control Server

| Feature | Termux (Recommended) | Control Server |
|---------|----------------------|----------------|
| **Philosophy** | Direct CLI usage | Remote access workaround |
| **Cost** | Free | €4.50/month |
| **Setup Time** | 10-15 min | 15-20 min |
| **Complexity** | Simple (no SSH) | More complex (SSH, server) |
| **Performance** | Phone CPU | Server CPU |
| **Battery Usage** | Minimal | None (offloaded) |
| **Offline Commands** | Yes (list, status) | No |
| **Requires Internet** | For provisioning only | For all commands |
| **Persistent** | When phone is on | 24/7 always-on |
| **Best For** | Most users | Specific use cases* |

**\*Control Server is only needed when:**
- You need 24/7 always-on availability (CI/CD pipelines)
- Multiple team members share the same Morpheus instance
- Running very long operations and phone can't stay on
- Integrating with automated workflows

**For 90% of users: Use Termux directly!** It's simpler, free, and the natural way to use a CLI tool.

## Alternative: Hybrid Approach

You can use both!

1. **Development/Testing:** Run Morpheus natively on Termux
2. **Production:** SSH to a control server for stability

```bash
# In Termux, create alias to control server
echo 'alias morpheus-prod="ssh root@YOUR_SERVER_IP morpheus"' >> ~/.bashrc
source ~/.bashrc

# Now use both:
morpheus plant cloud wood        # Local (Android)
morpheus-prod plant cloud forest # Remote (Server)
```

## Security Considerations

### Storing API Tokens

Your Hetzner API token is sensitive! Protect it:

```bash
# Make sure .bashrc is not world-readable
chmod 600 ~/.bashrc

# Never commit config files with tokens
# Use environment variables only
```

### SSH Private Keys

```bash
# Protect your SSH private key
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
```

### Termux Security

- Keep Termux updated: `pkg upgrade`
- Use strong device lock screen
- Consider encrypting your Android device

## Tips & Tricks

### 1. Background Provisioning

```bash
# Run in background (but keep Termux open)
morpheus plant cloud forest &

# Check if still running
jobs

# View logs
tail -f ~/.morpheus/morpheus.log  # If logging is enabled
```

### 2. Quick Status Check Widget

Create a shortcut script:

```bash
nano ~/check-forests.sh
```

Add:
```bash
#!/data/data/com.termux/files/usr/bin/bash
morpheus list
```

Make executable:
```bash
chmod +x ~/check-forests.sh
```

Run anytime:
```bash
~/check-forests.sh
```

### 3. Notifications on Completion

```bash
# Install termux-api
pkg install termux-api -y

# Provision with notification
morpheus plant cloud wood && termux-notification --title "Morpheus" --content "Forest provisioned!"
```

### 4. Save Commands History

Termux saves your command history. Search with:
- Press `Ctrl+R`
- Type part of command (e.g., "plant")
- Press `Enter` to run

## FAQ

**Q: Is this really the recommended way to use Morpheus on mobile?**  
A: Yes! Morpheus is a CLI tool. Termux is a terminal. Running it directly is the natural approach. The control server is only for specific use cases (24/7 availability, team collaboration, CI/CD).

**Q: Does this work on iPhone/iOS?**  
A: No. iOS doesn't support Termux. For iOS, you'll need to use the [Control Server approach](CONTROL_SERVER_SETUP.md) with SSH via a-Shell app.

**Q: Can I run Morpheus on a tablet?**  
A: Yes! Same steps as phone. More screen space = better experience.

**Q: Will this drain my battery?**  
A: No. Morpheus is CPU-light. Most time is spent waiting for Hetzner API responses, not computing.

**Q: Can I use mobile data instead of WiFi?**  
A: Yes, but be aware of data usage. Provisioning uses minimal data (~1-5MB per node), mostly API calls.

**Q: What if Termux crashes during provisioning?**  
A: Hetzner servers are already created. Check with `morpheus list` or Hetzner console. You can resume or teardown.

**Q: Can I provision from multiple devices?**  
A: Yes! Share the same `~/.morpheus/registry.json` via cloud storage (Syncthing, Dropbox, etc.) or use a shared control server.

**Q: Is this production-ready?**  
A: Absolutely! This is the primary way to use Morpheus on mobile. For team/enterprise use with specific requirements (24/7, CI/CD), consider a control server.

## Next Steps

Now that you have Morpheus running on Android:

1. **Test with a small forest:** `morpheus plant cloud wood`
2. **Explore NimsForest integration:** Install NimsForest on provisioned servers
3. **Set up monitoring:** Use `morpheus status` regularly
4. **Join the community:** Report issues, share tips!

## Resources

- **Termux Wiki:** https://wiki.termux.com/
- **Go on Android:** https://go.dev/
- **Hetzner Cloud:** https://console.hetzner.cloud/
- **Morpheus Issues:** https://github.com/nimsforest/morpheus/issues

---

**Happy provisioning from your pocket!** 📱🌲
