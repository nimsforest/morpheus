# Feature: DNS + Plant Integration

## Status: planning

## Summary

Wire up `morpheus plant` to automatically add DNS records after infrastructure is provisioned, and add verification commands to check NS delegation before proceeding.

## Tasks

### Phase 1: DNS Verification for Apex Domains

- [ ] Task 1.1 - Add `morpheus dns verify <domain>` command
  - Checks if NS records point to Hetzner nameservers
  - Works for both apex and subdomain zones
  - Shows clear pass/fail status

### Phase 2: Plant + DNS Integration

- [ ] Task 2.1 - Add `--domain` flag to `morpheus plant`
  - Associates forest with a DNS zone
  - Stores domain in forest metadata

- [ ] Task 2.2 - Auto-add A records after plant completes
  - Create A record for apex (@) pointing to first node IP
  - Create A records for each node (node1.domain, node2.domain, etc.)
  - Only if `--domain` flag is provided

- [ ] Task 2.3 - Auto-cleanup DNS records on teardown
  - Remove A records when `morpheus teardown` is called
  - Only if forest has associated domain

### Phase 3: Verification Flow (Optional)

- [ ] Task 3.1 - Add `--verify` flag to `morpheus plant`
  - Before provisioning, verify NS delegation is working
  - Fail early with helpful message if not configured

- [ ] Task 3.2 - Add TXT record verification (optional)
  - Create TXT record with unique token during `dns add`
  - `dns verify` checks if TXT record resolves
  - Confirms zone is live before plant proceeds

## Example Flow

```bash
# 1. Create zone
morpheus dns add apex nimsforest.com

# 2. Configure nameservers at registrar (manual)

# 3. Verify delegation works
morpheus dns verify nimsforest.com
# ✓ NS delegation verified

# 4. Plant with DNS integration
morpheus plant --domain nimsforest.com

# Output includes:
# ✓ Created node-1 at 1.2.3.4
# ✓ Created node-2 at 1.2.3.5
# ✓ Added DNS: nimsforest.com → 1.2.3.4
# ✓ Added DNS: node1.nimsforest.com → 1.2.3.4
# ✓ Added DNS: node2.nimsforest.com → 1.2.3.5
```

## Files

- `internal/commands/dns_simple.go` - Add verify command
- `internal/commands/plant.go` - Add --domain flag and DNS integration
- `internal/commands/teardown.go` - Add DNS cleanup
- `pkg/forest/provisioner.go` - Add DNS record creation after provisioning

## Notes

- DNS record creation should be best-effort (don't fail plant if DNS fails)
- Consider IPv6 AAAA records as well as A records
- Node naming: node1, node2, etc. or use forest-id prefix?
