package cloudinit

import (
	"bytes"
	"fmt"
	"text/template"
)

// NodeRole represents the type of node being provisioned
type NodeRole string

const (
	RoleEdge    NodeRole = "edge"
	RoleCompute NodeRole = "compute"
	RoleStorage NodeRole = "storage"
)

// TemplateData contains data for cloud-init template rendering
type TemplateData struct {
	NodeRole    NodeRole
	ForestID    string
	RegistryURL string // Optional: Morpheus registry for infrastructure state
	CallbackURL string // Optional: NimsForest callback URL for bootstrap trigger
	SSHKeys     []string

	// NimsForest auto-installation
	NimsForestInstall bool   // Auto-install NimsForest from GitHub releases
	NimsForestRepo    string // GitHub repo (e.g., "nimsforest/nimsforest")
	NimsForestBinary  string // Binary name pattern (e.g., "nimsforest-linux-amd64")
}

// EdgeNodeTemplate is the cloud-init script for edge nodes
// Morpheus responsibility: Infrastructure setup only (OS, network, firewall)
// NimsForest responsibility: NATS installation and configuration
const EdgeNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - ufw
  - jq
  - systemd

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "role": "{{.NodeRole}}",
        "provisioner": "morpheus",
        "provisioned_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "registry_url": "{{.RegistryURL}}",
        "callback_url": "{{.CallbackURL}}"
      }
    permissions: '0644'

  - path: /usr/local/bin/morpheus-bootstrap
    content: |
      #!/bin/bash
      # Morpheus bootstrap script - called by NimsForest
      set -e
      
      INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
      INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || echo "unknown")
      LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone || echo "unknown")
      
      echo "Instance IP: $INSTANCE_IP"
      echo "Instance ID: $INSTANCE_ID"
      echo "Location: $LOCATION"
      
      # Export for nimsforest
      export MORPHEUS_IP=$INSTANCE_IP
      export MORPHEUS_INSTANCE_ID=$INSTANCE_ID
      export MORPHEUS_LOCATION=$LOCATION
    permissions: '0755'

runcmd:
  # Configure firewall - NATS ports for nimsforest
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 4222/tcp comment 'NATS client port'
  - ufw allow 6222/tcp comment 'NATS cluster port'
  - ufw allow 8222/tcp comment 'NATS monitoring port'
  - ufw allow 7777/tcp comment 'NATS leafnode port'
  - ufw --force enable
  
  # Create directories for nimsforest (binaries, data, logs)
  - mkdir -p /opt/nimsforest/bin /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  - chown -R ubuntu:ubuntu /opt/nimsforest /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  
  # Prepare for direct binary deployment (systemd services managed by NimsForest)
  - systemctl daemon-reload
  
  # Get instance metadata
  - /usr/local/bin/morpheus-bootstrap
  
  {{if .NimsForestInstall}}
  # Download and install NimsForest from GitHub releases
  - |
    echo "📦 Installing NimsForest from GitHub releases..."
    NIMSFOREST_REPO="{{.NimsForestRepo}}"
    NIMSFOREST_BINARY="{{if .NimsForestBinary}}{{.NimsForestBinary}}{{else}}nimsforest-linux-amd64{{end}}"
    
    # Get latest release version from GitHub API
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/${NIMSFOREST_REPO}/releases/latest" | jq -r '.tag_name // empty')
    
    if [ -z "$LATEST_VERSION" ]; then
      echo "⚠️  Could not determine latest version, trying 'latest' tag..."
      LATEST_VERSION="latest"
    fi
    
    echo "📥 Downloading NimsForest ${LATEST_VERSION}..."
    DOWNLOAD_URL="https://github.com/${NIMSFOREST_REPO}/releases/download/${LATEST_VERSION}/${NIMSFOREST_BINARY}"
    
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "$DOWNLOAD_URL"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      echo "✅ NimsForest installed to /opt/nimsforest/bin/nimsforest"
      
      # Create systemd service for NimsForest
      cat > /etc/systemd/system/nimsforest.service << 'SERVICEEOF'
[Unit]
Description=NimsForest Service
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
ExecStart=/opt/nimsforest/bin/nimsforest start --forest-id {{.ForestID}}
Restart=always
RestartSec=5
Environment=FOREST_ID={{.ForestID}}
Environment=NODE_ROLE={{.NodeRole}}
WorkingDirectory=/var/lib/nimsforest

[Install]
WantedBy=multi-user.target
SERVICEEOF

      systemctl daemon-reload
      systemctl enable nimsforest
      systemctl start nimsforest
      echo "✅ NimsForest service started"
    else
      echo "⚠️  Failed to download NimsForest from ${DOWNLOAD_URL}"
      echo "    You can manually install later with:"
      echo "    curl -fsSL -o /opt/nimsforest/bin/nimsforest ${DOWNLOAD_URL}"
    fi
  {{end}}
  
  # Signal readiness to registry (infrastructure ready{{if not .NimsForestInstall}}, waiting for nimsforest{{end}})
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
    INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone || echo "unknown")
    
    if [ "{{.RegistryURL}}" != "" ]; then
      curl -X POST {{.RegistryURL}}/api/v1/nodes \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"role\": \"{{.NodeRole}}\",
          \"ip\": \"$INSTANCE_IP\",
          \"location\": \"$LOCATION\",
          \"status\": \"infrastructure_ready\",
          \"provisioner\": \"morpheus\"
        }" || echo "Registry notification failed (expected if registry not available)"
    fi
  
  # Trigger nimsforest bootstrap if callback URL provided
  - |
    if [ "{{.CallbackURL}}" != "" ]; then
      INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
      INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
      
      curl -X POST {{.CallbackURL}}/api/v1/bootstrap \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"node_ip\": \"$INSTANCE_IP\",
          \"role\": \"{{.NodeRole}}\"
        }" || echo "NimsForest callback failed (will retry via polling)"
    fi

