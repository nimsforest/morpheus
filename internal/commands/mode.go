package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nimsforest/morpheus/internal/ui"
	"github.com/nimsforest/morpheus/pkg/bootmode"
	"github.com/nimsforest/morpheus/pkg/machine/proxmox"
)

// HandleMode handles the mode command.
func HandleMode() {
	if len(os.Args) < 3 {
		printModeHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		handleModeList()
	case "status":
		handleModeStatus()
	case "linux", "windows":
		handleModeSwitch(subcommand)
	case "help", "--help", "-h":
		printModeHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode subcommand: %s\n\n", subcommand)
		printModeHelp()
		os.Exit(1)
	}
}

func printModeHelp() {
	fmt.Println("🎮 Morpheus Mode - VR Node Boot Mode Management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus mode <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list       List available boot modes")
	fmt.Println("  status     Show current mode and status")
	fmt.Println("  linux      Switch to Linux mode (CachyOS + WiVRN)")
	fmt.Println("  windows    Switch to Windows mode (SteamLink)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus mode status    # Check current mode")
	fmt.Println("  morpheus mode linux     # Switch to Linux for WiVRN VR")
	fmt.Println("  morpheus mode windows   # Switch to Windows for SteamVR")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  Configure Proxmox settings in ~/.morpheus/config.yaml:")
	fmt.Println()
	fmt.Println("  proxmox:")
	fmt.Println("    host: \"192.168.1.100\"")
	fmt.Println("    api_token_id: \"morpheus@pam!token\"")
	fmt.Println("    api_token_secret: \"${PROXMOX_API_TOKEN}\"")
	fmt.Println()
	fmt.Println("  vr:")
	fmt.Println("    linux:")
	fmt.Println("      vmid: 101")
	fmt.Println("    windows:")
	fmt.Println("      vmid: 102")
}

func loadProxmoxManager() (*bootmode.ProxmoxManager, error) {
	// Try to load config, but it's optional if env vars are set
	_, _ = LoadConfig()

	// Get Proxmox config from environment
	proxmoxConfig := proxmox.ProviderConfig{
		Host:           GetEnvOrDefault("PROXMOX_HOST", ""),
		Port:           8006,
		Node:           GetEnvOrDefault("PROXMOX_NODE", "pve"),
		APITokenID:     GetEnvOrDefault("PROXMOX_TOKEN_ID", ""),
		APITokenSecret: GetEnvOrDefault("PROXMOX_API_TOKEN", ""),
		VerifySSL:      false,
	}

	// Check if config is valid
	if proxmoxConfig.Host == "" || proxmoxConfig.APITokenSecret == "" {
		return nil, fmt.Errorf(`Proxmox not configured

Set these environment variables:
  export PROXMOX_HOST="192.168.1.100"
  export PROXMOX_API_TOKEN="your-api-token"
  export PROXMOX_TOKEN_ID="morpheus@pam!morpheus-token"

Optional:
  export PROXMOX_NODE="pve"           # Default: pve
  export PROXMOX_LINUX_VMID="101"     # Default: 101
  export PROXMOX_WINDOWS_VMID="102"   # Default: 102`)
	}

	// VR node config - get from environment or use defaults
	vrConfig := bootmode.VRNodeConfig{
		Linux: bootmode.VMConfig{
			VMID: GetEnvOrDefaultInt("PROXMOX_LINUX_VMID", 101),
			Name: "nimsforest-vr-linux",
		},
		Windows: bootmode.VMConfig{
			VMID: GetEnvOrDefaultInt("PROXMOX_WINDOWS_VMID", 102),
			Name: "nimsforest-vr-windows",
		},
		GPUPCI: GetEnvOrDefault("PROXMOX_GPU_PCI", "0000:01:00"),
	}

	return bootmode.NewProxmoxManager(proxmoxConfig, vrConfig)
}

