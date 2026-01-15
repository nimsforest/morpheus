#!/bin/bash
# Setup nimsforest.com as apex domain in Hetzner DNS
# Run this on a machine with HETZNER_API_TOKEN set

set -e

DOMAIN="nimsforest.com"

echo "Setting up $DOMAIN as apex domain..."
echo ""

# Check for API token
if [[ -z "$HETZNER_API_TOKEN" ]]; then
    echo "Error: HETZNER_API_TOKEN environment variable is not set"
    echo "Set it with: export HETZNER_API_TOKEN=\"your-token\""
    exit 1
fi

# Check if zone already exists
echo "Checking if zone exists..."
EXISTING=$(curl -s -H "Auth-API-Token: $HETZNER_API_TOKEN" \
    "https://dns.hetzner.com/api/v1/zones" | \
    grep -o "\"name\":\"$DOMAIN\"" || true)

if [[ -n "$EXISTING" ]]; then
    echo "Zone $DOMAIN already exists!"
    echo ""
    echo "To check zone details, run:"
    echo "  morpheus dns status $DOMAIN"
    exit 0
fi

# Create the zone
echo "Creating DNS zone for $DOMAIN..."
RESPONSE=$(curl -s -X POST "https://dns.hetzner.com/api/v1/zones" \
    -H "Auth-API-Token: $HETZNER_API_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"$DOMAIN\"}")

# Check for errors
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "Error creating zone:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

ZONE_ID=$(echo "$RESPONSE" | jq -r '.zone.id' 2>/dev/null)

echo ""
echo "Zone created successfully!"
echo "Zone ID: $ZONE_ID"
echo ""
echo "============================================"
echo "NEXT STEPS:"
echo "============================================"
echo ""
echo "1. Update your domain registrar's nameservers to:"
echo "   - hydrogen.ns.hetzner.com"
echo "   - oxygen.ns.hetzner.com"
echo "   - helium.ns.hetzner.de"
echo ""
echo "2. Wait for DNS propagation (up to 48 hours)"
echo ""
echo "3. Verify delegation with:"
echo "   morpheus dns verify $DOMAIN"
echo ""
echo "4. Add DNS records as needed:"
echo "   morpheus dns record create www.$DOMAIN CNAME $DOMAIN"
echo "   morpheus dns record create $DOMAIN A <your-server-ip>"
echo ""
