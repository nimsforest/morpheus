package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nimsforest/morpheus/pkg/cloudinit"
	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/provider"
	"github.com/nimsforest/morpheus/pkg/provider/hetzner"
	"github.com/nimsforest/morpheus/pkg/provider/local"
	"github.com/nimsforest/morpheus/pkg/updater"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "plant":
		handlePlant()
	case "list":
		handleList()
	case "status":
		handleStatus()
	case "teardown":
		handleTeardown()
	case "version":
		fmt.Printf("morpheus version %s\n", version)
	case "update":
		handleUpdate()
	case "check-update":
		handleCheckUpdate()
	case "check-ipv6":
		handleCheckIPv6()
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handlePlant() {
	// Parse arguments with smart defaults
	var deploymentType, size string
	
	// Check if we have enough arguments
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "❌ Missing arguments")
		fmt.Fprintln(os.Stderr, "")
		if isTermux() {
			fmt.Fprintln(os.Stderr, "Usage: morpheus plant <size>")
			fmt.Fprintln(os.Stderr, "       morpheus plant cloud <size>  (explicit)")
		} else {
			fmt.Fprintln(os.Stderr, "Usage: morpheus plant <cloud|local> <size>")
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Sizes:")
		fmt.Fprintln(os.Stderr, "  small  - 1 machine  (~5-7 min)   💰 ~€3-4/month")
		fmt.Fprintln(os.Stderr, "  medium - 3 machines (~15-20 min) 💰 ~€9-12/month")
		fmt.Fprintln(os.Stderr, "  large  - 5 machines (~25-35 min) 💰 ~€15-20/month")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		if isTermux() {
			fmt.Fprintln(os.Stderr, "  morpheus plant small        # Quick! Create 1 machine")
			fmt.Fprintln(os.Stderr, "  morpheus plant medium       # Create 3-machine cluster")
		} else {
			fmt.Fprintln(os.Stderr, "  morpheus plant cloud small  # Create 1 machine on Hetzner Cloud")
			fmt.Fprintln(os.Stderr, "  morpheus plant local small  # Create 1 machine locally (Docker)")
			fmt.Fprintln(os.Stderr, "  morpheus plant cloud medium # Create 3-machine cluster")
		}
		os.Exit(1)
	}

	// Smart argument parsing: support both 2 and 3 argument forms
	if len(os.Args) == 3 {
		// Two arguments: could be "plant small" or "plant cloud small"
		arg := os.Args[2]
		if isValidSize(arg) {
			// On Termux, default to cloud mode (Docker doesn't work on Android)
			// On Desktop, require explicit mode to prevent accidental cloud deployments
			if isTermux() {
				deploymentType = "cloud"
				size = arg
				fmt.Println("\n💡 Using cloud mode (default on Termux)")
			} else {
				// Desktop: require explicit mode to prevent billing surprises
				fmt.Fprintf(os.Stderr, "\n❌ Please specify deployment mode\n\n")
				fmt.Fprintf(os.Stderr, "Usage: morpheus plant <cloud|local> %s\n\n", arg)
				fmt.Fprintf(os.Stderr, "Options:\n")
				fmt.Fprintf(os.Stderr, "  cloud - Deploy to Hetzner Cloud (requires API token, incurs charges)\n")
				fmt.Fprintf(os.Stderr, "  local - Deploy locally with Docker (free, requires Docker running)\n\n")
				fmt.Fprintf(os.Stderr, "Examples:\n")
				fmt.Fprintf(os.Stderr, "  morpheus plant cloud %s   # Create on Hetzner Cloud\n", arg)
				fmt.Fprintf(os.Stderr, "  morpheus plant local %s   # Create locally with Docker\n", arg)
				os.Exit(1)
			}
		} else if arg == "cloud" || arg == "local" {
			// It's a deployment type without size
			fmt.Fprintf(os.Stderr, "❌ Missing size argument\n\n")
			fmt.Fprintf(os.Stderr, "Usage: morpheus plant %s <size>\n", arg)
			fmt.Fprintln(os.Stderr, "Sizes: small, medium, large")
			fmt.Fprintf(os.Stderr, "\nDid you mean: morpheus plant %s small\n", arg)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "❌ Unknown argument: %s\n\n", arg)
			fmt.Fprintln(os.Stderr, "Valid sizes: small, medium, large")
			fmt.Fprintln(os.Stderr, "Valid deployment types: cloud, local")
			fmt.Fprintln(os.Stderr, "\nExamples:")
			fmt.Fprintln(os.Stderr, "  morpheus plant small")
			fmt.Fprintln(os.Stderr, "  morpheus plant cloud medium")
			os.Exit(1)
		}
	} else if len(os.Args) >= 4 {
		// Three arguments: "plant cloud small"
		deploymentType = os.Args[2]
		size = os.Args[3]
	}

	// Validate deployment type
	if deploymentType != "cloud" && deploymentType != "local" {
		fmt.Fprintf(os.Stderr, "❌ Invalid deployment type: '%s'\n\n", deploymentType)
		fmt.Fprintln(os.Stderr, "Valid options: cloud, local")
		fmt.Fprintf(os.Stderr, "\nDid you mean: morpheus plant cloud %s\n", size)
		os.Exit(1)
	}

	// Validate size
	if !isValidSize(size) {
		fmt.Fprintf(os.Stderr, "❌ Invalid size: '%s'\n\n", size)
		fmt.Fprintln(os.Stderr, "Valid sizes:")
		fmt.Fprintln(os.Stderr, "  small  - 1 machine  (quick start, ~€3-4/mo)")
		fmt.Fprintln(os.Stderr, "  medium - 3 machines (small cluster, ~€9-12/mo)")
		fmt.Fprintln(os.Stderr, "  large  - 5 machines (large cluster, ~€15-20/mo)")
		fmt.Fprintln(os.Stderr, "")
		
		// Suggest closest match
		suggestion := suggestSize(size)
		if suggestion != "" {
			fmt.Fprintf(os.Stderr, "💡 Did you mean: morpheus plant %s\n", suggestion)
		}
		os.Exit(1)
	}

	// For local deployments, we don't need a full config file
	var cfg *config.Config
	var err error

	if deploymentType == "local" {
		cfg = getLocalConfig()
	} else {
		cfg, err = loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
			os.Exit(1)
		}

		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid config: %s\n", err)
			os.Exit(1)
		}
	}

	// Create provider based on deployment type
	var prov provider.Provider
	var providerName string

	if deploymentType == "local" {
		prov, err = local.NewProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create local provider: %s\n", err)
			fmt.Fprintln(os.Stderr, "\nMake sure Docker is installed and running:")
			fmt.Fprintln(os.Stderr, "  docker info")
			// Check if we're on Termux and provide helpful message
			if isTermux() {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "⚠️  You appear to be running on Termux (Android).")
				fmt.Fprintln(os.Stderr, "   Docker does NOT work on Termux due to Android kernel limitations.")
				fmt.Fprintln(os.Stderr, "   Please use cloud mode instead:")
				fmt.Fprintln(os.Stderr, "     morpheus plant cloud wood")
			}
			os.Exit(1)
		}
		providerName = "local (Docker)"
	} else {
		switch cfg.Infrastructure.Provider {
		case "hetzner":
			prov, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
				os.Exit(1)
			}
			providerName = "hetzner"
		default:
			fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", cfg.Infrastructure.Provider)
			os.Exit(1)
		}
	}

	// Create registry
	registryPath := getRegistryPath()
	registry, err := forest.NewRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create registry: %s\n", err)
		os.Exit(1)
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(prov, registry, cfg)

	// Generate forest ID
	forestID := fmt.Sprintf("forest-%d", time.Now().Unix())

	// Create context early for provider operations
	ctx := context.Background()

	// Determine machine profile, server type, and location
	var location, serverType, image string
	if deploymentType == "local" {
		location = "local"
		serverType = "local"
		image = "ubuntu:24.04"
	} else {
		// Use machine profile system for cloud deployments
		profile := provider.GetProfileForSize(size)
		
		// For Hetzner, select the best server type and locations
		if hetznerProv, ok := prov.(*hetzner.Provider); ok {
			// Get default locations if not configured
			preferredLocations := cfg.Infrastructure.Locations
			if len(preferredLocations) == 0 {
				preferredLocations = hetzner.GetDefaultLocations()
			}
			
			// Select best server type and available locations
			selectedType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, profile, preferredLocations)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n❌ Failed to select server type: %s\n", err)
				os.Exit(1)
			}
			
			serverType = selectedType
			location = availableLocations[0] // Use first available location
			image = "ubuntu-24.04" // Default to Ubuntu 24.04
			
			// Update available locations for fallback
			cfg.Infrastructure.Locations = availableLocations
		} else {
			// Non-Hetzner provider (shouldn't happen, but handle gracefully)
			fmt.Fprintln(os.Stderr, "Unknown cloud provider")
			os.Exit(1)
		}
	}

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   forestID,
		Size:       size,
		Location:   location,
		Role:       cloudinit.RoleEdge, // Default role
		ServerType: serverType,
		Image:      image,
	}

	// Display friendly provisioning header
	fmt.Printf("\n🌲 Planting your %s...\n", size)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	// Show what's being created
	nodeCount := getNodeCount(size)
	var timeEstimate string
	switch size {
	case "small":
		timeEstimate = "5-7 minutes"
	case "medium":
		timeEstimate = "15-20 minutes"
	case "large":
		timeEstimate = "25-35 minutes"
	}
	
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Forest ID:  %s\n", forestID)
	fmt.Printf("   Size:       %s (%d machine%s)\n", size, nodeCount, plural(nodeCount))
	if deploymentType == "cloud" {
		fmt.Printf("   Machine:    %s (with automatic fallback if unavailable)\n", serverType)
		fmt.Printf("   Location:   %s (with automatic fallback if unavailable)\n", hetzner.GetLocationDescription(location))
	} else {
		fmt.Printf("   Location:   %s\n", location)
	}
	fmt.Printf("   Provider:   %s\n", providerName)
	fmt.Printf("   Time:       ~%s\n\n", timeEstimate)
	
	if deploymentType == "cloud" {
		estimatedCost := hetzner.GetEstimatedCost(serverType) * float64(nodeCount)
		fmt.Printf("💰 Estimated cost: ~€%.2f/month\n", estimatedCost)
		fmt.Printf("   (IPv6-only, billed by minute, can teardown anytime)\n\n")
	}
	
	fmt.Println("🚀 Starting provisioning...")

	// For Hetzner cloud deployments, use the full fallback system that tries
	// alternative server types if the primary one fails in all locations.
	// This handles cases where Hetzner's API reports availability (via pricing data)
	// but actual provisioning fails with "unsupported location for server type".
	if deploymentType == "cloud" {
		if hetznerProv, ok := prov.(*hetzner.Provider); ok {
			profile := provider.GetProfileForSize(size)
			err = provisionWithFallback(ctx, provisioner, hetznerProv, req, profile)
		} else {
			// Non-Hetzner cloud (shouldn't happen currently)
			availableLocations := cfg.Infrastructure.Locations
			err = provisionWithLocationFallback(ctx, provisioner, req, availableLocations)
		}
	} else {
		// Local deployments - just try the single location
		err = provisioner.Provision(ctx, req)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Provisioning failed: %s\n", err)
		os.Exit(1)
	}

	// Success message with clear next steps
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ Success! Your %s is ready!\n", size)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	fmt.Printf("🎯 What's next?\n\n")
	
	if deploymentType == "cloud" {
		fmt.Printf("📊 Check your forest status:\n")
		fmt.Printf("   morpheus status %s\n\n", forestID)
		
		fmt.Printf("🌐 Your machines are ready for NATS deployment\n")
		fmt.Printf("   Infrastructure is configured and waiting\n\n")
		
		fmt.Printf("📋 View all your forests:\n")
		fmt.Printf("   morpheus list\n\n")
		
		fmt.Printf("🗑️  Clean up when done:\n")
		fmt.Printf("   morpheus teardown %s\n\n", forestID)
		
		fmt.Printf("💡 Tip: The infrastructure is ready. Deploy NATS with NimsForest\n")
		fmt.Printf("   or use the machines for your own applications.\n")
	} else {
		fmt.Printf("🐳 Your local Docker containers are running!\n\n")
		fmt.Printf("📊 Check status:\n")
		fmt.Printf("   morpheus status %s\n", forestID)
		fmt.Printf("   docker ps\n\n")
		
		fmt.Printf("🔍 Access a container:\n")
		fmt.Printf("   docker exec -it %s-node-1 bash\n\n", forestID)
		
		fmt.Printf("📋 View logs:\n")
		fmt.Printf("   docker logs %s-node-1\n\n", forestID)
		
		fmt.Printf("🗑️  Clean up when done:\n")
		fmt.Printf("   morpheus teardown %s\n", forestID)
	}
}

