# Configuration Simplification - Provider Abstraction

## Summary

Morpheus configuration has been simplified to hide provider-specific details (like Hetzner machine types and location codes) from users. Users now only need to specify their API token and optionally their SSH key name.

## What Changed

### Before (Old Config)
Users had to understand Hetzner-specific terminology:
```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cx22   # What is cx22? Users shouldn't need to know!
    image: ubuntu-24.04
    ssh_key: main
    ssh_key_path: ""
  locations:           # Users don't care about Hetzner location codes!
    - fsn1  # What is fsn1?
    - nbg1  # What is nbg1?
    - hel1
```

### After (New Config)
Simple, provider-agnostic configuration:
```yaml
infrastructure:
  provider: hetzner  # Just the provider name
  ssh:
    key_name: morpheus  # Simple, descriptive
    key_path: ""        # Optional
```

**That's it!** No machine types, no location codes, no provider-specific jargon.

## How It Works

### 1. Machine Profile System
- Created abstract machine profiles: `small`, `medium`, `large`
- Forest size automatically maps to appropriate profile:
  - `wood` → small profile (2 vCPU, 4GB RAM)
  - `forest` → small profile (edge nodes)
  - `jungle` → small profile (edge nodes)

### 2. Provider-Specific Mapping
- Each provider maps profiles to their machine types
- For Hetzner:
  - `small` → cx22 (primary), with fallbacks to cax11, cpx11
  - `medium` → cpx21, with fallbacks
  - `large` → cpx41, with fallbacks

### 3. Automatic Location Selection
- Providers define default recommended locations
- System automatically selects available locations based on:
  - Machine type availability
  - Geographic distribution
  - Reliability
- Automatic fallback if primary location is unavailable

## Benefits

### For Users
1. **Simpler Configuration**: Only need API token and (optionally) SSH key name
2. **No Provider Knowledge Required**: Don't need to learn Hetzner-specific terms
3. **Automatic Fallbacks**: System handles machine type/location unavailability
4. **Consistent Experience**: Same config works across different scenarios

### For Developers
1. **Easy to Add Providers**: Just implement profile mapping
2. **Centralized Logic**: Machine selection in one place
3. **Backward Compatible**: Legacy config format still works
4. **Testable**: Clear abstraction layers

## Migration Guide

### Existing Users
Your old config files will continue to work! The system supports both formats:

**Old format (still works):**
```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cx22
    image: ubuntu-24.04
    ssh_key: main
```

**New format (recommended):**
```yaml
infrastructure:
  provider: hetzner
  ssh:
    key_name: main
```

### New Users
Just copy `config.example.yaml` and set your API token. Done!

## Technical Details

### Files Changed
1. **pkg/provider/profile.go** - New machine profile abstraction
2. **pkg/provider/hetzner/profiles.go** - Hetzner-specific profile mapping
3. **pkg/config/config.go** - Updated config structure with backward compatibility
4. **cmd/morpheus/main.go** - Uses profile system instead of direct config
5. **pkg/forest/provisioner.go** - Accepts provider-specific parameters in request
6. **config.example.yaml** - Simplified user-facing config

### API Changes
- `ProvisionRequest` now includes `ServerType` and `Image` fields
- Config now has `SSH` section instead of nested in `Defaults`
- Legacy `Defaults` field is now optional (pointer) for backward compatibility

### Provider Interface
Providers can optionally implement automatic machine selection:
```go
type MachineProfileSelector interface {
    SelectBestServerType(ctx context.Context, profile MachineProfile, preferredLocations []string) (string, []string, error)
}
```

## Testing

All existing tests pass with the new system:
- ✅ Unit tests for profile mapping
- ✅ Config validation tests
- ✅ Backward compatibility tests
- ✅ Integration tests (with legacy and new format)

## Future Improvements

1. **Add More Profiles**: `gpu`, `storage-optimized`, etc.
2. **Cost Optimization**: Automatically select cheapest option within profile
3. **Geographic Preferences**: Allow users to specify region (EU, US, ASIA) instead of exact locations
4. **Multi-Provider**: Single config that works across Hetzner, AWS, GCP, etc.

## Conclusion

This change makes Morpheus significantly easier to use while maintaining full backward compatibility. Users can now focus on their infrastructure needs (small/medium/large) rather than provider-specific implementation details.
