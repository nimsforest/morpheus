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
// Returns multiple options in order of preference
func GetHetznerServerType(profile provider.MachineProfile) MachineTypeMapping {
	mappings := map[provider.MachineProfile]MachineTypeMapping{
		provider.ProfileSmall: {
			Primary: "cx22",
			Fallbacks: []string{
				"cax11", // ARM alternative (similar price, same RAM)
				"cpx11", // Dedicated vCPU (slightly more expensive but better performance)
				"cx21",  // Older generation
			},
			Architecture: "x86",
		},
		provider.ProfileMedium: {
			Primary: "cpx21",
			Fallbacks: []string{
				"cax21", // ARM alternative
				"cx32",  // Shared vCPU (cheaper but less consistent)
			},
			Architecture: "x86",
		},
		provider.ProfileLarge: {
			Primary: "cpx41",
			Fallbacks: []string{
				"cax41", // ARM alternative
				"cx52",  // Shared vCPU (cheaper but less consistent)
			},
			Architecture: "x86",
		},
	}

	return mappings[profile]
}

// SelectBestServerType selects the best available server type for a profile
// considering location availability
func (p *Provider) SelectBestServerType(ctx context.Context, profile provider.MachineProfile, preferredLocations []string) (string, []string, error) {
	mapping := GetHetznerServerType(profile)
	
	// Try primary first
	allOptions := append([]string{mapping.Primary}, mapping.Fallbacks...)
	
	// Filter out ARM types for now (ubuntu-24.04 doesn't support ARM on Hetzner)
	var x86Options []string
	for _, serverType := range allOptions {
		// Skip ARM-based server types (cax series)
		if len(serverType) >= 3 && serverType[:3] == "cax" {
			continue
		}
		x86Options = append(x86Options, serverType)
	}
	
	for _, serverType := range x86Options {
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
