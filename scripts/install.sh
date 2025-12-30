#!/bin/bash
# Morpheus CLI Universal Installer
# Works on Linux, macOS, and Termux/Android

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Repository information
REPO_OWNER="nimsforest"
REPO_NAME="morpheus"
BINARY_NAME="morpheus"

# Print colored message
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    local os_name
    os_name="$(uname -s)"
    
    case "$os_name" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os_name"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch_name
    arch_name="$(uname -m)"
    
    case "$arch_name" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7*|armv8l)
            echo "arm"
            ;;
        *)
            print_error "Unsupported architecture: $arch_name"
            exit 1
            ;;
    esac
}

# Check if running in Termux
is_termux() {
    if [ -n "$PREFIX" ] && [ -d "$PREFIX" ] && [[ "$PREFIX" == *"com.termux"* ]]; then
        return 0
    fi
    return 1
}

# Get latest release version from GitHub
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local version
    
    print_info "Fetching latest release information..."
    
    # Try curl first, then wget
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -sSL "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
    
    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi
    
    echo "$version"
}

# Download binary from GitHub releases
download_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local temp_file="$4"
    
    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${binary_name}"
    
    print_info "Downloading ${binary_name} from ${version}..."
    print_info "URL: ${download_url}"
    
    # Download using curl or wget
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sSL -f "$download_url" -o "$temp_file"; then
            print_error "Failed to download binary"
            return 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$temp_file"; then
            print_error "Failed to download binary"
            return 1
        fi
    fi
    
    # Make binary executable
    chmod +x "$temp_file"
    
    print_success "Binary downloaded successfully"
    return 0
}

# Verify binary works
verify_binary() {
    local binary_path="$1"
    
    print_info "Verifying binary..."
    
    if [ ! -f "$binary_path" ]; then
        print_error "Binary file not found: $binary_path"
        return 1
    fi
    
    if [ ! -x "$binary_path" ]; then
        print_error "Binary is not executable: $binary_path"
        return 1
    fi
    
    # Try to run version command
    if ! "$binary_path" version >/dev/null 2>&1; then
        print_error "Binary failed to execute 'version' command"
        return 1
    fi
    
    print_success "Binary verification passed"
    return 0
}

# Determine installation directory
get_install_dir() {
    # Termux: use $PREFIX/bin
    if is_termux; then
        echo "$PREFIX/bin"
        return 0
    fi
    
    # Check if /usr/local/bin is writable
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return 0
    fi
    
    # Check if we can write with sudo
    if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        echo "/usr/local/bin"
        return 0
    fi
    
    # Try to use sudo (will prompt for password)
    if command -v sudo >/dev/null 2>&1; then
        print_warning "/usr/local/bin requires sudo access"
        if sudo -v 2>/dev/null; then
            echo "/usr/local/bin"
            return 0
        fi
    fi
    
    # Fallback to ~/.local/bin
    print_warning "No sudo access available, using fallback location"
    local fallback_dir="$HOME/.local/bin"
    mkdir -p "$fallback_dir"
    echo "$fallback_dir"
}

# Install binary to destination
install_binary() {
    local source="$1"
    local dest_dir="$2"
    local dest_path="${dest_dir}/${BINARY_NAME}"
    
    print_info "Installing to ${dest_path}..."
    
    # Check if destination directory exists
    if [ ! -d "$dest_dir" ]; then
        print_error "Destination directory does not exist: $dest_dir"
        return 1
    fi
    
    # Install based on write permissions
    if [ -w "$dest_dir" ]; then
        # Direct copy
        if ! cp "$source" "$dest_path"; then
            print_error "Failed to install binary"
            return 1
        fi
    else
        # Use sudo
        if ! sudo cp "$source" "$dest_path"; then
            print_error "Failed to install binary with sudo"
            return 1
        fi
    fi
    
    print_success "Binary installed to ${dest_path}"
    return 0
}

# Check if binary is in PATH
check_path() {
    local install_dir="$1"
    
    if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
        print_warning "${install_dir} is not in your PATH"
        print_info "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "    export PATH=\"${install_dir}:\$PATH\""
        echo ""
    fi
}

# Cleanup temporary files
cleanup() {
    local temp_file="$1"
    if [ -n "$temp_file" ] && [ -f "$temp_file" ]; then
        rm -f "$temp_file"
    fi
}

# Main installation function
main() {
    print_info "Morpheus CLI Universal Installer"
    echo ""
    
    # Detect system information
    print_info "Detecting system information..."
    local os
    local arch
    local is_termux_env
    
    os=$(detect_os)
    arch=$(detect_arch)
    
    if is_termux; then
        is_termux_env="yes"
        print_info "Environment: Termux/Android"
    else
        is_termux_env="no"
        print_info "Environment: Standard ${os}"
    fi
    
    print_info "Operating System: ${os}"
    print_info "Architecture: ${arch}"
    echo ""
    
    # Get latest version
    local version
    version=$(get_latest_version)
    print_success "Latest version: ${version}"
    echo ""
    
    # Create temporary file
    local temp_file
    temp_file=$(mktemp)
    
    # Set up cleanup trap
    trap "cleanup '$temp_file'" EXIT INT TERM
    
    # Download binary
    if ! download_binary "$version" "$os" "$arch" "$temp_file"; then
        print_error "Installation failed: Could not download binary"
        exit 1
    fi
    echo ""
    
    # Verify binary
    if ! verify_binary "$temp_file"; then
        print_error "Installation failed: Binary verification failed"
        exit 1
    fi
    echo ""
    
    # Determine installation directory
    local install_dir
    install_dir=$(get_install_dir)
    print_info "Installation directory: ${install_dir}"
    echo ""
    
    # Install binary
    if ! install_binary "$temp_file" "$install_dir"; then
        print_error "Installation failed: Could not install binary"
        exit 1
    fi
    echo ""
    
    # Check PATH
    check_path "$install_dir"
    
    # Final success message
    print_success "Morpheus CLI has been successfully installed!"
    print_info "Run 'morpheus --help' to get started"
    
    # Show installed version
    local installed_version
    if command -v morpheus >/dev/null 2>&1; then
        echo ""
        print_info "Installed version:"
        morpheus version
    elif [ -x "${install_dir}/${BINARY_NAME}" ]; then
        echo ""
        print_info "Installed version:"
        "${install_dir}/${BINARY_NAME}" version
    fi
}

# Run main function
main "$@"