func handleModeList() {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ping to check connectivity
	if err := manager.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot connect to Proxmox: %s\n", err)
		os.Exit(1)
	}

	modes, err := manager.ListModes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list modes: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("🎮 Available Boot Modes")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("%-10s %-6s %-10s %-10s %s\n", "MODE", "VMID", "STATUS", "VR", "DESCRIPTION")
	fmt.Println("─────────────────────────────────────────────────────────────")

	var currentMode string
	for _, mode := range modes {
		statusIcon := "○"
		if mode.Status == bootmode.ModeStatusRunning {
			statusIcon = "●"
			currentMode = mode.Name
		}

		fmt.Printf("%-10s %-6d %s %-9s %-10s %s\n",
			mode.Name,
			mode.VMID,
			statusIcon,
			mode.Status,
			mode.VRSoftware,
			mode.Description,
		)
	}

	fmt.Println()
	if currentMode != "" {
		fmt.Printf("Current mode: %s\n", currentMode)
	} else {
		fmt.Println("No mode currently active")
	}
	fmt.Println()
	fmt.Println("💡 Switch modes: morpheus mode linux  or  morpheus mode windows")
}

func handleModeStatus() {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	current, err := manager.GetCurrentMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get current mode: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	if current == nil {
		fmt.Println("⚠️  No mode currently active")
		fmt.Println()
		fmt.Println("Start a mode with:")
		fmt.Println("  morpheus mode linux     # For WiVRN VR streaming")
		fmt.Println("  morpheus mode windows   # For SteamLink VR")
		return
	}

	fmt.Printf("🎮 Current Mode: %s\n", current.Name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("   VM:          %s (VMID %d)\n", current.Description, current.VMID)
	fmt.Printf("   Status:      %s\n", current.Status)
	if current.IPAddress != "" {
		fmt.Printf("   IP:          %s\n", current.IPAddress)
	}
	if current.Uptime > 0 {
		fmt.Printf("   Uptime:      %s\n", ui.FormatDuration(current.Uptime))
	}
	fmt.Printf("   VR Software: %s\n", current.VRSoftware)

	if len(current.Services) > 0 {
		fmt.Println()
		fmt.Println("   Services:")
		for _, svc := range current.Services {
			icon := "✓"
			if svc.Status != "active" {
				icon = "✗"
			}
			fmt.Printf("     %s %s: %s\n", icon, svc.Name, svc.Status)
		}
	}

	fmt.Println()
	otherMode := "windows"
	if current.Name == "windows" {
		otherMode = "linux"
	}
	fmt.Printf("💡 Switch to %s: morpheus mode %s\n", otherMode, otherMode)
}

func handleModeSwitch(targetMode string) {
	manager, err := loadProxmoxManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	// Parse options
	opts := bootmode.DefaultSwitchOptions()
	dryRun := false
	for _, arg := range os.Args[3:] {
		switch arg {
		case "--dry-run":
			dryRun = true
			opts.DryRun = true
		case "--force":
			opts.Force = true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get current mode for display
	current, _ := manager.GetCurrentMode(ctx)

	fmt.Println()
	if dryRun {
		fmt.Println("🔍 Dry run - no changes will be made")
		fmt.Println()
	}

	if current != nil {
		fmt.Printf("Switching %s → %s...\n", current.Name, targetMode)
	} else {
		fmt.Printf("Starting %s mode...\n", targetMode)
	}
	fmt.Println()

	result, err := manager.Switch(ctx, targetMode, opts)

	// Handle specific errors
	if _, ok := err.(*bootmode.AlreadyActiveError); ok {
		fmt.Printf("✅ Already in %s mode\n", targetMode)
		return
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Switch failed: %s\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("✅ Dry run complete - switch would succeed")
		return
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ Now in %s mode\n", targetMode)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if result.IPAddress != "" {
		fmt.Printf("   IP: %s\n", result.IPAddress)
	}
	fmt.Printf("   Duration: %s\n", result.Duration.Round(time.Second))

	if targetMode == "linux" {
		fmt.Println()
		fmt.Println("   🎮 WiVRN is ready for VR streaming")
	} else {
		fmt.Println()
		fmt.Println("   🎮 SteamLink is ready for VR streaming")
	}
}
