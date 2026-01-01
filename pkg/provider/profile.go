package provider

// MachineProfile defines abstract machine specifications
// that can be mapped to provider-specific machine types
type MachineProfile string

const (
	// ProfileSmall is suitable for edge nodes and small workloads
	// Typical specs: 2 vCPU, 4GB RAM
	ProfileSmall MachineProfile = "small"
	
	// ProfileMedium is suitable for compute nodes with moderate workloads
	// Typical specs: 3-4 vCPU, 8GB RAM
	ProfileMedium MachineProfile = "medium"
	
	// ProfileLarge is suitable for high-performance workloads
	// Typical specs: 8+ vCPU, 16GB+ RAM
	ProfileLarge MachineProfile = "large"
)

// GetProfileForSize returns the appropriate machine profile for a forest size
func GetProfileForSize(size string) MachineProfile {
	switch size {
	case "wood":
		// Single machine - use small profile for cost efficiency
		return ProfileSmall
	case "forest":
		// 3-machine cluster - use small profile (edge nodes)
		return ProfileSmall
	case "jungle":
		// 5-machine cluster - use small profile (edge nodes)
		return ProfileSmall
	default:
		return ProfileSmall
	}
}

// MachineProfileSpec contains the abstract specifications for a machine profile
type MachineProfileSpec struct {
	Profile     MachineProfile
	Description string
	MinCPU      int     // Minimum vCPUs
	MinRAM      int     // Minimum RAM in GB
	TargetCost  float64 // Target monthly cost in EUR
}

// GetProfileSpec returns the specification for a given profile
func GetProfileSpec(profile MachineProfile) MachineProfileSpec {
	specs := map[MachineProfile]MachineProfileSpec{
		ProfileSmall: {
			Profile:     ProfileSmall,
			Description: "Small instance for edge nodes and lightweight workloads",
			MinCPU:      2,
			MinRAM:      4,
			TargetCost:  3.50, // ~€3-4/month
		},
		ProfileMedium: {
			Profile:     ProfileMedium,
			Description: "Medium instance for compute workloads",
			MinCPU:      3,
			MinRAM:      8,
			TargetCost:  8.00, // ~€8-10/month
		},
		ProfileLarge: {
			Profile:     ProfileLarge,
			Description: "Large instance for high-performance workloads",
			MinCPU:      8,
			MinRAM:      16,
			TargetCost:  30.00, // ~€30-35/month
		},
	}
	
	return specs[profile]
}
