#!/data/data/com.termux/files/usr/bin/bash
# Morpheus Termux Compatibility Checker
# Run this script in Termux to verify your environment is ready for Morpheus

set -e

echo "🔍 Morpheus Termux Compatibility Check"
echo "======================================"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SUCCESS="${GREEN}✓${NC}"
FAILURE="${RED}✗${NC}"
WARNING="${YELLOW}⚠${NC}"

ERRORS=0
WARNINGS=0

# Check 1: Architecture
echo -n "Checking architecture... "
ARCH=$(uname -m)
if [[ "$ARCH" == "aarch64" ]] || [[ "$ARCH" == "arm64" ]]; then
    echo -e "$SUCCESS ARM64 detected ($ARCH)"
elif [[ "$ARCH" == "armv7l" ]] || [[ "$ARCH" == "armv8l" ]]; then
    echo -e "$SUCCESS ARM32 detected ($ARCH)"
else
    echo -e "$WARNING Unknown architecture: $ARCH"
    echo "   Morpheus may still work, but ARM64 is recommended."
    ((WARNINGS++))
fi

# Check 2: Operating System
echo -n "Checking operating system... "
OS=$(uname -s)
if [[ "$OS" == "Linux" ]]; then
    echo -e "$SUCCESS Linux (Android)"
else
    echo -e "$FAILURE Not Linux: $OS"
    echo "   Morpheus requires Linux (Android)."
    ((ERRORS++))
fi

# Check 3: Go installation
echo -n "Checking Go installation... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [[ $MAJOR -ge 1 ]] && [[ $MINOR -ge 20 ]]; then
        echo -e "$SUCCESS Go $GO_VERSION"
    else
        echo -e "$WARNING Go $GO_VERSION (recommended: 1.20+)"
        echo "   Morpheus may require Go 1.20 or higher."
        ((WARNINGS++))
    fi
else
    echo -e "$FAILURE Go not installed"
    echo "   Install with: pkg install golang"
    ((ERRORS++))
fi

# Check 4: Git
echo -n "Checking Git... "
if command -v git &> /dev/null; then
    GIT_VERSION=$(git --version | awk '{print $3}')
    echo -e "$SUCCESS Git $GIT_VERSION"
else
    echo -e "$FAILURE Git not installed"
    echo "   Install with: pkg install git"
    ((ERRORS++))
fi

# Check 5: Make
echo -n "Checking Make... "
if command -v make &> /dev/null; then
    MAKE_VERSION=$(make --version | head -1 | awk '{print $3}')
    echo -e "$SUCCESS Make $MAKE_VERSION"
else
    echo -e "$FAILURE Make not installed"
    echo "   Install with: pkg install make"
    ((ERRORS++))
fi

# Check 6: OpenSSH
echo -n "Checking OpenSSH... "
if command -v ssh &> /dev/null; then
    SSH_VERSION=$(ssh -V 2>&1 | awk '{print $1}')
    echo -e "$SUCCESS $SSH_VERSION"
else
    echo -e "$FAILURE OpenSSH not installed"
    echo "   Install with: pkg install openssh"
    ((ERRORS++))
fi

# Check 7: SSH Key
echo -n "Checking SSH key... "
if [[ -f "$HOME/.ssh/id_ed25519" ]] || [[ -f "$HOME/.ssh/id_rsa" ]]; then
    echo -e "$SUCCESS SSH key exists"
else
    echo -e "$WARNING No SSH key found"
    echo "   Generate with: ssh-keygen -t ed25519"
    ((WARNINGS++))
fi

# Check 8: Storage
echo -n "Checking available storage... "
AVAILABLE=$(df -h $HOME | tail -1 | awk '{print $4}')
AVAILABLE_MB=$(df -BM $HOME | tail -1 | awk '{print $4}' | sed 's/M//')
if [[ $AVAILABLE_MB -ge 500 ]]; then
    echo -e "$SUCCESS $AVAILABLE available"
else
    echo -e "$WARNING Only $AVAILABLE available"
    echo "   Morpheus requires ~500MB. Consider freeing up space."
    ((WARNINGS++))
fi

# Check 9: Internet connectivity
echo -n "Checking internet connectivity... "
if ping -c 1 8.8.8.8 &> /dev/null; then
    echo -e "$SUCCESS Connected"
else
    echo -e "$WARNING No internet connection"
    echo "   Internet required for building and provisioning."
    ((WARNINGS++))
fi

# Check 10: Termux environment
echo -n "Checking Termux environment... "
if [[ -d "/data/data/com.termux" ]]; then
    echo -e "$SUCCESS Termux detected"
else
    echo -e "$WARNING Not running in Termux"
    echo "   Some features may not work outside Termux."
    ((WARNINGS++))
fi

# Check 11: HETZNER_API_TOKEN
echo -n "Checking HETZNER_API_TOKEN... "
if [[ -n "$HETZNER_API_TOKEN" ]]; then
    TOKEN_LENGTH=${#HETZNER_API_TOKEN}
    echo -e "$SUCCESS Token set (length: $TOKEN_LENGTH)"
else
    echo -e "$WARNING HETZNER_API_TOKEN not set"
    echo "   Set with: export HETZNER_API_TOKEN=\"your_token\""
    echo "   Add to ~/.bashrc for persistence."
    ((WARNINGS++))
fi

# Check 12: Morpheus config
echo -n "Checking Morpheus config... "
if [[ -f "$HOME/.morpheus/config.yaml" ]] || [[ -f "./config.yaml" ]]; then
    echo -e "$SUCCESS Config file exists"
else
    echo -e "$WARNING No config file found"
    echo "   Create with: cp config.example.yaml ~/.morpheus/config.yaml"
    ((WARNINGS++))
fi

# Summary
echo ""
echo "======================================"
echo "Summary:"
echo "----"

if [[ $ERRORS -eq 0 ]] && [[ $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo ""
    echo "Your Termux environment is ready for Morpheus."
    echo ""
    echo "Next steps:"
    echo "  1. Clone Morpheus: git clone https://github.com/yourusername/morpheus.git"
    echo "  2. Build: cd morpheus && make build"
    echo "  3. Test: ./bin/morpheus version"
    exit 0
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}⚠ $WARNINGS warning(s)${NC}"
    echo ""
    echo "Your environment is mostly ready, but some optional features may not work."
    echo "Review warnings above and address them if needed."
    exit 0
else
    echo -e "${RED}✗ $ERRORS error(s), $WARNINGS warning(s)${NC}"
    echo ""
    echo "Your environment is not ready for Morpheus."
    echo "Please install missing packages:"
    echo ""
    echo "  pkg update && pkg upgrade -y"
    echo "  pkg install git golang make openssh -y"
    echo ""
    exit 1
fi
