package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestStorageBoxRegistry_Load_Empty(t *testing.T) {
	// Create a test server that returns 404 (file not found)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	data, err := registry.Load()
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil registry data")
	}
	if data.Version != 1 {
		t.Errorf("Expected version 1, got %d", data.Version)
	}
}

func TestStorageBoxRegistry_Load_Existing(t *testing.T) {
	// Create test data
	testData := NewRegistryData()
	testData.RegisterForest(&Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "active",
	})

	// Create a test server that returns the test data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("ETag", "\"test-etag\"")
			json.NewEncoder(w).Encode(testData)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	data, err := registry.Load()
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	forest, err := data.GetForest("test-forest")
	if err != nil {
		t.Fatalf("Failed to get forest: %v", err)
	}

	if forest.ID != "test-forest" {
		t.Errorf("Expected forest ID 'test-forest', got '%s'", forest.ID)
	}
}

func TestStorageBoxRegistry_Save(t *testing.T) {
	var savedData []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			// Read the request body
			buf := make([]byte, 10000)
			n, _ := r.Body.Read(buf)
			savedData = buf[:n]

			// Check auth
			user, pass, ok := r.BasicAuth()
			if !ok || user != "testuser" || pass != "testpass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.Header().Set("ETag", "\"new-etag\"")
			w.WriteHeader(http.StatusCreated)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "testuser", "testpass")

	data := NewRegistryData()
	data.RegisterForest(&Forest{
		ID:       "test-forest",
		Provider: "hetzner",
		Location: "hel1",
		Size:     "small",
		Status:   "active",
	})

	err := registry.Save(data)
	if err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Verify the saved data
	var saved RegistryData
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatalf("Failed to parse saved data: %v", err)
	}

	if len(saved.Forests) != 1 {
		t.Errorf("Expected 1 forest, got %d", len(saved.Forests))
	}
}

func TestStorageBoxRegistry_Save_OptimisticLocking(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("ETag", "\"original-etag\"")
			data := NewRegistryData()
			json.NewEncoder(w).Encode(data)
			return
		}
		if r.Method == "PUT" {
			requestCount++

			// Check If-Match header
			ifMatch := r.Header.Get("If-Match")
			if ifMatch != "" && ifMatch != "\"original-etag\"" {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	// Load first to get ETag
	_, err := registry.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Save should include If-Match header
	data := NewRegistryData()
	err = registry.Save(data)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 PUT request, got %d", requestCount)
	}
}

func TestStorageBoxRegistry_Save_ConcurrentModification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("ETag", "\"original-etag\"")
			data := NewRegistryData()
			json.NewEncoder(w).Encode(data)
			return
		}
		if r.Method == "PUT" {
			// Always return precondition failed to simulate concurrent modification
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	// Load first to get ETag
	_, err := registry.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Save should fail with ErrConcurrentModification
	data := NewRegistryData()
	err = registry.Save(data)
	if err != ErrConcurrentModification {
		t.Errorf("Expected ErrConcurrentModification, got %v", err)
	}
}

func TestStorageBoxRegistry_Update(t *testing.T) {
	var version int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == "GET" {
			data := NewRegistryData()
			data.Version = version
			w.Header().Set("ETag", "\"etag-"+string(rune('0'+version))+"\"")
			json.NewEncoder(w).Encode(data)
			return
		}
		if r.Method == "PUT" {
			// Check If-Match header
			ifMatch := r.Header.Get("If-Match")
			expectedETag := "\"etag-" + string(rune('0'+version)) + "\""
			if ifMatch != "" && ifMatch != expectedETag {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}

			// Parse and update version
			var data RegistryData
			json.NewDecoder(r.Body).Decode(&data)
			version = data.Version

			w.Header().Set("ETag", "\"etag-"+string(rune('0'+version))+"\"")
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	err := registry.Update(func(data *RegistryData) error {
		data.Version = 2
		return data.RegisterForest(&Forest{
			ID:       "test-forest",
			Provider: "hetzner",
			Location: "hel1",
			Size:     "small",
			Status:   "active",
		})
	})

	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify version was updated
	mu.Lock()
	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}
	mu.Unlock()
}

func TestStorageBoxRegistry_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Allow", "GET, PUT, OPTIONS")
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	err := registry.Ping()
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestStorageBoxRegistry_Ping_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "wrongpass")

	err := registry.Ping()
	if err == nil {
		t.Error("Expected error for unauthorized access")
	}
}

func TestStorageBoxRegistry_Load_EmptyFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{})
			return
		}
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "user", "pass")

	data, err := registry.Load()
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil registry data")
	}
	if data.Version != 1 {
		t.Errorf("Expected version 1, got %d", data.Version)
	}
}

func TestStorageBoxRegistry_BasicAuth(t *testing.T) {
	var receivedUser, receivedPass string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, receivedPass, _ = r.BasicAuth()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	registry := NewStorageBoxRegistry(server.URL+"/registry.json", "myuser", "mypass")
	registry.Load()

	if receivedUser != "myuser" {
		t.Errorf("Expected user 'myuser', got '%s'", receivedUser)
	}
	if receivedPass != "mypass" {
		t.Errorf("Expected pass 'mypass', got '%s'", receivedPass)
	}
}
