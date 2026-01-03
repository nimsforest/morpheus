package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LocalRegistry implements the Registry interface using a local JSON file
// This is similar to forest.Registry but uses the registry package types
type LocalRegistry struct {
	mu      sync.RWMutex
	forests map[string]*Forest
	nodes   map[string][]*Node
	path    string
}

// NewLocalRegistry creates a new local file-based registry
func NewLocalRegistry(path string) (*LocalRegistry, error) {
	r := &LocalRegistry{
		forests: make(map[string]*Forest),
		nodes:   make(map[string][]*Node),
		path:    path,
	}

	// Load existing registry if it exists
	if _, err := os.Stat(path); err == nil {
		if err := r.load(); err != nil {
			return nil, fmt.Errorf("failed to load registry: %w", err)
		}
	}

	return r, nil
}

// Ensure LocalRegistry implements Registry interface
var _ Registry = (*LocalRegistry)(nil)

// RegisterForest adds a new forest to the registry
func (r *LocalRegistry) RegisterForest(forest *Forest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.forests[forest.ID]; exists {
		return fmt.Errorf("forest already exists: %s", forest.ID)
	}

	if forest.CreatedAt.IsZero() {
		forest.CreatedAt = time.Now()
	}
	r.forests[forest.ID] = forest
	r.nodes[forest.ID] = []*Node{}

	return r.save()
}

// RegisterNode adds a node to a forest
func (r *LocalRegistry) RegisterNode(node *Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.forests[node.ForestID]; !exists {
		return fmt.Errorf("forest not found: %s", node.ForestID)
	}

	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	r.nodes[node.ForestID] = append(r.nodes[node.ForestID], node)

	return r.save()
}

// GetForest retrieves a forest by ID
func (r *LocalRegistry) GetForest(forestID string) (*Forest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	forest, exists := r.forests[forestID]
	if !exists {
		return nil, fmt.Errorf("forest not found: %s", forestID)
	}

	return forest, nil
}

// GetNodes retrieves all nodes for a forest
func (r *LocalRegistry) GetNodes(forestID string) ([]*Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes, exists := r.nodes[forestID]
	if !exists {
		return nil, fmt.Errorf("forest not found: %s", forestID)
	}

	return nodes, nil
}

// UpdateForest updates a forest's fields
func (r *LocalRegistry) UpdateForest(updated *Forest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	forest, exists := r.forests[updated.ID]
	if !exists {
		return fmt.Errorf("forest not found: %s", updated.ID)
	}

	// Update fields (preserve CreatedAt)
	createdAt := forest.CreatedAt
	*forest = *updated
	forest.CreatedAt = createdAt

	return r.save()
}

// UpdateForestStatus updates the status of a forest
func (r *LocalRegistry) UpdateForestStatus(forestID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	forest, exists := r.forests[forestID]
	if !exists {
		return fmt.Errorf("forest not found: %s", forestID)
	}

	forest.Status = status
	return r.save()
}

// UpdateNodeStatus updates the status of a node
func (r *LocalRegistry) UpdateNodeStatus(forestID, nodeID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes, exists := r.nodes[forestID]
	if !exists {
		return fmt.Errorf("forest not found: %s", forestID)
	}

	for _, node := range nodes {
		if node.ID == nodeID {
			node.Status = status
			return r.save()
		}
	}

	return fmt.Errorf("node not found: %s", nodeID)
}

// DeleteForest removes a forest and all its nodes
func (r *LocalRegistry) DeleteForest(forestID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.forests[forestID]; !exists {
		return fmt.Errorf("forest not found: %s", forestID)
	}

	delete(r.forests, forestID)
	delete(r.nodes, forestID)

	return r.save()
}

// ListForests returns all registered forests
func (r *LocalRegistry) ListForests() []*Forest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	forests := make([]*Forest, 0, len(r.forests))
	for _, forest := range r.forests {
		forests = append(forests, forest)
	}

	return forests
}

// load reads the registry from disk
func (r *LocalRegistry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var state struct {
		Forests map[string]*Forest `json:"forests"`
		Nodes   map[string][]*Node `json:"nodes"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	r.forests = state.Forests
	r.nodes = state.Nodes

	// Initialize maps if nil
	if r.forests == nil {
		r.forests = make(map[string]*Forest)
	}
	if r.nodes == nil {
		r.nodes = make(map[string][]*Node)
	}

	return nil
}

// save writes the registry to disk (must be called with lock held)
func (r *LocalRegistry) save() error {
	state := struct {
		Forests map[string]*Forest `json:"forests"`
		Nodes   map[string][]*Node `json:"nodes"`
	}{
		Forests: r.forests,
		Nodes:   r.nodes,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.path, data, 0644)
}
