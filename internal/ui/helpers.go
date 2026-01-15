// Package ui provides UI helper functions for the CLI.
package ui

import (
	"fmt"
	"time"
)

// Plural returns "s" if count is not 1, empty string otherwise.
func Plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// TruncateID truncates a node ID to maxLen characters.
func TruncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen-3] + "..."
}

// TruncateIP truncates an IP address to fit within maxLen characters.
func TruncateIP(ip string, maxLen int) string {
	if len(ip) <= maxLen {
		return ip
	}
	if maxLen < 10 {
		return ip[:maxLen]
	}
	prefixLen := (maxLen - 3) / 2
	suffixLen := maxLen - 3 - prefixLen
	return ip[:prefixLen] + "..." + ip[len(ip)-suffixLen:]
}

// ProgressBar creates a simple ASCII progress bar.
func ProgressBar(value, threshold float64) string {
	width := 24
	filled := int(value / 100.0 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	warning := ""
	if value > threshold {
		warning = " ⚠️"
	}
	return bar + warning
}

// FormatDuration formats a duration in a human-readable format.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// GetNodeCount returns the number of nodes for a given forest size (legacy support).
func GetNodeCount(size string) int {
	switch size {
	case "small":
		return 2
	case "medium":
		return 3
	case "large":
		return 5
	default:
		return 1
	}
}

// IsValidSize checks if a size is valid (legacy support).
func IsValidSize(size string) bool {
	validSizes := []string{"small", "medium", "large"}
	for _, valid := range validSizes {
		if size == valid {
			return true
		}
	}
	return false
}
