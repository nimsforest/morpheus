package registry

import (
	"testing"
	"time"
)

func TestNewRegistryData(t *testing.T) {
	data := NewRegistryData()

	if data == nil {
		t.Fatal("Expected non-nil registry data")
	}
	if data.Version != 1 {
		t.Errorf("Expected version 1, got %d", data.Version)
	}
	if data.Forests == nil {
		t.Error("Expected non-nil Forests map")
	}
	if data.Nodes == nil {
		t.Error("Expected non-nil Nodes map")
	}
}

func TestRegistryDataRegisterForest(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}

	err := data.RegisterForest(forest)
	if err != nil {
		t.Fatalf("Failed to register forest: %v", err)
	}

	// Verify forest was registered
	retrieved, err := data.GetForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to get forest: %v", err)
	}

	if retrieved.ID != forest.ID {
		t.Errorf("Expected ID '%s', got '%s'", forest.ID, retrieved.ID)
	}
	if retrieved.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestRegistryDataRegisterForestDuplicate(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}

	data.RegisterForest(forest)

	// Try to register again
	err := data.RegisterForest(forest)
	if err == nil {
		t.Error("Expected error when registering duplicate forest")
	}
}

func TestRegistryDataGetForestNotFound(t *testing.T) {
	data := NewRegistryData()

	_, err := data.GetForest("nonexistent")
	if err != ErrForestNotFound {
		t.Errorf("Expected ErrForestNotFound, got %v", err)
	}
}

func TestRegistryDataRegisterNode(t *testing.T) {
	data := NewRegistryData()

	// First register a forest
	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	// Now register a node
	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}

	err := data.RegisterNode(node)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Verify node was registered
	nodes, err := data.GetNodes("test-forest")
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	if nodes[0].ID != node.ID {
		t.Errorf("Expected node ID '%s', got '%s'", node.ID, nodes[0].ID)
	}
	if nodes[0].CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestRegistryDataRegisterNodeNoForest(t *testing.T) {
	data := NewRegistryData()

	node := &Node{
		ID:       "12345",
		ForestID: "nonexistent",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}

	err := data.RegisterNode(node)
	if err != ErrForestNotFound {
		t.Errorf("Expected ErrForestNotFound, got %v", err)
	}
}

func TestRegistryDataUpdateForestStatus(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	err := data.UpdateForestStatus("test-forest", "active")
	if err != nil {
		t.Fatalf("Failed to update forest status: %v", err)
	}

	retrieved, _ := data.GetForest("test-forest")
	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}
}

func TestRegistryDataUpdateForestStatusNotFound(t *testing.T) {
	data := NewRegistryData()

	err := data.UpdateForestStatus("nonexistent", "active")
	if err != ErrForestNotFound {
		t.Errorf("Expected ErrForestNotFound, got %v", err)
	}
}

func TestRegistryDataUpdateNodeStatus(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "provisioning",
	}
	data.RegisterNode(node)

	err := data.UpdateNodeStatus("test-forest", "12345", "active")
	if err != nil {
		t.Fatalf("Failed to update node status: %v", err)
	}

	nodes, _ := data.GetNodes("test-forest")
	if nodes[0].Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", nodes[0].Status)
	}
}

func TestRegistryDataUpdateNodeStatusNotFound(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	err := data.UpdateNodeStatus("test-forest", "nonexistent", "active")
	if err != ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound, got %v", err)
	}
}

func TestRegistryDataDeleteForest(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	err := data.DeleteForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to delete forest: %v", err)
	}

	// Verify forest is gone
	_, err = data.GetForest("test-forest")
	if err != ErrForestNotFound {
		t.Errorf("Expected ErrForestNotFound, got %v", err)
	}
}

func TestRegistryDataDeleteForestNotFound(t *testing.T) {
	data := NewRegistryData()

	err := data.DeleteForest("nonexistent")
	if err != ErrForestNotFound {
		t.Errorf("Expected ErrForestNotFound, got %v", err)
	}
}

func TestRegistryDataListForests(t *testing.T) {
	data := NewRegistryData()

	// Register multiple forests
	for i := 1; i <= 3; i++ {
		forest := &Forest{
			ID:       "forest-" + string(rune('0'+i)),
			Provider: "hetzner",
			Location: "hel1",
			Size:     "small",
			Status:   "active",
		}
		data.RegisterForest(forest)
	}

	forests := data.ListForests()
	if len(forests) != 3 {
		t.Errorf("Expected 3 forests, got %d", len(forests))
	}
}

func TestRegistryDataUpdatedAt(t *testing.T) {
	data := NewRegistryData()
	initialTime := data.UpdatedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	if !data.UpdatedAt.After(initialTime) {
		t.Error("Expected UpdatedAt to be updated after RegisterForest")
	}

	// Test node registration updates timestamp
	previousTime := data.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	node := &Node{
		ID:       "12345",
		ForestID: "test-forest",
		Role:     "edge",
		IP:       "2a01:4f8::1",
		Location: "hel1",
		Status:   "active",
	}
	data.RegisterNode(node)

	if !data.UpdatedAt.After(previousTime) {
		t.Error("Expected UpdatedAt to be updated after RegisterNode")
	}
}

func TestRegistryDataGetNodesEmptyForest(t *testing.T) {
	data := NewRegistryData()

	forest := &Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "provisioning",
	}
	data.RegisterForest(forest)

	nodes, err := data.GetNodes("test-forest")
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}
