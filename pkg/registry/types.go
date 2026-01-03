package registry

import (
	"errors"
	"time"
)

// ErrConcurrentModification is returned when a save fails due to concurrent writes
var ErrConcurrentModification = errors.New("concurrent modification detected")

// ErrForestNotFound is returned when a forest is not found
var ErrForestNotFound = errors.New("forest not found")

// ErrNodeNotFound is returned when a node is not found
var ErrNodeNotFound = errors.New("node not found")

// RegistryData represents the complete registry state stored in StorageBox
type RegistryData struct {
	Version   int                `json:"version"`
	UpdatedAt time.Time          `json:"updated_at"`
	Forests   map[string]*Forest `json:"forests"`
	Nodes     map[string][]*Node `json:"nodes"` // key is forest ID
}

// Forest represents a NATS forest deployment
type Forest struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"` // hetzner, local
	Location      string    `json:"location"`
	Size          string    `json:"size"` // small, medium, large
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	RegistryURL   string    `json:"registry_url,omitempty"` // URL used to access registry
	LastExpansion time.Time `json:"last_expansion,omitempty"`
}

// Node represents a server node in the forest
type Node struct {
	ID        string            `json:"id"`
	ForestID  string            `json:"forest_id"`
	IP        string            `json:"ip"`
	Role      string            `json:"role"` // edge, compute, storage
	Location  string            `json:"location"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// NewRegistryData creates an empty registry data structure
func NewRegistryData() *RegistryData {
	return &RegistryData{
		Version:   1,
		UpdatedAt: time.Now(),
		Forests:   make(map[string]*Forest),
		Nodes:     make(map[string][]*Node),
	}
}

// GetForest retrieves a forest by ID
func (r *RegistryData) GetForest(forestID string) (*Forest, error) {
	forest, exists := r.Forests[forestID]
	if !exists {
		return nil, ErrForestNotFound
	}
	return forest, nil
}

// GetNodes retrieves all nodes for a forest
func (r *RegistryData) GetNodes(forestID string) ([]*Node, error) {
	if _, exists := r.Forests[forestID]; !exists {
		return nil, ErrForestNotFound
	}
	nodes, exists := r.Nodes[forestID]
	if !exists {
		return []*Node{}, nil
	}
	return nodes, nil
}

// RegisterForest adds a new forest to the registry
func (r *RegistryData) RegisterForest(forest *Forest) error {
	if _, exists := r.Forests[forest.ID]; exists {
		return errors.New("forest already exists: " + forest.ID)
	}
	if forest.CreatedAt.IsZero() {
		forest.CreatedAt = time.Now()
	}
	r.Forests[forest.ID] = forest
	r.Nodes[forest.ID] = []*Node{}
	r.UpdatedAt = time.Now()
	return nil
}

// RegisterNode adds a node to a forest
func (r *RegistryData) RegisterNode(node *Node) error {
	if _, exists := r.Forests[node.ForestID]; !exists {
		return ErrForestNotFound
	}
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	r.Nodes[node.ForestID] = append(r.Nodes[node.ForestID], node)
	r.UpdatedAt = time.Now()
	return nil
}

// UpdateForestStatus updates the status of a forest
func (r *RegistryData) UpdateForestStatus(forestID, status string) error {
	forest, exists := r.Forests[forestID]
	if !exists {
		return ErrForestNotFound
	}
	forest.Status = status
	r.UpdatedAt = time.Now()
	return nil
}

// UpdateNodeStatus updates the status of a node
func (r *RegistryData) UpdateNodeStatus(forestID, nodeID, status string) error {
	nodes, exists := r.Nodes[forestID]
	if !exists {
		return ErrForestNotFound
	}
	for _, node := range nodes {
		if node.ID == nodeID {
			node.Status = status
			r.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrNodeNotFound
}

// DeleteForest removes a forest and all its nodes
func (r *RegistryData) DeleteForest(forestID string) error {
	if _, exists := r.Forests[forestID]; !exists {
		return ErrForestNotFound
	}
	delete(r.Forests, forestID)
	delete(r.Nodes, forestID)
	r.UpdatedAt = time.Now()
	return nil
}

// ListForests returns all registered forests
func (r *RegistryData) ListForests() []*Forest {
	forests := make([]*Forest, 0, len(r.Forests))
	for _, forest := range r.Forests {
		forests = append(forests, forest)
	}
	return forests
}
