package updater

import (
	"os"
	"runtime"
	"testing"
)

func TestNewUpdater(t *testing.T) {
	t.Run("create_updater", func(t *testing.T) {
		updater := NewUpdater("1.0.0")
		if updater == nil {
			t.Fatal("Updater should not be nil")
		}
		if updater.currentVersion != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %s", updater.currentVersion)
		}
	})
}

func TestIsRestrictedEnvironment(t *testing.T) {
	t.Run("normal_environment", func(t *testing.T) {
		// Save original env
		origTermux := os.Getenv("TERMUX_VERSION")
		defer func() {
			if origTermux != "" {
				os.Setenv("TERMUX_VERSION", origTermux)
			} else {
				os.Unsetenv("TERMUX_VERSION")
			}
		}()
		
		// Clear Termux env var
		os.Unsetenv("TERMUX_VERSION")
		
		// On non-Android systems without TERMUX_VERSION, should return false
		// (unless we're actually running on Android/Termux)
		result := isRestrictedEnvironment()
		
		// If we're on Linux, it might still detect Android, so we check the logic
		if runtime.GOOS != "linux" {
			if result {
				t.Error("Expected false for non-Android environment")
			}
		}
	})
	
	t.Run("termux_environment_via_env", func(t *testing.T) {
		// Save original env
		origTermux := os.Getenv("TERMUX_VERSION")
		defer func() {
			if origTermux != "" {
				os.Setenv("TERMUX_VERSION", origTermux)
			} else {
				os.Unsetenv("TERMUX_VERSION")
			}
		}()
		
		// Set Termux env var
		os.Setenv("TERMUX_VERSION", "0.118")
		
		result := isRestrictedEnvironment()
		if !result {
			t.Error("Expected true when TERMUX_VERSION is set")
		}
	})
}

func TestGetPlatform(t *testing.T) {
	platform := GetPlatform()
	expected := runtime.GOOS + "/" + runtime.GOARCH
	
	if platform != expected {
		t.Errorf("Expected platform %s, got %s", expected, platform)
	}
}
