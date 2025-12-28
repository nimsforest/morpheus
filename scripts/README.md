# Morpheus Scripts

This directory contains helper scripts for various Morpheus workflows.

## Universal Installer

### `install.sh`

**Universal installer for all platforms** - Works on Linux, macOS, and Termux/Android with automatic detection and binary download.

**Usage:**
```bash
# Download and run directly (recommended)
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh | bash

# Or from local clone
./scripts/install.sh
```

**Features:**
- ✅ Auto-detects OS (Linux, macOS/Darwin)
- ✅ Auto-detects architecture (x86_64→amd64, aarch64/arm64→arm64, armv7/armv8l→arm)
- ✅ Auto-detects Termux environment (checks $PREFIX)
- ✅ Downloads latest pre-built binary from GitHub releases
- ✅ Verifies binary works before installing (`morpheus version`)
- ✅ Smart installation location:
  - Termux: `$PREFIX/bin` (no sudo needed)
  - Linux/macOS: `/usr/local/bin` (with or without sudo)
  - Fallback: `~/.local/bin` (if no sudo available)
- ✅ Cleans up temporary files automatically
- ✅ Provides helpful PATH warnings if needed

**Binary Download Pattern:**
1. Fetches latest release from `https://api.github.com/repos/nimsforest/morpheus/releases/latest`
2. Parses `tag_name` from JSON response
3. Downloads: `https://github.com/nimsforest/morpheus/releases/download/{VERSION}/morpheus-{OS}-{ARCH}`
4. Examples: `morpheus-linux-arm64`, `morpheus-darwin-amd64`, etc.

**Exit Codes:**
- `0` - Installation successful
- `1` - Error during installation (unsupported OS/arch, download failed, verification failed, etc.)

**Why use this over building from source:**
- ⚡ Instant installation (no compilation time)
- 📦 No Go toolchain required
- ✅ Works on all supported platforms
- 🔒 Binary verification before installation

## Android/Termux Scripts

### `check-termux.sh`

Compatibility checker for Termux environments. Verifies that all required dependencies and configurations are in place before installing Morpheus.

**Usage:**
```bash
# Download and run directly (recommended)
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/check-termux.sh | bash

# Or from local clone
./scripts/check-termux.sh
```

**Checks:**
- ✓ Architecture (ARM64/ARM32)
- ✓ Operating System (Linux/Android)
- ✓ Go installation and version
- ✓ Git, Make, OpenSSH
- ✓ SSH key existence
- ✓ Available storage space
- ✓ Internet connectivity
- ✓ Termux environment
- ✓ Hetzner API token configuration
- ✓ Morpheus config file

**Exit Codes:**
- `0` - All checks passed or warnings only
- `1` - Critical errors found, installation will fail

### `install-termux.sh`

Automated installer for Morpheus on Termux. Handles the entire setup process non-interactively.

**Usage:**
```bash
# Download and run directly (recommended)
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# Or from local clone
./scripts/install-termux.sh

# With Hetzner token pre-configured
export HETZNER_API_TOKEN="your_token_here"
./scripts/install-termux.sh

# With custom options
MORPHEUS_FORCE_CLONE=1 MORPHEUS_SKIP_INSTALL=1 ./scripts/install-termux.sh
```

**What it does:**
1. Installs required packages (Go, Git, Make, OpenSSH)
2. Clones Morpheus repository
3. Builds Morpheus binary
4. Sets up configuration files
5. Generates SSH key (if needed)
6. Saves Hetzner API token (if provided)
7. Installs to PATH (by default)

**Environment Variables:**
- `HETZNER_API_TOKEN` - Your Hetzner Cloud API token (will be saved to `~/.bashrc`)
- `MORPHEUS_FORCE_CLONE=1` - Force re-clone repository if it already exists (default: skip)
- `MORPHEUS_SKIP_INSTALL=1` - Skip installing to PATH (default: install)

**Non-Interactive:** The script runs without prompts, using environment variables for configuration. This makes it suitable for automation, CI/CD, or scripted deployments.

## Contributing

When adding new scripts:

1. **Make executable:** `chmod +x scripts/your-script.sh`
2. **Add shebang:** Use `#!/data/data/com.termux/files/usr/bin/bash` for Termux scripts
3. **Document:** Add entry to this README
4. **Test:** Test on actual Termux before committing
5. **Error handling:** Use `set -e` and provide clear error messages

## Testing Scripts Locally

To test Termux scripts without Termux:

```bash
# Use bash instead of Termux shell
bash ./scripts/check-termux.sh
bash ./scripts/install-termux.sh
```

**Note:** Some checks will fail outside Termux, which is expected.

## Future Scripts (Planned)

- `backup-registry.sh` - Backup and restore Morpheus registry
- `sync-config.sh` - Sync config across devices
- `health-check.sh` - Check provisioned forests health
- `cleanup-old-forests.sh` - Remove forests older than N days

## See Also

- [Android/Termux Guide](../docs/ANDROID_TERMUX.md)
- [Control Server Setup](../docs/CONTROL_SERVER_SETUP.md)
