# Update Feature Implementation

## Summary

Added automatic update functionality to morpheus that checks for new versions via GitHub and can self-update.

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
   - Parses JSON response to get tag name (version)
   - Compares using semantic versioning

2. **Update Process**
   - Clones repository to `/tmp/morpheus-repo`
   - Builds binary with `go build`
   - Checks write permissions on current binary
   - Creates backup of current binary
   - Replaces binary with new version
   - Sets executable permissions
   - Cleans up temporary files

3. **Error Handling**
   - Network errors show manual update instructions
   - Permission errors suggest using sudo
   - Build failures are reported clearly
   - Backup is restored on installation failure

## Features

✅ **Works everywhere** - Desktop, Termux, any Linux/macOS system  
✅ **Safe** - Creates backup before updating  
✅ **Interactive** - Shows release notes and asks for confirmation  
✅ **Scriptable** - `check-update` for automation  
✅ **Fallback** - Shows manual update instructions on failure  
✅ **Self-contained** - No external dependencies except git and go  

## Requirements

- Git (for cloning repository)
- Go (for building from source)
- Write permission to morpheus binary location

## Edge Cases Handled

- No internet connection → Clear error message with manual instructions
- No releases published → Handles 404 gracefully
- No write permission → Suggests sudo or manual update
- Build failure → Shows error, doesn't corrupt existing binary
- Installation failure → Restores backup automatically

## Future Enhancements (Optional)

- [ ] Download pre-built binaries instead of building from source
- [ ] Automatic update checks on startup (configurable)
- [ ] Update notifications
- [ ] Rollback command to restore previous version
- [ ] Support for beta/rc channels

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
- `cmd/morpheus/main.go` - Added update handlers and commands
- `go.mod` - Fixed Go version (1.25 → 1.21)
- `README.md` - Added update documentation

**Total:** ~450 lines of code added (including tests and docs)
