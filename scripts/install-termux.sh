#!/data/data/com.termux/files/usr/bin/bash
# Morpheus Quick Install Script for Termux
# This script automates the entire setup process
#
# Non-interactive installation - uses environment variables for configuration:
#   MORPHEUS_FORCE_CLONE=1    - Force re-clone if directory exists (default: skip)
#   MORPHEUS_SKIP_INSTALL=1   - Skip installing to PATH (default: install)
#   MORPHEUS_BUILD_FROM_SOURCE=1 - Force build from source instead of downloading binary
#   HETZNER_API_TOKEN=xxx     - Hetzner API token (will be saved to ~/.bashrc)

set -e

echo "🌲 Morpheus Termux Quick Installer (Non-Interactive)"
echo "===================================================="
echo ""
echo "This script will:"
echo "  1. Install required packages (Git, OpenSSH, optional: Go)"
echo "  2. Download or build Morpheus binary"
echo "  3. Set up configuration"
echo "  4. Generate SSH key (if needed)"
echo "  5. Install to PATH"
echo ""
echo "Starting installation..."
echo ""

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    aarch64) BINARY_ARCH="arm64" ;;
    armv7* | armv8l) BINARY_ARCH="arm" ;;
    x86_64) BINARY_ARCH="amd64" ;;
    *)
        echo "⚠️  Unknown architecture: $ARCH. Will build from source."
        BINARY_ARCH=""
        ;;
esac

# Step 1: Update and install packages
echo ""
echo "📦 Step 1/5: Installing packages..."
echo "----"
pkg update -y

# Always install git and openssh
pkg install -y git openssh curl

# Only install Go if building from source or if binary download is disabled
if [[ "${MORPHEUS_BUILD_FROM_SOURCE}" == "1" || -z "$BINARY_ARCH" ]]; then
    echo "Installing Go, Make (building from source)..."
    pkg install -y golang make
fi

# Verify installations
echo ""
echo "Verifying installations:"
git --version
ssh -V 2>&1 | head -1
if command -v go &> /dev/null; then
    go version
fi

# Step 2: Get Morpheus binary
echo ""
echo "📥 Step 2/5: Getting Morpheus binary..."
echo "----"

# Create temp directory for binary
mkdir -p "$HOME/.morpheus-tmp"
BINARY_PATH="$HOME/.morpheus-tmp/morpheus"

# Try to download pre-built binary first (unless forced to build)
DOWNLOAD_SUCCESS=0
if [[ "${MORPHEUS_BUILD_FROM_SOURCE}" != "1" && -n "$BINARY_ARCH" ]]; then
    echo "Attempting to download pre-built binary for linux-$BINARY_ARCH..."
    
    # Get latest release info
    LATEST_URL="https://api.github.com/repos/nimsforest/morpheus/releases/latest"
    RELEASE_INFO=$(curl -s "$LATEST_URL")
    
    if [[ $? -eq 0 ]]; then
        VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
        DOWNLOAD_URL="https://github.com/nimsforest/morpheus/releases/download/${VERSION}/morpheus-linux-${BINARY_ARCH}"
        
        echo "Downloading Morpheus $VERSION for linux-$BINARY_ARCH..."
        if curl -L -f -o "$BINARY_PATH" "$DOWNLOAD_URL" 2>/dev/null; then
            chmod +x "$BINARY_PATH"
            
            # Verify the binary works
            if "$BINARY_PATH" version &>/dev/null; then
                echo "✓ Downloaded pre-built binary successfully!"
                DOWNLOAD_SUCCESS=1
            else
                echo "⚠️  Downloaded binary doesn't work. Will build from source."
                rm -f "$BINARY_PATH"
            fi
        else
            echo "⚠️  Download failed. Will build from source."
        fi
    else
        echo "⚠️  Could not fetch release info. Will build from source."
    fi
fi

# Build from source if download failed or was disabled
if [[ $DOWNLOAD_SUCCESS -eq 0 ]]; then
    echo ""
    echo "🔨 Building from source..."
    echo "----"
    
    # Make sure Go is installed
    if ! command -v go &> /dev/null; then
        echo "Installing Go and Make..."
        pkg install -y golang make
    fi
    
    # Clone repository if needed
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
            git pull || true
        fi
    else
        cd "$HOME"
        git clone https://github.com/nimsforest/morpheus.git
        cd morpheus
    fi
    
    echo "This may take 2-5 minutes on first build..."
    make deps
    make build
    
    # Copy built binary
    if [[ -f "./bin/morpheus" ]]; then
        cp "./bin/morpheus" "$BINARY_PATH"
        echo "✓ Build successful!"
    else
        echo "✗ Build failed. Check errors above."
        exit 1
    fi
fi

# Verify final binary
if [[ ! -f "$BINARY_PATH" ]]; then
    echo "✗ Binary not found at $BINARY_PATH"
    exit 1
fi

echo ""
echo "Binary version:"
"$BINARY_PATH" version

# Step 3: Configuration
echo ""
echo "⚙️  Step 3/5: Setting up configuration..."
echo "----"

mkdir -p "$HOME/.morpheus"

# Get config.example.yaml
if [[ ! -f "$HOME/.morpheus/config.yaml" ]]; then
    if [[ -f "$HOME/morpheus/config.example.yaml" ]]; then
        cp "$HOME/morpheus/config.example.yaml" "$HOME/.morpheus/config.yaml"
    else
        # Download from GitHub if not available locally
        echo "Downloading config.example.yaml..."
        curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/config.example.yaml -o "$HOME/.morpheus/config.yaml"
    fi
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

# Step 4: SSH Key
echo ""
echo "🔑 Step 4/5: Checking SSH key..."
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

# Step 5: Install to PATH (default: yes)
echo ""
echo "📦 Step 5/5: Installing morpheus to PATH..."
echo "----"

if [[ "${MORPHEUS_SKIP_INSTALL}" == "1" ]]; then
    echo "⚠️  MORPHEUS_SKIP_INSTALL=1, skipping PATH installation"
    echo "   Binary available at: $BINARY_PATH"
else
    # Termux doesn't use sudo and has a different bin path
    TERMUX_BIN="$PREFIX/bin"
    if [[ -z "$PREFIX" ]]; then
        TERMUX_BIN="/data/data/com.termux/files/usr/bin"
    fi
    
    if [[ ! -f "$BINARY_PATH" ]]; then
        echo "✗ Error: Binary not found at $BINARY_PATH"
        exit 1
    fi
    
    cp "$BINARY_PATH" "$TERMUX_BIN/morpheus"
    chmod +x "$TERMUX_BIN/morpheus"
    echo "✓ Morpheus installed to $TERMUX_BIN/"
    echo "  You can now run 'morpheus' from anywhere."
    
    # Clean up temp directory
    rm -rf "$HOME/.morpheus-tmp"
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
