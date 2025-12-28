# Morpheus Scripts

This directory contains helper scripts for various Morpheus workflows.

## Universal Installer

### `install.sh`

**Universal installer that works on all platforms.** Auto-detects OS and architecture, then downloads and installs the appropriate pre-built binary.

**Usage:**
```bash
# Works on Linux, macOS, and Termux
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install.sh | bash
```

**Supported Platforms:**
- **Linux**: x86_64 (amd64), aarch64 (arm64), armv7/armv8l (arm)
- **macOS**: x86_64 (Intel), arm64 (Apple Silicon)
- **Termux**: aarch64 (most Android), armv7/armv8l

**What it does:**
1. ✅ Detects your OS (Linux/Darwin/Termux)
2. ✅ Detects your architecture
3. ✅ Fetches latest release from GitHub
4. ✅ Downloads pre-built binary for your platform
5. ✅ Verifies binary works (`morpheus version`)
6. ✅ Installs to appropriate location:
   - Termux: `$PREFIX/bin/morpheus`
   - Linux/macOS with sudo: `/usr/local/bin/morpheus`
   - Linux/macOS without sudo: `~/.local/bin/morpheus`

**Why use this?**
- 🚀 Fast: Downloads binary (no compilation)
- 🔒 Safe: No dependencies on Go/Make
- 🌍 Universal: One command for all platforms
- ✨ Simple: Auto-detects everything

---

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

**Legacy Termux-specific installer with full setup automation.**

This script is more comprehensive than `install.sh` - it sets up configuration, generates SSH keys, and can build from source as a fallback.

**Usage:**
```bash
# Download and run directly
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# With Hetzner token pre-configured
export HETZNER_API_TOKEN="your_token_here"
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# Force build from source instead of binary
export MORPHEUS_BUILD_FROM_SOURCE=1
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

**What it does:**
1. Installs required packages (Git, OpenSSH, curl)
2. Downloads pre-built binary (fast)
3. Falls back to building from source if download fails
4. Sets up `~/.morpheus/config.yaml`
5. Generates SSH key (if needed)
6. Saves Hetzner API token to `~/.bashrc` (if provided)
7. Installs to PATH

**Environment Variables:**
- `HETZNER_API_TOKEN` - Your Hetzner Cloud API token (saved to `~/.bashrc`)
- `MORPHEUS_BUILD_FROM_SOURCE=1` - Force build from source (skips binary download)
- `MORPHEUS_FORCE_CLONE=1` - Force re-clone repository if it exists
- `MORPHEUS_SKIP_INSTALL=1` - Skip installing to PATH

**When to use this vs `install.sh`:**
- Use `install.sh` (universal): Just want to install Morpheus binary quickly
- Use `install-termux.sh`: Need full Termux setup (SSH keys, config, API token saving)

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
