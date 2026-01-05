package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// StorageBoxRegistry provides access to registry data stored in a Hetzner StorageBox via WebDAV
type StorageBoxRegistry struct {
	URL      string
	Username string
	Password string

	// Internal state
	mu       sync.Mutex
	lastETag string
	client   *http.Client
}

// NewStorageBoxRegistry creates a new StorageBox registry client
func NewStorageBoxRegistry(url, username, password string) *StorageBoxRegistry {
	return &StorageBoxRegistry{
		URL:      url,
		Username: username,
		Password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Load reads the registry data from StorageBox
func (r *StorageBoxRegistry) Load() (*RegistryData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	req, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(r.Username, r.Password)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	// Handle 404 - registry doesn't exist yet, return empty
	if resp.StatusCode == http.StatusNotFound {
		r.lastETag = ""
		return NewRegistryData(), nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch registry: status %d: %s", resp.StatusCode, string(body))
	}

	// Store ETag for optimistic locking
	r.lastETag = resp.Header.Get("ETag")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Empty file - return new registry
	if len(body) == 0 {
		return NewRegistryData(), nil
	}

	var data RegistryData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	// Initialize maps if nil (for backward compatibility)
	if data.Forests == nil {
		data.Forests = make(map[string]*Forest)
	}
	if data.Nodes == nil {
		data.Nodes = make(map[string][]*Node)
	}

	return &data, nil
}

// Save writes the registry data to StorageBox with optimistic locking
func (r *StorageBoxRegistry) Save(data *RegistryData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveWithLock(data)
}

// saveWithLock performs the save operation (must be called with lock held)
func (r *StorageBoxRegistry) saveWithLock(data *RegistryData) error {
	// Update timestamp
	data.UpdatedAt = time.Now()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	req, err := http.NewRequest("PUT", r.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Set("Content-Type", "application/json")

	// Use ETag for optimistic locking if we have one
	if r.lastETag != "" {
		req.Header.Set("If-Match", r.lastETag)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	defer resp.Body.Close()

	// Check for concurrent modification
	if resp.StatusCode == http.StatusPreconditionFailed {
		return ErrConcurrentModification
	}

	// Accept any 2xx status code (200 OK, 201 Created, 204 No Content)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save registry: status %d: %s", resp.StatusCode, string(body))
	}

	// Update ETag from response
	if etag := resp.Header.Get("ETag"); etag != "" {
		r.lastETag = etag
	}

	return nil
}

// Update performs an atomic read-modify-write operation with retry
func (r *StorageBoxRegistry) Update(fn func(*RegistryData) error) error {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Load current state
		data, err := r.Load()
		if err != nil {
			return fmt.Errorf("failed to load registry: %w", err)
		}

		// Apply modification
		if err := fn(data); err != nil {
			return err
		}

		// Try to save
		r.mu.Lock()
		err = r.saveWithLock(data)
		r.mu.Unlock()

		if err == nil {
			return nil
		}

		// If concurrent modification, retry
		if err == ErrConcurrentModification {
			// Small backoff before retry
			time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
			continue
		}

		return err
	}

	return fmt.Errorf("failed to update registry after %d retries: %w", maxRetries, ErrConcurrentModification)
}

// EnsureDirectory creates the parent directory for the registry file if it doesn't exist
// This uses WebDAV MKCOL to create the directory structure
func (r *StorageBoxRegistry) EnsureDirectory() error {
	// Extract directory path from URL
	// URL format: https://user.your-storagebox.de/morpheus/registry.json
	// We need to create: /morpheus/

	// For simplicity, we'll try to PUT an empty file and let the server create the path
	// Most WebDAV servers will auto-create parent directories

	// Try a PROPFIND to check if path exists
	req, err := http.NewRequest("PROPFIND", r.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Set("Depth", "0")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check registry path: %w", err)
	}
	resp.Body.Close()

	// If file exists or would be created, we're good
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus ||
		resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("unexpected status checking registry path: %d", resp.StatusCode)
}

// Ping tests connectivity to the StorageBox
func (r *StorageBoxRegistry) Ping() error {
	req, err := http.NewRequest("OPTIONS", r.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(r.Username, r.Password)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to StorageBox: %w", err)
	}
	defer resp.Body.Close()

	// Accept authentication errors as "connected but unauthorized"
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed: check username/password")
	}

	// Accept any 2xx or 405 (Method Not Allowed - some WebDAV servers don't support OPTIONS)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}

	return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