// provisionWithLocationFallback tries to provision a forest, automatically
// falling back to alternative locations if the primary location is unavailable
func provisionWithLocationFallback(ctx context.Context, provisioner *forest.Provisioner, req forest.ProvisionRequest, locations []string) error {
	var lastErr error
	var attemptedLocations []string

	// Try each configured location in order
	for _, location := range locations {
		attemptedLocations = append(attemptedLocations, location)
		req.Location = location

		err := provisioner.Provision(ctx, req)
		if err == nil {
			// Success!
			return nil
		}

		lastErr = err

		// Check if the error is a location availability error
		errStr := err.Error()
		if containsLocationError(errStr) {
			fmt.Printf("⚠️  Location %s is unavailable for %s, trying next location...\n\n", location, req.ServerType)
			continue
		}

		// If it's not a location error, don't try other locations
		break
	}

	// All locations failed or encountered a non-location error
	if containsLocationError(lastErr.Error()) && len(attemptedLocations) > 0 {
		return fmt.Errorf("all configured locations are unavailable (%s): %w\n\n"+
			"Hetzner may be experiencing capacity issues. Try again later or update your config with different locations:\n"+
			"  Available locations: ash (Ashburn, USA), fsn1 (Falkenstein, Germany), nbg1 (Nuremberg, Germany), \n"+
			"                       hel1 (Helsinki, Finland), hil (Hillsboro, USA), sin (Singapore)",
			joinLocations(attemptedLocations), lastErr)
	}

	return lastErr
}

