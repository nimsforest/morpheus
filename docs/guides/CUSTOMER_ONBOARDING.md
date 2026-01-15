# Customer Onboarding Guide

This guide walks through the process of onboarding a new customer with DNS delegation to Hetzner DNS.

## Prerequisites

Before starting, ensure you have:

1. **Hetzner Cloud Account** - Create one at [console.hetzner.cloud](https://console.hetzner.cloud/)
2. **Hetzner DNS API Token** - Generate a token for the customer's Hetzner project
3. **Customer Domain** - The subdomain that will be delegated (e.g., `services.customer.com`)
4. **Morpheus Installed** - See [README](../../README.md) for installation

## Onboarding Process

### Step 1: Create Hetzner Project (Manual)

1. Log in to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Create a new project named `customer-<name>` (e.g., `customer-acme`)
3. Navigate to **Security > API Tokens**
4. Generate a new token with **Read & Write** permissions
5. Store the token securely (e.g., in a password manager)

### Step 2: Initialize Customer in Morpheus

```bash
morpheus customer init <customer-id> --domain <delegated-subdomain>
```

Example:
```bash
morpheus customer init acme --domain services.acme.example.com --name "ACME Corp"
```

When prompted, enter the Hetzner DNS token. You can provide:
- A direct API token
- An environment variable reference (e.g., `${ACME_DNS_TOKEN}`)

### Step 3: Create DNS Zone

Create the zone for the delegated subdomain:

```bash
morpheus dns zone create services.acme.example.com --customer acme
```

This creates the zone in Hetzner DNS and returns the nameservers.

### Step 4: Customer Adds NS Records

The customer must add NS records at their domain registrar pointing the delegated subdomain to Hetzner nameservers:

```
services.acme.example.com  NS  hydrogen.ns.hetzner.com
services.acme.example.com  NS  oxygen.ns.hetzner.com
services.acme.example.com  NS  helium.ns.hetzner.de
```

**Important:** DNS propagation can take up to 48 hours.

### Step 5: Verify Delegation

After the customer adds NS records, verify the delegation:

```bash
morpheus customer verify acme
```

This checks that:
- NS records are properly configured
- The subdomain resolves to Hetzner nameservers

### Step 6: Add DNS Records

Once delegation is verified, you can manage DNS records:

```bash
# Add an A record
morpheus dns record create www.services.acme.example.com A 1.2.3.4 --customer acme

# Add a CNAME record
morpheus dns record create api.services.acme.example.com CNAME www.services.acme.example.com --customer acme

# List records
morpheus dns record list services.acme.example.com --customer acme
```

## Using Ventures (Optional)

For standardized service deployments, use ventures:

```bash
# List available venture templates
morpheus venture list

# Enable a venture for the customer
morpheus venture enable acme experiencenet --server-ip 1.2.3.4

# Check venture status
morpheus venture status acme experiencenet
```

## Verification Steps

### Check Zone Exists

```bash
morpheus dns zone get services.acme.example.com --customer acme
```

### Check NS Delegation

```bash
# Using Morpheus
morpheus customer verify acme

# Using dig (manual)
dig NS services.acme.example.com
```

Expected output should show Hetzner nameservers:
```
services.acme.example.com. 86400 IN NS hydrogen.ns.hetzner.com.
services.acme.example.com. 86400 IN NS oxygen.ns.hetzner.com.
services.acme.example.com. 86400 IN NS helium.ns.hetzner.de.
```

### Check Record Resolution

```bash
# Using dig
dig A www.services.acme.example.com
```

## Troubleshooting

### "DNS lookup failed"

**Cause:** NS records not yet propagated or domain doesn't exist.

**Solution:**
- Wait 24-48 hours for DNS propagation
- Verify NS records are correctly added at the registrar
- Check with `dig NS <domain>` to see current state

### "No DNS token configured"

**Cause:** Customer token not set or environment variable not defined.

**Solution:**
- Check `~/.morpheus/customers.yaml` for the token entry
- If using env var reference, ensure the variable is exported
- Re-run `morpheus customer init` to update the token

### "Zone not found"

**Cause:** Zone hasn't been created in Hetzner DNS.

**Solution:**
```bash
morpheus dns zone create <domain> --customer <customer-id>
```

### Partial NS Delegation

**Cause:** Not all required NS records are configured.

**Solution:**
Ensure ALL three NS records are added at the registrar:
- `hydrogen.ns.hetzner.com`
- `oxygen.ns.hetzner.com`
- `helium.ns.hetzner.de`

### Records Not Resolving

**Cause:** DNS cache or propagation delay.

**Solution:**
- Wait for TTL to expire (default: 300 seconds for records)
- Try querying Hetzner nameservers directly:
  ```bash
  dig @hydrogen.ns.hetzner.com A www.services.acme.example.com
  ```

## Configuration Reference

Customer configuration is stored in `~/.morpheus/customers.yaml`:

```yaml
customers:
  - id: acme
    name: ACME Corp
    domain: services.acme.example.com
    hetzner:
      dns_token: ${ACME_DNS_TOKEN}  # or direct token
```

## Related Documentation

- [DNS Delegation Architecture](../architecture/DNS_DELEGATION.md)
- [Hetzner DNS API](https://dns.hetzner.com/api-docs)
