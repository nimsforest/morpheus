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

	// NimsForest auto-installation (with embedded NATS)
	NimsForestInstall     bool   // Auto-install NimsForest
	NimsForestDownloadURL string // URL to download binary (e.g., "https://nimsforest.io/bin/nimsforest")

	// Node identification (for embedded NATS peer discovery)
	NodeID string // Unique node ID (e.g., "myforest-node-1")
	NodeIP string // Node's public IP (empty = use metadata service)

	// StorageBox mount for shared registry (enables NATS peer discovery)
	StorageBoxHost     string // CIFS host: uXXXXX.your-storagebox.de
	StorageBoxUser     string // StorageBox username: uXXXXX
	StorageBoxPassword string // StorageBox password
}

// EdgeNodeTemplate is the cloud-init script for edge nodes
// Morpheus responsibility: Infrastructure setup only (OS, network, firewall)
// NimsForest responsibility: Application logic (with embedded NATS)
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
  - cifs-utils
  - python3

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "node_id": "{{.NodeID}}",
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
  
  {{if .StorageBoxHost}}
  # Mount StorageBox for shared registry (enables NATS peer discovery)
  - |
    echo "📁 Mounting StorageBox for shared registry..."
    mkdir -p /mnt/forest
    
    # Add to fstab for persistence across reboots
    echo "//{{.StorageBoxHost}}/backup /mnt/forest cifs user={{.StorageBoxUser}},pass={{.StorageBoxPassword}},uid=root,gid=root,_netdev,nofail 0 0" >> /etc/fstab
    
    # Mount now
    if mount /mnt/forest; then
      echo "✅ StorageBox mounted at /mnt/forest"
    else
      echo "⚠️  Mount failed - NimsForest will retry on startup"
    fi
  
  # Register this node in shared registry (for NATS peer discovery)
  - |
    echo "📝 Registering node in shared registry..."
    REGISTRY=/mnt/forest/registry.json
    
    # Get node IP from metadata or hostname
    NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
    
    # Wait for mount to be available (max 30 seconds)
    for i in $(seq 1 15); do
      [ -d /mnt/forest ] && mountpoint -q /mnt/forest && break
      sleep 2
    done
    
    # Create registry if missing
    [ -f "$REGISTRY" ] || echo '{"nodes":{}}' > "$REGISTRY"
    
    # Use flock for atomic update
    flock "$REGISTRY" python3 << PYEOF
import json
import sys

forest_id = "{{.ForestID}}"
node_id = "{{.NodeID}}"
node_ip = "${NODE_IP}"

try:
    with open('/mnt/forest/registry.json', 'r+') as f:
        reg = json.load(f)
        if 'nodes' not in reg:
            reg['nodes'] = {}
        if forest_id not in reg['nodes']:
            reg['nodes'][forest_id] = []
        
        # Add node if not exists
        if not any(n.get('id') == node_id for n in reg['nodes'][forest_id]):
            reg['nodes'][forest_id].append({
                "id": node_id,
                "ip": node_ip,
                "forest_id": forest_id,
                "status": "provisioning"
            })
            print(f"✅ Node {node_id} registered in shared registry (IP: {node_ip})")
        else:
            print(f"ℹ️  Node {node_id} already in registry")
        
        f.seek(0)
        f.truncate()
        json.dump(reg, f, indent=2)
except Exception as e:
    print(f"⚠️  Registry update failed: {e}", file=sys.stderr)
