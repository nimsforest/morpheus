# DNS Management via Hetzner

This document describes how Morpheus manages DNS via Hetzner DNS.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Hetzner DNS                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  NimsForest Project          Customer Project (Acme)                │
│  ┌─────────────────────┐     ┌─────────────────────────────────┐   │
│  │ nimsforest.com      │     │ experiencenet.acme.com          │   │
│  │ experiencenet.io    │     │ nimsforest.acme.com             │   │
│  └─────────────────────┘     └─────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                    │                         │
                    │                         │
                    ▼                         ▼
              Morpheus CLI              Morpheus CLI
           (maintainer token)        (maintainer token)
```

### Key Concepts

- **One Hetzner project per customer**: Each customer has an isolated Hetzner project
- **One Morpheus API token per maintainer**: Maintainers use their own tokens
- **DNS zones in customer's project**: Zones for delegated subdomains live in the customer's project
- **NimsForest compute**: Runs on Hetzner servers within the same project
- **ExperienceNet edge nodes**: May be bare metal outside Hetzner, but DNS records pointing to them are managed through Hetzner DNS

---

## Use Case 1: We Control the Apex Domain (Internal Ventures)

For domains we own (e.g., `nimsforest.com`, `experiencenet.io`):

### Setup

1. Set nameservers at the registrar to Hetzner:
   ```
   hydrogen.ns.hetzner.com
   oxygen.ns.hetzner.com
   helium.ns.hetzner.de
   ```

2. Create zone in Hetzner DNS

3. Morpheus manages all records

### Example Zone Structure

```
┌──────────────────────────────────────────────────────────┐
│  nimsforest.com (Hetzner DNS - NimsForest Project)       │
│                                                          │
│  @              A      <server-ip>                       │
│  @              AAAA   <server-ipv6>                     │
│  www            CNAME  @                                 │
│  api            A      <api-server-ip>                   │
│  app            A      <app-server-ip>                   │
│  *.forests      A      <forest-lb-ip>                    │
└──────────────────────────────────────────────────────────┘
```

---

## Use Case 2: Customer Delegates a Subdomain

Customers delegate a subdomain to our DNS by adding NS records pointing to Hetzner's nameservers on their side. The subdomain corresponds to the venture service they're using.

### Customer DNS Setup

Customer adds NS records on their existing DNS provider:

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

### Customer Onboarding Flow

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
│  4. [Morpheus] Provision records based on enabled venture services     │
│         │                                                              │
│         ▼                                                              │
│  5. [Customer] Add NS records on their DNS provider                    │
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

4. **Morpheus provisions records**
   - Based on which venture services are enabled
   - Based on where infrastructure is deployed

5. **Customer configures NS records**
   - Customer adds NS records pointing to Hetzner nameservers
   - Provide customer with copy-paste instructions

### Root Domain Hosting

If NimsForest manages the customer's marketing website, we create a record under the delegated subdomain (e.g., `www.nimsforest.customer.com`). Customer then adds:

**Option A: CNAME + ALIAS (Preferred)**

```
www.customer.com     CNAME  www.nimsforest.customer.com
customer.com         ALIAS  www.nimsforest.customer.com  (or ANAME)
```

**Option B: Static IP (Fallback)**

For customers whose DNS provider doesn't support ALIAS/ANAME at the apex:

```
www.customer.com     CNAME  www.nimsforest.customer.com
customer.com         A      <static-ip-we-provision>
```

---

## What We Need to Build

- Zone creation and record management via Hetzner DNS API
- Support for multiple venture prefixes per customer

---

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
  -H "Auth-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "experiencenet.customer.com", "ttl": 86400}'
```

### Example: Create Record

```bash
curl -X POST "https://dns.hetzner.com/api/v1/records" \
  -H "Auth-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "zone_id": "<zone-id>",
    "name": "www",
    "type": "A",
    "value": "1.2.3.4",
    "ttl": 300
  }'
```

---

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

---

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

---

## Related Documentation

- [Hetzner API Mapping](HETZNER_MAPPING.md) - General Hetzner API patterns
- [Separation of Concerns](SEPARATION_OF_CONCERNS.md) - How Morpheus and NimsForest work together
