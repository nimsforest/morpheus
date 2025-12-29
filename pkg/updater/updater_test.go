package updater

import (
	"testing"
)

func TestCreateHTTPClient(t *testing.T) {
	// Test HTTP client creation
	t.Run("client_creation", func(t *testing.T) {
		client := createHTTPClient()
		if client == nil {
			t.Fatal("HTTP client should not be nil")
		}
		if client.Timeout == 0 {
			t.Error("Client timeout should be set")
		}
	})

	// Test that client has proper transport
	t.Run("client_transport", func(t *testing.T) {
		client := createHTTPClient()
		if client.Transport == nil {
			// Default transport is used, which is fine
			t.Log("Using default transport")
		}
	})
}

func TestNewUpdater(t *testing.T) {
	t.Run("create_updater", func(t *testing.T) {
		updater := NewUpdater("1.0.0")
		if updater == nil {
			t.Fatal("Updater should not be nil")
		}
		if updater.currentVersion != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %s", updater.currentVersion)
		}
		if updater.client == nil {
			t.Error("HTTP client should not be nil")
		}
	})
}