PYEOF
  {{end}}
  
  {{if .NimsForestInstall}}
  # Download and install NimsForest (with embedded NATS)
  - |
    echo "📦 Installing NimsForest..."
    DOWNLOAD_URL="{{.NimsForestDownloadURL}}"
    
    echo "📥 Downloading from ${DOWNLOAD_URL}..."
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "$DOWNLOAD_URL"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      echo "✅ NimsForest installed"
      
      # Get node IP for embedded NATS cluster
      NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
      
      cat > /etc/systemd/system/nimsforest.service << SERVICEEOF
      [Unit]
      Description=NimsForest (with embedded NATS)
      After=network-online.target{{if .StorageBoxHost}} mnt-forest.mount{{end}}
      Wants=network-online.target
      
      [Service]
      Type=simple
      User=root
      ExecStart=/opt/nimsforest/bin/nimsforest start --forest-id {{.ForestID}}
      Restart=always
      RestartSec=5
      Environment=FOREST_ID={{.ForestID}}
      Environment=NODE_ID={{.NodeID}}
      Environment=NODE_ROLE={{.NodeRole}}
      Environment=NODE_IP=${NODE_IP}
      {{if .StorageBoxHost}}Environment=REGISTRY_PATH=/mnt/forest/registry.json{{end}}
      WorkingDirectory=/var/lib/nimsforest
      
      [Install]
      WantedBy=multi-user.target
      SERVICEEOF
      
      sed -i 's/^      //' /etc/systemd/system/nimsforest.service
      systemctl daemon-reload
      systemctl enable nimsforest
      systemctl start nimsforest
      echo "✅ NimsForest service started (with embedded NATS)"
    else
      echo "⚠️  Failed to download NimsForest from ${DOWNLOAD_URL}"
    fi
  {{end}}
  
  # Signal readiness to registry
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
// NimsForest responsibility: Worker/compute service installation (with embedded NATS)
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
  - cifs-utils
  - python3

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "node_id": "{{.NodeID}}",
        "role": "{{.NodeRole}}",
        "provisioner": "morpheus",
        "provisioned_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "registry_url": "{{.RegistryURL}}",
        "callback_url": "{{.CallbackURL}}"
      }
    permissions: '0644'

runcmd:
  # Configure firewall - NATS ports for nimsforest (embedded NATS)
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 4222/tcp comment 'NATS client port'
  - ufw allow 6222/tcp comment 'NATS cluster port'
  - ufw allow 8222/tcp comment 'NATS monitoring port'
  - ufw --force enable
  
  # Create directories for nimsforest (binaries, data, logs)
  - mkdir -p /opt/nimsforest/bin /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  - chown -R ubuntu:ubuntu /opt/nimsforest /var/lib/nimsforest /var/log/nimsforest /etc/nimsforest
  
  # Prepare for direct binary deployment
  - systemctl daemon-reload
  
  {{if .StorageBoxHost}}
  # Mount StorageBox for shared registry (enables NATS peer discovery)
  - |
    echo "📁 Mounting StorageBox for shared registry..."
    mkdir -p /mnt/forest
    echo "//{{.StorageBoxHost}}/backup /mnt/forest cifs user={{.StorageBoxUser}},pass={{.StorageBoxPassword}},uid=root,gid=root,_netdev,nofail 0 0" >> /etc/fstab
    if mount /mnt/forest; then
      echo "✅ StorageBox mounted at /mnt/forest"
    else
      echo "⚠️  Mount failed - NimsForest will retry on startup"
    fi
  
  # Register this node in shared registry
  - |
    echo "📝 Registering node in shared registry..."
    REGISTRY=/mnt/forest/registry.json
    NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
    
    for i in $(seq 1 15); do
      [ -d /mnt/forest ] && mountpoint -q /mnt/forest && break
      sleep 2
    done
    
    [ -f "$REGISTRY" ] || echo '{"nodes":{}}' > "$REGISTRY"
    
    flock "$REGISTRY" python3 << PYEOF
import json
forest_id = "{{.ForestID}}"
node_id = "{{.NodeID}}"
node_ip = "${NODE_IP}"
try:
    with open('/mnt/forest/registry.json', 'r+') as f:
        reg = json.load(f)
        if 'nodes' not in reg:
            reg['nodes'] = {}
        if forest_id not in reg['nodes']:
            reg['nodes'][forest_id] = []
        if not any(n.get('id') == node_id for n in reg['nodes'][forest_id]):
            reg['nodes'][forest_id].append({"id": node_id, "ip": node_ip, "forest_id": forest_id, "status": "provisioning"})
            print(f"✅ Node {node_id} registered")
        f.seek(0)
        f.truncate()
        json.dump(reg, f, indent=2)
