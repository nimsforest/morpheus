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
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handlePlant() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus plant <cloud|local> <size>")
		fmt.Fprintln(os.Stderr, "Sizes: wood (1 node), forest (3 nodes), jungle (5 nodes)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus plant cloud wood   # Create 1-node forest on Hetzner")
		fmt.Fprintln(os.Stderr, "  morpheus plant local wood   # Create 1-node forest locally (Docker)")
		os.Exit(1)
	}

	deploymentType := os.Args[2]
	size := os.Args[3]

	if deploymentType != "cloud" && deploymentType != "local" {
		fmt.Fprintf(os.Stderr, "Invalid deployment type: %s (must be: cloud or local)\n", deploymentType)
		os.Exit(1)
	}

	if size != "wood" && size != "forest" && size != "jungle" {
		fmt.Fprintf(os.Stderr, "Invalid size: %s (must be: wood, forest, or jungle)\n", size)
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

	// Determine location
	var location string
	if deploymentType == "local" {
		location = "local"
	} else {
		if len(cfg.Infrastructure.Locations) == 0 {
			fmt.Fprintln(os.Stderr, "No locations configured in config")
			os.Exit(1)
		}
		// Try to find an available location
		location = cfg.Infrastructure.Locations[0]
	}

	// Create provision request
	req := forest.ProvisionRequest{
		ForestID: forestID,
		Size:     size,
		Location: location,
		Role:     cloudinit.RoleEdge, // Default role
	}

	// Provision with automatic location fallback
	fmt.Printf("\n🌲 Morpheus - Infrastructure Provisioning\n")
	fmt.Printf("=========================================\n")
	fmt.Printf("Forest ID: %s\n", forestID)
	fmt.Printf("Size: %s\n", size)
	fmt.Printf("Location: %s\n", location)
	fmt.Printf("Provider: %s\n\n", providerName)

	ctx := context.Background()
	err = provisionWithLocationFallback(ctx, provisioner, req, cfg.Infrastructure.Locations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Provisioning failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Forest provisioned successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  - Check status: morpheus status %s\n", forestID)
	fmt.Printf("  - List all: morpheus list\n")
	fmt.Printf("  - Teardown: morpheus teardown %s\n", forestID)
	if deploymentType == "local" {
		fmt.Printf("\nLocal mode tips:\n")
		fmt.Printf("  - Access container: docker exec -it <container-name> bash\n")
		fmt.Printf("  - View logs: docker logs <container-name>\n")
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
			fmt.Printf("⚠️  Location %s is unavailable, trying next location...\n\n", location)
			continue
		}

		// If it's not a location error, don't try other locations
		break
	}

	// All locations failed or encountered a non-location error
	if containsLocationError(lastErr.Error()) && len(attemptedLocations) > 1 {
		return fmt.Errorf("all configured locations are unavailable (%s): %w\n\n"+
			"Hetzner may be experiencing capacity issues. Try again later or update your config with different locations:\n"+
			"  Available locations: ash (Ashburn, USA), fsn1 (Falkenstein, Germany), nbg1 (Nuremberg, Germany), \n"+
			"                       hel1 (Helsinki, Finland), hil (Hillsboro, USA), sin (Singapore)",
			joinLocations(attemptedLocations), lastErr)
	}

	return lastErr
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
			Defaults: config.DefaultsConfig{
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
		fmt.Println("No forests found.")
		fmt.Println("\nCreate one with: morpheus plant cloud <size>")
		return
	}

	fmt.Println("FOREST ID            SIZE    LOCATION  STATUS       CREATED")
	fmt.Println("----------------------------------------------------------------------------")

	for _, f := range forests {
		fmt.Printf("%-20s %-7s %-9s %-12s %s\n",
			f.ID,
			f.Size,
			f.Location,
			f.Status,
			f.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
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
	fmt.Printf("Size: %s\n", forestInfo.Size)
	fmt.Printf("Location: %s\n", forestInfo.Location)
	fmt.Printf("Provider: %s\n", forestInfo.Provider)
	fmt.Printf("Status: %s\n", forestInfo.Status)
	fmt.Printf("Created: %s\n\n", forestInfo.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(nodes) > 0 {
		fmt.Printf("Nodes (%d):\n", len(nodes))
		fmt.Println("ID        ROLE   IP             LOCATION  STATUS")
		fmt.Println("-----------------------------------------------------------")
		for _, node := range nodes {
			fmt.Printf("%-9s %-6s %-14s %-9s %s\n",
				node.ID,
				node.Role,
				node.IP,
				node.Location,
				node.Status,
			)
		}
	} else {
		fmt.Println("No nodes registered yet.")
	}
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

	// Confirm
	fmt.Printf("⚠️  This will permanently delete forest: %s\n", forestID)
	fmt.Print("Are you sure? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Teardown cancelled.")
		return
	}

	// Teardown
	ctx := context.Background()
	if err := provisioner.Teardown(ctx, forestID); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Teardown failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Forest %s has been torn down successfully!\n", forestID)
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

func printHelp() {
	fmt.Println("Morpheus - Nims Forest Infrastructure Provisioning Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  plant <cloud|local> <size>  Provision a new forest")
	fmt.Println("                              Deployment types:")
	fmt.Println("                                cloud - Provision on Hetzner Cloud")
	fmt.Println("                                local - Provision locally using Docker")
	fmt.Println("                              Sizes: wood (1 node), forest (3 nodes), jungle (5 nodes)")
	fmt.Println("  list                        List all forests")
	fmt.Println("  status <forest-id>          Show detailed forest status")
	fmt.Println("  teardown <forest-id>        Delete a forest and all its resources")
	fmt.Println("  version                     Show version information")
	fmt.Println("  update                      Check for updates and install if available")
	fmt.Println("  check-update                Check for updates without installing")
	fmt.Println("  help                        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus plant cloud wood     # Create 1-node forest on Hetzner")
	fmt.Println("  morpheus plant local wood     # Create 1-node forest locally (Docker)")
	fmt.Println("  morpheus plant cloud forest   # Create 3-node forest on Hetzner")
	fmt.Println("  morpheus plant local forest   # Create 3-node forest locally (Docker)")
	fmt.Println("  morpheus list                 # List all forests")
	fmt.Println("  morpheus status forest-12345  # Show forest details")
	fmt.Println("  morpheus teardown forest-12345 # Delete forest")
	fmt.Println("  morpheus update               # Update to latest version")
	fmt.Println()
	fmt.Println("Local Mode:")
	fmt.Println("  Local mode uses Docker to create forest containers on your machine.")
	fmt.Println("  No cloud account or API token required - great for development!")
	fmt.Println("  Requirements: Docker must be installed and running.")
	fmt.Println()
	fmt.Println("  ⚠️  Termux users: Local mode does NOT work on Android/Termux!")
	fmt.Println("      Docker cannot run on Termux due to Android kernel limitations.")
	fmt.Println("      Use 'morpheus plant cloud <size>' instead.")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Morpheus looks for config.yaml in:")
	fmt.Println("    - ./config.yaml")
	fmt.Println("    - ~/.morpheus/config.yaml")
	fmt.Println("    - /etc/morpheus/config.yaml")
	fmt.Println("  Note: Local mode doesn't require a config file.")
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/nimsforest/morpheus")
}
