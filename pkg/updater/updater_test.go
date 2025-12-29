package updater

import (
	"os"
	"testing"
)

func TestCreateTLSConfig(t *testing.T) {
	// Test normal TLS configuration
	t.Run("normal_tls_config", func(t *testing.T) {
		config := createTLSConfig()
		if config == nil {
			t.Fatal("TLS config should not be nil")
		}
		if config.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be false by default")
		}
		if config.RootCAs == nil {
			t.Error("RootCAs should not be nil")
		}
	})

	// Test with MORPHEUS_SKIP_TLS_VERIFY environment variable
	t.Run("skip_tls_verify", func(t *testing.T) {
		os.Setenv("MORPHEUS_SKIP_TLS_VERIFY", "1")
		defer os.Unsetenv("MORPHEUS_SKIP_TLS_VERIFY")

		config := createTLSConfig()
		if config == nil {
			t.Fatal("TLS config should not be nil")
		}
		if !config.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be true when MORPHEUS_SKIP_TLS_VERIFY=1")
		}
	})
}

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
			t.Error("Client transport should not be nil")
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
