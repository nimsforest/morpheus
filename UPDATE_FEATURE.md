# Update Feature Implementation

## Summary

Added automatic update functionality to morpheus that checks for new versions via GitHub and can self-update. **Version numbers are now automatically synchronized with git release tags**, eliminating the need to manually update hardcoded versions.

## New Commands

### `morpheus update`
Interactive command that:
1. Checks GitHub API for the latest release
2. Compares with current version (1.1.0)
3. Shows release notes
4. Asks for user confirmation
5. Clones the repository, builds, and installs the new version
6. Creates a backup of the current binary

**Example:**
```bash
$ morpheus update
🔍 Checking for updates...

Current version: 1.1.0
Latest version:  1.2.0

🎉 A new version is available!

Release notes:
─────────────────────────────────────────────────
- Added automatic update feature
- Fixed bug in forest provisioning
- Improved error messages
─────────────────────────────────────────────────

View full release: https://github.com/nimsforest/morpheus/releases/tag/v1.2.0

Do you want to update now? (yes/no): yes

📦 Cloning latest version from GitHub...
🔨 Building latest version...
📋 Backing up current version to /usr/local/bin/morpheus.backup
✨ Installing update to /usr/local/bin/morpheus

✅ Update completed successfully!

Run 'morpheus version' to verify the update.
Backup of previous version saved at: /usr/local/bin/morpheus.backup
```

### `morpheus check-update`
Non-interactive command for automation/scripts:
```bash
$ morpheus check-update
Update available: 1.1.0 → 1.2.0
Run 'morpheus update' to install.

$ echo $?
0  # Exit code 0 whether update is available or not
```

## Implementation Details

### New Package: `pkg/updater`

**`pkg/updater/updater.go`**
- `Updater` struct with version checking and update logic
- `CheckForUpdate()` - Queries GitHub API for latest release
- `PerformUpdate()` - Clones repo, builds, and installs new version
- Handles permission checking and backup creation

**`pkg/updater/version/version.go`**
- `Compare()` - Semantic version comparison (supports x.y.z format)
- `IsNewer()` - Helper to check if version is newer
- Handles version prefixes (v1.0.0 vs 1.0.0)
- Parses and compares major, minor, patch components

### Tests

**`pkg/updater/version/version_test.go`**
- Comprehensive tests for version comparison
- Tests for versions with/without 'v' prefix
- Tests for pre-release versions
- Tests for different version lengths

All tests pass ✅

## How It Works

1. **Version Check**
   - Queries GitHub API: `https://api.github.com/repos/nimsforest/morpheus/releases/latest`
   - Parses JSON response to get tag name (version) and release assets
   - Compares using semantic versioning

2. **Update Process**
   - Detects your platform (OS + architecture)
   - Downloads pre-built binary for your platform from GitHub releases
   - Verifies downloaded binary works correctly
   - Checks write permissions on current binary
   - Creates backup of current binary
   - Replaces binary with new version atomically
   - Cleans up temporary files

3. **Version Synchronization** ✨
   - Version is **automatically injected at build time** via `-ldflags`
   - CI/CD builds use `git describe --tags` for version detection
   - Pre-built binaries in releases have correct versions baked in
   - No more manual version updates needed!

4. **Error Handling**
   - Network errors show manual update instructions
   - Permission errors suggest using sudo
   - Binary verification failures prevent installation
   - Backup is restored on installation failure
   - Platform detection handles all supported architectures

## Features

✅ **Fast** - Downloads pre-built binaries (seconds vs minutes)  
✅ **Works everywhere** - Desktop, Termux, any Linux/macOS system  
✅ **Safe** - Creates backup before updating  
✅ **Interactive** - Shows release notes and asks for confirmation  
✅ **Scriptable** - `check-update` for automation  
✅ **Reliable** - No build dependencies, no compilation errors  
✅ **Verified** - Tests binary before installation  
✅ **Auto-versioning** - Version automatically syncs with git tags  