// provisionWithFallback tries to provision a forest, automatically falling back
// to alternative server types and locations if the primary ones are unavailable.
// This handles cases where Hetzner's API reports a server type is available in a
// location (via pricing data) but actual provisioning fails.
//
// Priority order:
// 1. cx22 in Helsinki (hel1)
// 2. cpx11 in Helsinki, then Nuremberg (nbg1), then others
// 3. cx21 in Helsinki, then Nuremberg, then others
func provisionWithFallback(ctx context.Context, provisioner *forest.Provisioner, hetznerProv *hetzner.Provider, req forest.ProvisionRequest, profile provider.MachineProfile) error {
	// Get all server type options for this profile
	mapping := hetzner.GetHetznerServerType(profile)
	allServerTypes := append([]string{mapping.Primary}, mapping.Fallbacks...)

	// Preferred location order: Helsinki first, then Nuremberg, then others
	preferredLocations := hetzner.GetDefaultLocations()

	var lastErr error
	var attemptedCombos []string
	isFirstAttempt := true

	// Try each server type
	for serverTypeIdx, serverType := range allServerTypes {
		// Get available locations for this server type
		availableLocations, err := hetznerProv.GetAvailableLocations(ctx, serverType)
		if err != nil {
			fmt.Printf("   ⚠️  Could not check availability for %s: %v\n", serverType, err)
			continue
		}
		if len(availableLocations) == 0 {
			fmt.Printf("   ⚠️  Server type %s has no available locations\n", serverType)
			continue
		}

		// Reorder available locations to match preferred order
		orderedLocations := orderLocationsByPreference(availableLocations, preferredLocations)

		// Show info when switching to fallback server type
		if serverTypeIdx > 0 && len(attemptedCombos) > 0 {
			fmt.Printf("\n📦 Trying alternative server type: %s (~€%.2f/mo)\n", 
				serverType, hetzner.GetEstimatedCost(serverType))
		}

		// Try each location for this server type (in preferred order)
		for _, location := range orderedLocations {
			attemptedCombos = append(attemptedCombos, fmt.Sprintf("%s@%s", serverType, location))

			// Update request with current server type and location
			req.ServerType = serverType
			req.Location = location

			if !isFirstAttempt {
				fmt.Printf("   📍 Trying %s in %s...\n", serverType, hetzner.GetLocationDescription(location))
			}
			isFirstAttempt = false

			err := provisioner.Provision(ctx, req)
			if err == nil {
				// Success!
				return nil
			}

			lastErr = err

			// Check if the error is a location/server type availability error
			errStr := err.Error()
			if containsLocationError(errStr) {
				fmt.Printf("   ⚠️  %s not available in %s, trying next option...\n", serverType, location)
				continue
			}

			// If it's not a location error, this is a real error - stop trying
			return err
		}
	}

	// All combinations failed
	if lastErr != nil && containsLocationError(lastErr.Error()) {
		return fmt.Errorf("no server type/location combination available\n\n"+
			"Tried %d combinations across server types: %s, %s\n\n"+
			"This usually means Hetzner is experiencing high demand or capacity issues.\n"+
			"Please try again in a few minutes.\n"+
			"For status updates, check: https://status.hetzner.com/",
			len(attemptedCombos), mapping.Primary, joinLocations(mapping.Fallbacks))
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no server type available for profile")
}

// orderLocationsByPreference reorders available locations to match the preferred order.
// Locations in preferredOrder come first (in that order), followed by any remaining locations.
func orderLocationsByPreference(available, preferredOrder []string) []string {
	availableSet := make(map[string]bool)
	for _, loc := range available {
		availableSet[loc] = true
	}

	var result []string

	// First, add locations in preferred order (if available)
	for _, loc := range preferredOrder {
		if availableSet[loc] {
			result = append(result, loc)
			delete(availableSet, loc)
		}
	}

	// Then add any remaining available locations
	for _, loc := range available {
		if availableSet[loc] {
			result = append(result, loc)
		}
	}

	return result
}

// handleServerTypeLocationMismatch presents an interactive menu when server type
// is not available in any configured location
func handleServerTypeLocationMismatch(ctx context.Context, locationAware provider.LocationAwareProvider, serverType string, configuredLocations []string) (string, []string) {
	fmt.Fprintf(os.Stderr, "\n❌ Server type '%s' is not available in your configured locations:\n", serverType)
	fmt.Fprintf(os.Stderr, "   Configured: %s\n\n", joinLocations(configuredLocations))

	// Get available locations for current server type
	availableForType, _ := locationAware.GetAvailableLocations(ctx, serverType)

	// Define recommended server types with their descriptions
	recommendedTypes := []struct {
		name        string
		description string
	}{
		{"cx22", "2 vCPU (shared AMD), 4 GB RAM - ~€3.29/mo"},
		{"cpx11", "2 vCPU (dedicated AMD), 2 GB RAM - ~€4.49/mo"},
		{"cpx21", "3 vCPU (dedicated AMD), 4 GB RAM - ~€8.49/mo"},
		{"cax11", "2 vCPU (ARM), 4 GB RAM - ~€3.79/mo"},
	}

	fmt.Println("What would you like to do?")
	fmt.Println()

	// Option 1: Use a different location (if available for current server type)
	if len(availableForType) > 0 {
		fmt.Printf("  [1] Use a different location for '%s'\n", serverType)
		fmt.Printf("      Available: %s\n", joinLocations(availableForType))
	} else {
		fmt.Printf("  [1] (Not available - '%s' has no available locations)\n", serverType)
	}
	fmt.Println()

	// Option 2: Change server type
	fmt.Println("  [2] Change server type (recommended)")
	fmt.Println("      Suggested server types:")
	for _, st := range recommendedTypes {
		locs, _ := locationAware.GetAvailableLocations(ctx, st.name)
		if len(locs) > 0 {
			fmt.Printf("        • %s: %s\n", st.name, st.description)
			fmt.Printf("          Locations: %s\n", joinLocations(locs))
		}
	}
	fmt.Println()

	// Option 3: Exit
	fmt.Println("  [3] Exit and update config manually")
	fmt.Println()

	fmt.Print("Enter choice (1/2/3): ")
	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		if len(availableForType) == 0 {
			fmt.Println("\n❌ No locations available for this server type.")
			return "", nil
		}
		// Let user pick a location
		fmt.Printf("\nAvailable locations for '%s':\n", serverType)
		for i, loc := range availableForType {
			locDesc := hetzner.GetLocationDescription(loc)
			fmt.Printf("  [%d] %s - %s\n", i+1, loc, locDesc)
		}
		fmt.Print("\nEnter location number: ")
		var locChoice int
		fmt.Scanln(&locChoice)
		if locChoice < 1 || locChoice > len(availableForType) {
			fmt.Println("\n❌ Invalid choice.")
			return "", nil
		}
		selectedLoc := availableForType[locChoice-1]
		fmt.Printf("\n✓ Using location: %s\n\n", selectedLoc)
		return serverType, []string{selectedLoc}

	case "2":
		// Let user pick a server type
		var availableTypes []struct {
			name        string
			description string
			locations   []string
		}
		for _, st := range recommendedTypes {
			locs, _ := locationAware.GetAvailableLocations(ctx, st.name)
			if len(locs) > 0 {
				availableTypes = append(availableTypes, struct {
					name        string
					description string
					locations   []string
				}{st.name, st.description, locs})
			}
		}

		if len(availableTypes) == 0 {
			fmt.Println("\n❌ No recommended server types are available. Please check Hetzner status.")
			return "", nil
		}

		fmt.Println("\nAvailable server types:")
		for i, st := range availableTypes {
			fmt.Printf("  [%d] %s: %s\n", i+1, st.name, st.description)
		}
		fmt.Print("\nEnter server type number: ")
		var typeChoice int
		fmt.Scanln(&typeChoice)
		if typeChoice < 1 || typeChoice > len(availableTypes) {
			fmt.Println("\n❌ Invalid choice.")
			return "", nil
		}
		selected := availableTypes[typeChoice-1]

		// Filter locations to prefer configured ones
		var useLocations []string
		for _, configLoc := range configuredLocations {
			for _, availLoc := range selected.locations {
				if configLoc == availLoc {
					useLocations = append(useLocations, configLoc)
				}
			}
		}
		// If none of the configured locations work, use all available
		if len(useLocations) == 0 {
			useLocations = selected.locations
		}

		fmt.Printf("\n✓ Using server type: %s\n", selected.name)
		fmt.Printf("✓ Available locations: %s\n\n", joinLocations(useLocations))
		return selected.name, useLocations

	case "3":
		fmt.Println("\nTo fix this, update your config.yaml:")
		fmt.Println("  1. Change server_type to one that's available in your locations, or")
		fmt.Println("  2. Change locations to ones that support your server type")
		fmt.Println("\nConfig file locations:")
		fmt.Println("  - ./config.yaml")
		fmt.Println("  - ~/.morpheus/config.yaml")
		return "", nil

	default:
		fmt.Println("\n❌ Invalid choice.")
		return "", nil
	}
}


