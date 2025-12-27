# Morpheus Control Server Setup

This guide covers setting up a dedicated server to run Morpheus, especially useful for:
- Mobile-only workflows (working from phone/tablet)
- Private GitHub repositories
- CI/CD automation
- Multi-user team environments

## Why a Control Server?

Morpheus is a CLI tool that doesn't need to run 24/7. However, you might want a dedicated server if:

- ✅ You work primarily from a mobile device
- ✅ Your repository is private and needs authentication
- ✅ You want a consistent environment for infrastructure management
- ✅ Multiple team members need shared access
- ✅ You're integrating with automation/CI/CD

## Quick Setup

### Prerequisites

- Hetzner Cloud account
- GitHub account with access to Morpheus repository
- SSH client on your device (Termux, JuiceSSH, Blink Shell, etc.)

### Step 1: Create Control Server

**Via Hetzner Cloud Console (Web UI):**

1. Go to https://console.hetzner.cloud/
2. Create new project (e.g., "morpheus-control")
3. Add Server:
   - **Name:** `morpheus-control`
   - **Location:** Any (e.g., `fsn1`)
   - **Image:** Ubuntu 24.04
   - **Type:** CPX11 (2 vCPU, 2GB RAM, ~€4.50/month)
   - **SSH Key:** Add your SSH public key
4. Click "Create & Buy"
5. Note the server IP address

**Via Hetzner CLI:**

```bash
# If you have hcloud CLI already
hcloud server create \
  --name morpheus-control \
  --type cpx11 \
  --image ubuntu-24.04 \
  --ssh-key main \
  --location fsn1
```

### Step 2: Initial Server Setup

```bash
# SSH to your new server
ssh root@YOUR_SERVER_IP

# Update system
apt update && apt upgrade -y

# Install required tools
apt install -y curl wget git make
```

### Step 3: Install Go

```bash
# Download and install Go 1.23+
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
rm go1.23.4.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

### Step 4: Install GitHub CLI

```bash
# Install GitHub CLI
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
  dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
  tee /etc/apt/sources.list.d/github-cli.list > /dev/null

apt update
apt install gh -y

# Verify
gh --version
```

### Step 5: Authenticate with GitHub (Private Repo)

**Method A: Token Authentication (Easiest for Mobile)**

1. Create a Personal Access Token:
   - Go to: https://github.com/settings/tokens/new
   - **Note:** "Morpheus Control Server"
   - **Expiration:** 90 days or longer
   - **Scopes:** Check `repo` (Full control of private repositories)
   - Click **Generate token**
   - **Copy the token** (starts with `ghp_...`)

2. Authenticate on server:

```bash
# Paste your token when prompted (it won't be visible)
echo 'ghp_YOUR_TOKEN_HERE' | gh auth login --with-token

# Verify authentication
gh auth status
```

**Method B: Browser Authentication**

```bash
# Follow interactive prompts
gh auth login

# Choose:
# - GitHub.com
# - HTTPS
# - Login with a web browser
# - Enter the code shown in your mobile browser
```

**Method C: Direct Git with Token (No gh CLI needed)**

```bash
# Clone directly with token in URL
git clone https://ghp_YOUR_TOKEN_HERE@github.com/yourusername/morpheus.git
```

### Step 6: Clone and Build Morpheus

```bash
# Clone repository
gh repo clone yourusername/morpheus
# Or if using method C above, skip this

cd morpheus

# Download dependencies
make deps

# Build binary
make build

# Install globally (optional, but convenient)
make install

# Verify installation
morpheus version
```

### Step 7: Configure Morpheus

```bash
# Get your Hetzner API token from:
# https://console.hetzner.cloud/ → Security → API Tokens

# Set environment variable
export HETZNER_API_TOKEN="your_hetzner_api_token"

# Make it persistent
echo 'export HETZNER_API_TOKEN="your_hetzner_api_token"' >> ~/.bashrc

# Copy example config
cp config.example.yaml config.yaml

# Edit config (use nano, vi, or your preferred editor)
nano config.yaml
```

**Update config.yaml:**

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cpx31
    image: ubuntu-24.04
    ssh_key: main  # ← Change to match your Hetzner SSH key name
  locations:
    - fsn1
    - nbg1
    - hel1

integration:
  nimsforest_url: ""
  registry_url: ""

secrets:
  hetzner_api_token: "${HETZNER_API_TOKEN}"
```

### Step 8: Test Morpheus

```bash
# Test installation
morpheus version
morpheus help

# Optional: Provision a test forest
morpheus plant cloud wood

# Check status
morpheus list

# Clean up test
morpheus teardown forest-XXXXX
```

## Usage from Mobile

### Save SSH Connection

Most mobile SSH clients let you save connections:

**Termux (Android):**
```bash
# Add alias to your local ~/.bashrc
alias morpheus-server='ssh root@YOUR_SERVER_IP'

# Then just:
morpheus-server
```

**Blink Shell (iOS):**
- Settings → Hosts → Add New Host
- Name: `morpheus`
- Host: `YOUR_SERVER_IP`
- User: `root`
- Then connect with: `ssh morpheus`

### Useful Aliases

Add these to `~/.bashrc` on your control server for faster typing:

