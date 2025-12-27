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

### Planned
- Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- Auto-scaling based on load
- Built-in monitoring
- Web UI for forest management
- Private network support
- Load balancer integration
- Backup and disaster recovery