// containsLocationError checks if an error message indicates a location availability issue
func containsLocationError(errMsg string) bool {
	locationErrorPhrases := []string{
		"server location disabled",
		"resource_unavailable",
		"location not available",
		"location disabled",
		"datacenter not available",
		"unsupported location",
		"unsupported location for server type",
	}

	for _, phrase := range locationErrorPhrases {
		if contains(errMsg, phrase) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring performs a simple substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// joinLocations joins location names with commas
func joinLocations(locations []string) string {
	result := ""
	for i, loc := range locations {
		if i > 0 {
			result += ", "
		}
		result += loc
	}
	return result
}

// getLocalConfig returns a minimal config for local deployments
func getLocalConfig() *config.Config {
	return &config.Config{
		Infrastructure: config.InfrastructureConfig{
			Provider: "local",
			Defaults: &config.DefaultsConfig{
				ServerType: "local",
				Image:      "ubuntu:24.04",
			},
			Locations: []string{"local"},
		},
	}
}

func handleList() {
	registryPath := getRegistryPath()
	registry, err := forest.NewRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load registry: %s\n", err)
		os.Exit(1)
	}

	forests := registry.ListForests()

	if len(forests) == 0 {
		fmt.Println("🌲 No forests yet!")
		fmt.Println()
		if isTermux() {
			fmt.Println("Create your first forest:")
			fmt.Println("  morpheus plant small        # Quick start with 1 machine")
		} else {
			fmt.Println("Create your first forest:")
			fmt.Println("  morpheus plant cloud small  # 1 machine on cloud")
			fmt.Println("  morpheus plant local small  # 1 machine locally (Docker)")
		}
		return
	}

	fmt.Printf("🌲 Your Forests (%d)\n", len(forests))
	fmt.Println()
	fmt.Println("FOREST ID            SIZE    LOCATION  STATUS       CREATED")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, f := range forests {
		statusIcon := "✅"
		if f.Status == "provisioning" {
			statusIcon = "⏳"
		} else if f.Status != "active" {
			statusIcon = "⚠️ "
		}
		
		fmt.Printf("%-20s %-7s %-9s %s %-11s %s\n",
			f.ID,
			f.Size,
			f.Location,
			statusIcon,
			f.Status,
			f.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	
	fmt.Println()
	fmt.Println("💡 Tip: Use 'morpheus status <forest-id>' to see detailed information")
}

func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus status <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	registryPath := getRegistryPath()
	registry, err := forest.NewRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load registry: %s\n", err)
		os.Exit(1)
	}

	forestInfo, err := registry.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get forest: %s\n", err)
		os.Exit(1)
	}

	nodes, err := registry.GetNodes(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get nodes: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("🌲 Forest: %s\n", forestInfo.ID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	
	statusIcon := "✅"
	if forestInfo.Status == "provisioning" {
		statusIcon = "⏳"
	} else if forestInfo.Status != "active" {
		statusIcon = "⚠️ "
	}
	
	fmt.Printf("📊 Overview:\n")
	fmt.Printf("   Status:   %s %s\n", statusIcon, forestInfo.Status)
	fmt.Printf("   Size:     %s (%d machine%s)\n", forestInfo.Size, len(nodes), plural(len(nodes)))
	fmt.Printf("   Location: %s\n", forestInfo.Location)
	fmt.Printf("   Provider: %s\n", forestInfo.Provider)
	fmt.Printf("   Created:  %s\n", forestInfo.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(nodes) > 0 {
		fmt.Printf("\n🖥️  Machines (%d):\n", len(nodes))
		fmt.Println()
		fmt.Println("   ID        ROLE   IPV6 ADDRESS        LOCATION  STATUS")
		fmt.Println("   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, node := range nodes {
			nodeStatusIcon := "✅"
			if node.Status != "active" {
				nodeStatusIcon = "⏳"
			}
			fmt.Printf("   %-9s %-6s %-19s %-9s %s %s\n",
				node.ID,
				node.Role,
				truncateIP(node.IP, 19),
				node.Location,
				nodeStatusIcon,
				node.Status,
			)
		}
		
		fmt.Println()
		fmt.Printf("💡 SSH into machines:\n")
		for i, node := range nodes {
			if i < 2 { // Show first 2 examples
				fmt.Printf("   ssh root@[%s]\n", node.IP)
			}
		}
		if len(nodes) > 2 {
			fmt.Printf("   ... (%d more machine%s)\n", len(nodes)-2, plural(len(nodes)-2))
		}
	} else {
		fmt.Println("\n⏳ No machines registered yet (still provisioning)")
	}
	
	fmt.Println()
	fmt.Printf("🗑️  Teardown: morpheus teardown %s\n", forestInfo.ID)
}

func handleTeardown() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus teardown <forest-id>")
		os.Exit(1)
	}

	forestID := os.Args[2]

	// First, get the forest info to determine the provider
	registryPath := getRegistryPath()
	registry, err := forest.NewRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create registry: %s\n", err)
		os.Exit(1)
	}

	// Get forest info to determine provider type
	forestInfo, err := registry.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get forest info: %s\n", err)
		os.Exit(1)
	}

	// Create provider based on forest's provider
	var prov provider.Provider
	var cfg *config.Config

	if forestInfo.Provider == "local" {
		prov, err = local.NewProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create local provider: %s\n", err)
			os.Exit(1)
		}
		cfg = getLocalConfig()
	} else {
		cfg, err = loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
			os.Exit(1)
		}

		switch cfg.Infrastructure.Provider {
		case "hetzner":
			prov, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", cfg.Infrastructure.Provider)
			os.Exit(1)
		}
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(prov, registry, cfg)

	// Show what will be deleted
	nodes, _ := registry.GetNodes(forestID)
	
	fmt.Printf("\n⚠️  About to permanently delete:\n")
	fmt.Printf("   Forest: %s\n", forestID)
	fmt.Printf("   Size:   %s (%d machine%s)\n", forestInfo.Size, len(nodes), plural(len(nodes)))
	if len(nodes) > 0 {
		fmt.Printf("   Machines:\n")
		for _, node := range nodes {
			fmt.Printf("      • %s (%s)\n", node.ID, node.IP)
		}
	}
	fmt.Println()
	fmt.Printf("💰 This will stop billing for these resources\n")
	fmt.Println()
	fmt.Print("Type 'yes' to confirm deletion: ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("\n✅ Teardown cancelled - your forest is safe!")
		return
	}

	// Teardown
	fmt.Println()
	ctx := context.Background()
	if err := provisioner.Teardown(ctx, forestID); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Teardown failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ Forest %s deleted successfully!\n", forestID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💰 Resources have been removed and billing stopped")
	fmt.Println()
	fmt.Println("💡 View your remaining forests: morpheus list")
}

