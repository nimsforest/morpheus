package forest

import (
	"testing"
)

func TestGetNodeCount(t *testing.T) {
	tests := []struct {
		size     string
		expected int
	}{
		{"wood", 1},
		{"forest", 3},
		{"jungle", 5},
		{"unknown", 1}, // default
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			count := getNodeCount(tt.size)
			if count != tt.expected {
				t.Errorf("For size '%s', expected %d nodes, got %d",
					tt.size, tt.expected, count)
			}
		})
	}
}
