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

	// RustDesk server installation
	RustDeskInstall bool   // Auto-install RustDesk server (hbbs + hbbr)
	RustDeskVersion string // RustDesk server version (e.g., "1.1.11")
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
  # Configure firewall - NATS ports for embedded NATS + NimsForest webview
  - ufw allow 22/tcp comment 'SSH'
  - ufw allow 4222/tcp comment 'NATS client'
  - ufw allow 6222/tcp comment 'NATS cluster'
  - ufw allow 8222/tcp comment 'NATS monitoring'
  - ufw allow 8080/tcp comment 'NimsForest webview'
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
  # Download and install NimsForest (binary with embedded NATS)
  - |
    echo "📦 Installing NimsForest from {{.NimsForestDownloadURL}}..."
    if curl -fsSL -o /opt/nimsforest/bin/nimsforest "{{.NimsForestDownloadURL}}"; then
      chmod +x /opt/nimsforest/bin/nimsforest
      ln -sf /opt/nimsforest/bin/nimsforest /usr/local/bin/forest
      /opt/nimsforest/bin/nimsforest version || echo "NimsForest binary ready"
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
      
      # Create systemd service for NimsForest (standalone mode with embedded NATS)
      cat > /etc/systemd/system/nimsforest.service <<'SERVICEEOF'
    [Unit]
    Description=NimsForest - Event-Driven Organizational Orchestration
    After=network-online.target{{if .StorageBoxHost}} mnt-forest.mount{{end}}
    Wants=network-online.target

    [Service]
    Type=simple
    ExecStart=/opt/nimsforest/bin/nimsforest standalone
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
      
      # Create systemd service for NimsForest webview
      cat > /etc/systemd/system/nimsforest-webview.service <<'WEBVIEWEOF'
    [Unit]
    Description=NimsForest Webview - Isometric Visualization
    After=nimsforest.service
    Requires=nimsforest.service

    [Service]
    Type=simple
    ExecStart=/opt/nimsforest/bin/nimsforest viewmodel webview --port=8080
    Restart=always
    RestartSec=5
    WorkingDirectory=/var/lib/nimsforest

    [Install]
    WantedBy=multi-user.target
    WEBVIEWEOF
      
      sed -i 's/^    //' /etc/systemd/system/nimsforest.service
      sed -i 's/^    //' /etc/systemd/system/nimsforest-webview.service
      systemctl daemon-reload
      systemctl enable nimsforest nimsforest-webview
      systemctl start nimsforest
      sleep 2
      systemctl start nimsforest-webview
      echo "✅ NimsForest started with embedded NATS"
      echo "✅ NimsForest webview available at http://$(hostname -I | awk '{print $1}'):8080/"
    else
      echo "❌ Failed to download NimsForest"
    fi
  {{end}}

{{if .RustDeskInstall}}
  # Download and install RustDesk Server (hbbs + hbbr)
  - |
    echo "📦 Installing RustDesk Server..."
    RUSTDESK_VERSION="{{if .RustDeskVersion}}{{.RustDeskVersion}}{{else}}1.1.11{{end}}"
    RUSTDESK_DIR="/opt/rustdesk"
    mkdir -p $RUSTDESK_DIR

    # Download rustdesk-server binaries
    cd /tmp
    wget -q "https://github.com/rustdesk/rustdesk-server/releases/download/${RUSTDESK_VERSION}/rustdesk-server-linux-amd64.zip" -O rustdesk-server.zip

    if [ -f rustdesk-server.zip ]; then
      apt-get install -y unzip
      unzip -o rustdesk-server.zip -d $RUSTDESK_DIR
      chmod +x $RUSTDESK_DIR/hbbs $RUSTDESK_DIR/hbbr
      rm rustdesk-server.zip
      echo "✅ RustDesk Server binaries installed"
    else
      echo "❌ Failed to download RustDesk Server"
      exit 1
    fi

    # Configure firewall for RustDesk
    ufw allow 21115/tcp comment 'RustDesk NAT test'
    ufw allow 21116/tcp comment 'RustDesk ID server TCP'
    ufw allow 21116/udp comment 'RustDesk ID server UDP'
    ufw allow 21117/tcp comment 'RustDesk relay'
    ufw allow 21118/tcp comment 'RustDesk websocket ID'
    ufw allow 21119/tcp comment 'RustDesk websocket relay'

    # Create systemd service for hbbs (ID/Rendezvous server)
    cat > /etc/systemd/system/rustdesk-hbbs.service <<'HBBSEOF'
[Unit]
Description=RustDesk ID/Rendezvous Server
After=network.target

[Service]
Type=simple
ExecStart=/opt/rustdesk/hbbs
WorkingDirectory=/opt/rustdesk
Restart=always
RestartSec=5
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
HBBSEOF

    # Create systemd service for hbbr (Relay server)
    cat > /etc/systemd/system/rustdesk-hbbr.service <<'HBBREOF'
[Unit]
Description=RustDesk Relay Server
After=network.target rustdesk-hbbs.service
Wants=rustdesk-hbbs.service

[Service]
Type=simple
ExecStart=/opt/rustdesk/hbbr
WorkingDirectory=/opt/rustdesk
Restart=always
RestartSec=5
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
HBBREOF

    # Enable and start RustDesk services
    systemctl daemon-reload
    systemctl enable rustdesk-hbbs rustdesk-hbbr
    systemctl start rustdesk-hbbs
    sleep 2
    systemctl start rustdesk-hbbr

    # Display public key for client configuration
    sleep 3
    if [ -f /opt/rustdesk/id_ed25519.pub ]; then
      echo "✅ RustDesk Server started"
      echo "🔑 Public Key (for client configuration):"
      cat /opt/rustdesk/id_ed25519.pub
    else
      echo "✅ RustDesk Server started (key will be generated on first connection)"
    fi

    # Get server IP for display
    SERVER_IP=$(curl -s http://169.254.169.254/hetzner/v1/metadata/public-ipv4 2>/dev/null | grep -v "not found" | head -1)
    if [ -z "$SERVER_IP" ]; then
      SERVER_IP=$(hostname -I | awk '{print $1}')
    fi
    echo "🌐 RustDesk Server IP: $SERVER_IP"
    echo "📝 Client config: ID Server = $SERVER_IP, Relay Server = $SERVER_IP"
{{end}}

final_message: "Node ready.{{if .NimsForestInstall}} NimsForest running.{{end}}{{if .RustDeskInstall}} RustDesk Server running.{{end}}"
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
