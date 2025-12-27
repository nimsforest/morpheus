package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerateEdgeNode(t *testing.T) {
	data := TemplateData{
		NodeRole:     RoleEdge,
		ForestID:     "test-forest",
		NATSServers:  []string{"10.0.0.1", "10.0.0.2"},
		RegistryURL:  "http://registry.example.com",
	}

	script, err := Generate(RoleEdge, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Check for cloud-config header
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}

	// Check for NATS installation
	if !strings.Contains(script, "nats-server") {
		t.Error("Script should contain NATS server installation")
	}

	// Check for forest ID
	if !strings.Contains(script, "test-forest") {
		t.Error("Script should contain forest ID")
	}

	// Check for NATS cluster configuration
	if !strings.Contains(script, "10.0.0.1") {
		t.Error("Script should contain NATS server addresses")
	}

	// Check for firewall configuration
	if !strings.Contains(script, "ufw") {
		t.Error("Script should contain UFW firewall configuration")
	}

	// Check for registry URL
	if !strings.Contains(script, "http://registry.example.com") {
		t.Error("Script should contain registry URL")
	}
}

func TestGenerateComputeNode(t *testing.T) {
	data := TemplateData{
		NodeRole:    RoleCompute,
		ForestID:    "test-forest",
		RegistryURL: "http://registry.example.com",
	}

	script, err := Generate(RoleCompute, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Check for cloud-config header
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}

	// Check for Docker installation
	if !strings.Contains(script, "docker") {
		t.Error("Script should contain Docker installation")
	}

	// Check for forest ID
	if !strings.Contains(script, "test-forest") {
		t.Error("Script should contain forest ID")
	}
}

func TestGenerateStorageNode(t *testing.T) {
	data := TemplateData{
		NodeRole:    RoleStorage,
		ForestID:    "test-forest",
		RegistryURL: "http://registry.example.com",
	}

	script, err := Generate(RoleStorage, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Check for cloud-config header
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}

	// Check for NFS installation
	if !strings.Contains(script, "nfs") {
		t.Error("Script should contain NFS server installation")
	}

	// Check for forest ID
	if !strings.Contains(script, "test-forest") {
		t.Error("Script should contain forest ID")
	}
}

func TestGenerateInvalidRole(t *testing.T) {
	data := TemplateData{
		NodeRole:    NodeRole("invalid"),
		ForestID:    "test-forest",
		RegistryURL: "http://registry.example.com",
	}

	_, err := Generate(data.NodeRole, data)
	if err == nil {
		t.Error("Expected error for invalid node role")
	}
}

func TestGenerateWithoutNATSServers(t *testing.T) {
	data := TemplateData{
		NodeRole:     RoleEdge,
		ForestID:     "test-forest",
		NATSServers:  []string{}, // Empty servers list
		RegistryURL:  "http://registry.example.com",
	}

	script, err := Generate(RoleEdge, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Should still generate valid script
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}
}

func TestNodeRoleConstants(t *testing.T) {
	if RoleEdge != "edge" {
		t.Errorf("Expected RoleEdge to be 'edge', got '%s'", RoleEdge)
	}
	if RoleCompute != "compute" {
		t.Errorf("Expected RoleCompute to be 'compute', got '%s'", RoleCompute)
	}
	if RoleStorage != "storage" {
		t.Errorf("Expected RoleStorage to be 'storage', got '%s'", RoleStorage)
	}
}