```bash
# Morpheus shortcuts
alias mp='morpheus plant cloud'
alias ml='morpheus list'
alias ms='morpheus status'
alias mt='morpheus teardown'
alias mh='morpheus help'

# Quick commands
alias forests='morpheus list'
alias plant-wood='morpheus plant cloud wood'
alias plant-forest='morpheus plant cloud forest'
```

Then reload:
```bash
source ~/.bashrc
```

**Usage:**
```bash
# Instead of: morpheus plant cloud wood
mp wood

# Instead of: morpheus list
ml

# Instead of: morpheus status forest-123
ms forest-123
```

## Security Best Practices

### 1. Protect Your Tokens

```bash
# Never commit tokens to git
echo ".env" >> .gitignore
echo "config.yaml" >> .gitignore

# Use environment variables
export HETZNER_API_TOKEN="xxx"
export GITHUB_TOKEN="ghp_xxx"
```

### 2. Rotate Tokens Regularly

- GitHub tokens: Set 90-day expiration
- Hetzner tokens: Rotate every 6 months
- Update server when rotating:

```bash
ssh root@YOUR_SERVER_IP
nano ~/.bashrc  # Update HETZNER_API_TOKEN
source ~/.bashrc
```

### 3. Secure SSH Access

```bash
# Disable password authentication (key-only)
nano /etc/ssh/sshd_config
# Set: PasswordAuthentication no
systemctl restart sshd

# Optional: Change SSH port
# Set: Port 2222
# Remember to update Hetzner firewall rules
```

### 4. Firewall Configuration

```bash
# Enable UFW
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp comment 'SSH'
ufw --force enable

# Check status
ufw status
```

### 5. Regular Updates

```bash
# Schedule weekly updates
crontab -e

# Add:
0 2 * * 0 apt update && apt upgrade -y && apt autoremove -y
```

## Troubleshooting

### "Permission denied (publickey)"

```bash
# Verify SSH key is added to Hetzner
hcloud ssh-key list

# Or add it manually:
cat ~/.ssh/id_ed25519.pub | ssh root@SERVER_IP "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

### "gh: command not found"

```bash
# Reinstall GitHub CLI
apt update && apt install gh -y
```

### "failed to load config"

```bash
# Check config exists
ls -la ~/.morpheus/config.yaml
ls -la ./config.yaml

# Check environment variable
echo $HETZNER_API_TOKEN

# Reload bashrc
source ~/.bashrc
```

### "dial tcp: lookup api.hetzner.cloud: no such host"

```bash
# Check DNS
cat /etc/resolv.conf

# Test connectivity
ping -c 3 api.hetzner.cloud
curl https://api.hetzner.cloud/v1/datacenters
```

### GitHub Token Expired

```bash
# Generate new token at: https://github.com/settings/tokens
# Re-authenticate:
echo 'ghp_NEW_TOKEN' | gh auth login --with-token
gh auth status
```

## Cost Estimate

**Monthly Costs (Hetzner):**

- **Control Server:** CPX11 (~€4.50/month)
- **Provisioned Forests:**
  - Wood (1 node): CPX31 = ~€18/month
  - Forest (3 nodes): 3x CPX31 = ~€54/month
  - Jungle (5 nodes): 5x CPX31 = ~€90/month

**Total Example:** Control server + 1 forest = ~€58.50/month

**Tips to Save:**
- Use smaller instance types for testing (CPX11/CPX21)
- Teardown test forests when not in use
- Hetzner charges by the minute, so you can create/destroy frequently

## Advanced: Multi-User Setup

### Create Non-Root User

```bash
# Create morpheus user
adduser morpheus
usermod -aG sudo morpheus

# Setup for morpheus user
su - morpheus
cd /home/morpheus
gh repo clone yourusername/morpheus
cd morpheus
make build

# Each user has their own registry
ls ~/.morpheus/registry.json
```

### Shared Registry (Optional)

```bash
# Create shared directory
mkdir -p /opt/morpheus/shared
chmod 755 /opt/morpheus/shared

# Point users to shared registry
export MORPHEUS_REGISTRY=/opt/morpheus/shared/registry.json
```

## Alternative: Docker-based Setup

If you prefer running Morpheus in a container:

```bash
# Create Dockerfile
cat > Dockerfile <<'EOF'
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/morpheus /usr/local/bin/
ENTRYPOINT ["morpheus"]
EOF

# Build and run
docker build -t morpheus:latest .
docker run -it \
  -e HETZNER_API_TOKEN=$HETZNER_API_TOKEN \
  -v ~/.morpheus:/root/.morpheus \
  morpheus:latest plant cloud wood
```

## Next Steps

After setting up your control server:

1. **Test provisioning:** `morpheus plant cloud wood`
2. **Check status:** `morpheus list`
3. **Set up NimsForest** (if using): See NimsForest documentation
4. **Create automation** (optional): See [CI/CD Integration](CI_CD_INTEGRATION.md)
5. **Join community:** GitHub Discussions for questions

## Related Documentation

- [README.md](../README.md) - Main documentation
- [SEPARATION_OF_CONCERNS.md](SEPARATION_OF_CONCERNS.md) - Architecture overview
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Development guide

## Support

- **Issues:** https://github.com/yourusername/morpheus/issues
- **Discussions:** https://github.com/yourusername/morpheus/discussions
- **Security:** security@yourproject.com

---

**Last Updated:** December 27, 2025  
**Tested On:** Ubuntu 24.04 LTS, Hetzner Cloud