func loadConfig() (*config.Config, error) {
	// Try multiple config locations
	configPaths := []string{
		"./config.yaml",
		filepath.Join(os.Getenv("HOME"), ".morpheus", "config.yaml"),
		"/etc/morpheus/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return config.LoadConfig(path)
		}
	}

	return nil, fmt.Errorf("no config file found (tried: %v)", configPaths)
}

func getRegistryPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}

	registryDir := filepath.Join(homeDir, ".morpheus")
	os.MkdirAll(registryDir, 0755)

	return filepath.Join(registryDir, "registry.json")
}

func handleUpdate() {
	u := updater.NewUpdater(version)

	fmt.Println("🔍 Checking for updates...")
	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check for updates: %s\n", err)
		fmt.Fprintf(os.Stderr, "\nYou can manually download the latest release from:\n")
		fmt.Fprintf(os.Stderr, "  https://github.com/nimsforest/morpheus/releases/latest\n")
		os.Exit(1)
	}

	fmt.Printf("\nCurrent version: %s\n", info.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", info.LatestVersion)

	if !info.Available {
		fmt.Println("\n✅ You are already running the latest version!")
		return
	}

	fmt.Println("\n🎉 A new version is available!")
	fmt.Println("\nRelease notes:")
	fmt.Println("─────────────────────────────────────────────────")
	if info.ReleaseNotes != "" {
		fmt.Println(info.ReleaseNotes)
	} else {
		fmt.Println("No release notes available.")
	}
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Printf("\nView full release: %s\n", info.UpdateURL)

	// Ask for confirmation
	fmt.Print("\nDo you want to update now? (yes/no): ")
	var response string
	fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		fmt.Println("\nUpdate cancelled.")
		fmt.Printf("To update later, run: morpheus update\n")
		return
	}

	// Perform update
	fmt.Println()
	if err := u.PerformUpdate(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Update failed: %s\n", err)
		os.Exit(1)
	}
}

