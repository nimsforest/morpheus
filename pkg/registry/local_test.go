package registry

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewLocalRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, err := NewLocalRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}
}

func TestLocalRegistryRegisterForest(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}

	err := registry.RegisterForest(forest)
	if err != nil {
		t.Fatalf("Failed to register forest: %v", err)
	}

	// Verify forest was registered
	retrieved, err := registry.GetForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to get forest: %v", err)
	}

	if retrieved.ID != forest.ID {
		t.Errorf("Expected ID '%s', got '%s'", forest.ID, retrieved.ID)
	}
	if retrieved.Size != forest.Size {
		t.Errorf("Expected size '%s', got '%s'", forest.Size, retrieved.Size)
	}
}

func TestLocalRegistryRegisterForestDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}

	registry.RegisterForest(forest)

	// Try to register again
	err := registry.RegisterForest(forest)
	if err == nil {
		t.Error("Expected error when registering duplicate forest")
	}
}

func TestLocalRegistryRegisterNode(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	// First register a forest
	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	// Now register a node
	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}

	err := registry.RegisterNode(node)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Verify node was registered
	nodes, err := registry.GetNodes("test-forest")
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	if nodes[0].ID != node.ID {
		t.Errorf("Expected node ID '%s', got '%s'", node.ID, nodes[0].ID)
	}
}

func TestLocalRegistryRegisterNodeNoForest(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	node := &Node{
		ID:       "12345",
		ForestID: "nonexistent",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}

	err := registry.RegisterNode(node)
	if err == nil {
		t.Error("Expected error when registering node for nonexistent forest")
	}
}

func TestLocalRegistryUpdateForestStatus(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	err := registry.UpdateForestStatus("test-forest", "active")
	if err != nil {
		t.Fatalf("Failed to update forest status: %v", err)
	}

	retrieved, _ := registry.GetForest("test-forest")
	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}
}

func TestLocalRegistryUpdateNodeStatus(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "provisioning",
	}
	registry.RegisterNode(node)

	err := registry.UpdateNodeStatus("test-forest", "12345", "active")
	if err != nil {
		t.Fatalf("Failed to update node status: %v", err)
	}

	nodes, _ := registry.GetNodes("test-forest")
	if nodes[0].Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", nodes[0].Status)
	}
}

func TestLocalRegistryDeleteForest(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	err := registry.DeleteForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to delete forest: %v", err)
	}

	// Verify forest is gone
	_, err = registry.GetForest("test-forest")
	if err == nil {
		t.Error("Expected error when getting deleted forest")
	}
}

func TestLocalRegistryListForests(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	// Register multiple forests
	for i := 1; i <= 3; i++ {
		forest := &Forest{
			ID:       "forest-" + string(rune('0'+i)),
			Size:     "small",
			Location: "hel1",
			Provider: "hetzner",
			Status:   "active",
		}
		registry.RegisterForest(forest)
	}

	forests := registry.ListForests()
	if len(forests) != 3 {
		t.Errorf("Expected 3 forests, got %d", len(forests))
	}
}

func TestLocalRegistryPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create and populate registry
	registry1, _ := NewLocalRegistry(registryPath)
	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "active",
	}
	registry1.RegisterForest(forest)

	// Create new registry instance and load from disk
	registry2, err := NewLocalRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Verify data was persisted
	retrieved, err := registry2.GetForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to get forest from loaded registry: %v", err)
	}

	if retrieved.ID != forest.ID {
		t.Errorf("Expected ID '%s', got '%s'", forest.ID, retrieved.ID)
	}
}

func TestLocalRegistryForestTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	before := time.Now()
	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)
	after := time.Now()

	retrieved, _ := registry.GetForest("test-forest")
	if retrieved.CreatedAt.Before(before) || retrieved.CreatedAt.After(after) {
		t.Error("CreatedAt timestamp is not within expected range")
	}
}

func TestLocalRegistryNodeTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	before := time.Now()
	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}
	registry.RegisterNode(node)
	after := time.Now()

	nodes, _ := registry.GetNodes("test-forest")
	if nodes[0].CreatedAt.Before(before) || nodes[0].CreatedAt.After(after) {
		t.Error("CreatedAt timestamp is not within expected range")
	}
}

func TestLocalRegistryUpdateForest(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry, _ := NewLocalRegistry(registryPath)

	forest := &Forest{
		ID:       "test-forest",
		Size:     "small",
		Location: "hel1",
		Provider: "hetzner",
		Status:   "provisioning",
	}
	registry.RegisterForest(forest)

	// Get the created timestamp
	original, _ := registry.GetForest("test-forest")
	originalCreatedAt := original.CreatedAt

	// Update the forest
	updated := &Forest{
		ID:       "test-forest",
		Size:     "medium",
		Location: "nbg1",
		Provider: "hetzner",
		Status:   "active",
	}

	err := registry.UpdateForest(updated)
	if err != nil {
		t.Fatalf("Failed to update forest: %v", err)
	}

	// Verify update
	retrieved, _ := registry.GetForest("test-forest")
	if retrieved.Size != "medium" {
		t.Errorf("Expected size 'medium', got '%s'", retrieved.Size)
	}
	if retrieved.Location != "nbg1" {
		t.Errorf("Expected location 'nbg1', got '%s'", retrieved.Location)
	}
	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}

	// Verify CreatedAt was preserved
	if !retrieved.CreatedAt.Equal(originalCreatedAt) {
		t.Error("CreatedAt should be preserved after update")
	}
}
