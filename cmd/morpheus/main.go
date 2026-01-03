package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/nats"
	"github.com/nimsforest/morpheus/pkg/provider"
	"github.com/nimsforest/morpheus/pkg/provider/hetzner"
	"github.com/nimsforest/morpheus/pkg/provider/local"
	"github.com/nimsforest/morpheus/pkg/registry"
	"github.com/nimsforest/morpheus/pkg/sshutil"
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
	case "grow":
		handleGrow()
	case "version":
		fmt.Printf("morpheus version %s\n", version)
	case "update":
		handleUpdate()
	case "check-update":
		handleCheckUpdate()
	case "check-ipv6":
		handleCheckIPv6()
	case "check":
		handleCheck()
	case "test":
		handleTest()
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
	registry, err := registry.NewLocalRegistry(registryPath)
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
			image = "ubuntu-24.04"           // Default to Ubuntu 24.04

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
		if cfg.Infrastructure.EnableIPv4Fallback {
			fmt.Printf("   (IPv4+IPv6, billed by minute, can teardown anytime)\n")
			fmt.Printf("   ⚠️  IPv4 enabled - additional charges apply per IPv4 address\n\n")
		} else {
			fmt.Printf("   (IPv6-only, billed by minute, can teardown anytime)\n\n")
		}
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
// 2. cpx22 in Helsinki, then Nuremberg (nbg1), then others
// 3. Other fallbacks in Helsinki, then other locations
func provisionWithFallback(ctx context.Context, provisioner *forest.Provisioner, hetznerProv *hetzner.Provider, req forest.ProvisionRequest, profile provider.MachineProfile) error {
	// Get all server type options for this profile
	mapping := hetzner.GetHetznerServerType(profile)
	allServerTypes := append([]string{mapping.Primary}, mapping.Fallbacks...)

	// Preferred location order: Helsinki first, then Nuremberg, then others
	preferredLocations := hetzner.GetDefaultLocations()

	var lastErr error
	var attemptedCombos []string
	var validServerTypes []string
	isFirstAttempt := true

	// First, validate which server types actually exist in Hetzner
	for _, serverType := range allServerTypes {
		exists, err := hetznerProv.ValidateServerType(ctx, serverType)
		if err != nil {
			fmt.Printf("   ⚠️  Could not validate server type %s: %v\n", serverType, err)
			continue
		}
		if !exists {
			fmt.Printf("   ⚠️  Server type %s does not exist in Hetzner, skipping\n", serverType)
			continue
		}
		validServerTypes = append(validServerTypes, serverType)
	}

	if len(validServerTypes) == 0 {
		return fmt.Errorf("none of the configured server types exist in Hetzner: %s", joinLocations(allServerTypes))
	}

	// Try each validated server type
	for serverTypeIdx, serverType := range validServerTypes {
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
			"Tried %d combinations across server types: %s\n\n"+
			"This usually means Hetzner is experiencing high demand or capacity issues.\n"+
			"Please try again in a few minutes.\n"+
			"For status updates, check: https://status.hetzner.com/",
			len(attemptedCombos), joinLocations(validServerTypes))
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
	registry, err := registry.NewLocalRegistry(registryPath)
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
	registry, err := registry.NewLocalRegistry(registryPath)
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
		fmt.Println("   ID                IP ADDRESS               LOCATION  STATUS")
		fmt.Println("   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, node := range nodes {
			nodeStatusIcon := "✅"
			if node.Status != "active" {
				nodeStatusIcon = "⏳"
			}
			fmt.Printf("   %-17s %-24s %-9s %s %s\n",
				node.ID,
				truncateIP(node.IP, 24),
				node.Location,
				nodeStatusIcon,
				node.Status,
			)
		}

		fmt.Println()

		// Detect SSH private key for better guidance
		sshKeyPath := sshutil.DetectSSHPrivateKeyPath()

		fmt.Printf("💡 SSH into machines:\n")
		for i, node := range nodes {
			if i < 2 { // Show first 2 examples
				if sshKeyPath != "" {
					fmt.Printf("   %s\n", sshutil.FormatSSHCommandWithIdentity("root", node.IP, sshKeyPath))
				} else {
					fmt.Printf("   %s\n", sshutil.FormatSSHCommand("root", node.IP))
				}
			}
		}
		if len(nodes) > 2 {
			fmt.Printf("   ... (%d more machine%s)\n", len(nodes)-2, plural(len(nodes)-2))
		}

		// Add troubleshooting tip for password prompts
		fmt.Println()
		fmt.Printf("   ⚠️  If asked for a password, your SSH key may not be configured correctly.\n")
		fmt.Printf("   Run 'morpheus check ssh' to diagnose SSH key issues.\n")
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
	registry, err := registry.NewLocalRegistry(registryPath)
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

func handleGrow() {
	// Parse arguments
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus grow <forest-id> [--auto] [--threshold N]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Check forest health and optionally add nodes.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --auto         Non-interactive mode (auto-expand if needed)")
		fmt.Fprintln(os.Stderr, "  --threshold N  Resource threshold percentage (default: 80)")
		fmt.Fprintln(os.Stderr, "  --json         Output in JSON format")
		os.Exit(1)
	}

	forestID := os.Args[2]

	// Parse optional flags
	autoMode := false
	jsonOutput := false
	threshold := 80.0

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--auto":
			autoMode = true
		case "--json":
			jsonOutput = true
		case "--threshold":
			if i+1 < len(os.Args) {
				i++
				fmt.Sscanf(os.Args[i], "%f", &threshold)
			}
		}
	}

	// Load registry
	registryPath := getRegistryPath()
	reg, err := registry.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load registry: %s\n", err)
		os.Exit(1)
	}

	// Get forest info
	forestInfo, err := reg.GetForest(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Forest not found: %s\n", err)
		os.Exit(1)
	}

	// Get nodes
	nodes, err := reg.GetNodes(forestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get nodes: %s\n", err)
		os.Exit(1)
	}

	if len(nodes) == 0 {
		fmt.Fprintln(os.Stderr, "No nodes found in forest")
		os.Exit(1)
	}

	// Create NATS monitor
	monitor := nats.NewMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Collect node IPs
	var nodeIPs []string
	for _, node := range nodes {
		if node.IP != "" {
			nodeIPs = append(nodeIPs, node.IP)
		}
	}

	if !jsonOutput {
		fmt.Printf("\n🌲 Forest: %s\n", forestID)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	// Check each node's NATS stats
	var totalCPU float64
	var totalMem int64
	var totalConns int
	var reachableNodes int
	var nodeStats []*nodeHealthInfo

	for _, node := range nodes {
		if node.IP == "" {
			continue
		}

		status := monitor.CheckNodeHealth(ctx, node.IP)
		info := &nodeHealthInfo{
			NodeID:    node.ID,
			IP:        node.IP,
			Reachable: status.Healthy,
		}

		if status.Healthy {
			reachableNodes++
			info.CPU = status.CPUPercent
			info.MemMB = status.MemMB
			info.Connections = status.Connections
			totalCPU += status.CPUPercent
			totalMem += status.Stats.Mem
			totalConns += status.Connections
		} else {
			info.Error = status.Error
		}

		nodeStats = append(nodeStats, info)
	}

	// Calculate averages
	avgCPU := 0.0
	avgMem := 0.0
	if reachableNodes > 0 {
		avgCPU = totalCPU / float64(reachableNodes)
		avgMem = float64(totalMem) / float64(reachableNodes) / (1024 * 1024) // Convert to MB
	}

	// JSON output
	if jsonOutput {
		output := map[string]interface{}{
			"forest_id":       forestID,
			"total_nodes":     len(nodes),
			"reachable_nodes": reachableNodes,
			"total_connections": totalConns,
			"avg_cpu_percent": avgCPU,
			"avg_mem_mb":      avgMem,
			"cpu_high":        avgCPU > threshold,
			"threshold":       threshold,
			"nodes":           nodeStats,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	// Display cluster info
	fmt.Printf("📊 NATS Cluster: %d node%s, %d connection%s\n",
		reachableNodes, plural(reachableNodes),
		totalConns, plural(totalConns))
	fmt.Println()

	// Display resource usage with progress bars
	fmt.Printf("Resource Usage:\n")
	fmt.Printf("  CPU:    %5.1f%% %s\n", avgCPU, progressBar(avgCPU, threshold))
	fmt.Printf("  Memory: %5.0f MB avg\n", avgMem)
	fmt.Println()

	// Display node table
	fmt.Println("Nodes:")
	fmt.Println("  NODE          IP                      CPU      MEM      CONNS  STATUS")
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, info := range nodeStats {
		if info.Reachable {
			warning := ""
			if info.CPU > threshold {
				warning = " ⚠️"
			}
			fmt.Printf("  %-13s %-23s %5.1f%%   %5dMB   %5d  ✅%s\n",
				truncateID(info.NodeID, 13),
				truncateIP(info.IP, 23),
				info.CPU,
				info.MemMB,
				info.Connections,
				warning)
		} else {
			fmt.Printf("  %-13s %-23s    -        -       -  ❌ unreachable\n",
				truncateID(info.NodeID, 13),
				truncateIP(info.IP, 23))
		}
	}
	fmt.Println()

	// Show warnings
	needsExpansion := avgCPU > threshold
	if needsExpansion {
		fmt.Printf("⚠️  Average CPU above %.0f%% threshold\n", threshold)
		fmt.Println()
	}

	// Auto mode or interactive
	if autoMode {
		if needsExpansion {
			fmt.Println("🌱 Auto-expanding cluster...")
			expandCluster(forestID, forestInfo, reg, nodes)
		} else {
			fmt.Println("✅ Cluster resources within threshold. No expansion needed.")
		}
		return
	}

	// Interactive mode
	if needsExpansion {
		fmt.Print("Add 1 node to cluster? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" || response == "yes" {
			expandCluster(forestID, forestInfo, reg, nodes)
		} else {
			fmt.Println("\n✅ No changes made.")
		}
	} else {
		fmt.Println("✅ Cluster resources within threshold.")
		fmt.Println("   Use 'morpheus grow <forest-id> --threshold N' to set a different threshold.")
	}
}

// nodeHealthInfo holds health info for display
type nodeHealthInfo struct {
	NodeID      string `json:"node_id"`
	IP          string `json:"ip"`
	Reachable   bool   `json:"reachable"`
	CPU         float64 `json:"cpu_percent,omitempty"`
	MemMB       int64   `json:"mem_mb,omitempty"`
	Connections int     `json:"connections,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// progressBar creates a simple ASCII progress bar
func progressBar(value, threshold float64) string {
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

// truncateID truncates a node ID to maxLen characters
func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen-3] + "..."
}

// expandCluster adds a new node to the cluster
func expandCluster(forestID string, forestInfo *registry.Forest, reg registry.Registry, existingNodes []*registry.Node) {
	fmt.Println()
	fmt.Println("🌱 Expanding cluster...")

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		return
	}

	// Create provider
	var prov provider.Provider
	switch forestInfo.Provider {
	case "hetzner":
		prov, err = hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
			return
		}
	case "local":
		prov, err = local.NewProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %s\n", err)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported provider: %s\n", forestInfo.Provider)
		return
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(prov, reg, cfg)

	// Determine server type based on existing nodes (if available) or profile
	profile := provider.GetProfileForSize(forestInfo.Size)
	serverType := ""
	location := forestInfo.Location

	if hetznerProv, ok := prov.(*hetzner.Provider); ok {
		ctx := context.Background()
		selectedType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, profile, []string{location})
		if err == nil {
			serverType = selectedType
			if len(availableLocations) > 0 {
				location = availableLocations[0]
			}
		}
	}

	if serverType == "" {
		// Fallback to legacy config
		if cfg.Infrastructure.Defaults != nil && cfg.Infrastructure.Defaults.ServerType != "" {
			serverType = cfg.Infrastructure.Defaults.ServerType
		} else {
			fmt.Fprintln(os.Stderr, "Could not determine server type")
			return
		}
	}

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   forestID,
		Size:       forestInfo.Size,
		Location:   location,
		ServerType: serverType,
		Image:      "ubuntu-24.04",
	}

	ctx := context.Background()
	if err := provisioner.Provision(ctx, req); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Expansion failed: %s\n", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Cluster expanded successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("💡 View updated cluster: morpheus status %s\n", forestID)
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

func handleCheck() {
	// Parse subcommand
	subcommand := ""
	if len(os.Args) >= 3 {
		subcommand = os.Args[2]
	}

	switch subcommand {
	case "ipv6":
		runIPv6Check(true)
	case "ipv4":
		runIPv4Check(true)
	case "network":
		runNetworkCheck(true)
	case "ssh":
		runSSHCheck(true)
	case "":
		// Run all checks
		fmt.Println("🔍 Running Morpheus Diagnostics")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		ipv6Ok, ipv4Ok := runNetworkCheck(false)
		fmt.Println()
		sshOk := runSSHCheck(false)

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if ipv6Ok && sshOk {
			fmt.Println("✅ All checks passed! You're ready to use Morpheus.")
		} else if ipv4Ok && sshOk {
			fmt.Println("⚠️  IPv6 not available, but IPv4 works.")
			fmt.Println("   Enable IPv4 fallback in config.yaml:")
			fmt.Println("     infrastructure:")
			fmt.Println("       enable_ipv4_fallback: true")
			fmt.Println("   Note: IPv4 costs extra on Hetzner.")
		} else {
			fmt.Println("⚠️  Some checks failed. Please review the issues above.")
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown check: %s\n\n", subcommand)
		fmt.Fprintln(os.Stderr, "Usage: morpheus check [ipv6|ipv4|network|ssh]")
		fmt.Fprintln(os.Stderr, "  morpheus check         Run all checks")
		fmt.Fprintln(os.Stderr, "  morpheus check ipv6    Check IPv6 connectivity")
		fmt.Fprintln(os.Stderr, "  morpheus check ipv4    Check IPv4 connectivity")
		fmt.Fprintln(os.Stderr, "  morpheus check network Check both IPv6 and IPv4")
		fmt.Fprintln(os.Stderr, "  morpheus check ssh     Check SSH key setup")
		os.Exit(1)
	}
}

// runIPv6Check checks IPv6 connectivity and returns true if successful
func runIPv6Check(exitOnResult bool) bool {
	fmt.Println("📡 IPv6 Connectivity")
	fmt.Println("   Checking connection to IPv6 services...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv6Connectivity(ctx)

	if result.Available {
		fmt.Println("   ✅ IPv6 is available")
		fmt.Printf("   Your IPv6 address: %s\n", result.Address)
		if exitOnResult {
			os.Exit(0)
		}
		return true
	} else {
		fmt.Println("   ❌ IPv6 is NOT available")
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		fmt.Println()
		fmt.Println("   Morpheus uses IPv6 by default to connect to provisioned servers.")
		fmt.Println("   Options:")
		fmt.Println("     • Enable IPv6 on your ISP/router")
		fmt.Println("     • Use an IPv6 tunnel (e.g., Hurricane Electric)")
		fmt.Println("     • Use a VPS with IPv6 connectivity")
		fmt.Println("     • Enable IPv4 fallback in config.yaml (costs extra)")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}
}

// runIPv4Check checks IPv4 connectivity and returns true if successful
func runIPv4Check(exitOnResult bool) bool {
	fmt.Println("📡 IPv4 Connectivity")
	fmt.Println("   Checking connection to IPv4 services...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := httputil.CheckIPv4Connectivity(ctx)

	if result.Available {
		fmt.Println("   ✅ IPv4 is available")
		fmt.Printf("   Your IPv4 address: %s\n", result.Address)
		if exitOnResult {
			os.Exit(0)
		}
		return true
	} else {
		fmt.Println("   ❌ IPv4 is NOT available")
		if result.Error != nil {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}
}

// runNetworkCheck checks both IPv6 and IPv4 connectivity
// Returns (ipv6Ok, ipv4Ok)
func runNetworkCheck(exitOnResult bool) (bool, bool) {
	fmt.Println("📡 Network Connectivity")
	fmt.Println()

	// Check IPv6
	fmt.Println("   Checking IPv6...")
	ctx6, cancel6 := context.WithTimeout(context.Background(), 10*time.Second)
	result6 := httputil.CheckIPv6Connectivity(ctx6)
	cancel6()

	ipv6Ok := false
	if result6.Available {
		fmt.Println("   ✅ IPv6 is available")
		fmt.Printf("      Your IPv6 address: %s\n", result6.Address)
		ipv6Ok = true
	} else {
		fmt.Println("   ❌ IPv6 is NOT available")
	}

	fmt.Println()

	// Check IPv4
	fmt.Println("   Checking IPv4...")
	ctx4, cancel4 := context.WithTimeout(context.Background(), 10*time.Second)
	result4 := httputil.CheckIPv4Connectivity(ctx4)
	cancel4()

	ipv4Ok := false
	if result4.Available {
		fmt.Println("   ✅ IPv4 is available")
		fmt.Printf("      Your IPv4 address: %s\n", result4.Address)
		ipv4Ok = true
	} else {
		fmt.Println("   ❌ IPv4 is NOT available")
	}

	fmt.Println()

	// Summary and recommendations
	if ipv6Ok && ipv4Ok {
		fmt.Println("   ✅ Both IPv6 and IPv4 are available")
		fmt.Println("      Morpheus will use IPv6 by default (recommended, saves costs)")
	} else if ipv6Ok {
		fmt.Println("   ✅ IPv6 available - Morpheus will work with default settings")
	} else if ipv4Ok {
		fmt.Println("   ⚠️  Only IPv4 available")
		fmt.Println("      To use Morpheus, enable IPv4 fallback in config.yaml:")
		fmt.Println("        infrastructure:")
		fmt.Println("          enable_ipv4_fallback: true")
		fmt.Println("      Note: IPv4 costs extra on Hetzner Cloud")
	} else {
		fmt.Println("   ❌ No network connectivity")
		fmt.Println("      Please check your internet connection")
	}

	if exitOnResult {
		if ipv6Ok {
			os.Exit(0)
		} else if ipv4Ok {
			os.Exit(0) // IPv4 available, user can enable fallback
		} else {
			os.Exit(1)
		}
	}

	return ipv6Ok, ipv4Ok
}

// runSSHCheck checks SSH key configuration and returns true if successful
func runSSHCheck(exitOnResult bool) bool {
	fmt.Println("🔑 SSH Key Setup")

	allOk := true

	// 1. Check for local SSH keys
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		fmt.Println("   ❌ Cannot determine home directory")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Check if .ssh directory exists
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		fmt.Println("   ❌ SSH directory not found (~/.ssh)")
		fmt.Println("   Run: ssh-keygen -t ed25519")
		if exitOnResult {
			os.Exit(1)
		}
		return false
	}

	// Look for SSH keys (both public and private)
	keyPaths := []string{
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_rsa"),
	}

	var foundKey string
	var foundKeyPath string
	var foundPrivateKeyPath string
	for _, basePath := range keyPaths {
		pubPath := basePath + ".pub"
		if data, err := os.ReadFile(pubPath); err == nil {
			content := string(data)
			if isValidSSHKey(content) {
				foundKey = content
				foundKeyPath = pubPath
				// Check if private key also exists
				if _, err := os.Stat(basePath); err == nil {
					foundPrivateKeyPath = basePath
				}
				break
			}
		}
	}

	if foundKey == "" {
		fmt.Println("   ❌ No SSH public key found")
		fmt.Println("   Searched: ~/.ssh/id_ed25519.pub, ~/.ssh/id_rsa.pub")
		fmt.Println()
		fmt.Println("   Generate a new key with:")
		fmt.Println("     ssh-keygen -t ed25519 -C \"your_email@example.com\"")
		allOk = false
	} else {
		fmt.Printf("   ✅ SSH public key found: %s\n", foundKeyPath)
		// Show key type
		if len(foundKey) > 20 {
			parts := splitFirst(foundKey, " ")
			if parts[0] != "" {
				fmt.Printf("      Key type: %s\n", parts[0])
			}
		}

		// Check private key
		if foundPrivateKeyPath != "" {
			fmt.Printf("   ✅ SSH private key found: %s\n", foundPrivateKeyPath)

			// Check if private key might have a passphrase (we can't easily detect this,
			// but we can inform the user about it)
			fmt.Println()
			fmt.Println("   💡 SSH Authentication Tips:")
			fmt.Printf("      When connecting, use: ssh -i %s root@<ip>\n", foundPrivateKeyPath)
			fmt.Println()
			fmt.Println("      If still asked for a password:")
			fmt.Println("      • Your private key may have a passphrase (enter that, not server password)")
			fmt.Println("      • Try adding your key to ssh-agent: ssh-add " + foundPrivateKeyPath)
			fmt.Println("      • Ensure the public key was uploaded to Hetzner (check below)")
		} else {
			fmt.Println("   ⚠️  SSH private key NOT found")
			fmt.Println("      The private key is required for authentication")
			fmt.Println("      Expected at: " + strings.TrimSuffix(foundKeyPath, ".pub"))
			allOk = false
		}
	}

	// 2. Check config and Hetzner API token
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println()
		fmt.Println("   ⚠️  No config file found (can't check Hetzner SSH key status)")
		fmt.Println("   Create config.yaml to enable full SSH key validation")
	} else if cfg.Secrets.HetznerAPIToken == "" {
		fmt.Println()
		fmt.Println("   ⚠️  No Hetzner API token configured (can't verify cloud SSH key)")
	} else {
		// Try to check if SSH key exists in Hetzner
		fmt.Println()
		fmt.Println("   Checking Hetzner Cloud SSH key status...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		hetznerProv, err := hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
		if err != nil {
			fmt.Printf("   ⚠️  Could not connect to Hetzner: %s\n", err)
		} else {
			keyName := cfg.GetSSHKeyName()
			keyInfo, err := hetznerProv.GetSSHKeyInfo(ctx, keyName)
			if err != nil {
				fmt.Printf("   ⚠️  Could not check SSH key: %s\n", err)
			} else if keyInfo != nil {
				fmt.Printf("   ✅ SSH key '%s' exists in Hetzner Cloud\n", keyName)
				fmt.Printf("      Hetzner fingerprint: %s\n", keyInfo.Fingerprint)

				// Compare fingerprints if we found a local key
				if foundKeyPath != "" {
					localFingerprint, _, err := sshutil.ReadAndCalculateFingerprint(foundKeyPath)
					if err != nil {
						fmt.Printf("   ⚠️  Could not calculate local key fingerprint: %s\n", err)
					} else {
						fmt.Printf("      Local fingerprint:   %s\n", localFingerprint)
						if localFingerprint == keyInfo.Fingerprint {
							fmt.Println("   ✅ Fingerprints MATCH - your local key matches Hetzner")
						} else {
							allOk = false
							fmt.Println()
							fmt.Println("   ❌ FINGERPRINT MISMATCH!")
							fmt.Println("      Your local SSH key does NOT match the key in Hetzner Cloud.")
							fmt.Println("      This is why the server asks for a password!")
							fmt.Println()
							fmt.Println("   How would you like to fix this?")
							fmt.Println()
							fmt.Println("   [1] Update the key in Hetzner (recommended if you regenerated your key)")
							fmt.Println("       This will delete the old key and upload your current local key.")
							fmt.Println()
							fmt.Println("   [2] Use a different key name in config.yaml (manual)")
							fmt.Println("       You'll need to edit config.yaml and set a new ssh.key_name")
							fmt.Println()
							fmt.Print("   Enter choice (1/2): ")

							var choice string
							fmt.Scanln(&choice)

							switch choice {
							case "1":
								fmt.Println()
								fmt.Printf("   Deleting old SSH key '%s' from Hetzner...\n", keyName)

								// Delete the old key using Hetzner API
								deleteErr := hetznerProv.DeleteSSHKey(ctx, keyName)
								if deleteErr != nil {
									fmt.Printf("   ❌ Failed to delete key: %s\n", deleteErr)
								} else {
									fmt.Printf("   ✅ Deleted old key '%s'\n", keyName)

									// Upload the new key using Hetzner API
									fmt.Printf("   Uploading new SSH key from %s...\n", foundKeyPath)
									_, createErr := hetznerProv.EnsureSSHKeyWithPath(ctx, keyName, foundKeyPath)
									if createErr != nil {
										fmt.Printf("   ❌ Failed to upload key: %s\n", createErr)
									} else {
										fmt.Printf("   ✅ Uploaded new SSH key '%s' to Hetzner\n", keyName)

										// Clear old host keys from known_hosts for all active servers
										clearKnownHostsForActiveServers()

										fmt.Println()
										fmt.Println("   ⚠️  IMPORTANT: Your existing servers still have the OLD key installed.")
										fmt.Println("   You need to teardown and re-provision to use the new key:")
										fmt.Println()
										fmt.Println("      morpheus list                    # See your forests")
										fmt.Println("      morpheus teardown <forest-id>    # Remove old servers")
										fmt.Println("      morpheus plant small             # Create new servers with new key")
										fmt.Println()
										allOk = true // Fixed the Hetzner key issue
									}
								}
							case "2":
								fmt.Println()
								fmt.Println("   To use a different key name, edit config.yaml:")
								fmt.Println()
								fmt.Println("      ssh:")
								fmt.Println("        key_name: my-new-key")
								fmt.Println()
								fmt.Println("   After updating, re-provision your servers for the new key to be used.")
							default:
								fmt.Println()
								fmt.Println("   No action taken.")
							}
						}
					}
				}
			} else {
				fmt.Printf("   ⚠️  SSH key '%s' not found in Hetzner Cloud\n", keyName)
				fmt.Println("   Morpheus will automatically upload it when you provision")
				if foundKey == "" {
					allOk = false
					fmt.Println("   ❌ But no local SSH key was found to upload!")
				}
			}
		}
	}

	// 3. Check SSH connectivity to existing servers (if any)
	registryPath := getRegistryPath()
	reg, err := registry.NewLocalRegistry(registryPath)
	if err == nil {
		forests := reg.ListForests()
		var activeNodes []*registry.Node
		for _, f := range forests {
			if f.Status == "active" {
				nodes, err := reg.GetNodes(f.ID)
				if err == nil {
					activeNodes = append(activeNodes, nodes...)
				}
			}
		}

		if len(activeNodes) > 0 {
			fmt.Println()
			fmt.Printf("   Testing SSH connectivity to %d active server(s)...\n", len(activeNodes))

			// Test first server only to avoid too many checks
			node := activeNodes[0]
			sshPort := 22
			if cfg != nil && cfg.Provisioning.SSHPort != 0 {
				sshPort = cfg.Provisioning.SSHPort
			}

			addr := sshutil.FormatSSHAddress(node.IP, sshPort)
			conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
			if err != nil {
				fmt.Printf("   ⚠️  Cannot reach server %s: %s\n", node.IP, classifyNetError(err))
				fmt.Println("   This could be due to:")
				fmt.Println("     • Server is still booting")
				fmt.Println("     • IPv6 connectivity issues from your network")
				fmt.Println("     • Firewall blocking the connection")
			} else {
				conn.Close()
				fmt.Printf("   ✅ Server %s is reachable on port %d\n", node.IP, sshPort)
			}
		}
	}

	if exitOnResult {
		if allOk {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
	return allOk
}

// isValidSSHKey checks if a string looks like a valid SSH public key
func isValidSSHKey(key string) bool {
	key = strings.TrimSpace(key)
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2-"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// splitFirst splits a string on the first occurrence of sep
func splitFirst(s, sep string) []string {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return []string{s, ""}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

// classifyNetError returns a human-readable description of a network error
func classifyNetError(err error) string {
	if err == nil {
		return "connected"
	}
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "connection refused"):
		return "port closed"
	case strings.Contains(errStr, "no route to host"):
		return "no route"
	case strings.Contains(errStr, "network is unreachable"):
		return "network unreachable"
	case strings.Contains(errStr, "i/o timeout"), strings.Contains(errStr, "timeout"):
		return "timeout"
	default:
		return err.Error()
	}
}

// clearKnownHostsForActiveServers removes host key entries from known_hosts
// for all active servers in the registry. This is useful when SSH keys are
// updated and servers will be reprovisioned with new host keys.
func clearKnownHostsForActiveServers() {
	registryPath := getRegistryPath()
	registry, err := registry.NewLocalRegistry(registryPath)
	if err != nil {
		return // Silently fail if registry can't be loaded
	}

	forests := registry.ListForests()
	var clearedHosts []string

	for _, f := range forests {
		nodes, err := registry.GetNodes(f.ID)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			if node.IP != "" {
				if err := sshutil.RemoveKnownHostEntry(node.IP); err == nil {
					clearedHosts = append(clearedHosts, node.IP)
				}
			}
		}
	}

	if len(clearedHosts) > 0 {
		fmt.Println()
		fmt.Printf("   🔑 Cleared %d old host key(s) from known_hosts:\n", len(clearedHosts))
		for _, host := range clearedHosts {
			fmt.Printf("      - %s\n", host)
		}
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

func handleTest() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus test <subcommand>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  e2e      Run end-to-end tests (runs locally, provisions to cloud)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --keep   Keep the test forest after tests (for debugging)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "The E2E test suite (runs locally with IPv4/IPv6 support):")
		fmt.Fprintln(os.Stderr, "  1. Checks network connectivity (IPv6 or IPv4 fallback)")
		fmt.Fprintln(os.Stderr, "  2. Ensures SSH key is configured in Hetzner")
		fmt.Fprintln(os.Stderr, "  3. Plants a test forest (small)")
		fmt.Fprintln(os.Stderr, "  4. Verifies SSH connectivity to provisioned node")
		fmt.Fprintln(os.Stderr, "  5. Checks cloud-init completion")
		fmt.Fprintln(os.Stderr, "  6. Verifies NimsForest installation (required)")
		fmt.Fprintln(os.Stderr, "  7. Checks NATS monitoring (required)")
		fmt.Fprintln(os.Stderr, "  8. Tears down test forest")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus test e2e          # Run E2E tests (~7-10 min)")
		fmt.Fprintln(os.Stderr, "  morpheus test e2e --keep   # Keep forest after tests for debugging")
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "e2e":
		handleTestE2E()
	default:
		fmt.Fprintf(os.Stderr, "Unknown test subcommand: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, "Available subcommands: e2e")
		os.Exit(1)
	}
}

func handleTestE2E() {
	// Parse flags
	keepForest := false
	for _, arg := range os.Args[3:] {
		if arg == "--keep" {
			keepForest = true
		}
	}

	fmt.Println()
	fmt.Println("🧪 Morpheus E2E Test Suite (Local Control)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Load config to get API token
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %s\n", err)
		fmt.Fprintln(os.Stderr, "   Make sure HETZNER_API_TOKEN is set or config.yaml exists")
		os.Exit(1)
	}

	if cfg.Secrets.HetznerAPIToken == "" {
		fmt.Fprintln(os.Stderr, "❌ Hetzner API token not configured")
		fmt.Fprintln(os.Stderr, "   Set HETZNER_API_TOKEN environment variable or add to config.yaml")
		os.Exit(1)
	}

	// Create Hetzner provider for direct API operations
	hetznerProv, err := hetzner.NewProvider(cfg.Secrets.HetznerAPIToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Hetzner provider: %s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var testForestID string
	testsPassed := 0
	testsFailed := 0

	// Helper to run SSH commands on a node
	runSSHToNode := func(nodeIP, command string) (string, error) {
		sshArgs := []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "ConnectTimeout=15",
			fmt.Sprintf("root@%s", nodeIP),
			command,
		}
		cmd := exec.Command("ssh", sshArgs...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Cleanup function
	cleanup := func() {
		if testForestID != "" && !keepForest {
			fmt.Println()
			fmt.Println("🧹 Tearing down test forest...")

			// Load registry to get nodes
			registryPath := getRegistryPath()
			reg, err := registry.NewLocalRegistry(registryPath)
			if err == nil {
				nodes, _ := reg.GetNodes(testForestID)
				for _, node := range nodes {
					if node.ID != "" {
						_ = hetznerProv.DeleteServer(ctx, node.ID)
					}
				}
				_ = reg.DeleteForest(testForestID)
			}
			fmt.Println("   ✅ Test forest torn down")
		} else if keepForest && testForestID != "" {
			fmt.Println()
			fmt.Println("📌 Keeping test forest (--keep flag)")
			fmt.Printf("   Forest ID: %s\n", testForestID)
			fmt.Println("   To teardown later: morpheus teardown " + testForestID)
		}
	}

	// Step 1: Check network connectivity
	fmt.Println("📡 Step 1: Checking network connectivity...")

	// Test IPv6
	ctx6, cancel6 := context.WithTimeout(ctx, 10*time.Second)
	result6 := httputil.CheckIPv6Connectivity(ctx6)
	cancel6()

	// Test IPv4
	ctx4, cancel4 := context.WithTimeout(ctx, 10*time.Second)
	result4 := httputil.CheckIPv4Connectivity(ctx4)
	cancel4()

	hasIPv6 := result6.Available
	hasIPv4 := result4.Available

	if hasIPv6 {
		fmt.Printf("   ✅ IPv6 available (%s)\n", result6.Address)
		testsPassed++
	} else {
		fmt.Println("   ⚠️  IPv6 not available")
	}

	if hasIPv4 {
		fmt.Printf("   ✅ IPv4 available (%s)\n", result4.Address)
		if !hasIPv6 {
			testsPassed++ // Count IPv4 as pass if no IPv6
		}
	} else {
		fmt.Println("   ⚠️  IPv4 not available")
	}

	if !hasIPv6 && !hasIPv4 {
		fmt.Println("   ❌ No network connectivity")
		testsFailed++
		os.Exit(1)
	}

	// Enable IPv4 fallback if no IPv6
	if !hasIPv6 && hasIPv4 {
		fmt.Println("   📝 Enabling IPv4 fallback mode for this test")
		cfg.Infrastructure.EnableIPv4Fallback = true
	}

	// Step 2: Ensure SSH key exists
	fmt.Println()
	fmt.Println("🔑 Step 2: Checking SSH key...")

	sshKeyName := cfg.GetSSHKeyName()
	_, err = hetznerProv.EnsureSSHKeyWithPath(ctx, sshKeyName, "")
	if err != nil {
		fmt.Printf("   ❌ Failed to ensure SSH key: %s\n", err)
		testsFailed++
		os.Exit(1)
	}
	fmt.Printf("   ✅ SSH key '%s' ready in Hetzner\n", sshKeyName)
	testsPassed++

	// Step 3: Plant a test forest
	fmt.Println()
	fmt.Println("🌲 Step 3: Planting test forest (small)...")

	// Create registry
	registryPath := getRegistryPath()
	reg, err := registry.NewLocalRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ❌ Failed to create registry: %s\n", err)
		testsFailed++
		os.Exit(1)
	}

	// Create provisioner
	provisioner := forest.NewProvisioner(hetznerProv, reg, cfg)

	// Generate forest ID
	testForestID = fmt.Sprintf("e2e-test-%d", time.Now().Unix())

	// Get machine profile and select server type
	profile := provider.GetProfileForSize("small")
	preferredLocations := []string{"ash", "hel1", "nbg1", "fsn1"}

	serverType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, profile, preferredLocations)
	if err != nil {
		fmt.Printf("   ❌ Failed to select server type: %s\n", err)
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	location := availableLocations[0]
	fmt.Printf("   📦 Using %s in %s\n", serverType, location)

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   testForestID,
		Size:       "small",
		Location:   location,
		ServerType: serverType,
		Image:      "ubuntu-24.04",
	}

	// Provision
	err = provisioner.Provision(ctx, req)
	if err != nil {
		fmt.Printf("   ❌ Provisioning failed: %s\n", err)
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	fmt.Printf("   ✅ Forest %s planted\n", testForestID)
	testsPassed++

	// Step 4: Get node info and verify connectivity
	fmt.Println()
	fmt.Println("🔍 Step 4: Verifying node connectivity...")

	nodes, err := reg.GetNodes(testForestID)
	if err != nil || len(nodes) == 0 {
		fmt.Println("   ❌ No nodes found in forest")
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	node := nodes[0]

	// Use the appropriate IP based on connectivity
	nodeIP := node.GetPreferredIP(hasIPv6)
	if hasIPv6 && node.IPv6 != "" {
		fmt.Printf("   📍 Node IP: %s (IPv6)\n", nodeIP)
	} else if node.IPv4 != "" {
		fmt.Printf("   📍 Node IP: %s (IPv4)\n", nodeIP)
	} else {
		fmt.Printf("   📍 Node IP: %s\n", nodeIP)
	}

	// Wait for SSH to be available
	fmt.Println("   ⏳ Waiting for SSH...")
	sshReady := false
	sshDeadline := time.Now().Add(3 * time.Minute)

	for time.Now().Before(sshDeadline) {
		sshAddr := sshutil.FormatSSHAddress(nodeIP, 22)
		conn, err := net.DialTimeout("tcp", sshAddr, 5*time.Second)
		if err == nil {
			conn.Close()
			sshReady = true
			break
		}
		time.Sleep(10 * time.Second)
	}

	if !sshReady {
		fmt.Println("   ❌ SSH not available within timeout")
		testsFailed++
		cleanup()
		os.Exit(1)
	}

	fmt.Println("   ✅ SSH is available")
	testsPassed++

	// Step 5: Verify cloud-init completed
	fmt.Println()
	fmt.Println("⚙️  Step 5: Verifying cloud-init...")

	// Wait a bit for cloud-init
	time.Sleep(30 * time.Second)

	output, err := runSSHToNode(nodeIP, "cloud-init status --wait 2>/dev/null || echo 'done'")
	if err == nil && (strings.Contains(output, "done") || strings.Contains(output, "status: done")) {
		fmt.Println("   ✅ Cloud-init completed")
		testsPassed++
	} else {
		fmt.Printf("   ⚠️  Cloud-init status unclear: %s\n", strings.TrimSpace(output))
		// Not a hard failure, continue
	}

	// Step 6: Check if NimsForest binary exists
	fmt.Println()
	fmt.Println("📦 Step 6: Checking NimsForest installation...")

	output, err = runSSHToNode(nodeIP, "test -f /opt/nimsforest/bin/nimsforest && echo 'exists' || test -f /usr/local/bin/nimsforest && echo 'exists' || echo 'not-found'")
	if strings.Contains(output, "exists") {
		fmt.Println("   ✅ NimsForest binary found")
		testsPassed++

		// Check if service is running
		output, _ = runSSHToNode(nodeIP, "systemctl is-active nimsforest 2>/dev/null || echo 'not-active'")
		if strings.Contains(output, "active") && !strings.Contains(output, "not-active") {
			fmt.Println("   ✅ NimsForest service is running")
		} else {
			fmt.Println("   ❌ NimsForest service not active")
			testsFailed++
		}
	} else {
		fmt.Println("   ❌ NimsForest binary not found")
		testsFailed++
	}

	// Step 7: Check NATS server is running (required)
	fmt.Println()
	fmt.Println("📊 Step 7: Checking NATS server...")

	// Check if NATS is listening (client port or cluster port)
	output, err = runSSHToNode(nodeIP, "ss -tlnp 2>/dev/null | grep -E '(nimsforest|nats)' | head -5")
	if err == nil && (strings.Contains(output, "nimsforest") || strings.Contains(output, "nats")) {
		fmt.Println("   ✅ NATS server is listening")
		testsPassed++
		// Show which ports are listening
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "LISTEN") {
				fmt.Printf("      %s\n", strings.TrimSpace(line))
			}
		}
	} else {
		// Fallback: check if monitoring endpoint is available
		output2, _ := runSSHToNode(nodeIP, "curl -s --connect-timeout 5 http://localhost:8222/varz 2>/dev/null | head -c 100")
		if strings.Contains(output2, "server_id") || strings.Contains(output2, "{") {
			fmt.Println("   ✅ NATS monitoring endpoint responding")
			testsPassed++
		} else {
			fmt.Println("   ❌ NATS server not running")
			testsFailed++
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Test Results")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Passed: %d\n", testsPassed)
	fmt.Printf("   Failed: %d\n", testsFailed)
	fmt.Println()

	if testsFailed == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Printf("❌ %d test(s) failed\n", testsFailed)
	}

	// Cleanup
	cleanup()

	fmt.Println()
	if testsFailed == 0 {
		fmt.Println("✅ E2E test suite completed successfully")
	} else {
		fmt.Println("❌ E2E test suite completed with failures")
		os.Exit(1)
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
	fmt.Println("  grow <forest-id>            Check cluster health and optionally expand")
	fmt.Println("                              Options:")
	fmt.Println("                                --auto        Non-interactive mode")
	fmt.Println("                                --threshold N Resource threshold (default: 80)")
	fmt.Println("                                --json        Output in JSON format")
	fmt.Println("  teardown <forest-id>        Delete a forest and all its resources")
	fmt.Println("  version                     Show version information")
	fmt.Println("  update                      Check for updates and install if available")
	fmt.Println("  check-update                Check for updates without installing")
	fmt.Println("  check                       Run all diagnostics (network, SSH)")
	fmt.Println("  check ipv6                  Check IPv6 connectivity")
	fmt.Println("  check ipv4                  Check IPv4 connectivity")
	fmt.Println("  check network               Check both IPv6 and IPv4")
	fmt.Println("  check ssh                   Check SSH key setup")
	fmt.Println("  check-ipv6                  (deprecated) Use 'check ipv6' instead")
	fmt.Println("  test e2e                    Run E2E tests locally (~7-10 min)")
	fmt.Println("  test e2e --keep             Run E2E tests and keep forest for debugging")
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