final_message: "Morpheus infrastructure provisioning complete. Ready for NimsForest bootstrap."
`

// ComputeNodeTemplate is the cloud-init script for compute nodes
// Morpheus responsibility: Infrastructure setup only
// NimsForest responsibility: Worker/compute service installation
const ComputeNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - ufw
  - jq
  - systemd

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "role": "{{.NodeRole}}",
        "provisioner": "morpheus",
        "provisioned_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "registry_url": "{{.RegistryURL}}",
        "callback_url": "{{.CallbackURL}}"
      }
    permissions: '0644'

runcmd:
  # Configure firewall - minimal for compute
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 4222/tcp comment 'NATS client connection'
  - ufw --force enable
  
  # Create directories for nimsforest (binaries, data, logs)
  - mkdir -p /opt/nimsforest/bin /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  - chown -R ubuntu:ubuntu /opt/nimsforest /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  
  # Prepare for direct binary deployment
  - systemctl daemon-reload
  
  # Signal readiness
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
    INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone || echo "unknown")
    
    if [ "{{.RegistryURL}}" != "" ]; then
      curl -X POST {{.RegistryURL}}/api/v1/nodes \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"role\": \"{{.NodeRole}}\",
          \"ip\": \"$INSTANCE_IP\",
          \"location\": \"$LOCATION\",
          \"status\": \"infrastructure_ready\",
          \"provisioner\": \"morpheus\"
        }" || echo "Registry notification failed"
    fi
  
  # Trigger nimsforest bootstrap
  - |
    if [ "{{.CallbackURL}}" != "" ]; then
      INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
      INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
      
      curl -X POST {{.CallbackURL}}/api/v1/bootstrap \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"node_ip\": \"$INSTANCE_IP\",
          \"role\": \"{{.NodeRole}}\"
        }" || echo "NimsForest callback failed"
    fi

final_message: "Morpheus infrastructure provisioning complete. Ready for NimsForest bootstrap."
`

// StorageNodeTemplate is the cloud-init script for storage nodes
// Morpheus responsibility: Infrastructure setup (NFS, firewall)
// NimsForest responsibility: Storage orchestration and management
const StorageNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - nfs-kernel-server
  - ufw
  - jq

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "role": "{{.NodeRole}}",
        "provisioner": "morpheus",
        "provisioned_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "registry_url": "{{.RegistryURL}}",
        "callback_url": "{{.CallbackURL}}"
      }
    permissions: '0644'

  - path: /etc/exports
    content: |
      # NFS exports - managed by NimsForest
      /mnt/nimsforest-storage *(rw,sync,no_subtree_check,no_root_squash)

runcmd:
  # Setup base storage directory
  - mkdir -p /mnt/nimsforest-storage
  - chmod 755 /mnt/nimsforest-storage
  - chown ubuntu:ubuntu /mnt/nimsforest-storage
  
  # Configure NFS
  - systemctl enable nfs-kernel-server
  - systemctl start nfs-kernel-server
  - exportfs -ra
  
  # Configure firewall
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 2049/tcp comment 'NFS'
  - ufw allow 111/tcp comment 'RPC'
  - ufw allow 111/udp comment 'RPC'
  - ufw allow 4222/tcp comment 'NATS client'
  - ufw --force enable
  
  # Signal readiness
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
    INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone || echo "unknown")
    
    if [ "{{.RegistryURL}}" != "" ]; then
      curl -X POST {{.RegistryURL}}/api/v1/nodes \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"role\": \"{{.NodeRole}}\",
          \"ip\": \"$INSTANCE_IP\",
          \"location\": \"$LOCATION\",
          \"status\": \"infrastructure_ready\",
          \"provisioner\": \"morpheus\",
          \"storage_path\": \"/mnt/nimsforest-storage\"
        }" || echo "Registry notification failed"
    fi
  
  # Trigger nimsforest bootstrap
  - |
    if [ "{{.CallbackURL}}" != "" ]; then
      INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || echo "unknown")
      INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id || hostname)
      
      curl -X POST {{.CallbackURL}}/api/v1/bootstrap \
        -H "Content-Type: application/json" \
        -d "{
          \"forest_id\": \"{{.ForestID}}\",
          \"node_id\": \"$INSTANCE_ID\",
          \"node_ip\": \"$INSTANCE_IP\",
          \"role\": \"{{.NodeRole}}\"
        }" || echo "NimsForest callback failed"
    fi

final_message: "Morpheus infrastructure provisioning complete. Ready for NimsForest bootstrap."
`

// Generate creates a cloud-init script for the given role and data
func Generate(role NodeRole, data TemplateData) (string, error) {
	var tmplStr string

	switch role {
	case RoleEdge:
		tmplStr = EdgeNodeTemplate
	case RoleCompute:
		tmplStr = ComputeNodeTemplate
	case RoleStorage:
		tmplStr = StorageNodeTemplate
	default:
		return "", fmt.Errorf("unknown node role: %s", role)
	}

	// Add template functions
	funcMap := template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("cloudinit").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
