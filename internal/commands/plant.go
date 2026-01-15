package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nimsforest/morpheus/internal/ui"
	"github.com/nimsforest/morpheus/pkg/forest"
	"github.com/nimsforest/morpheus/pkg/machine/hetzner"
)

// HandlePlant handles the plant command.
func HandlePlant() {
	// Parse arguments - simplified CLI
	// morpheus plant             -> 2 nodes (default)
	// morpheus plant --nodes 3   -> 3 nodes

	nodeCount := 2

	// Parse arguments
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--nodes", "-n":
			if i+1 < len(os.Args) {
				i++
				n, err := strconv.Atoi(os.Args[i])
				if err != nil || n < 1 {
					fmt.Fprintf(os.Stderr, "❌ Invalid node count: %s\n", os.Args[i])
					os.Exit(1)
				}
				nodeCount = n
			} else {
				fmt.Fprintln(os.Stderr, "❌ --nodes requires a number")
				os.Exit(1)
			}
		case "--help", "-h":
			fmt.Println("Usage: morpheus plant [options]")
			fmt.Println()
			fmt.Println("Create a new forest with the specified number of nodes.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --nodes, -n N   Number of nodes to create (default: 2)")
			fmt.Println("  --help, -h      Show this help")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  morpheus plant              # Create 2-node cluster")
			fmt.Println("  morpheus plant --nodes 3    # Create 3-node forest")
			os.Exit(0)
		default:
			// Support legacy size arguments for backward compatibility
			if ui.IsValidSize(arg) {
				nodeCount = ui.GetNodeCount(arg)
			} else {
				fmt.Fprintf(os.Stderr, "❌ Unknown argument: %s\n", arg)
				fmt.Fprintln(os.Stderr, "Use 'morpheus plant --help' for usage")
				os.Exit(1)
			}
		}
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %s\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %s\n", err)
		os.Exit(1)
	}

	// Create machine provider based on configuration
	machineProv, providerName, err := CreateMachineProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// Create storage
	storageProv, err := CreateStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create storage: %s\n", err)
		os.Exit(1)
	}

	// Create DNS provider if configured
	dnsProv := CreateDNSProvider(cfg)

	// Create provisioner
	var provisioner *forest.Provisioner
	if dnsProv != nil {
		provisioner = forest.NewProvisionerWithDNS(machineProv, storageProv, dnsProv, cfg)
	} else {
		provisioner = forest.NewProvisioner(machineProv, storageProv, cfg)
	}

	// Generate forest ID
	forestID := fmt.Sprintf("forest-%d", time.Now().Unix())

	// Create context early for provider operations
	ctx := context.Background()

	// Determine server type, location, and image from config
	var location, serverType, image string

	// For Hetzner, select the best server type and locations
	if hetznerProv, ok := machineProv.(*hetzner.Provider); ok {
		// Get default locations if not configured
		preferredLocations := []string{cfg.GetLocation()}
		if preferredLocations[0] == "" {
			preferredLocations = hetzner.GetDefaultLocations()
		}

		// Select best server type and available locations using config
		selectedType, availableLocations, err := hetznerProv.SelectBestServerType(ctx, cfg.GetServerType(), cfg.GetServerTypeFallback(), preferredLocations)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to select server type: %s\n", err)
			os.Exit(1)
		}

		serverType = selectedType
		location = availableLocations[0] // Use first available location
		image = cfg.GetImage()
	} else {
		// Non-Hetzner provider
		serverType = cfg.GetServerType()
		location = cfg.GetLocation()
		image = cfg.GetImage()
	}

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID:   forestID,
		NodeCount:  nodeCount,
		Location:   location,
		ServerType: serverType,
		Image:      image,
	}

	// Display friendly provisioning header
	fmt.Printf("\n🌲 Planting your forest...\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Show what's being created
	var timeEstimate string
	switch {
	case nodeCount == 1:
		timeEstimate = "3-5 minutes"
	case nodeCount <= 3:
		timeEstimate = "7-15 minutes"
	default:
		timeEstimate = "15-30 minutes"
	}

	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Forest ID:  %s\n", forestID)
	fmt.Printf("   Nodes:      %d\n", nodeCount)
	fmt.Printf("   Machine:    %s (with automatic fallback if unavailable)\n", serverType)
	fmt.Printf("   Location:   %s (with automatic fallback if unavailable)\n", hetzner.GetLocationDescription(location))
	fmt.Printf("   Provider:   %s\n", providerName)
	fmt.Printf("   Time:       ~%s\n\n", timeEstimate)

	estimatedCost := hetzner.GetEstimatedCost(serverType) * float64(nodeCount)
	fmt.Printf("💰 Estimated cost: ~€%.2f/month\n", estimatedCost)
	if cfg.IsIPv4Enabled() {
		fmt.Printf("   (IPv4+IPv6, billed by minute, can teardown anytime)\n")
		fmt.Printf("   ⚠️  IPv4 enabled - additional charges apply per IPv4 address\n\n")
	} else {
		fmt.Printf("   (IPv6-only, billed by minute, can teardown anytime)\n\n")
	}

	fmt.Println("🚀 Starting provisioning...")

	// Use the full fallback system for Hetzner
	if hetznerProv, ok := machineProv.(*hetzner.Provider); ok {
		err = provisionWithFallback(ctx, provisioner, hetznerProv, req, cfg.GetServerType(), cfg.GetServerTypeFallback())
	} else {
		err = provisioner.Provision(ctx, req)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Provisioning failed: %s\n", err)
		os.Exit(1)
	}

	// Success message with clear next steps
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ Success! Your forest is ready!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("🎯 What's next?\n\n")

	fmt.Printf("📊 Check your forest status:\n")
	fmt.Printf("   morpheus status %s\n\n", forestID)

	fmt.Printf("🌐 Your machines are ready for NATS deployment\n")
	fmt.Printf("   Infrastructure is configured and waiting\n\n")

	fmt.Printf("📋 View all your forests:\n")
	fmt.Printf("   morpheus list\n\n")

	fmt.Printf("🌱 Add more nodes:\n")
	fmt.Printf("   morpheus grow %s --nodes 2\n\n", forestID)

	fmt.Printf("🗑️  Clean up when done:\n")
	fmt.Printf("   morpheus teardown %s\n\n", forestID)
}

