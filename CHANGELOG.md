# Changelog

All notable changes to this project will be documented in this file.

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
- Comprehensive documentation

### Features
- Automatic Hetzner Cloud server provisioning
- Server lifecycle management
- Forest registry with JSON persistence
- NATS server auto-installation and clustering
- SSH key management
- Firewall configuration
- Progress indicators and colored output

## [Unreleased]

### Planned
- Multi-cloud support (AWS, GCP, Azure, OVH, Vultr)
- Auto-scaling based on load
- Built-in monitoring
- Web UI
- Backup and disaster recovery
