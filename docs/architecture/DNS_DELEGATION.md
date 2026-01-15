# DNS Delegation for Customer Subdomains

This document describes the architecture for managing DNS for customer subdomains using Hetzner DNS.

## Overview

Customers delegate a subdomain to our DNS by adding NS records pointing to Hetzner's nameservers on their side. The subdomain corresponds to the venture service they're using (e.g., `experiencenet.customer.com`, `nimsforest.customer.com`). Morpheus manages all records via Hetzner DNS.

## Architecture

```
Customer's DNS Provider                    Hetzner DNS (via Morpheus)
┌─────────────────────────┐               ┌─────────────────────────────────┐
│                         │               │                                 │
│  customer.com           │               │  experiencenet.customer.com     │
│  └─ experiencenet  NS ──┼──────────────>│  └─ www          A  1.2.3.4     │
│                         │               │  └─ api          A  1.2.3.5     │
│  └─ nimsforest     NS ──┼──────────────>│                                 │
│                         │               │  nimsforest.customer.com        │
└─────────────────────────┘               │  └─ www          A  5.6.7.8     │
                                          │  └─ nats         A  5.6.7.9     │
                                          └─────────────────────────────────┘
```

### Key Concepts

- **One Hetzner project per customer**: Each customer has an isolated Hetzner project
- **Project-scoped API token**: Each project has its own Morpheus API token (token determines project/org scope)
- **Delegated zones**: DNS zones for all delegated subdomains live in the customer's project
- **NimsForest compute**: Runs on Hetzner servers within the same project
- **ExperienceNet edge nodes**: May be bare metal outside Hetzner, but DNS records are managed through Hetzner DNS

## Customer DNS Setup

Customers add NS records on their existing DNS provider to delegate the subdomain:

```
experiencenet.customer.com  NS  hydrogen.ns.hetzner.com
experiencenet.customer.com  NS  oxygen.ns.hetzner.com
experiencenet.customer.com  NS  helium.ns.hetzner.de
```

For multiple venture services, each subdomain is delegated separately:

```
experiencenet.customer.com  NS  hydrogen.ns.hetzner.com
experiencenet.customer.com  NS  oxygen.ns.hetzner.com
experiencenet.customer.com  NS  helium.ns.hetzner.de

nimsforest.customer.com     NS  hydrogen.ns.hetzner.com
nimsforest.customer.com     NS  oxygen.ns.hetzner.com
nimsforest.customer.com     NS  helium.ns.hetzner.de
```

## Root Domain Hosting

When NimsForest manages the customer's marketing website, we create a record under the delegated subdomain (e.g., `www.nimsforest.customer.com`). Customer then configures:

### Option A: CNAME/ALIAS (Preferred)

```
www.customer.com     CNAME  www.nimsforest.customer.com
customer.com         ALIAS  www.nimsforest.customer.com  (or ANAME)
```

### Option B: Static IP (Fallback)

For customers whose DNS provider doesn't support ALIAS/ANAME at the apex, we provision a static IP and they use an A record:

```
www.customer.com     CNAME  www.nimsforest.customer.com
customer.com         A      <static-ip-we-provision>
```

## Customer Onboarding Flow

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Onboarding Process                              │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  1. [Manual] Create Hetzner project for customer                       │
│         │                                                              │
│         ▼                                                              │
│  2. [Manual] Generate API token, store in Bitwarden                    │
│         │                                                              │
│         ▼                                                              │
│  3. [Morpheus] Create DNS zone(s) using project-specific token         │
│         │                                                              │
│         ▼                                                              │
│  4. [Customer] Add NS records on their DNS provider                    │
│         │                                                              │
│         ▼                                                              │
│  5. [Morpheus] Provision records based on enabled venture services     │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Detailed Steps

1. **Create Hetzner project** (Manual)
   - Create a new project in Hetzner Cloud Console for the customer
   - Name convention: `customer-<name>` (e.g., `customer-acme`)

2. **Generate and store API token** (Manual)
   - Generate API token in the Hetzner project
   - Store token as secure note in Bitwarden
   - Reference format: `morpheus/<customer-name>/hetzner-dns-token`

3. **Morpheus creates zone(s)**
   - Using the customer-specific token
   - Zone name = delegated subdomain (e.g., `experiencenet.customer.com`)

4. **Customer configures NS records**
   - Customer adds NS records pointing to Hetzner nameservers
   - Provide customer with copy-paste instructions

5. **Morpheus provisions records**
   - Based on which venture services are enabled
   - Based on where infrastructure is deployed

## Hetzner DNS API

- **Documentation**: https://dns.hetzner.com/api-docs
- **Authentication**: Bearer token in header (token scoped to project)
- **Key endpoints**:
  - `POST /zones` - Create zone
  - `GET /zones` - List zones
  - `POST /records` - Add records
  - `GET /records` - List records
  - `DELETE /records/{id}` - Delete record

### Example: Create Zone

```bash
curl -X POST "https://dns.hetzner.com/api/v1/zones" \
  -H "Auth-API-Token: <customer-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "experiencenet.customer.com", "ttl": 86400}'
```

### Example: Create Record

```bash
curl -X POST "https://dns.hetzner.com/api/v1/records" \
  -H "Auth-API-Token: <customer-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "zone_id": "<zone-id>",
    "name": "www",
    "type": "A",
    "value": "1.2.3.4",
    "ttl": 300
  }'
```

## Security Considerations

### Token Isolation

Each customer has an isolated API token, so a compromised token only affects that customer's DNS zones and records.

### Token Storage

- Tokens stored as secure notes in Bitwarden
- Never committed to version control
- Passed to Morpheus via environment variables or secure config

### Trust Model

Document the following in customer terms of service:
- NimsForest manages DNS records for delegated subdomains
- Customer retains control of their root domain
- Changes to DNS are automated based on infrastructure deployments

## Post-MVP Enhancements

### Audit Logging

- Log all DNS changes with timestamp, actor, and before/after state
- Integrate with centralized logging system

### Alerting

- Alert on unexpected record modifications
- Alert on zone deletion attempts
- Alert on bulk record changes

### Token Rotation

- Define rotation policy (e.g., quarterly)
- Automate rotation process
- Update Bitwarden entries automatically

## Related Documentation

- [Hetzner API Mapping](HETZNER_MAPPING.md) - General Hetzner API patterns
- [Separation of Concerns](SEPARATION_OF_CONCERNS.md) - How Morpheus and NimsForest work together
