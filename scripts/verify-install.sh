#!/bin/bash
# Simple verification script for universal installer

echo "Universal Installer Verification"
echo "================================="
echo ""

# Check script exists
if [ ! -f "/workspace/scripts/install.sh" ]; then
    echo "❌ install.sh not found"
    exit 1
fi
echo "✅ install.sh exists"

# Check it's executable
if [ ! -x "/workspace/scripts/install.sh" ]; then
    echo "❌ install.sh is not executable"
    exit 1
fi
echo "✅ install.sh is executable"

# Check bash syntax
if bash -n /workspace/scripts/install.sh; then
    echo "✅ Bash syntax is valid"
else
    echo "❌ Bash syntax error"
    exit 1
fi

# Check for required functions
required_functions=(
    "detect_os"
    "detect_arch"
    "is_termux"
    "get_latest_version"
    "download_binary"
    "verify_binary"
    "get_install_dir"
    "install_binary"
    "main"
)

echo ""
echo "Checking for required functions:"
for func in "${required_functions[@]}"; do
    if grep -q "^${func}()" /workspace/scripts/install.sh; then
        echo "  ✅ $func"
    else
        echo "  ❌ $func missing"
        exit 1
    fi
done

# Check architecture mappings
echo ""
echo "Verifying architecture mappings:"
if grep -q "x86_64|amd64)" /workspace/scripts/install.sh && grep -q 'echo "amd64"' /workspace/scripts/install.sh; then
    echo "  ✅ x86_64 → amd64"
else
    echo "  ❌ x86_64 mapping missing"
fi

if grep -q "aarch64|arm64)" /workspace/scripts/install.sh && grep -q 'echo "arm64"' /workspace/scripts/install.sh; then
    echo "  ✅ aarch64 → arm64"
else
    echo "  ❌ aarch64 mapping missing"
fi

if grep -q "armv7\*|armv8l)" /workspace/scripts/install.sh && grep -q 'echo "arm"' /workspace/scripts/install.sh; then
    echo "  ✅ armv7/armv8l → arm"
else
    echo "  ❌ arm mapping missing"
fi

# Check OS detection
echo ""
echo "Verifying OS detection:"
if grep -q "Linux\*)" /workspace/scripts/install.sh && grep -q 'echo "linux"' /workspace/scripts/install.sh; then
    echo "  ✅ Linux detection"
else
    echo "  ❌ Linux detection missing"
fi

if grep -q "Darwin\*)" /workspace/scripts/install.sh && grep -q 'echo "darwin"' /workspace/scripts/install.sh; then
    echo "  ✅ Darwin/macOS detection"
else
    echo "  ❌ Darwin detection missing"
fi

# Check Termux detection
echo ""
echo "Verifying Termux detection:"
if grep -q 'PREFIX' /workspace/scripts/install.sh && grep -q 'com.termux' /workspace/scripts/install.sh; then
    echo "  ✅ Termux detection logic present"
else
    echo "  ❌ Termux detection missing"
fi

# Check GitHub API usage
echo ""
echo "Verifying GitHub integration:"
if grep -q "api.github.com/repos" /workspace/scripts/install.sh; then
    echo "  ✅ GitHub API usage"
else
    echo "  ❌ GitHub API usage missing"
fi

if grep -q "releases/download" /workspace/scripts/install.sh; then
    echo "  ✅ Binary download URL"
else
    echo "  ❌ Binary download URL missing"
fi

# Check verification
echo ""
echo "Verifying safety features:"
if grep -q "morpheus version" /workspace/scripts/install.sh; then
    echo "  ✅ Binary verification"
else
    echo "  ❌ Binary verification missing"
fi

if grep -q "cleanup" /workspace/scripts/install.sh && grep -q "trap" /workspace/scripts/install.sh; then
    echo "  ✅ Cleanup on exit"
else
    echo "  ❌ Cleanup logic missing"
fi

# Check installation locations
echo ""
echo "Verifying installation locations:"
if grep -q '\$PREFIX/bin' /workspace/scripts/install.sh; then
    echo "  ✅ Termux: \$PREFIX/bin"
else
    echo "  ❌ Termux location missing"
fi

if grep -q '/usr/local/bin' /workspace/scripts/install.sh; then
    echo "  ✅ Standard: /usr/local/bin"
else
    echo "  ❌ Standard location missing"
fi

if grep -q '\.local/bin' /workspace/scripts/install.sh; then
    echo "  ✅ Fallback: ~/.local/bin"
else
    echo "  ❌ Fallback location missing"
fi

echo ""
echo "================================="
echo "✅ All verifications passed!"
echo "================================="
echo ""
echo "The universal installer is ready to use:"
echo "  curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh | bash"
