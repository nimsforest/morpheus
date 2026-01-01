package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerateEdgeNode(t *testing.T) {
	data := TemplateData{
		NodeRole:    RoleEdge,
		ForestID:    "test-forest",
		RegistryURL: "http://registry.example.com",
		CallbackURL: "http://nimsforest.example.com",
	}

	script, err := Generate(RoleEdge, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Check for cloud-config header
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}

	// Check for forest ID
	if !strings.Contains(script, "test-forest") {
		t.Error("Script should contain forest ID")
	}

	// Check for firewall configuration (Morpheus responsibility)
	if !strings.Contains(script, "ufw") {
		t.Error("Script should contain UFW firewall configuration")
	}

	// Check for NATS ports in firewall (infrastructure preparation)
	if !strings.Contains(script, "4222") {
		t.Error("Script should configure NATS client port in firewall")
	}

	// Check for callback URL (NimsForest integration)
	if !strings.Contains(script, "nimsforest.example.com") {
		t.Error("Script should contain NimsForest callback URL")
	}

	// Check for morpheus metadata file
	if !strings.Contains(script, "/etc/morpheus/node-info.json") {
		t.Error("Script should create Morpheus metadata file")
	}

	// Check that NATS installation is NOT in Morpheus template
	if strings.Contains(script, "nats-server-v2") || strings.Contains(script, "nats-server.tar.gz") {
		t.Error("Script should NOT install NATS (that's NimsForest's responsibility)")
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

	// Check for directory creation (not Docker)
	if !strings.Contains(script, "/opt/nimsforest/bin") {
		t.Error("Script should create /opt/nimsforest/bin directory")
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

func TestGenerateWithoutCallbacks(t *testing.T) {
	data := TemplateData{
		NodeRole:    RoleEdge,
		ForestID:    "test-forest",
		RegistryURL: "", // No registry
		CallbackURL: "", // No callback
	}

	script, err := Generate(RoleEdge, data)
	if err != nil {
		t.Fatalf("Failed to generate cloud-init script: %v", err)
	}

	// Should still generate valid script
	if !strings.HasPrefix(script, "#cloud-config") {
		t.Error("Script should start with #cloud-config")
	}

	// Should still have basic infrastructure setup
	if !strings.Contains(script, "ufw") {
		t.Error("Script should configure firewall even without callbacks")
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
