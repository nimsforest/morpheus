package updater

import (
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
