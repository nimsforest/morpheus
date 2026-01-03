# Troubleshooting Guide

This guide helps resolve common issues when using Morpheus.

## Provisioning Errors

### Server Type Not Available in Configured Locations

**Error message:**
```
❌ Provisioning failed: all configured locations are unavailable (fsn1, nbg1, hel1): 
failed to provision node forest-xxx-node-1: failed to create server: 
unsupported location for server type (invalid_input)
```

**What happened:**
The server type configured in your `config.yaml` (e.g., `cx11`) is not available in any of your configured locations (e.g., `fsn1`, `nbg1`, `hel1`). Hetzner Cloud has different server types available in different locations.

**Solution:**
Morpheus will automatically detect this issue and present an interactive menu with options:

1. **Use a different location** - If your server type is available in other locations, you can select one
2. **Change server type** (recommended) - Select from recommended server types that are available in your locations
3. **Exit and update config manually** - Manually edit your `config.yaml`

**Example interactive menu:**
```
What would you like to do?

  [1] Use a different location for 'cx11'
      Available: ash, hil

  [2] Change server type (recommended)
      Suggested server types:
        • cx22: 2 vCPU (shared AMD), 4 GB RAM - ~€3.29/mo
          Locations: fsn1, nbg1, hel1, ash, hil
        • cpx11: 2 vCPU (dedicated AMD), 2 GB RAM - ~€4.49/mo
          Locations: fsn1, nbg1, hel1
        • cax11: 2 vCPU (ARM), 4 GB RAM - ~€3.79/mo
          Locations: fsn1, nbg1, hel1

  [3] Exit and update config manually

Enter choice (1/2/3):
```

**Manual fix:**
If you prefer to fix this manually, update your `config.yaml`:

```yaml
infrastructure:
  provider: hetzner
  defaults:
    server_type: cx22  # Change to an available type
    # ... other settings
  locations:
    - fsn1  # Falkenstein, Germany
    - nbg1  # Nuremberg, Germany
    - hel1  # Helsinki, Finland
```

**Recommended server types:**
- `cx22` - 2 vCPU (shared), 4 GB RAM - €3.29/mo - Available in most locations
- `cpx11` - 2 vCPU (dedicated), 2 GB RAM - €4.49/mo - Better performance
- `cpx21` - 3 vCPU (dedicated), 4 GB RAM - €8.49/mo - Production workloads
- `cax11` - 2 vCPU (ARM), 4 GB RAM - €3.79/mo - ARM-based, cost-effective

**Check availability:**
You can check server type availability in Hetzner Cloud Console:
- https://console.hetzner.cloud/ → Your Project → Servers → Create Server
- Select a location to see available server types

### Location Temporarily Unavailable

**Error message:**
```
⚠️  Location fsn1 is unavailable, trying next location...
```

**What happened:**
Hetzner Cloud may temporarily have capacity issues in a specific location. Morpheus automatically tries other configured locations.

**Solution:**
No action needed - Morpheus will automatically fall back to alternative locations. If all locations fail, you'll see the interactive menu (see above).

## Authentication Errors

### Invalid API Token

**Error message:**
```
Failed to create provider: API token contains invalid characters: newline (\n) at position 64
```

**What happened:**
Your Hetzner API token contains invalid characters (newlines, spaces, etc.). This often happens when copying from the console.

**Solution:**
1. Go to Hetzner Cloud Console: https://console.hetzner.cloud/
2. Navigate to your project → Security → API Tokens
3. Copy the token carefully (no extra spaces or newlines)
4. Set it in your environment or config:
   ```bash
   export HETZNER_API_TOKEN="your_token_here"
   ```
5. Verify the token is set correctly:
   ```bash
   echo "$HETZNER_API_TOKEN" | od -c  # Should show no \n or \r
   ```

### Unauthorized Error

**Error message:**
```
failed to get server type: Unauthorized

This usually means:
  1. The API token is invalid, revoked, or expired
  2. The token was copied incorrectly (missing characters)
  3. The token doesn't have the required permissions
```

**Solution:**
1. Generate a new API token in Hetzner Cloud Console
2. Ensure it has **Read & Write** permissions
3. Update your token:
   ```bash
   export HETZNER_API_TOKEN="your_new_token"
   ```

## IPv6 Connectivity Issues

### IPv6 Not Available

**Error message:**
```
❌ IPv6 connectivity is NOT available
```

