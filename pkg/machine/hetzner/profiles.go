package hetzner

import (
	"context"
	"fmt"
)

// SelectBestServerType selects the best available server type considering location availability.
// It tries the primary serverType first, then falls back to the provided fallbacks in order.
//
// Priority: Primary server type is always preferred over fallbacks.
// If the primary server type is available anywhere, it will be used even if not
// in the preferred locations. This ensures cost optimization.
func (p *Provider) SelectBestServerType(ctx context.Context, serverType string, fallbacks []string, preferredLocations []string) (string, []string, error) {
	// Try primary first, then fallbacks in order
	allOptions := append([]string{serverType}, fallbacks...)

	// First pass: try to find a server type that works with preferred locations
	for _, st := range allOptions {
		// Get locations where this server type is available
		availableLocations, err := p.GetAvailableLocations(ctx, st)
		if err != nil {
			// Skip this server type if we can't get its locations
			continue
		}

		if len(availableLocations) == 0 {
			continue
		}

		// If we have preferred locations, filter to those (preserving preferred order)
		if len(preferredLocations) > 0 {
			matchingLocations := intersectLocationsPreserveOrder(preferredLocations, availableLocations)
			if len(matchingLocations) > 0 {
				return st, matchingLocations, nil
			}
		} else {
			// No preferred locations, use all available
			return st, availableLocations, nil
		}
	}

	// Second pass: if no server type works with preferred locations,
	// use the PRIMARY server type with ANY available location (prioritize cost)
	availableLocations, err := p.GetAvailableLocations(ctx, serverType)
	if err == nil && len(availableLocations) > 0 {
		// Primary type is available somewhere, use it
		return serverType, availableLocations, nil
	}

	// Third pass: try fallbacks with any available location
	for _, st := range fallbacks {
		availableLocations, err := p.GetAvailableLocations(ctx, st)
		if err != nil {
			continue
		}
		if len(availableLocations) > 0 {
			return st, availableLocations, nil
		}
	}

	return "", nil, fmt.Errorf("no suitable server type found: %s (fallbacks: %v)", serverType, fallbacks)
}

// GetDefaultLocations returns a recommended set of locations for Hetzner.
// These are prioritized by reliability and server type availability.
// Helsinki (hel1) is preferred first, then German locations (nbg1, fsn1).
func GetDefaultLocations() []string {
	return []string{
		"hel1", // Helsinki, Finland - Preferred location
		"nbg1", // Nuremberg, Germany - Second choice
		"fsn1", // Falkenstein, Germany - Third choice
		"ash",  // Ashburn, USA - US East Coast
		"hil",  // Hillsboro, USA - US West Coast
	}
}

// GetLocationDescription returns human-readable location descriptions
func GetLocationDescription(loc string) string {
	descriptions := map[string]string{
		"fsn1": "Falkenstein, Germany (EU)",
		"nbg1": "Nuremberg, Germany (EU)",
		"hel1": "Helsinki, Finland (EU)",
		"ash":  "Ashburn, VA, USA",
		"hil":  "Hillsboro, OR, USA",
		"sin":  "Singapore, Asia",
	}
	if desc, ok := descriptions[loc]; ok {
		return desc
	}
	return loc
}

// intersectLocationsPreserveOrder returns locations that exist in both lists.
// The result order follows the preferred list order (first argument).
// This ensures user's preferred location order is respected.
func intersectLocationsPreserveOrder(preferred, available []string) []string {
	availableSet := make(map[string]bool)
	for _, item := range available {
		availableSet[item] = true
	}

	var result []string
	for _, item := range preferred {
		if availableSet[item] {
			result = append(result, item)
		}
	}
	return result
}

// GetEstimatedCost returns the estimated monthly cost for a server type
func GetEstimatedCost(serverType string) float64 {
	// Approximate monthly costs in EUR (as of 2024)
	// Note: Prices may vary; these are estimates for reference
	costs := map[string]float64{
		"cx11":  3.29,
		"cx21":  3.29,
		"cx22":  3.29,
		"cx23":  4.49,
		"cx31":  6.29,
		"cx32":  6.29,
		"cx41":  12.29,
		"cx42":  12.29,
		"cx51":  24.29,
		"cx52":  24.29,
		"cpx11": 4.49,
		"cpx21": 8.49,
		"cpx22": 11.49,
		"cpx31": 15.49,
		"cpx41": 29.49,
		"cpx51": 57.49,
		"ccx13": 12.49,
		"ccx23": 24.49,
		"ccx33": 48.49,
		"cax11": 3.79,
		"cax21": 7.59,
		"cax31": 15.19,
		"cax41": 30.39,
	}

	if cost, ok := costs[serverType]; ok {
		return cost
	}

	// Default estimate
	return 5.0
}
