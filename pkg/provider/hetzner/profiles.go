package hetzner

import (
	"context"
	"fmt"

	"github.com/nimsforest/morpheus/pkg/provider"
)

// MachineTypeMapping maps machine profiles to Hetzner-specific server types
// with fallback options if the primary type is unavailable
type MachineTypeMapping struct {
	Primary    string   // Primary choice
	Fallbacks  []string // Fallback options in order of preference
	Architecture string // "x86" or "arm"
}

// GetHetznerServerType returns the appropriate Hetzner server type for a machine profile
// 
// Morpheus is opinionated: We use Ubuntu 24.04 and x86 architecture only.
// ARM types (cax series) are NOT included because ubuntu-24.04 doesn't support ARM on Hetzner.
//
// Returns multiple x86 options in order of preference for fallback if primary is unavailable.
func GetHetznerServerType(profile provider.MachineProfile) MachineTypeMapping {
	mappings := map[provider.MachineProfile]MachineTypeMapping{
		provider.ProfileSmall: {
			Primary: "cx22",  // 2 vCPU (shared AMD), 4 GB RAM - ~€3.29/mo
			Fallbacks: []string{
				"cpx11", // 2 vCPU (dedicated AMD), 2 GB RAM - ~€4.49/mo (better performance)
				"cx21",  // 2 vCPU (shared Intel), 4 GB RAM - ~€3.29/mo (older gen)
			},
			Architecture: "x86",
		},
		provider.ProfileMedium: {
			Primary: "cpx21", // 3 vCPU (dedicated AMD), 4 GB RAM - ~€8.49/mo
			Fallbacks: []string{
				"cx32",  // 4 vCPU (shared AMD), 8 GB RAM - ~€6.29/mo (cheaper but shared)
				"cpx31", // 4 vCPU (dedicated AMD), 8 GB RAM - ~€15.49/mo (more powerful)
			},
			Architecture: "x86",
		},
		provider.ProfileLarge: {
			Primary: "cpx41", // 8 vCPU (dedicated AMD), 16 GB RAM - ~€29.49/mo
			Fallbacks: []string{
				"cpx51", // 16 vCPU (dedicated AMD), 32 GB RAM - ~€57.49/mo (more powerful)
				"cx52",  // 16 vCPU (shared AMD), 32 GB RAM - ~€24.29/mo (cheaper but shared)
			},
			Architecture: "x86",
		},
	}

	return mappings[profile]
}

// SelectBestServerType selects the best available server type for a profile
// considering location availability
//
// All server types from GetHetznerServerType are x86-only (opinionated for Ubuntu compatibility)
func (p *Provider) SelectBestServerType(ctx context.Context, profile provider.MachineProfile, preferredLocations []string) (string, []string, error) {
	mapping := GetHetznerServerType(profile)
	
	// Try primary first, then fallbacks in order
	allOptions := append([]string{mapping.Primary}, mapping.Fallbacks...)
	
	for _, serverType := range allOptions {
		// Get locations where this server type is available
		availableLocations, err := p.GetAvailableLocations(ctx, serverType)
		if err != nil {
			// Skip this server type if we can't get its locations
			continue
		}
		
		if len(availableLocations) == 0 {
			continue
		}
		
		// If we have preferred locations, filter to those
		if len(preferredLocations) > 0 {
			matchingLocations := intersectLocations(availableLocations, preferredLocations)
			if len(matchingLocations) > 0 {
				return serverType, matchingLocations, nil
			}
		} else {
			// No preferred locations, use all available
			return serverType, availableLocations, nil
		}
	}
	
	return "", nil, fmt.Errorf("no suitable server type found for profile %s", profile)
}

// GetDefaultLocations returns a recommended set of locations for Hetzner
// These are prioritized by reliability and geographic distribution
func GetDefaultLocations() []string {
	return []string{
		"fsn1", // Falkenstein, Germany - Most established DC
		"nbg1", // Nuremberg, Germany - Second German DC
		"hel1", // Helsinki, Finland - Northern Europe
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

// intersectLocations returns locations that exist in both lists
func intersectLocations(a, b []string) []string {
	set := make(map[string]bool)
	for _, item := range b {
		set[item] = true
	}
	
	var result []string
	for _, item := range a {
		if set[item] {
			result = append(result, item)
		}
	}
	return result
}

// GetEstimatedCost returns the estimated monthly cost for a server type
func GetEstimatedCost(serverType string) float64 {
	// Approximate monthly costs in EUR (as of 2024)
	costs := map[string]float64{
		"cx22":  3.29,
		"cx21":  3.29,
		"cx32":  6.29,
		"cx42":  12.29,
		"cx52":  24.29,
		"cpx11": 4.49,
		"cpx21": 8.49,
		"cpx31": 15.49,
		"cpx41": 29.49,
		"cpx51": 57.49,
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