**What happened:**
Morpheus requires IPv6 connectivity because Hetzner Cloud uses IPv6-only by default (IPv4 costs extra). Your network doesn't have IPv6 enabled.

**Solution:**
Check IPv6 connectivity:
```bash
morpheus check-ipv6
```

Options to get IPv6:
1. **Enable IPv6 on your ISP/router** (best option)
2. **Use an IPv6 tunnel service** (e.g., Hurricane Electric)
3. **Use a VPS/server with IPv6** to run Morpheus
4. **Use Termux on Android** (most mobile networks support IPv6)

See [docs/guides/IPV6_SETUP.md](IPV6_SETUP.md) for detailed IPv6 setup instructions.

## Docker Issues (Local Mode)

### Docker Not Available

**Error message:**
```
Failed to create local provider: Cannot connect to the Docker daemon
```

**What happened:**
Docker is not installed or not running. Local mode requires Docker.

**Solution:**
1. Install Docker:
   ```bash
   # Ubuntu/Debian
   curl -fsSL https://get.docker.com | sh
   
   # Start Docker
   sudo systemctl start docker
   ```

2. Verify Docker is running:
   ```bash
   docker info
   ```

**Termux users:**
Docker does **NOT** work on Termux/Android due to kernel limitations. Use cloud mode instead:
```bash
morpheus plant cloud wood
```

## SSH Key Issues

### SSH Key Not Found

**Error message:**
```
SSH key 'main' not found in Hetzner Cloud and could not read local key
```

**What happened:**
Morpheus couldn't find the SSH public key locally to auto-upload it to Hetzner Cloud.

**Solution:**
1. Check if you have an SSH key:
   ```bash
   ls -la ~/.ssh/*.pub
   ```

2. If no key exists, generate one:
   ```bash
   ssh-keygen -t ed25519 -C "your_email@example.com"
   ```

3. Verify the key name in your config matches your SSH key:
   ```yaml
   infrastructure:
     defaults:
       ssh_key: main  # Should match ~/.ssh/main.pub or ~/.ssh/id_ed25519.pub
   ```

4. Optionally specify a custom path:
   ```yaml
   infrastructure:
     defaults:
       ssh_key: main
       ssh_key_path: "~/.ssh/custom_key.pub"
   ```

## Update Issues

### Update Check Fails

**Error message:**
```
Failed to check for updates: connection refused
```

**What happened:**
Network connectivity issue or GitHub is temporarily unavailable.

**Solution:**
1. Check internet connectivity:
   ```bash
   curl -I https://github.com
   ```

2. Try again later - GitHub may be experiencing issues

3. Manually download from:
   https://github.com/nimsforest/morpheus/releases/latest

## Common Configuration Mistakes

### Missing Config File

**Error message:**
```
no config file found (tried: [./config.yaml ~/.morpheus/config.yaml /etc/morpheus/config.yaml])
```

**Solution:**
1. Create a config file from the example:
   ```bash
   cp config.example.yaml ~/.morpheus/config.yaml
   ```

2. Edit with your settings:
   ```bash
   nano ~/.morpheus/config.yaml
   ```

### Invalid YAML Syntax

**Error message:**
```
Invalid config: yaml: unmarshal errors
```

**Solution:**
1. Validate YAML syntax:
   ```bash
   cat ~/.morpheus/config.yaml | python3 -c 'import yaml, sys; yaml.safe_load(sys.stdin)'
   ```

2. Common YAML mistakes:
   - Incorrect indentation (use 2 spaces, not tabs)
   - Missing colons after keys
   - Unquoted strings with special characters

## Getting Help

If you're still experiencing issues:

1. **Check the documentation:**
   - [README.md](../../README.md) - Getting started
   - [ANDROID_TERMUX.md](ANDROID_TERMUX.md) - Android/Termux setup
   - [IPV6_SETUP.md](IPV6_SETUP.md) - IPv6 configuration
   - [CONTROL_SERVER_SETUP.md](CONTROL_SERVER_SETUP.md) - Control server setup

2. **Check GitHub issues:**
   - https://github.com/nimsforest/morpheus/issues

3. **Open a new issue:**
   - Include the full error message
   - Include your `morpheus version`
   - Include relevant config (remove sensitive data like API tokens)
   - Include your OS and environment (Termux, Linux, etc.)

4. **Enable verbose logging:**
   ```bash
   export MORPHEUS_DEBUG=1
   morpheus plant cloud wood
   ```
