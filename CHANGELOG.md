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

See [RELEASE.md](RELEASE.md) for complete release process documentation.

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
- Fixed "text file busy" error when updating morpheus on Termux/Android
  - Replaced file copy with atomic rename operation for binary updates
  - Removed write permission check that was opening the running executable
  - Update process now works correctly even while morpheus is running
- Verified no CGO dependencies blocking Android support
- Verified no platform-specific build tags
- Ensured pure Go implementation for cross-platform compatibility

## [Unreleased]

### Improved - Network Resilience in Update System

**Enhanced Update System:** Morpheus updater now handles network issues gracefully!

When `morpheus update` encounters network problems (DNS failures, connection issues, timeouts), it now:

- ✅ **Automatic Retry Logic:** Retries failed requests up to 3 times with exponential backoff
- ✅ **Better DNS Handling:** Custom HTTP transport with improved IPv4/IPv6 dual-stack support
- ✅ **Enhanced Error Messages:** Provides specific troubleshooting advice based on error type:
  - DNS resolution failures → Suggests checking DNS configuration
  - Localhost DNS issues → Detects misconfigured localhost DNS and provides fix instructions
  - Connection refused → Advises on firewall/proxy checks
  - Timeouts → Suggests network connectivity troubleshooting

**Common DNS Issue Fixed (especially on Termux/Android):**
```
Error: lookup api.github.com on [::1]:53: connection refused

Now shows:
DNS configuration issue detected:
  • Your system is trying to use localhost as DNS server
  • [On Linux] Check /etc/resolv.conf for incorrect DNS settings
  • [On Termux] Disable Private DNS in Android Settings
  • [On Termux] Restart Termux or disable VPN apps
```

**Termux-Specific Improvements:**
- Added comprehensive Termux DNS troubleshooting guide
- Updated Android/Termux documentation with DNS fixes
- Most common fix: Disable "Private DNS" in Android Settings

**Technical Details:**
- Added `maxRetries = 3` constant for network operations
- Implemented exponential backoff (2s, 4s, 6s intervals)
- Custom `net.Dialer` with optimized timeout settings
- Intelligent error detection and classification
- Only retries transient network errors (not API errors)

**Benefits:**
- More reliable updates on unstable networks
- Better user experience with actionable error messages
- Handles common DNS misconfigurations gracefully
- Especially helpful for Termux/mobile environments with varying network quality

See [UPDATER_NETWORK_IMPROVEMENTS.md](UPDATER_NETWORK_IMPROVEMENTS.md) for complete technical documentation.

### Planned
- Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- Auto-scaling based on load
- Built-in monitoring
- Web UI for forest management
- Private network support
- Load balancer integration
- Backup and disaster recovery
- iOS support via a-Shell (similar to Termux)
