package forest

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Registry manages the forest and its nodes
type Registry struct {
	mu      sync.RWMutex
	forests map[string]*Forest
	nodes   map[string][]*Node
	path    string
}

// Forest represents a NATS forest deployment
type Forest struct {
	ID        string    `json:"id"`
	Size      string    `json:"size"` // wood, forest, jungle
	Location  string    `json:"location"`
	Provider  string    `json:"provider"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Node represents a server node in the forest
type Node struct {
	ID        string            `json:"id"`
	ForestID  string            `json:"forest_id"`
	Role      string            `json:"role"` // edge, compute, storage
	IP        string            `json:"ip"`
	Location  string            `json:"location"`
	Capacity  string            `json:"capacity,omitempty"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// NewRegistry creates a new forest registry
func NewRegistry(path string) (*Registry, error) {
	r := &Registry{
		forests: make(map[string]*Forest),
		nodes:   make(map[string][]*Node),
		path:    path,
	}

	// Load existing registry if it exists
	if _, err := os.Stat(path); err == nil {
		if err := r.Load(); err != nil {
			return nil, fmt.Errorf("failed to load registry: %w", err)
		}
	}

	return r, nil
}

// RegisterForest adds a new forest to the registry
func (r *Registry) RegisterForest(forest *Forest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.forests[forest.ID]; exists {
		return fmt.Errorf("forest already exists: %s", forest.ID)
	}

	forest.CreatedAt = time.Now()
	r.forests[forest.ID] = forest
	r.nodes[forest.ID] = []*Node{}

	return r.save()
}

// RegisterNode adds a node to a forest
func (r *Registry) RegisterNode(node *Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.forests[node.ForestID]; !exists {
		return fmt.Errorf("forest not found: %s", node.ForestID)
	}

	node.CreatedAt = time.Now()
	r.nodes[node.ForestID] = append(r.nodes[node.ForestID], node)

	return r.save()
}

// GetForest retrieves a forest by ID
func (r *Registry) GetForest(forestID string) (*Forest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	forest, exists := r.forests[forestID]
	if !exists {
		return nil, fmt.Errorf("forest not found: %s", forestID)
	}

	return forest, nil
}

// GetNodes retrieves all nodes for a forest
func (r *Registry) GetNodes(forestID string) ([]*Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes, exists := r.nodes[forestID]
	if !exists {
		return nil, fmt.Errorf("forest not found: %s", forestID)
	}

	return nodes, nil
}

// UpdateForest updates a forest's fields
func (r *Registry) UpdateForest(updated *Forest) error {
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
func (r *Registry) UpdateForestStatus(forestID, status string) error {
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
func (r *Registry) UpdateNodeStatus(forestID, nodeID, status string) error {
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
func (r *Registry) DeleteForest(forestID string) error {
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
func (r *Registry) ListForests() []*Forest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	forests := make([]*Forest, 0, len(r.forests))
	for _, forest := range r.forests {
		forests = append(forests, forest)
	}

	return forests
}

// Load reads the registry from disk
func (r *Registry) Load() error {
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

	return nil
}

// save writes the registry to disk (must be called with lock held)
func (r *Registry) save() error {
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
