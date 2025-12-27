# Morpheus Control Server Setup

**TL;DR:** Run Morpheus from a cheap Hetzner server using your phone with Termux.

## Why a Control Server?

Morpheus is a CLI tool. If you work from a laptop, just run it there. But if you:
- ✅ Work from your phone/tablet
- ✅ Have a private GitHub repository
- ✅ Want a persistent environment

Then set up a small Hetzner server (~€4.50/month) to run Morpheus.

## Prerequisites

- Hetzner Cloud account
- GitHub account with access to Morpheus repository
- **Termux** app on your phone ([Android](https://f-droid.org/en/packages/com.termux/) | [iOS: use a-Shell](https://apps.apple.com/us/app/a-shell/id1473805438))

## Setup

### Step 1: Install Termux

**Android:** Install from [F-Droid](https://f-droid.org/en/packages/com.termux/) (NOT Google Play - it's outdated)

**iOS:** Install [a-Shell](https://apps.apple.com/us/app/a-shell/id1473805438) (similar to Termux)

### Step 2: Generate SSH Key in Termux

```bash
# Open Termux, install openssh
pkg install openssh

# Generate SSH key
ssh-keygen -t ed25519 -C "morpheus-phone"
# Press Enter for all prompts (default location)

# Show your public key
cat ~/.ssh/id_ed25519.pub
# Copy this entire output
```

### Step 3: Create Control Server

Open your phone browser, go to https://console.hetzner.cloud/

1. Create new project: "morpheus"
2. Go to Security → SSH Keys → Add SSH Key
   - Paste the public key from Termux
   - Name it "phone"
3. Add Server:
   - **Name:** `morpheus`
   - **Location:** `fsn1` (or any)
   - **Image:** Ubuntu 24.04
   - **Type:** CPX11 (~€4.50/month)
   - **SSH Key:** Select "phone"
4. Click "Create & Buy"
5. **Copy the server IP**

### Step 4: Connect from Termux

```bash
# In Termux, connect to your server
ssh root@YOUR_SERVER_IP

# If it asks "Are you sure?", type: yes
```

You're now on your control server!

### Step 5: Install Everything (One Command)

```bash
# Run this entire block (copy/paste into Termux):
apt update && apt install -y curl wget git make golang-go gh && \
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && \
source ~/.bashrc
```

This installs: Git, Go, GitHub CLI, Make

### Step 6: Authenticate GitHub (For Private Repos)

**Create GitHub Token:**
1. Open in phone browser: https://github.com/settings/tokens/new
2. Name: "morpheus"
3. Expiration: 90 days
4. Check: `repo` (full control)
5. Generate and **copy the token** (starts with `ghp_...`)

**Use the token:**
```bash
# Paste your token when you see it
echo 'ghp_YOUR_TOKEN_HERE' | gh auth login --with-token
```

### Step 7: Install Morpheus

```bash
# Clone and build
gh repo clone yourusername/morpheus
cd morpheus
make deps && make build && make install

# Verify
morpheus version
```

### Step 8: Configure Morpheus

**Get Hetzner API Token:**
1. Phone browser: https://console.hetzner.cloud/
2. Security → API Tokens → Generate
3. Permissions: Read & Write
4. Copy the token

**Set it up:**
```bash
# Add your token
echo 'export HETZNER_API_TOKEN="your_token_here"' >> ~/.bashrc
source ~/.bashrc

# Create config
cp config.example.yaml config.yaml
nano config.yaml
# Change ssh_key from "main" to "phone" (the name you used in Hetzner)
# Ctrl+X to save
```

### Step 9: Test It

```bash
morpheus version
morpheus plant cloud wood  # Creates 1 server
morpheus list              # Shows your forests
```

**Done!** 🎉

## Daily Usage from Phone

### Save SSH Connection in Termux

```bash
# In Termux on phone, edit config:
nano ~/.ssh/config

# Add this:
Host morpheus
  HostName YOUR_SERVER_IP
  User root
  IdentityFile ~/.ssh/id_ed25519

# Save (Ctrl+X)

# Now connect easily:
ssh morpheus
```

Then just use Morpheus commands:

```bash
morpheus plant cloud wood
morpheus list
morpheus status forest-123
morpheus teardown forest-123
```

## Common Issues

**Can't connect via SSH:**
```bash
# Make sure you copied the right public key to Hetzner
cat ~/.ssh/id_ed25519.pub
```

**"gh: command not found":**
```bash
apt update && apt install gh -y
```

**GitHub token expired:**
```bash
# Generate new token: https://github.com/settings/tokens
echo 'ghp_NEW_TOKEN' | gh auth login --with-token
```

**"failed to load config":**
```bash
# Check token is set
echo $HETZNER_API_TOKEN
# If empty, set it again:
echo 'export HETZNER_API_TOKEN="your_token"' >> ~/.bashrc
source ~/.bashrc
```

## Costs

- **Control Server:** CPX11 = €4.50/month
- **Each Forest Node:** CPX31 = €18/month

Example: Control server + 3-node forest = €4.50 + €54 = **€58.50/month**

Hetzner charges by the minute, so tear down forests when not needed!

## That's It!

You now have:
- ✅ Morpheus running on a Hetzner server
- ✅ Access from your phone via Termux

**Daily workflow:**
```bash
# Open Termux
ssh morpheus

# Use Morpheus commands
morpheus plant cloud wood
morpheus list
morpheus status forest-123
morpheus teardown forest-123
```

---

**Questions?** Open an issue: https://github.com/yourusername/morpheus/issues
