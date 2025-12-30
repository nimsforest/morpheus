# Release Process

This document describes how to create a new release of Morpheus.

## Overview

Morpheus uses **automated releases** via GitHub Actions. When you push a version tag, the workflow automatically:
1. Builds binaries for multiple platforms
2. Creates a GitHub release
3. Attaches binaries as downloadable assets
4. Extracts release notes from CHANGELOG.md

## Binaries

Pre-built binaries are hosted on GitHub Releases for:
- **Linux AMD64** - Standard Linux servers and desktops
- **Linux ARM64** - ARM servers and 64-bit Android/Termux
- **Linux ARM** - 32-bit Android/Termux devices
- **macOS AMD64** - Intel Macs
- **macOS ARM64** - Apple Silicon Macs

Users can download them directly or use `morpheus update` to automatically update.

## Release Steps

### 1. Update CHANGELOG.md

Move unreleased changes to a versioned section:

```markdown
## [1.2.0] - 2025-12-28

### Added
- Feature X
- Feature Y

### Fixed
- Bug A
- Bug B

## [Unreleased]
```

**Important**: Make sure the version number is in brackets `[1.2.0]` and matches the tag you'll create (without the `v` prefix).

### 2. Commit Changes

```bash
git add CHANGELOG.md
git commit -m "docs: prepare for v1.2.0 release"
git push origin main
```

### 3. Create and Push Tag

```bash
# Create annotated tag
git tag -a v1.2.0 -m "Release v1.2.0"

# Push tag to GitHub
git push origin v1.2.0
```

**That's it!** The GitHub Actions workflow will automatically:
- Detect the new tag
- Extract version number (e.g., `v1.2.0`)
- Build binaries for all platforms
- Extract release notes from CHANGELOG.md for version `1.2.0`
- Create GitHub release with binaries attached

### 4. Monitor Release

Watch the workflow progress:
```bash
gh run watch
```

Or check: https://github.com/nimsforest/morpheus/actions/workflows/release.yml

### 5. Verify Release

Once complete, check:
```bash
gh release view v1.2.0
```

Or visit: https://github.com/nimsforest/morpheus/releases/tag/v1.2.0

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** (1.x.x) - Breaking changes
- **MINOR** (x.1.x) - New features (backwards compatible)
- **PATCH** (x.x.1) - Bug fixes

Examples:
- `v1.2.0` - Added Android/Termux support (new feature)
- `v1.2.1` - Fixed update bug (bug fix)
- `v2.0.0` - Changed API (breaking change)

## Hotfix Releases

For urgent bug fixes:

```bash
# Create fix on main or hotfix branch
git checkout main
# ... make fixes ...
git commit -m "fix: critical bug"
git push

# Tag and release
git tag v1.2.1
git push origin v1.2.1
```

## Pre-release / Beta Releases

For testing before stable release:

```bash
# Tag with pre-release suffix
git tag v1.3.0-beta.1
git push origin v1.3.0-beta.1

# GitHub will mark it as "Pre-release"
```

Users can test with:
```bash
# Download specific version
gh release download v1.3.0-beta.1
```

## Rollback a Release

If a release has critical issues:

### Option 1: Delete Release (if just published)
```bash
gh release delete v1.2.0 --yes
git tag -d v1.2.0
git push origin :refs/tags/v1.2.0
```

### Option 2: Mark as Pre-release (if users already downloaded)
```bash
gh release edit v1.2.0 --prerelease
```

Then release a hotfix:
```bash
git tag v1.2.1
git push origin v1.2.1
```

## Troubleshooting

### Release workflow failed

Check the workflow logs:
```bash
gh run list --workflow=release.yml
gh run view <run-id> --log
```

Common issues:
- **CHANGELOG parsing failed** - Check `## [1.2.0]` format is correct
- **Build failed** - Check Go version compatibility
- **Permission denied** - Check GitHub Actions has write permission (should be automatic)

### Binary doesn't work

Test locally before releasing:
```bash
# Build for specific platform
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=v1.2.0" -o morpheus-test ./cmd/morpheus

# Test binary
./morpheus-test version
```

### Release notes are empty

The workflow extracts notes from CHANGELOG.md between:
```markdown
## [1.2.0]
...content...
## [1.1.0]  <- stops here
```

Make sure:
- Version is in brackets: `[1.2.0]`
- There's a previous version section below it
- The version number matches the tag (without `v`)

## Manual Release (Fallback)

If automation fails, create release manually:

```bash
# Build binaries
make build

# Create release
gh release create v1.2.0 \
  --title "v1.2.0" \
  --notes "See CHANGELOG.md" \
  bin/morpheus
```

## Post-Release

After release:
1. **Announce** - Update README badges if needed
2. **Test** - Verify `morpheus update` works for users
3. **Monitor** - Watch for issues in GitHub Issues
4. **Document** - Update documentation for new features

## User Update Process

Once released, users can update via:

```bash
# Automatic update
morpheus update

# Or download specific version
gh release download v1.2.0
```

The Termux installer will also automatically download the latest pre-built binary:
```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

## Checklist

Before releasing:
- [ ] CHANGELOG.md updated with version and date
- [ ] Version follows semantic versioning
- [ ] All tests pass (`make test`)
- [ ] Documentation updated for new features
- [ ] Changes committed and pushed to main
- [ ] Tag created and pushed
- [ ] Release workflow completed successfully
- [ ] Binaries work on target platforms
- [ ] Release notes are accurate

## Questions?

See GitHub Actions workflow: `.github/workflows/release.yml`