except Exception as e:
    print(f"⚠️  Registry update failed: {e}")
PYEOF
  {{end}}
  
  {{if .NimsForestInstall}}
  # Download and install NimsForest (with embedded NATS)
  - |
    echo "📦 Installing NimsForest..."
    DOWNLOAD_URL="{{.NimsForestDownloadURL}}"
    
    echo "📥 Downloading from ${DOWNLOAD_URL}..."
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "$DOWNLOAD_URL"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      echo "✅ NimsForest installed"
      
      NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
      
      cat > /etc/systemd/system/nimsforest.service << SERVICEEOF
      [Unit]
      Description=NimsForest (with embedded NATS)
      After=network-online.target{{if .StorageBoxHost}} mnt-forest.mount{{end}}
      Wants=network-online.target
      
      [Service]
      Type=simple
      User=root
      ExecStart=/opt/nimsforest/bin/nimsforest start --forest-id {{.ForestID}}
      Restart=always
      RestartSec=5
      Environment=FOREST_ID={{.ForestID}}
      Environment=NODE_ID={{.NodeID}}
      Environment=NODE_ROLE={{.NodeRole}}
      Environment=NODE_IP=${NODE_IP}
      {{if .StorageBoxHost}}Environment=REGISTRY_PATH=/mnt/forest/registry.json{{end}}
      WorkingDirectory=/var/lib/nimsforest
      
      [Install]
      WantedBy=multi-user.target
      SERVICEEOF

      sed -i 's/^      //' /etc/systemd/system/nimsforest.service
      systemctl daemon-reload
      systemctl enable nimsforest
      systemctl start nimsforest
      echo "✅ NimsForest service started (with embedded NATS)"
    else
      echo "⚠️  Failed to download NimsForest from ${DOWNLOAD_URL}"
    fi
  {{end}}
  
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
          \"status\": \"{{if .NimsForestInstall}}active{{else}}infrastructure_ready{{end}}\",
          \"provisioner\": \"morpheus\"
        }" || echo "Registry notification failed"
    fi
  
  # Trigger nimsforest bootstrap (only if not auto-installed)
  {{if not .NimsForestInstall}}
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
  {{end}}