## Requirements

- Internet connection (to download binary)
- Write permission to morpheus binary location
- ~10MB disk space for download

**No longer required:**
- ❌ Git
- ❌ Go compiler
- ❌ Build tools

## Edge Cases Handled

- No internet connection → Clear error message with manual instructions
- No releases published → Handles 404 gracefully
- No write permission → Suggests sudo or manual update
- Binary download failure → Shows error with fallback instructions
- Binary verification failure → Prevents installation, shows error
- Installation failure → Restores backup automatically
- Platform detection → Supports all major platforms (Linux/macOS, amd64/arm64/arm)

## Future Enhancements (Optional)

- [x] ~~Download pre-built binaries instead of building from source~~ ✅ **DONE**
- [ ] Automatic update checks on startup (configurable)
- [ ] Update notifications
- [ ] Rollback command to restore previous version
- [ ] Support for beta/rc channels
- [ ] Progress bar for downloads
- [ ] Resume interrupted downloads

## Testing

```bash
# Run all tests
make test

# Test specific package
go test ./pkg/updater/version/...

# Build and test commands
make build
./bin/morpheus version
./bin/morpheus help
./bin/morpheus check-update
```

## Documentation Updates

Updated README.md with:
- New commands in quick reference
- Update section with usage examples
- Manual update instructions as fallback

## Files Changed/Added

**New Files:**
- `pkg/updater/updater.go` (213 lines)
- `pkg/updater/version/version.go` (54 lines)
- `pkg/updater/version/version_test.go` (62 lines)

**Modified Files:**
- `cmd/morpheus/main.go` - Added update handlers and commands, changed version to build-time variable
- `go.mod` - Fixed Go version (1.25 → 1.21)
- `README.md` - Added update documentation
- `Makefile` - Added automatic version injection via -ldflags
- `.github/workflows/build.yml` - Added git tag fetching for version detection
- `.github/workflows/release.yml` - Builds multi-platform binaries
- `pkg/updater/updater.go` - Downloads pre-built binaries instead of building from source ✨ **UPDATED**
- `scripts/install-termux.sh` - Prioritizes binary downloads over source builds ✨ **UPDATED**

**Total:** ~450 lines of code added (including tests and docs)

## Version Synchronization Details ✨

### How Version Injection Works

The version is automatically determined from git tags and injected at build time:

1. **Local Development Builds** (`make build`):
   ```bash
   # Makefile extracts version
   VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
   LDFLAGS=-ldflags "-X main.version=$(VERSION)"
   
   # Example output: v1.1.0-3-g1787a62-dirty
   # - v1.1.0: latest tag
   # - 3: commits since tag
   # - g1787a62: commit hash
   # - dirty: uncommitted changes
   ```

2. **CI/CD Builds** (Release Workflow):
   - GitHub Actions workflow fetches all tags (`fetch-depth: 0`)
   - Builds binaries for all platforms with version from tag
   - Uploads pre-built binaries to GitHub releases

3. **Self-Update** (User Update):
   - Downloads pre-built binary from GitHub releases
   - Binary already has correct version baked in
   - No cloning or building required!

### Benefits

- ✅ No manual version updates needed
- ✅ Version always matches git tags
- ✅ Easy to see if binary is from a release or development build
- ✅ Commit hash included for traceability (development builds)
- ✅ Works seamlessly across all build methods
- ✅ Fast updates (download vs build)
- ✅ No build dependencies required

### For Maintainers

To release a new version:

1. Update `CHANGELOG.md` with release notes
2. Tag the commit: `git tag v1.2.0`
3. Push the tag: `git push origin v1.2.0`
4. GitHub Actions automatically:
   - Builds binaries for all platforms
   - Creates GitHub release
   - Uploads binaries
5. Users run: `morpheus update` to download the new version ✨

No need to manually update version numbers anywhere!
