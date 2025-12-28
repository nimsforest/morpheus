#!/data/data/com.termux/files/usr/bin/bash
# Morpheus Quick Install Script for Termux
# This script automates the entire setup process
#
# Non-interactive installation - uses environment variables for configuration:
#   MORPHEUS_FORCE_CLONE=1    - Force re-clone if directory exists (default: skip)
#   MORPHEUS_SKIP_INSTALL=1   - Skip installing to PATH (default: install)
#   HETZNER_API_TOKEN=xxx     - Hetzner API token (will be saved to ~/.bashrc)

set -e

echo "🌲 Morpheus Termux Quick Installer (Non-Interactive)"
echo "===================================================="
echo ""
echo "This script will:"
echo "  1. Install required packages (Go, Git, Make, OpenSSH)"
echo "  2. Clone Morpheus repository"
echo "  3. Build Morpheus binary"
echo "  4. Set up configuration"
echo "  5. Generate SSH key (if needed)"
echo "  6. Install to PATH"
echo ""
echo "Starting installation..."
echo ""

# Step 1: Update and install packages
echo ""
echo "📦 Step 1/5: Installing packages..."
echo "----"
pkg update -y
pkg install -y git golang make openssh

# Verify installations
echo ""
echo "Verifying installations:"
go version
git --version
make --version
ssh -V 2>&1 | head -1

# Step 2: Clone repository
echo ""
echo "📥 Step 2/5: Cloning Morpheus repository..."
echo "----"
if [[ -d "$HOME/morpheus" ]]; then
    if [[ "${MORPHEUS_FORCE_CLONE}" == "1" ]]; then
        echo "⚠️  Directory $HOME/morpheus exists. MORPHEUS_FORCE_CLONE=1, removing..."
        rm -rf "$HOME/morpheus"
        cd "$HOME"
        git clone https://github.com/nimsforest/morpheus.git
        cd morpheus
    else
        echo "⚠️  Directory $HOME/morpheus already exists. Using existing directory."
        echo "    (Set MORPHEUS_FORCE_CLONE=1 to force re-clone)"
        cd "$HOME/morpheus"
    fi
else
    cd "$HOME"
    git clone https://github.com/nimsforest/morpheus.git
    cd morpheus
fi

# Step 3: Build
echo ""
echo "🔨 Step 3/5: Building Morpheus..."
echo "----"
echo "This may take 2-5 minutes on first build..."
make deps
make build

# Verify build
if [[ -f "./bin/morpheus" ]]; then
    echo "✓ Build successful!"
    ./bin/morpheus version
else
    echo "✗ Build failed. Check errors above."
    exit 1
fi

# Step 4: Configuration
echo ""
echo "⚙️  Step 4/5: Setting up configuration..."
echo "----"

mkdir -p "$HOME/.morpheus"

if [[ ! -f "$HOME/.morpheus/config.yaml" ]]; then
    cp config.example.yaml "$HOME/.morpheus/config.yaml"
    echo "✓ Created config at ~/.morpheus/config.yaml"
else
    echo "⚠️  Config already exists at ~/.morpheus/config.yaml"
fi

# Check for API token
if [[ -z "$HETZNER_API_TOKEN" ]]; then
    echo ""
    echo "⚠️  HETZNER_API_TOKEN not set."
    echo ""
    echo "To get your Hetzner API token:"
    echo "  1. Open: https://console.hetzner.cloud/"
    echo "  2. Go to: Security → API Tokens"
    echo "  3. Generate new token (Read & Write permissions)"
    echo "  4. Copy the token"
    echo ""
    echo "Set it before running this script:"
    echo "  export HETZNER_API_TOKEN=\"your_token\""
    echo ""
    echo "Or add it to ~/.bashrc for persistence:"
    echo "  echo 'export HETZNER_API_TOKEN=\"your_token\"' >> ~/.bashrc"
    echo ""
else
    # Token is set, save it to .bashrc if not already there
    if ! grep -q "HETZNER_API_TOKEN" "$HOME/.bashrc" 2>/dev/null; then
        echo "export HETZNER_API_TOKEN=\"$HETZNER_API_TOKEN\"" >> "$HOME/.bashrc"
        echo "✓ Token saved to ~/.bashrc"
    else
        echo "✓ HETZNER_API_TOKEN is set"
    fi
fi

# Step 5: SSH Key
echo ""
echo "🔑 Step 5/5: Checking SSH key..."
echo "----"

if [[ -f "$HOME/.ssh/id_ed25519" ]]; then
    echo "✓ SSH key already exists: ~/.ssh/id_ed25519"
else
    echo "No SSH key found. Generating new key..."
    mkdir -p "$HOME/.ssh"
    ssh-keygen -t ed25519 -C "morpheus-android" -f "$HOME/.ssh/id_ed25519" -N ""
    echo "✓ SSH key generated"
fi

echo ""
echo "📋 Your SSH public key:"
echo "----"
cat "$HOME/.ssh/id_ed25519.pub"
echo ""
echo "You need to upload this key to Hetzner:"
echo "  1. Open: https://console.hetzner.cloud/"
echo "  2. Go to: Security → SSH Keys"
echo "  3. Click 'Add SSH Key'"
echo "  4. Paste the key above"
echo "  5. Name it: android"
echo ""

# Install to PATH (default: yes)
echo ""
if [[ "${MORPHEUS_SKIP_INSTALL}" == "1" ]]; then
    echo "⚠️  MORPHEUS_SKIP_INSTALL=1, skipping PATH installation"
    echo "   Run with: ~/morpheus/bin/morpheus"
else
    echo "📦 Installing morpheus to PATH..."
    make install
    echo "✓ Morpheus installed to /data/data/com.termux/files/usr/bin/"
    echo "  You can now run 'morpheus' from anywhere."
fi

# Final instructions
echo ""
echo "==================================="
echo "✅ Installation Complete!"
echo "==================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Upload SSH key to Hetzner (see above)"
echo ""
echo "2. Edit config to match your SSH key name:"
echo "   nano ~/.morpheus/config.yaml"
echo "   (Change 'ssh_key: main' to 'ssh_key: android')"
echo ""
echo "3. Test Morpheus:"
echo "   morpheus version"
echo ""
echo "4. Create your first forest:"
echo "   morpheus plant cloud wood"
echo ""
echo "5. Check status:"
echo "   morpheus list"
echo ""
echo "For help: morpheus help"
echo "Documentation: ~/morpheus/docs/ANDROID_TERMUX.md"
echo ""
echo "Happy provisioning! 🌲"
