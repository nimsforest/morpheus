# Environment Setup & Team Sharing Guide

This guide covers how to configure Morpheus using environment variables and securely share configurations across your team.

## Quick Start

```bash
# Copy the example file
cp .env.example .env

# Edit with your values
nano .env  # or your preferred editor

# Verify it's working
morpheus check config
```

## Environment Variables

Morpheus supports the following environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `HETZNER_API_TOKEN` | Yes* | Hetzner Cloud API token for provisioning |
| `HETZNER_DNS_TOKEN` | No | Hetzner DNS API token (falls back to API token) |
| `STORAGEBOX_PASSWORD` | No | Password for Hetzner StorageBox shared registry |
| `MORPHEUS_CONFIG_PATH` | No | Override default config file location |

*Required when using Hetzner as the machine provider.

### Getting Your Tokens

#### Hetzner Cloud API Token
1. Go to [Hetzner Cloud Console](https://console.hetzner.cloud/)
2. Select your project (or create one)
3. Navigate to **Security** → **API Tokens**
4. Click **Generate API Token**
5. Give it a name and select **Read & Write** permissions
6. Copy the token (it's only shown once!)

#### Hetzner DNS API Token
1. Go to [Hetzner DNS Console](https://dns.hetzner.com/)
2. Click **Settings** → **API Tokens**
3. Click **Create access token**
4. Copy the token

## Configuration Hierarchy

Morpheus loads configuration in this order (later sources override earlier):

1. **Default values** (built into the application)
2. **Config file** (`config.yaml` or path from `MORPHEUS_CONFIG_PATH`)
3. **Environment variables** (highest priority)

This means you can:
- Keep non-sensitive settings in `config.yaml` (committed to git)
- Keep secrets in `.env` (gitignored, never committed)

## Secure Team Sharing Methods

### 1. Password Manager (Recommended for Small Teams)

Store your `.env` file in a team password manager like 1Password, Bitwarden, or LastPass.

**Setup with 1Password:**
```bash
# Store the .env file
op document create .env --title "Morpheus .env" --vault "Team"

# Retrieve later
op document get "Morpheus .env" --vault "Team" > .env
```

**Setup with Bitwarden:**
```bash
# Store as a secure note with file attachment
bw create item --template secure-note.json

# Or use the web vault to upload the file as an attachment
```

### 2. Encrypted File Sharing

Use `age` (modern) or `gpg` (traditional) to encrypt before sharing.

**Using age (recommended):**
```bash
# Install age
# macOS: brew install age
# Linux: apt install age / dnf install age

# Encrypt with a passphrase (share passphrase separately!)
age -p .env > .env.age

# Share .env.age via Slack, email, etc.

# Decrypt
age -d .env.age > .env
```

**Using age with team keys:**
```bash
# Each team member generates a key
age-keygen -o key.txt
# Share the PUBLIC key (starts with age1...)

# Encrypt for multiple recipients
age -r age1abc... -r age1def... -r age1ghi... .env > .env.age

# Anyone with their private key can decrypt
age -d -i key.txt .env.age > .env
```

**Using GPG:**
```bash
# Encrypt with passphrase
gpg -c .env
# Creates .env.gpg

# Decrypt
gpg -d .env.gpg > .env
```

### 3. Git-Crypt (Encrypted Files in Repository)

Keep encrypted secrets in the repository itself.

```bash
# Install git-crypt
# macOS: brew install git-crypt
# Linux: apt install git-crypt

# Initialize in your repo
git-crypt init

# Add team members' GPG keys
git-crypt add-gpg-user USER_GPG_KEY_ID

# Create .gitattributes to specify encrypted files
echo ".env.team filter=git-crypt diff=git-crypt" >> .gitattributes

# Create a team env file (will be encrypted on push)
cp .env .env.team
git add .env.team .gitattributes
git commit -m "Add encrypted team env"
```

Team members with access can see decrypted files automatically.

### 4. SOPS (Secrets OPerationS)

Mozilla SOPS supports multiple backends (AWS KMS, GCP KMS, age, PGP).

```bash
# Install sops
# macOS: brew install sops
# Linux: Download from GitHub releases

# Create .sops.yaml config
cat > .sops.yaml << 'EOF'
creation_rules:
  - path_regex: \.env\.enc$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
EOF

# Encrypt
sops -e .env > .env.enc

# Decrypt
sops -d .env.enc > .env

# Edit in place (decrypts, opens editor, re-encrypts)
sops .env.enc
```

### 5. Secret Management Services (Production)

For production or larger teams, use dedicated secret management:

**Doppler:**
```bash
# Install CLI
brew install dopplerhq/cli/doppler

# Login and setup
doppler login
doppler setup

# Run with secrets injected
doppler run -- morpheus plant
```

**HashiCorp Vault:**
```bash
# Store secrets
vault kv put secret/morpheus \
  HETZNER_API_TOKEN="xxx" \
  STORAGEBOX_PASSWORD="yyy"

# Retrieve and export
export $(vault kv get -format=json secret/morpheus | jq -r '.data.data | to_entries | .[] | "\(.key)=\(.value)"')
```

**AWS Secrets Manager:**
```bash
# Store
aws secretsmanager create-secret \
  --name morpheus/env \
  --secret-string file://.env

# Retrieve
aws secretsmanager get-secret-value \
  --secret-id morpheus/env \
  --query SecretString \
  --output text > .env
```

## CI/CD Configuration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Morpheus
        env:
          HETZNER_API_TOKEN: ${{ secrets.HETZNER_API_TOKEN }}
          STORAGEBOX_PASSWORD: ${{ secrets.STORAGEBOX_PASSWORD }}
        run: |
          morpheus plant --nodes 3
```

Add secrets in: **Repository Settings** → **Secrets and variables** → **Actions**

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  script:
    - morpheus plant --nodes 3
  variables:
    HETZNER_API_TOKEN: $HETZNER_API_TOKEN  # Set in CI/CD Variables
```

## Best Practices

1. **Never commit `.env` files** - Always keep real secrets out of version control
2. **Use different tokens per environment** - Development, staging, production
3. **Rotate tokens regularly** - Especially after team member departures
4. **Limit token permissions** - Use read-only tokens where possible
5. **Audit access** - Track who has access to which secrets
6. **Use short-lived tokens** - When supported by the provider

## Troubleshooting

### "hetzner_api_token is required"
```bash
# Check if the variable is set
echo $HETZNER_API_TOKEN

# Check if .env is being loaded
cat .env | grep HETZNER_API_TOKEN

# Ensure no extra whitespace
export HETZNER_API_TOKEN="$(echo $HETZNER_API_TOKEN | tr -d '[:space:]')"
```

### Token works in shell but not in app
```bash
# Some shells don't export by default
export HETZNER_API_TOKEN="your-token"

# Or source the .env file
set -a && source .env && set +a
```

### Different configs for different environments
```bash
# Use separate config files
MORPHEUS_CONFIG_PATH=./config.production.yaml morpheus plant

# Or use .env.production
cp .env.production .env
```

## See Also

- [config.example.yaml](../../config.example.yaml) - Full configuration reference
- [.env.example](../../.env.example) - Environment variable template
- [CONTRIBUTING.md](../development/CONTRIBUTING.md) - Development setup