func handleCheckUpdate() {
	u := updater.NewUpdater(version)

	info, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check for updates: %s\n", err)
		os.Exit(1)
	}

	if info.Available {
		fmt.Printf("Update available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
		fmt.Printf("Run 'morpheus update' to install.\n")
		os.Exit(0)
	} else {
		fmt.Printf("Already up to date: %s\n", info.CurrentVersion)
		os.Exit(0)
	}
}

func handleCheckIPv6() {
	fmt.Println("🔍 Checking IPv6 connectivity...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv6Connectivity(ctx)

	if result.Available {
		fmt.Println("✅ IPv6 connectivity is available!")
		fmt.Printf("   Your IPv6 address: %s\n", result.Address)
		fmt.Println()
		fmt.Println("You can use Morpheus to provision IPv6-only infrastructure on Hetzner Cloud.")
		os.Exit(0)
	} else {
		fmt.Println("❌ IPv6 connectivity is NOT available")
		fmt.Println()
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
			fmt.Println()
		}
		fmt.Println("Morpheus requires IPv6 connectivity to provision infrastructure.")
		fmt.Println("Hetzner Cloud uses IPv6-only by default (IPv4 costs extra).")
		fmt.Println()
		fmt.Println("Options to get IPv6:")
		fmt.Println("  1. Enable IPv6 on your ISP/router")
		fmt.Println("  2. Use an IPv6 tunnel service (e.g., Hurricane Electric)")
		fmt.Println("  3. Use a VPS/server with IPv6 to run Morpheus")
		fmt.Println()
		fmt.Println("For more information, see:")
		fmt.Println("  https://github.com/nimsforest/morpheus/blob/main/docs/guides/IPV6_SETUP.md")
		os.Exit(1)
	}
}

