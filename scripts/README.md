# Morpheus Scripts

This directory contains helper scripts for various Morpheus workflows.

## Android/Termux Scripts

### `check-termux.sh`

Compatibility checker for Termux environments. Verifies that all required dependencies and configurations are in place before installing Morpheus.

**Usage:**
```bash
# From local clone
./scripts/check-termux.sh

# Or download and run directly
curl -sSL https://raw.githubusercontent.com/yourusername/morpheus/main/scripts/check-termux.sh | bash
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

Automated installer for Morpheus on Termux. Handles the entire setup process interactively.

**Usage:**
```bash
# From local clone
./scripts/install-termux.sh

# Or download and run directly
curl -sSL https://raw.githubusercontent.com/yourusername/morpheus/main/scripts/install-termux.sh | bash
```

**What it does:**
1. Installs required packages (Go, Git, Make, OpenSSH)
2. Clones Morpheus repository
3. Builds Morpheus binary
4. Sets up configuration files
5. Generates SSH key (if needed)
6. Guides through Hetzner API token setup
7. Optionally installs to PATH

**Interactive:** The script asks for confirmation at each major step and guides you through token and SSH key setup.

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
