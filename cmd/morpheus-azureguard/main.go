package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/guard"
	"github.com/nimsforest/morpheus/pkg/guard/azure"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		handleCreate()
	case "status":
		handleStatus()
	case "list":
		handleList()
	case "teardown":
		handleTeardown()
	case "peer":
		handlePeer()
	case "version":
		fmt.Printf("morpheus-azureguard version %s\n", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("🛡️  morpheus-azureguard — WireGuard Gateway VM Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus-azureguard <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create                   Create a new guard VM")
	fmt.Println("    --config <path|->      WireGuard config file (required)")
	fmt.Println("    --mesh-cidrs <cidrs>   Comma-separated mesh CIDRs")
	fmt.Println("    --location <loc>       Azure location (default: from config)")
	fmt.Println()
	fmt.Println("  status <guard-id>        Show guard details")
	fmt.Println("  list                     List all guards")
	fmt.Println("  teardown <guard-id>      Delete a guard and all resources")
	fmt.Println()
	fmt.Println("  peer <guard-id>          Peer a workload VNet to the guard VNet")
	fmt.Println("    --vnet <resource-id>   Remote VNet resource ID (required)")
	fmt.Println("    --subnet <resource-id> Remote subnet for route table (optional)")
	fmt.Println()
	fmt.Println("  version                  Show version")
	fmt.Println("  help                     Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus-azureguard create --config /path/to/wg0.conf --mesh-cidrs 10.200.0.0/16")
	fmt.Println("  hydraguard venue config azure-westeu | morpheus-azureguard create --config -")
	fmt.Println("  morpheus-azureguard peer guard-1738123456 --vnet /subscriptions/.../virtualNetworks/workload-vnet")
	fmt.Println("  morpheus-azureguard status guard-1738123456")
	fmt.Println("  morpheus-azureguard list")
	fmt.Println("  morpheus-azureguard teardown guard-1738123456")
}

func loadConfig() *config.Config {
	cfg, err := loadConfigFromPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %s\n", err)
		os.Exit(1)
	}
	if err := cfg.ValidateGuard(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Invalid config: %s\n", err)
		os.Exit(1)
	}
	return cfg
}

func loadConfigFromPaths() (*config.Config, error) {
	paths := []string{
		"./config.yaml",
	}
	home := os.Getenv("HOME")
	if home != "" {
		paths = append(paths, home+"/.morpheus/config.yaml")
	}
	paths = append(paths, "/etc/morpheus/config.yaml")

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return config.LoadConfig(path)
		}
	}
	return nil, fmt.Errorf("no config file found (tried: %v)", paths)
}

func createProvider(cfg *config.Config) *azure.Provider {
	az := cfg.Machine.Azure
	prov, err := azure.NewProvider(
		az.SubscriptionID, az.TenantID, az.ClientID, az.ClientSecret,
		az.ResourceGroup, az.Location, az.VMSize, az.Image,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Azure provider: %s\n", err)
		os.Exit(1)
	}
	return prov
}

// ── create ──────────────────────────────────────────────────────────────────

func handleCreate() {
	var configPath, location string
	var meshCIDRs []string

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--config":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "❌ --config requires a path or '-' for stdin")
				os.Exit(1)
			}
			i++
			configPath = os.Args[i]
		case "--mesh-cidrs":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "❌ --mesh-cidrs requires comma-separated CIDRs")
				os.Exit(1)
			}
			i++
			meshCIDRs = strings.Split(os.Args[i], ",")
		case "--location":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "❌ --location requires a value")
				os.Exit(1)
			}
			i++
			location = os.Args[i]
		case "--help", "-h":
			fmt.Println("Usage: morpheus-azureguard create --config <path|-> [--mesh-cidrs <cidrs>] [--location <loc>]")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "❌ Unknown argument: %s\n", os.Args[i])
			os.Exit(1)
		}
	}

	if configPath == "" {
		fmt.Fprintln(os.Stderr, "❌ --config is required")
		fmt.Fprintln(os.Stderr, "Usage: morpheus-azureguard create --config <path|-> [--mesh-cidrs <cidrs>]")
		os.Exit(1)
	}

	// Read WireGuard config
	var wgConf string
	if configPath == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to read from stdin: %s\n", err)
			os.Exit(1)
		}
		wgConf = string(data)
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to read config file: %s\n", err)
			os.Exit(1)
		}
		wgConf = string(data)
	}

	if strings.TrimSpace(wgConf) == "" {
		fmt.Fprintln(os.Stderr, "❌ WireGuard config is empty")
		os.Exit(1)
	}

	cfg := loadConfig()
	prov := createProvider(cfg)
	provisioner := guard.NewProvisioner(prov, cfg)

	ctx := context.Background()
	g, err := provisioner.Provision(ctx, guard.CreateGuardRequest{
		Location:      location,
		WireGuardConf: wgConf,
		MeshCIDRs:     meshCIDRs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Create failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ Guard created successfully!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Printf("   Guard ID:    %s\n", g.ID)
	fmt.Printf("   Public IP:   %s\n", g.PublicIP)
	fmt.Printf("   Private IP:  %s\n", g.PrivateIP)
	fmt.Printf("   VNet:        %s\n", g.VNetID)
	fmt.Printf("   Location:    %s\n", g.Location)
	fmt.Println()
	fmt.Printf("🔗 Peer a workload VNet:\n")
	fmt.Printf("   morpheus-azureguard peer %s --vnet <workload-vnet-resource-id>\n\n", g.ID)
	fmt.Printf("🔍 Check status:\n")
	fmt.Printf("   morpheus-azureguard status %s\n\n", g.ID)
	fmt.Printf("🗑️  Teardown:\n")
	fmt.Printf("   morpheus-azureguard teardown %s\n", g.ID)
}

// ── status ──────────────────────────────────────────────────────────────────

func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus-azureguard status <guard-id>")
		os.Exit(1)
	}

	guardID := os.Args[2]
	cfg := loadConfig()
	prov := createProvider(cfg)

	ctx := context.Background()
	g, err := prov.GetGuard(ctx, guardID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get guard: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🛡️  Guard: %s\n", g.ID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("   Status:      %s\n", g.Status)
	fmt.Printf("   Location:    %s\n", g.Location)
	fmt.Printf("   Public IP:   %s\n", g.PublicIP)
	fmt.Printf("   Private IP:  %s\n", g.PrivateIP)
	fmt.Printf("   WG Port:     %d\n", g.WireGuardPort)
	if len(g.MeshCIDRs) > 0 {
		fmt.Printf("   Mesh CIDRs:  %s\n", strings.Join(g.MeshCIDRs, ", "))
	}
	fmt.Printf("   VNet:        %s\n", g.VNetID)
	fmt.Printf("   RG:          %s\n", g.ResourceGroup)
	if len(g.Peerings) > 0 {
		fmt.Printf("\n   Peerings:\n")
		for _, p := range g.Peerings {
			fmt.Printf("     • %s -> %s\n", p.Name, p.RemoteVNetID)
		}
	}
	fmt.Println()
}

