package cloudinit

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateData contains data for cloud-init template rendering
type TemplateData struct {
	ForestID    string
	RegistryURL string // Optional: Morpheus registry for infrastructure state
	CallbackURL string // Optional: NimsForest callback URL for bootstrap trigger
	SSHKeys     []string

	// NimsForest auto-installation (with embedded NATS)
	NimsForestInstall     bool   // Auto-install NimsForest
	NimsForestDownloadURL string // URL to download binary

	// Node identification (for embedded NATS peer discovery)
	NodeID    string // Unique node ID (e.g., "myforest-node-1")
	NodeIndex int    // Node index (0-based) in the forest
	NodeCount int    // Total number of nodes in the forest (1=standalone, 3+=cluster)

	// StorageBox mount for shared registry (enables NATS peer discovery)
	StorageBoxHost     string // CIFS host: uXXXXX.your-storagebox.de
	StorageBoxUser     string // StorageBox username: uXXXXX
	StorageBoxPassword string // StorageBox password
}

// NodeTemplate is the cloud-init script for all forest nodes
// All nodes run NimsForest with embedded NATS
const NodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - ufw
  - jq
  - cifs-utils

write_files:
  - path: /etc/nimsforest/node-info.json
    content: |
      {
        "forest_id": "{{.ForestID}}",
        "node_id": "{{.NodeID}}",
        "node_index": {{.NodeIndex}},
        "cluster_size": {{.NodeCount}},
        "provisioner": "morpheus"
      }
    permissions: '0644'

runcmd:
  # Configure firewall - NATS ports for embedded NATS
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 4222/tcp comment 'NATS client'
  - ufw allow 6222/tcp comment 'NATS cluster'
  - ufw allow 8222/tcp comment 'NATS monitoring'
  - ufw --force enable
  
  # Create directories for nimsforest
  - mkdir -p /opt/nimsforest/bin /var/lib/nimsforest /var/log/nimsforest
  
  {{if .StorageBoxHost}}
  # Mount StorageBox for shared registry
  - |
    echo "📁 Mounting StorageBox..."
    mkdir -p /mnt/forest
    echo "//{{.StorageBoxHost}}/backup /mnt/forest cifs user={{.StorageBoxUser}},pass={{.StorageBoxPassword}},uid=root,gid=root,_netdev,nofail 0 0" >> /etc/fstab
    mount /mnt/forest || echo "⚠️  Mount failed - will retry"
  
  # Register node in shared registry
  - |
    REGISTRY=/mnt/forest/registry.json
    FOREST_ID="{{.ForestID}}"
    NODE_ID="{{.NodeID}}"
    NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null || hostname -I | awk '{print $1}')
    
    # Wait for mount
    for i in $(seq 1 15); do
      mountpoint -q /mnt/forest && break
      sleep 2
    done
    
    [ -f "$REGISTRY" ] || echo '{"nodes":{}}' > "$REGISTRY"
    
    flock "$REGISTRY" sh -c '
      TEMP=$(mktemp)
      if jq -e ".nodes[\"'"$FOREST_ID"'\"] | map(select(.id == \"'"$NODE_ID"'\")) | length > 0" "$REGISTRY" >/dev/null 2>&1; then
        echo "ℹ️  Node '"$NODE_ID"' already registered"
      else
        jq ".nodes[\"'"$FOREST_ID"'\"] //= [] | .nodes[\"'"$FOREST_ID"'\"] += [{\"id\": \"'"$NODE_ID"'\", \"ip\": \"'"$NODE_IP"'\", \"forest_id\": \"'"$FOREST_ID"'\"}]" "$REGISTRY" > "$TEMP" && mv "$TEMP" "$REGISTRY"
        echo "✅ Node '"$NODE_ID"' registered (IP: '"$NODE_IP"')"
      fi
    '
  {{end}}
  
  {{if .NimsForestInstall}}
  # Download and install NimsForest
  - |
    echo "📦 Installing NimsForest..."
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "{{.NimsForestDownloadURL}}"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      ln -sf /opt/nimsforest/bin/nimsforest /usr/local/bin/forest
      echo "✅ NimsForest installed"
      
      # Get node IP - try IPv4 first (more reliable), then IPv6, then hostname
      NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv4 2>/dev/null | grep -v "not found" | head -1)
      if [ -z "$NODE_IP" ]; then
        NODE_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv6 2>/dev/null | grep -v "not found" | head -1)
      fi
      if [ -z "$NODE_IP" ]; then
        NODE_IP=$(hostname -I | awk '{print $1}')
      fi
      echo "Node IP: $NODE_IP"
      echo "Cluster mode: {{if eq .NodeCount 1}}standalone{{else}}cluster ({{.NodeCount}} nodes){{end}}"
      
      # Create local registry if no shared storage configured
      {{if not .StorageBoxHost}}
      mkdir -p /mnt/forest
      echo "{\"nodes\":{\"{{.ForestID}}\":[{\"id\":\"{{.NodeID}}\",\"ip\":\"$NODE_IP\",\"forest_id\":\"{{.ForestID}}\"}]}}" > /mnt/forest/registry.json
      {{end}}
      
      # Create systemd service for cluster mode
      cat > /etc/systemd/system/nimsforest.service <<'SERVICEEOF'
    [Unit]
    Description=NimsForest
    After=network-online.target{{if .StorageBoxHost}} mnt-forest.mount{{end}}
    Wants=network-online.target

    [Service]
    Type=simple
    ExecStart=/opt/nimsforest/bin/nimsforest start
    Restart=always
    RestartSec=5
    Environment=NATS_CLUSTER_NODE_INFO=/etc/nimsforest/node-info.json
    Environment=NATS_CLUSTER_REGISTRY=/mnt/forest/registry.json
    Environment=JETSTREAM_DIR=/var/lib/nimsforest/jetstream
    Environment=NATS_CLUSTER_SIZE={{.NodeCount}}
    WorkingDirectory=/var/lib/nimsforest

    [Install]
    WantedBy=multi-user.target
    SERVICEEOF
      
      sed -i 's/^    //' /etc/systemd/system/nimsforest.service
      systemctl daemon-reload
      systemctl enable nimsforest
      systemctl start nimsforest
      echo "✅ NimsForest started"
    else
      echo "❌ Failed to download NimsForest"
    fi
  {{end}}

final_message: "Node ready.{{if .NimsForestInstall}} NimsForest running.{{end}}"
`

// Generate creates a cloud-init script for a forest node
func Generate(data TemplateData) (string, error) {
	tmpl, err := template.New("cloudinit").Parse(NodeTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