final_message: "Morpheus provisioning complete.{{if .NimsForestInstall}} NimsForest installed and running.{{else}} Ready for NimsForest bootstrap.{{end}}"
`

// StorageNodeTemplate is the cloud-init script for storage nodes
// Morpheus responsibility: Infrastructure setup (NFS, firewall)
// NimsForest responsibility: Storage orchestration and management (with embedded NATS)
const StorageNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - nfs-kernel-server
  - ufw
  - jq
  - cifs-utils
  - python3

write_files:
  - path: /etc/morpheus/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "node_id": "{{.NodeID}}",
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
  
  # Configure firewall (includes NATS ports for embedded NATS)
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 2049/tcp comment 'NFS'
  - ufw allow 111/tcp comment 'RPC'
  - ufw allow 111/udp comment 'RPC'
  - ufw allow 4222/tcp comment 'NATS client port'
  - ufw allow 6222/tcp comment 'NATS cluster port'
  - ufw allow 8222/tcp comment 'NATS monitoring port'
  - ufw --force enable
  
  {{if .StorageBoxHost}}
  # Mount StorageBox for shared registry (enables NATS peer discovery)
  - |
    echo "📁 Mounting StorageBox for shared registry..."
    mkdir -p /mnt/forest
    echo "//{{.StorageBoxHost}}/backup /mnt/forest cifs user={{.StorageBoxUser}},pass={{.StorageBoxPassword}},uid=root,gid=root,_netdev,nofail 0 0" >> /etc/fstab
    if mount /mnt/forest; then
      echo "✅ StorageBox mounted at /mnt/forest"
    else
      echo "⚠️  Mount failed - NimsForest will retry on startup"
    fi
  
  # Register this node in shared registry
  - |
    echo "📝 Registering node in shared registry..."
    REGISTRY=/mnt/forest/registry.json
    NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
    
    for i in $(seq 1 15); do
      [ -d /mnt/forest ] && mountpoint -q /mnt/forest && break
      sleep 2
    done
    
    [ -f "$REGISTRY" ] || echo '{"nodes":{}}' > "$REGISTRY"
    
    flock "$REGISTRY" python3 << PYEOF
import json
forest_id = "{{.ForestID}}"
node_id = "{{.NodeID}}"
node_ip = "${NODE_IP}"
try:
    with open('/mnt/forest/registry.json', 'r+') as f:
        reg = json.load(f)
        if 'nodes' not in reg:
            reg['nodes'] = {}
        if forest_id not in reg['nodes']:
            reg['nodes'][forest_id] = []
        if not any(n.get('id') == node_id for n in reg['nodes'][forest_id]):
            reg['nodes'][forest_id].append({"id": node_id, "ip": node_ip, "forest_id": forest_id, "status": "provisioning", "storage_path": "/mnt/nimsforest-storage"})
            print(f"✅ Node {node_id} registered")
        f.seek(0)
        f.truncate()
        json.dump(reg, f, indent=2)
except Exception as e:
    print(f"⚠️  Registry update failed: {e}")
PYEOF
  {{end}}
  
  {{if .NimsForestInstall}}
  # Download and install NimsForest (with embedded NATS)
  - |
    echo "📦 Installing NimsForest..."
    DOWNLOAD_URL="{{.NimsForestDownloadURL}}"
    
    echo "📥 Downloading from ${DOWNLOAD_URL}..."
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "$DOWNLOAD_URL"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      echo "✅ NimsForest installed"
      
      NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
      
      cat > /etc/systemd/system/nimsforest.service << SERVICEEOF
      [Unit]
      Description=NimsForest (with embedded NATS)
      After=network-online.target{{if .StorageBoxHost}} mnt-forest.mount{{end}}
      Wants=network-online.target
      
      [Service]
      Type=simple
      User=root
      ExecStart=/opt/nimsforest/bin/nimsforest start --forest-id {{.ForestID}}
      Restart=always
      RestartSec=5
      Environment=FOREST_ID={{.ForestID}}
      Environment=NODE_ID={{.NodeID}}
      Environment=NODE_ROLE={{.NodeRole}}
      Environment=NODE_IP=${NODE_IP}
      {{if .StorageBoxHost}}Environment=REGISTRY_PATH=/mnt/forest/registry.json{{end}}
      WorkingDirectory=/var/lib/nimsforest
      
      [Install]
      WantedBy=multi-user.target
      SERVICEEOF

      sed -i 's/^      //' /etc/systemd/system/nimsforest.service
      systemctl daemon-reload
      systemctl enable nimsforest
      systemctl start nimsforest
      echo "✅ NimsForest service started (with embedded NATS)"
    else
      echo "⚠️  Failed to download NimsForest from ${DOWNLOAD_URL}"
    fi
  {{end}}
  
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
          \"status\": \"{{if .NimsForestInstall}}active{{else}}infrastructure_ready{{end}}\",
          \"provisioner\": \"morpheus\",
          \"storage_path\": \"/mnt/nimsforest-storage\"
        }" || echo "Registry notification failed"
    fi
  
  {{if not .NimsForestInstall}}
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
  {{end}}

final_message: "Morpheus provisioning complete.{{if .NimsForestInstall}} NimsForest installed and running.{{else}} Ready for NimsForest bootstrap.{{end}}"
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
