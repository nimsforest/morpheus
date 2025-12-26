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
	NodeRole     NodeRole
	ForestID     string
	NATSServers  []string
	RegistryURL  string
	SSHKeys      []string
}

// EdgeNodeTemplate is the cloud-init script for edge nodes
const EdgeNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - docker.io
  - ufw

write_files:
  - path: /etc/systemd/system/nats-server.service
    content: |
      [Unit]
      Description=NATS Server
      After=network.target

      [Service]
      Type=simple
      ExecStart=/usr/local/bin/nats-server -c /etc/nats/nats-server.conf
      Restart=on-failure
      User=nats
      Group=nats

      [Install]
      WantedBy=multi-user.target

  - path: /etc/nats/nats-server.conf
    content: |
      port: 4222
      http_port: 8222
      
      server_name: {{.ForestID}}-edge
      
      jetstream {
        store_dir: /var/lib/nats
        max_memory_store: 1GB
        max_file_store: 10GB
      }

      cluster {
        name: {{.ForestID}}
        port: 6222
        {{- if .NATSServers}}
        routes = [
          {{- range $i, $server := .NATSServers}}
          nats://{{$server}}:6222{{if lt $i (sub (len $.NATSServers) 1)}},{{end}}
          {{- end}}
        ]
        {{- end}}
      }

      accounts {
        $SYS {
          users = [
            { user: "admin", password: "$NATS_ADMIN_PASSWORD" }
          ]
        }
      }

runcmd:
  # Install NATS server
  - curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.tar.gz -o /tmp/nats-server.tar.gz
  - tar -xzf /tmp/nats-server.tar.gz -C /tmp
  - mv /tmp/nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
  - chmod +x /usr/local/bin/nats-server
  - rm -rf /tmp/nats-server*
  
  # Create NATS user and directories
  - useradd -r -s /bin/false nats
  - mkdir -p /var/lib/nats /etc/nats
  - chown -R nats:nats /var/lib/nats /etc/nats
  
  # Configure firewall
  - ufw allow 22/tcp
  - ufw allow 4222/tcp
  - ufw allow 6222/tcp
  - ufw allow 8222/tcp
  - ufw --force enable
  
  # Start NATS server
  - systemctl daemon-reload
  - systemctl enable nats-server
  - systemctl start nats-server
  
  # Register with forest registry
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
    curl -X POST {{.RegistryURL}}/nodes \
      -H "Content-Type: application/json" \
      -d "{\"forest_id\":\"{{.ForestID}}\",\"role\":\"edge\",\"ip\":\"$INSTANCE_IP\",\"location\":\"$LOCATION\",\"status\":\"active\"}"

final_message: "Edge node {{.ForestID}} is ready!"
`

// ComputeNodeTemplate is the cloud-init script for compute nodes
const ComputeNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - docker.io
  - ufw

write_files:
  - path: /etc/systemd/system/nats-client.service
    content: |
      [Unit]
      Description=NATS Client Worker
      After=network.target

      [Service]
      Type=simple
      ExecStart=/usr/local/bin/worker --forest={{.ForestID}} --role=compute
      Restart=on-failure
      User=worker
      Group=worker

      [Install]
      WantedBy=multi-user.target

runcmd:
  # Configure firewall
  - ufw allow 22/tcp
  - ufw --force enable
  
  # Setup Docker
  - systemctl enable docker
  - systemctl start docker
  - usermod -aG docker ubuntu
  
  # Create worker user
  - useradd -r -s /bin/false worker
  
  # Register with forest registry
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
    curl -X POST {{.RegistryURL}}/nodes \
      -H "Content-Type: application/json" \
      -d "{\"forest_id\":\"{{.ForestID}}\",\"role\":\"compute\",\"ip\":\"$INSTANCE_IP\",\"location\":\"$LOCATION\",\"status\":\"active\"}"

final_message: "Compute node {{.ForestID}} is ready!"
`

// StorageNodeTemplate is the cloud-init script for storage nodes
const StorageNodeTemplate = `#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - nfs-kernel-server
  - ufw

write_files:
  - path: /etc/exports
    content: |
      /mnt/forest-storage *(rw,sync,no_subtree_check,no_root_squash)

runcmd:
  # Setup storage directory
  - mkdir -p /mnt/forest-storage
  - chmod 777 /mnt/forest-storage
  
  # Configure NFS
  - systemctl enable nfs-kernel-server
  - systemctl start nfs-kernel-server
  - exportfs -ra
  
  # Configure firewall
  - ufw allow 22/tcp
  - ufw allow 2049/tcp
  - ufw --force enable
  
  # Register with forest registry
  - |
    INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
    LOCATION=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
    curl -X POST {{.RegistryURL}}/nodes \
      -H "Content-Type: application/json" \
      -d "{\"forest_id\":\"{{.ForestID}}\",\"role\":\"storage\",\"ip\":\"$INSTANCE_IP\",\"location\":\"$LOCATION\",\"capacity\":\"100GB\",\"status\":\"active\"}"

final_message: "Storage node {{.ForestID}} is ready!"
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