// provisionWithFallback tries to provision a forest, automatically falling back
// to alternative server types and locations if the primary ones are unavailable.
func provisionWithFallback(ctx context.Context, provisioner *forest.Provisioner, hetznerProv *hetzner.Provider, req forest.ProvisionRequest, serverType string, fallbacks []string) error {
	// Get all server type options from config
	allServerTypes := append([]string{serverType}, fallbacks...)

	// Preferred location order: Helsinki first, then Nuremberg, then others
	preferredLocations := hetzner.GetDefaultLocations()

	var lastErr error
	var attemptedCombos []string
	var validServerTypes []string
	isFirstAttempt := true

	// First, validate which server types actually exist in Hetzner
	for _, st := range allServerTypes {
		exists, err := hetznerProv.ValidateServerType(ctx, st)
		if err != nil {
			fmt.Printf("   ⚠️  Could not validate server type %s: %v\n", st, err)
			continue
		}
		if !exists {
			fmt.Printf("   ⚠️  Server type %s does not exist in Hetzner, skipping\n", st)
			continue
		}
		validServerTypes = append(validServerTypes, st)
	}

	if len(validServerTypes) == 0 {
		return fmt.Errorf("none of the configured server types exist in Hetzner: %s", JoinLocations(allServerTypes))
	}

	// Try each validated server type
	for serverTypeIdx, st := range validServerTypes {
		// Get available locations for this server type
		availableLocations, err := hetznerProv.GetAvailableLocations(ctx, st)
		if err != nil {
			fmt.Printf("   ⚠️  Could not check availability for %s: %v\n", st, err)
			continue
		}
		if len(availableLocations) == 0 {
			fmt.Printf("   ⚠️  Server type %s has no available locations\n", st)
			continue
		}

		// Reorder available locations to match preferred order
		orderedLocations := OrderLocationsByPreference(availableLocations, preferredLocations)

		// Show info when switching to fallback server type
		if serverTypeIdx > 0 && len(attemptedCombos) > 0 {
			fmt.Printf("\n📦 Trying alternative server type: %s (~€%.2f/mo)\n",
				st, hetzner.GetEstimatedCost(st))
		}

		// Try each location for this server type (in preferred order)
		for _, location := range orderedLocations {
			attemptedCombos = append(attemptedCombos, fmt.Sprintf("%s@%s", st, location))

			// Update request with current server type and location
			req.ServerType = st
			req.Location = location

			if !isFirstAttempt {
				fmt.Printf("   📍 Trying %s in %s...\n", st, hetzner.GetLocationDescription(location))
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
			if ContainsLocationError(errStr) {
				fmt.Printf("   ⚠️  %s not available in %s, trying next option...\n", st, location)
				continue
			}

			// If it's not a location error, this is a real error - stop trying
			return err
		}
	}

	// All combinations failed
	if lastErr != nil && ContainsLocationError(lastErr.Error()) {
		return fmt.Errorf("no server type/location combination available\n\n"+
			"Tried %d combinations across server types: %s\n\n"+
			"This usually means Hetzner is experiencing high demand or capacity issues.\n"+
			"Please try again in a few minutes.\n"+
			"For status updates, check: https://status.hetzner.com/",
			len(attemptedCombos), JoinLocations(validServerTypes))
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no server type available")
}
