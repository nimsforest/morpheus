package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"v1 greater major", "2.0.0", "1.0.0", 1},
		{"v1 less major", "1.0.0", "2.0.0", -1},
		{"v1 greater minor", "1.2.0", "1.1.0", 1},
		{"v1 less minor", "1.1.0", "1.2.0", -1},
		{"v1 greater patch", "1.0.2", "1.0.1", 1},
		{"v1 less patch", "1.0.1", "1.0.2", -1},
		{"with v prefix", "v1.2.3", "v1.2.2", 1},
		{"mixed v prefix", "v1.2.3", "1.2.3", 0},
		{"different lengths 1", "1.2", "1.2.0", 0},
		{"different lengths 2", "1.2.1", "1.2", 1},
		{"complex version", "1.10.0", "1.9.0", 1},
		{"pre-release ignored", "1.2.3-beta", "1.2.3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%q, %q) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name           string
		newVersion     string
		currentVersion string
		expected       bool
	}{
		{"newer version", "1.2.0", "1.1.0", true},
		{"same version", "1.1.0", "1.1.0", false},
		{"older version", "1.0.0", "1.1.0", false},
		{"major version bump", "2.0.0", "1.9.9", true},
		{"with v prefix", "v1.2.0", "v1.1.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNewer(tt.newVersion, tt.currentVersion)
			if result != tt.expected {
				t.Errorf("IsNewer(%q, %q) = %v, expected %v", tt.newVersion, tt.currentVersion, result, tt.expected)
			}
		})
	}
}
