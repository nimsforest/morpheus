# Size Names Update - Breaking Change

## Summary

Changed forest size names from nature-themed words to simple, clear descriptors:
- `wood` → `small`
- `forest` → `medium`
- `jungle` → `large`

**No backward compatibility** - old names are completely removed.

## Rationale

1. **Clarity**: `small`, `medium`, `large` are universally understood
2. **Professional**: Nature names were cute but potentially confusing
3. **Consistency**: Aligns with machine profile system (ProfileSmall, ProfileMedium, ProfileLarge)
4. **International**: Easier for non-native English speakers

## Changes Made

### Command Interface
```bash
# OLD (no longer works)
morpheus plant cloud wood
morpheus plant cloud forest
morpheus plant cloud jungle

# NEW
morpheus plant cloud small
morpheus plant cloud medium
morpheus plant cloud large
```

### Files Modified
1. **cmd/morpheus/main.go** - Command parsing and validation
2. **pkg/forest/provisioner.go** - Size constants in comments
3. **pkg/forest/provisioner_test.go** - Test cases
4. **pkg/forest/registry.go** - Documentation
5. **pkg/forest/registry_test.go** - Test fixtures
6. **README.md** - All examples and documentation
7. **docs/guides/TERMUX_QUICKSTART.md** - User guides
8. **scripts/install-termux.sh** - Installation scripts
9. **config.example.yaml** - Configuration examples

### Size Specifications

| Size   | Machines | Est. Time  | Est. Cost/mo | Use Case          |
|--------|----------|------------|--------------|-------------------|
| small  | 1        | 5-7 min    | €3-4         | Testing, dev      |
| medium | 3        | 15-20 min  | €9-12        | Small production  |
| large  | 5        | 25-35 min  | €15-20       | Larger deployment |

### Testing

All tests pass:
```bash
✅ pkg/cloudinit
✅ pkg/config  
✅ pkg/forest
✅ pkg/httputil
✅ pkg/provider/hetzner
✅ pkg/provider/local
✅ pkg/updater
✅ pkg/updater/version
```

## Migration Guide for Users

### For Existing Scripts/Automation
Update any scripts that use the old size names:
```bash
# OLD
morpheus plant cloud wood

# NEW  
morpheus plant cloud small
```

### For Existing Forests
No migration needed - existing forests continue to run. Only new deployments use the new names.

### Error Messages
If users try to use old names:
```bash
$ morpheus plant cloud wood
❌ Invalid size: 'wood'

Valid sizes:
  small  - 1 machine  (quick start, ~€3-4/mo)
  medium - 3 machines (small cluster, ~€9-12/mo)
  large  - 5 machines (large cluster, ~€15-20/mo)

💡 Did you mean: morpheus plant wood
```

## Benefits

1. **Clearer Intent**: Size names directly communicate scale
2. **Easier Documentation**: No need to explain "wood = 1 machine"
3. **Better UX**: Users can guess what "medium" means
4. **Scalability**: Easy to add "xlarge", "2xlarge" etc. in future
5. **Professionalism**: More enterprise-friendly terminology

## Implementation Details

### Validation Function
```go
func isValidSize(size string) bool {
    validSizes := []string{"small", "medium", "large"}
    for _, valid := range validSizes {
        if size == valid {
            return true
        }
    }
    return false
}
```

### Node Count Mapping
```go
func getNodeCount(size string) int {
    switch size {
    case "small":  return 1
    case "medium": return 3
    case "large":  return 5
    default:       return 1
    }
}
```

## Related Changes

This update pairs with the provider abstraction changes:
- Users no longer need to know about Hetzner machine types (cx22, cpx11, etc.)
- Users no longer need to configure locations (fsn1, nbg1, etc.)
- Size names map to abstract machine profiles
- System automatically selects appropriate infrastructure

## Future Considerations

### Potential Additions
- `xlarge` - 7-10 machines
- `2xlarge` - 10+ machines
- Custom sizes: `morpheus plant cloud custom --nodes=15`

### Alternative Considered
- Numeric: `morpheus plant cloud 1`, `morpheus plant cloud 3`
  - Rejected: Less descriptive, harder to remember
- T-shirt sizes: `xs`, `s`, `m`, `l`, `xl`
  - Rejected: Too casual, confusion about what "xs" means

## Conclusion

The change to `small`, `medium`, `large` provides a better user experience through clarity and simplicity. The breaking change is justified by significantly improved UX and consistency with the overall architecture.