// isTermux checks if we're running on Termux (Android)
func isTermux() bool {
	// Check for Termux-specific environment variable
	if os.Getenv("TERMUX_VERSION") != "" {
		return true
	}
	// Check for Termux prefix path
	if os.Getenv("PREFIX") == "/data/data/com.termux/files/usr" {
		return true
	}
	// Check if home directory is in Termux path
	home := os.Getenv("HOME")
	if home != "" && (home == "/data/data/com.termux/files/home" ||
		filepath.HasPrefix(home, "/data/data/com.termux")) {
		return true
	}
	return false
}

// plural returns "s" if count is not 1, empty string otherwise
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}


// truncateIP truncates an IP address to fit within maxLen characters
func truncateIP(ip string, maxLen int) string {
	if len(ip) <= maxLen {
		return ip
	}
	// For IPv6, show first part and last part with ellipsis
	if maxLen < 10 {
		return ip[:maxLen]
	}
	prefixLen := (maxLen - 3) / 2
	suffixLen := maxLen - 3 - prefixLen
	return ip[:prefixLen] + "..." + ip[len(ip)-suffixLen:]
}

// getNodeCount returns the number of nodes for a given forest size
func getNodeCount(size string) int {
	switch size {
	case "small":
		return 1
	case "medium":
		return 3
	case "large":
		return 5
	default:
		return 1
	}
}

