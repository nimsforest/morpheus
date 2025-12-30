# Universal Installer for Morpheus CLI

## Overview

The universal installer (`scripts/install.sh`) provides a **single-script solution** for installing Morpheus CLI across all supported platforms: Linux, macOS, and Termux/Android.

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh | bash
```

That's it! The script handles everything automatically.

## How It Works

### 1. Platform Detection

The script automatically detects:

- **Operating System**: Linux or macOS (Darwin)
- **Architecture**: Maps system architecture to binary names:
  - `x86_64`, `amd64` → `amd64`
  - `aarch64`, `arm64` → `arm64`
  - `armv7*`, `armv8l` → `arm`
- **Environment**: Detects if running in Termux by checking `$PREFIX` environment variable

### 2. Binary Download

Downloads pre-built binaries from GitHub releases:

1. **Fetch latest release**:
   ```
   GET https://api.github.com/repos/nimsforest/morpheus/releases/latest
   ```
   Parses `tag_name` from JSON response

2. **Download binary**:
   ```
   https://github.com/nimsforest/morpheus/releases/download/{VERSION}/morpheus-{OS}-{ARCH}
   ```
   
   Examples:
   - `morpheus-linux-amd64`
   - `morpheus-linux-arm64`
   - `morpheus-darwin-amd64`
   - `morpheus-darwin-arm64`

3. **Uses curl or wget**: Automatically detects which tool is available

### 3. Verification

Before installation, the script:

- Checks if the binary file exists
- Verifies the binary is executable
- Runs `morpheus version` to ensure it works properly
- Exits with error if verification fails

### 4. Smart Installation

Chooses the appropriate installation location:

| Environment | Location | Requires sudo? |
|-------------|----------|----------------|
| Termux | `$PREFIX/bin/morpheus` | No |
| Linux/macOS (writable) | `/usr/local/bin/morpheus` | No |
| Linux/macOS (with sudo) | `/usr/local/bin/morpheus` | Yes (prompted) |
| Fallback | `~/.local/bin/morpheus` | No |

The script tries each option in order and uses the first one that works.

### 5. PATH Management

After installation:

- Checks if installation directory is in `$PATH`
- Provides helpful instructions if PATH needs updating
- Example output:
  ```
  [WARNING] /home/user/.local/bin is not in your PATH
  [INFO] Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):
  
      export PATH="/home/user/.local/bin:$PATH"
  ```

### 6. Cleanup

Automatically removes temporary files on:
- Successful installation
- Installation failure
- Script interruption (Ctrl+C)

## Features

### ✅ Universal Compatibility

- Works on Linux (all architectures)
- Works on macOS (Intel and Apple Silicon)
- Works on Termux/Android (ARM64, ARM)
- Single script for all platforms

### ✅ Safe Installation

- Verifies binary before installing
- Tests execution with `morpheus version`
- Cleans up on failure
- No destructive operations without verification

### ✅ Smart Fallbacks

- Tries multiple installation locations
- Falls back to user directory if sudo unavailable
- Works with or without sudo access
- Provides clear error messages

### ✅ Zero Configuration

- No environment variables required
- No user input needed
- Fully automated process
- Downloads latest version automatically

## Comparison with Other Install Methods

| Method | Speed | Requirements | Use Case |
|--------|-------|--------------|----------|
| **Universal Installer** | ⚡ Instant | curl/wget only | **Recommended for everyone** |
| `install-termux.sh` | 🐌 5-10 min | Go, Git, Make | Termux users who want to build from source |
| Build from source | 🐌 5-10 min | Go 1.25+, Git, Make | Developers, custom builds |

## Advanced Usage

### Specify Version

```bash
# Download the script
curl -sSL -o install.sh https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh
chmod +x install.sh

# Edit the script to specify a version
# Change get_latest_version() to return your desired version

# Run it
./install.sh
```

### Custom Installation Directory

The script uses standard locations, but you can modify it:

```bash
# Download and edit
curl -sSL -o install.sh https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh
chmod +x install.sh

# Edit get_install_dir() function to return your custom path
# Then run
./install.sh
```

### Test Without Installing

```bash
# Download to temp file
temp_file=$(mktemp)
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

# Map architecture
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  armv7*|armv8l) arch="arm" ;;
esac

