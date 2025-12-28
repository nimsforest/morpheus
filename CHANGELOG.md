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

## [Unreleased]

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

### Changed
- **Mobile approach repositioned:** Termux is now the primary/recommended approach for mobile users
- Control server repositioned as alternative for specific use cases (24/7, teams, CI/CD)
- Go version set to 1.25 (Termux installs whatever current version is available)
- Updated README badges to reflect Go 1.25
- Enhanced prerequisites section - clarified Go is auto-installed on Termux
- Reorganized mobile documentation to emphasize Termux-first approach
- Updated CONTROL_SERVER_SETUP.md to clarify it's for specific scenarios only

### Fixed
- Verified no CGO dependencies blocking Android support
- Verified no platform-specific build tags
- Ensured pure Go implementation for cross-platform compatibility

## [Unreleased]

### Planned
- Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- Auto-scaling based on load
- Built-in monitoring
- Web UI for forest management
- Private network support
- Load balancer integration
- Backup and disaster recovery
- iOS support via a-Shell (similar to Termux)