// ── list ────────────────────────────────────────────────────────────────────

func handleList() {
	cfg := loadConfig()
	prov := createProvider(cfg)

	ctx := context.Background()
	guards, err := prov.ListGuards(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list guards: %s\n", err)
		os.Exit(1)
	}

	if len(guards) == 0 {
		fmt.Println("\nNo guards found.")
		fmt.Println("Create one with: morpheus-azureguard create --config <wg0.conf>")
		return
	}

	fmt.Printf("\n🛡️  Guards (%d)\n", len(guards))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	for _, g := range guards {
		fmt.Printf("  %-25s  %-12s  %-15s  %s\n", g.ID, g.Status, g.PublicIP, g.Location)
	}
	fmt.Println()
}

// ── teardown ────────────────────────────────────────────────────────────────

func handleTeardown() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus-azureguard teardown <guard-id>")
		os.Exit(1)
	}

	guardID := os.Args[2]
	cfg := loadConfig()
	prov := createProvider(cfg)

	ctx := context.Background()

	// Show what will be deleted
	g, err := prov.GetGuard(ctx, guardID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Guard not found: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n⚠️  About to permanently delete:\n")
	fmt.Printf("   Guard:     %s\n", g.ID)
	fmt.Printf("   Location:  %s\n", g.Location)
	fmt.Printf("   Public IP: %s\n", g.PublicIP)
	fmt.Printf("   RG:        %s\n", g.ResourceGroup)
	fmt.Println()
	fmt.Print("Type 'yes' to confirm deletion: ")

	var response string
	fmt.Scanln(&response)
	if response != "yes" {
		fmt.Println("\n✅ Teardown cancelled.")
		return
	}

	provisioner := guard.NewProvisioner(prov, cfg)
	if err := provisioner.Teardown(ctx, guardID); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Teardown failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ Guard %s deleted successfully!\n", guardID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// ── peer ────────────────────────────────────────────────────────────────────

func handlePeer() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus-azureguard peer <guard-id> --vnet <resource-id>")
		os.Exit(1)
	}

	guardID := os.Args[2]
	var remoteVNetID, remoteSubnetID string

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--vnet":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "❌ --vnet requires a resource ID")
				os.Exit(1)
			}
			i++
			remoteVNetID = os.Args[i]
		case "--subnet":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "❌ --subnet requires a resource ID")
				os.Exit(1)
			}
			i++
			remoteSubnetID = os.Args[i]
		case "--help", "-h":
			fmt.Println("Usage: morpheus-azureguard peer <guard-id> --vnet <resource-id> [--subnet <resource-id>]")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "❌ Unknown argument: %s\n", os.Args[i])
			os.Exit(1)
		}
	}

	if remoteVNetID == "" {
		fmt.Fprintln(os.Stderr, "❌ --vnet is required")
		os.Exit(1)
	}

	cfg := loadConfig()
	prov := createProvider(cfg)
	ctx := context.Background()

	// Get guard info from Azure
	g, err := prov.GetGuard(ctx, guardID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Guard not found: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🔗 Peering guard %s to workload VNet\n", guardID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("   Guard VNet:  %s\n", g.VNetID)
	fmt.Printf("   Remote VNet: %s\n", remoteVNetID)
	fmt.Println()

	peeringName := fmt.Sprintf("%s-peer", guardID)
	err = prov.PeerNetwork(ctx, guard.PeerRequest{
		GuardID:        guardID,
		GuardVNetID:    g.VNetID,
		RemoteVNetID:   remoteVNetID,
		PeeringName:    peeringName,
		GuardPrivateIP: g.PrivateIP,
		MeshCIDRs:      g.MeshCIDRs,
		SubnetID:       remoteSubnetID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Peering failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("   ✅ Peering established\n")
	if len(g.MeshCIDRs) > 0 && remoteSubnetID != "" {
		fmt.Printf("   ✅ Route table created for mesh CIDRs: %s\n", strings.Join(g.MeshCIDRs, ", "))
	}
	fmt.Println()
}
