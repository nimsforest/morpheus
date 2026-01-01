# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [1.1.0] - 2025-12-26

### Changed - Separation of Concerns

**BREAKING**: Morpheus no longer installs NATS. This is now NimsForest's responsibility.

- Removed NATS installation from cloud-init templates
- Removed NATS configuration and clustering setup
- Added integration with NimsForest via callbacks
- Added `/etc/morpheus/node-info.json` metadata file
- Added `integration` section to configuration
- Node status now `infrastructure_ready` instead of `active`

**Migration**: Update config.yaml to add `integration` section. NimsForest must handle NATS installation.

### Added
- Integration configuration for NimsForest callbacks
- Node metadata file for handoff to NimsForest
- Morpheus bootstrap script at `/usr/local/bin/morpheus-bootstrap`
- Enhanced firewall configuration for NATS ports
- Comprehensive test suite (66.4% coverage)
- Separation of concerns documentation

### Fixed
- Cloud-init templates now focus on infrastructure only
- Clear boundary between infrastructure and application layers
- Improved error messages and validation

## [1.0.0] - 2025-12-26

### Added
- Initial release of Morpheus
- Hetzner Cloud integration with hcloud-go/v2
- Automated server provisioning with cloud-init
- Forest registry for tracking deployments
- CLI commands: plant, teardown, list, status
- Support for wood (1 node), forest (3 nodes), jungle (5 nodes)
- Cloud-init templates for edge, compute, and storage nodes
- YAML configuration management
- Environment variable support for API tokens
- Automatic rollback on failures
- Multi-location support (fsn1, nbg1, hel1)

## [1.2.0] - 2025-12-28

### Added - Android/Termux Support (Primary Mobile Approach)

**NEW**: Morpheus can now run natively on Android devices via Termux!

**Philosophy:** Morpheus is a CLI tool, Termux is a terminal - running directly on Android is the natural approach. Control servers are only for specific use cases (24/7 availability, teams, CI/CD).

- ✅ Full Android/ARM64 and ARM32 support
- ✅ Native Termux compatibility (no emulation needed)
- ✅ Direct CLI usage - no SSH overhead
- ✅ Free alternative to control server (~€4.50/month savings)
- ✅ Comprehensive Android/Termux installation guide
- ✅ Automated installer script (`scripts/install-termux.sh`)
- ✅ Compatibility checker script (`scripts/check-termux.sh`)
- ✅ Mobile-friendly documentation and workflows
- ✅ No CGO dependencies (pure Go)
- ✅ No platform-specific system calls
- ✅ Go version set to 1.25 (Termux automatically installs current version)

**Documentation:**
- Added `docs/ANDROID_TERMUX.md` - Complete guide for running Morpheus on Android
- Added `scripts/README.md` - Documentation for helper scripts
- Updated main README with Android/Termux quick start links

**Scripts:**
- `scripts/check-termux.sh` - Verify Termux environment compatibility
- `scripts/install-termux.sh` - Automated installation for Termux

**Use Cases:**
- Personal infrastructure management from mobile device
- Testing and development on the go
- Cost savings vs control server (~€4.50/month)
- Offline command support (list, status)

**Installation:**
```bash
# One-command install
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash

# Or manual installation - see docs/ANDROID_TERMUX.md
```

### Added - Automated Releases

**NEW**: GitHub releases are now fully automated with pre-built binaries!

When a version tag is pushed (e.g., `v1.2.0`), GitHub Actions automatically:
- ✅ Builds binaries for multiple platforms (Linux amd64/arm64/arm, macOS amd64/arm64)
- ✅ Creates GitHub release with binaries attached
- ✅ Extracts release notes from CHANGELOG.md
- ✅ Publishes to GitHub Releases (hosted on GitHub CDN)

**Benefits:**
- Faster installation - no need to build from source (10x faster!)
- Easier updates - pre-built binaries for all platforms
- Consistent releases - automated process reduces human error
- Better user experience - especially for Termux users (no Go compilation needed)

**Features:**
- Updated Termux installer to prefer downloading pre-built binaries
- Falls back to building from source if download fails
- Added `MORPHEUS_BUILD_FROM_SOURCE=1` flag to force source build
- Created RELEASE.md documentation for maintainers
- Binaries hosted on GitHub Releases (free, fast CDN)

**For Maintainers:**
```bash
# Release is now as simple as:
git tag v1.2.0
git push origin v1.2.0
# Done! GitHub Actions handles the rest
```

See [docs/development/RELEASE.md](docs/development/RELEASE.md) for complete release process documentation.

### Added - Automatic SSH Key Upload

**NEW**: Morpheus now automatically uploads SSH keys to Hetzner Cloud!

Users no longer need to manually upload SSH keys through the Hetzner console. When provisioning servers, Morpheus will:

1. Check if the configured SSH key exists in Hetzner Cloud
2. If not found, automatically read the SSH public key from local filesystem
3. Upload the key to Hetzner Cloud with the configured name
4. Proceed with server provisioning

**Features:**
- ✅ Automatic SSH key detection from common locations (`~/.ssh/id_ed25519.pub`, `~/.ssh/id_rsa.pub`)
- ✅ Support for custom SSH key paths via `ssh_key_path` config option
- ✅ Tilde expansion for home directory paths
- ✅ Validation of SSH public key format (RSA, ED25519, ECDSA, DSS)
- ✅ Falls back to common locations if custom path not found
- ✅ Comprehensive test coverage for SSH key functionality

**Configuration:**
```yaml
infrastructure:
  defaults:
    ssh_key: main              # Key name in Hetzner Cloud
    ssh_key_path: ""           # Optional: custom path to local SSH public key
```

