package registry

// Registry defines the interface for managing forest and node state
// This interface can be implemented by both local (file-based) and remote (StorageBox) registries
type Registry interface {
	// RegisterForest adds a new forest to the registry
	RegisterForest(forest *Forest) error

	// RegisterNode adds a node to a forest
	RegisterNode(node *Node) error

	// GetForest retrieves a forest by ID
	GetForest(forestID string) (*Forest, error)

	// GetNodes retrieves all nodes for a forest
	GetNodes(forestID string) ([]*Node, error)

	// UpdateForest updates a forest's fields
	UpdateForest(updated *Forest) error

	// UpdateForestStatus updates the status of a forest
	UpdateForestStatus(forestID, status string) error

	// UpdateNodeStatus updates the status of a node
	UpdateNodeStatus(forestID, nodeID, status string) error

	// DeleteForest removes a forest and all its nodes
	DeleteForest(forestID string) error

	// ListForests returns all registered forests
	ListForests() []*Forest
}

// Ensure implementations satisfy the interface
var _ Registry = (*RemoteRegistry)(nil)

// RemoteRegistry wraps StorageBoxRegistry to implement the Registry interface
type RemoteRegistry struct {
	storage *StorageBoxRegistry
}

// NewRemoteRegistry creates a new remote registry backed by StorageBox
func NewRemoteRegistry(storage *StorageBoxRegistry) *RemoteRegistry {
	return &RemoteRegistry{storage: storage}
}

// RegisterForest adds a new forest to the registry
func (r *RemoteRegistry) RegisterForest(forest *Forest) error {
	return r.storage.Update(func(data *RegistryData) error {
		return data.RegisterForest(forest)
	})
}

// RegisterNode adds a node to a forest
func (r *RemoteRegistry) RegisterNode(node *Node) error {
	return r.storage.Update(func(data *RegistryData) error {
		return data.RegisterNode(node)
	})
}

// GetForest retrieves a forest by ID
func (r *RemoteRegistry) GetForest(forestID string) (*Forest, error) {
	data, err := r.storage.Load()
	if err != nil {
		return nil, err
	}
	return data.GetForest(forestID)
}

// GetNodes retrieves all nodes for a forest
func (r *RemoteRegistry) GetNodes(forestID string) ([]*Node, error) {
	data, err := r.storage.Load()
	if err != nil {
		return nil, err
	}
	return data.GetNodes(forestID)
}

// UpdateForest updates a forest's fields
func (r *RemoteRegistry) UpdateForest(updated *Forest) error {
	return r.storage.Update(func(data *RegistryData) error {
		existing, err := data.GetForest(updated.ID)
		if err != nil {
			return err
		}
		// Preserve CreatedAt
		createdAt := existing.CreatedAt
		*existing = *updated
		existing.CreatedAt = createdAt
		data.UpdatedAt = existing.CreatedAt // will be overwritten by Update
		return nil
	})
}

// UpdateForestStatus updates the status of a forest
func (r *RemoteRegistry) UpdateForestStatus(forestID, status string) error {
	return r.storage.Update(func(data *RegistryData) error {
		return data.UpdateForestStatus(forestID, status)
	})
}

// UpdateNodeStatus updates the status of a node
func (r *RemoteRegistry) UpdateNodeStatus(forestID, nodeID, status string) error {
	return r.storage.Update(func(data *RegistryData) error {
		return data.UpdateNodeStatus(forestID, nodeID, status)
	})
}

// DeleteForest removes a forest and all its nodes
func (r *RemoteRegistry) DeleteForest(forestID string) error {
	return r.storage.Update(func(data *RegistryData) error {
		return data.DeleteForest(forestID)
	})
}

// ListForests returns all registered forests
func (r *RemoteRegistry) ListForests() []*Forest {
	data, err := r.storage.Load()
	if err != nil {
		return []*Forest{}
	}
	return data.ListForests()
}

// Ping tests connectivity to the remote storage
func (r *RemoteRegistry) Ping() error {
	return r.storage.Ping()
}