# Download binary
version=$(curl -sSL https://api.github.com/repos/nimsforest/morpheus/releases/latest | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
curl -sSL "https://github.com/nimsforest/morpheus/releases/download/${version}/morpheus-${os}-${arch}" -o "$temp_file"
chmod +x "$temp_file"

# Test it
"$temp_file" version

# Clean up
rm "$temp_file"
```

## Troubleshooting

### "Unsupported operating system"

The script only supports:
- Linux (all architectures)
- macOS/Darwin (Intel and Apple Silicon)

Windows is not supported. Use WSL2 or a Linux VM.

### "Unsupported architecture"

Currently supported architectures:
- `x86_64` (amd64)
- `aarch64` (arm64)
- `armv7*` (arm)
- `armv8l` (arm)

If you have a different architecture, you'll need to build from source.

### "Neither curl nor wget is available"

Install one of them:

```bash
# Debian/Ubuntu
sudo apt install curl

# macOS
brew install curl

# Termux
pkg install curl
```

### "Failed to download binary"

Possible causes:
1. **No internet connection**: Check your network
2. **GitHub is down**: Try again later
3. **Binary doesn't exist for your platform**: Check releases at https://github.com/nimsforest/morpheus/releases

### "Binary failed to execute 'version' command"

Possible causes:
1. **Incompatible binary**: Architecture mismatch
2. **Missing dependencies**: Check system libraries
3. **Corrupted download**: Try again

### Installation succeeds but `morpheus` not found

The installation directory is not in your PATH. Add it:

```bash
# For ~/.local/bin
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# For Termux ($PREFIX/bin is usually already in PATH)
# No action needed

# For /usr/local/bin (usually already in PATH)
# No action needed
```

## Script Output Example

```
[INFO] Morpheus CLI Universal Installer

[INFO] Detecting system information...
[INFO] Environment: Standard linux
[INFO] Operating System: linux
[INFO] Architecture: arm64

[INFO] Fetching latest release information...
[SUCCESS] Latest version: v1.2.0

[INFO] Downloading morpheus-linux-arm64 from v1.2.0...
[INFO] URL: https://github.com/nimsforest/morpheus/releases/download/v1.2.0/morpheus-linux-arm64
[SUCCESS] Binary downloaded successfully

[INFO] Verifying binary...
[SUCCESS] Binary verification passed

[INFO] Installation directory: /usr/local/bin

[INFO] Installing to /usr/local/bin/morpheus...
[SUCCESS] Binary installed to /usr/local/bin/morpheus

[SUCCESS] Morpheus CLI has been successfully installed!
[INFO] Run 'morpheus --help' to get started

[INFO] Installed version:
Morpheus CLI v1.2.0
```

## Technical Details

### Dependencies

**Required**:
- Bash shell
- One of: `curl` or `wget`
- One of: `grep`, `sed`
- `uname` (for OS/arch detection)
- `chmod` (for making binary executable)

**Optional**:
- `sudo` (only if installing to `/usr/local/bin` without write access)

### Error Handling

The script uses:
- `set -e`: Exit on any error
- `trap`: Clean up temporary files on exit
- Return codes: All functions return 0 on success, 1 on failure
- Color-coded output: Red for errors, yellow for warnings, green for success

### Security Considerations

- Downloads from official GitHub releases only
- Verifies binary execution before installation
- Uses HTTPS for all downloads
- No arbitrary code execution
- No modification of system files beyond installation directory

### Color Output

The script uses ANSI color codes:
- 🔵 Blue: Informational messages
- 🟢 Green: Success messages
- 🟡 Yellow: Warnings
- 🔴 Red: Errors

Colors automatically work in most terminals, including Termux.

## Integration with Existing Installers

The universal installer complements (but doesn't replace) existing installation methods:

- **`install-termux.sh`**: Still useful for users who want to build from source on Termux
- **`make install`**: Still used by developers during local development
- **Universal installer**: Best for end-users who want quick, automated installation

All three methods result in a working Morpheus installation.

## Future Enhancements

Potential improvements:

- [ ] Support for Windows (download .exe)
- [ ] Verify binary checksum/signature
- [ ] Allow version specification via environment variable
- [ ] Progress indicator for large downloads
- [ ] Automatic update check on execution
- [ ] Support for installing to custom directory via flag

## Contributing

If you want to improve the installer:

1. Test on your platform first
2. Ensure backward compatibility
3. Update this documentation
4. Test error cases
5. Verify cleanup works properly

## See Also

- [scripts/install.sh](scripts/install.sh) - The actual installer script
- [scripts/README.md](scripts/README.md) - Overview of all scripts
- [README.md](README.md) - Main Morpheus documentation
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

---

**Status**: Production Ready ✅  
**Created**: December 28, 2025  
**Tested**: Linux (x86_64, ARM64), macOS (Intel, Apple Silicon), Termux (ARM64)