// isValidSize checks if a size is valid
func isValidSize(size string) bool {
	validSizes := []string{"small", "medium", "large"}
	for _, valid := range validSizes {
		if size == valid {
			return true
		}
	}
	return false
}

// suggestSize suggests a size based on user input
func suggestSize(input string) string {
	if len(input) == 0 {
		return ""
	}
	
	firstChar := input[0]
	switch firstChar {
	case 's', 'S':
		return "small"
	case 'm', 'M':
		return "medium"
	case 'l', 'L':
		return "large"
	default:
		return ""
	}
}

func printHelp() {
	isOnTermux := isTermux()
	
	fmt.Println("🌲 Morpheus - Nims Forest Infrastructure Provisioning")
	fmt.Println()
	
	if isOnTermux {
		fmt.Println("Quick Start (Termux):")
		fmt.Println("  morpheus plant small        # Create 1 machine on Hetzner (~5-7 min)")
		fmt.Println("  morpheus plant medium       # Create 3 machines (~15-20 min)")
		fmt.Println()
	}
	
	fmt.Println("Usage:")
	fmt.Println("  morpheus <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	if isOnTermux {
		fmt.Println("  plant <size>                Plant a forest (cloud mode)")
		fmt.Println("                              Sizes:")
		fmt.Println("                                small  - 1 machine  (~5-7 min, €3-4/mo)")
		fmt.Println("                                medium - 3 machines (~15-20 min, €9-12/mo)")
		fmt.Println("                                large  - 5 machines (~25-35 min, €15-20/mo)")
	} else {
		fmt.Println("  plant <cloud|local> <size>  Provision a new forest")
		fmt.Println("                              Deployment types:")
		fmt.Println("                                cloud - Provision on Hetzner Cloud (requires API token)")
		fmt.Println("                                local - Provision locally using Docker (free)")
		fmt.Println("                              Sizes:")
		fmt.Println("                                small  - 1 machine  (~5-7 min)")
		fmt.Println("                                medium - 3 machines (~15-20 min)")
		fmt.Println("                                large  - 5 machines (~25-35 min)")
	}
	fmt.Println("  list                        List all forests")
	fmt.Println("  status <forest-id>          Show detailed forest status")
	fmt.Println("  teardown <forest-id>        Delete a forest and all its resources")
	fmt.Println("  version                     Show version information")
	fmt.Println("  update                      Check for updates and install if available")
	fmt.Println("  check-update                Check for updates without installing")
	fmt.Println("  check-ipv6                  Check if IPv6 connectivity is available")
	fmt.Println("  help                        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	if isOnTermux {
		fmt.Println("  morpheus plant small        # Quick! Create 1 machine")
		fmt.Println("  morpheus plant medium       # Create 3-machine cluster")
		fmt.Println("  morpheus list               # View all your forests")
		fmt.Println("  morpheus status forest-123  # Check forest details")
		fmt.Println("  morpheus teardown forest-123 # Clean up resources")
	} else {
		fmt.Println("  morpheus plant cloud small  # Create 1 machine on Hetzner Cloud")
		fmt.Println("  morpheus plant local small  # Create 1 machine locally (Docker)")
		fmt.Println("  morpheus plant cloud medium # Create 3-machine cluster")
		fmt.Println("  morpheus list               # View all forests")
		fmt.Println("  morpheus status forest-123  # Check forest details")
		fmt.Println("  morpheus teardown forest-123 # Clean up resources")
		fmt.Println("  morpheus update             # Update to latest version")
	}
	fmt.Println()
	
	if !isOnTermux {
		fmt.Println("Local Mode:")
		fmt.Println("  Local mode uses Docker to create containers on your machine.")
		fmt.Println("  No cloud account required - great for development!")
		fmt.Println("  Requirements: Docker must be installed and running.")
		fmt.Println()
	}
	
	fmt.Println("Configuration:")
	fmt.Println("  Morpheus looks for config.yaml in:")
	fmt.Println("    - ./config.yaml")
	fmt.Println("    - ~/.morpheus/config.yaml")
	fmt.Println("    - /etc/morpheus/config.yaml")
	if !isOnTermux {
		fmt.Println("  Note: Local mode doesn't require a config file.")
	}
	fmt.Println()
	fmt.Println("More info: https://github.com/nimsforest/morpheus")
}