**Benefits:**
- Simplified setup - one less manual step
- Better automation - fully scriptable infrastructure provisioning
- Improved user experience - especially for new users
- Works seamlessly with Termux and desktop environments

**Documentation:**
- Updated README.md with SSH key auto-upload information
- Updated CONTROL_SERVER_SETUP.md with new workflow
- Updated config.example.yaml with ssh_key_path option
- Added comprehensive tests for SSH key functionality

### Changed
- **Mobile approach repositioned:** Termux is now the primary/recommended approach for mobile users
- Control server repositioned as alternative for specific use cases (24/7, teams, CI/CD)
- Go version set to 1.25 (Termux installs whatever current version is available)
- Updated README badges to reflect Go 1.25
- Enhanced prerequisites section - clarified Go is auto-installed on Termux
- Reorganized mobile documentation to emphasize Termux-first approach
- Updated CONTROL_SERVER_SETUP.md to clarify it's for specific scenarios only

### Fixed
- **CRITICAL: Fixed SIGSYS crash on Termux/Android** - Updater now works on all platforms
  - Root cause: `exec.Command` was triggering `faccessat2` syscall (Linux 5.8+) not available on Android
  - Solution: Replaced curl with Go's native `net/http` package with smart TLS certificate handling
  - Eliminated all external command dependencies (`exec.Command("curl", ...)`)
  - Added restricted environment detection for Termux/Android
  - Binary verification now skipped on restricted platforms to avoid syscall issues
  - **Impact:** Update command no longer crashes with "SIGSYS: bad system call" on Termux
- **Fixed TLS certificate issues on minimal distros** - Works everywhere now
  - Smart certificate loading from multiple common locations across distros
  - Supports Debian/Ubuntu, Fedora/RHEL, Alpine, OpenSUSE, FreeBSD, Termux, etc.
  - Falls back to insecure connection on Termux only (with clear warning) as last resort
  - Maintains security on normal systems (no insecure connections)
  - Paths checked: `/etc/ssl/certs/ca-certificates.crt`, `/data/data/com.termux/files/usr/etc/tls/cert.pem`, etc.
- Fixed "text file busy" error when updating morpheus on Termux/Android
  - Replaced file copy with atomic rename operation for binary updates
  - Removed write permission check that was opening the running executable
  - Update process now works correctly even while morpheus is running
- Improved cross-platform compatibility
  - Works on systems without curl installed
  - Handles TLS certificates intelligently across all platforms
  - Better error messages for network failures
  - More maintainable pure Go implementation
- Verified no CGO dependencies blocking Android support
- Verified no platform-specific build tags
- Ensured pure Go implementation for cross-platform compatibility

## [Unreleased]

### Changed - Direct Binary Deployment

**IMPORTANT**: Removed Docker from cloud deployments. Morpheus now prepares infrastructure for direct Go binary deployment.

**Philosophy:** For single-binary applications like NATS, Docker adds unnecessary overhead. Cloud VMs already provide isolation, so we deploy Go binaries directly via systemd.

**Changes:**
- ✅ Removed `docker.io` from cloud-init package installation
- ✅ Added `/opt/nimsforest/bin` directory for Go binaries
- ✅ Added `/etc/nimsforest` directory for configuration files
- ✅ Replaced Docker setup with systemd preparation
- ✅ Updated cloud-init templates (EdgeNode, ComputeNode)
- ✅ Updated documentation to reflect binary deployment approach

**Benefits:**
- 🚀 Faster startup (no Docker daemon, no container layer)
- 💾 Lower memory usage (~200MB saved per node)
- 🔧 Simpler debugging (standard Linux processes)
- 🔒 Reduced attack surface (no Docker daemon)
- 📦 Native integration (systemd, journald, standard Linux tools)

**Important Notes:**
- **Cloud mode** (production): No Docker. Direct binaries via systemd.
- **Local mode** (development): Still uses Docker for local testing.
- NimsForest should download NATS binary to `/opt/nimsforest/bin/`
- NimsForest should create systemd service at `/etc/systemd/system/nats.service`
- See `docs/architecture/BINARY_DEPLOYMENT.md` for complete guide

**Directory Structure:**
```
/opt/nimsforest/bin/         # Go binaries (nats-server)
/var/lib/nimsforest/         # Data storage (JetStream)
/var/log/nimsforest/         # Logs
/etc/nimsforest/             # Configuration (nats.conf)
```

**For NimsForest developers:** Update bootstrap scripts to:
1. Download NATS binary from GitHub releases
2. Create systemd service file
3. Start service via `systemctl start nats`
4. No Docker commands needed

See [docs/architecture/BINARY_DEPLOYMENT.md](docs/architecture/BINARY_DEPLOYMENT.md) for detailed implementation guide.

### Added - OS Selection Guide

**NEW**: Comprehensive guide for choosing between Ubuntu and Debian for different node types.

**Documentation:**
- Added `docs/architecture/OS_SELECTION.md` - Ubuntu vs Debian decision guide
- Covers Forest nodes (NATS, CPU/RAM heavy) vs Nims nodes (GPU-dependent)
- Performance comparison, resource usage analysis
- GPU driver support comparison
- Recommendation: Use Ubuntu 24.04 LTS for all nodes

**Key Findings:**
- Ubuntu required for GPU support (Nims nodes)
- Debian saves only ~50MB RAM (negligible on 8GB+ servers)
- Ubuntu easier to troubleshoot with better community support
- Keeping one OS across all nodes simplifies operations

**Current Configuration:** Already using Ubuntu 24.04 (optimal choice) ✅

### Planned
- Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- Auto-scaling based on load
- Built-in monitoring
- Web UI for forest management
- Private network support
- Load balancer integration
- Backup and disaster recovery
- iOS support via a-Shell (similar to Termux)
