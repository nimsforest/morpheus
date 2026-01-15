package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nimsforest/morpheus/pkg/dns"
)

// HandleDNSAdd handles "morpheus dns add <type> <domain>"
// Types: apex (we control domain) or subdomain (delegated from parent)
func HandleDNSAdd() {
	// Check for help flag first
	for _, arg := range os.Args[3:] {
		if arg == "--help" || arg == "-h" {
			printDNSAddHelp()
			os.Exit(0)
		}
	}

	if len(os.Args) < 5 {
		printDNSAddHelp()
		os.Exit(1)
	}

	zoneType := os.Args[3] // "apex" or "subdomain"
	domain := os.Args[4]
	var customerID string

	// Validate zone type
	if zoneType != "apex" && zoneType != "subdomain" {
		fmt.Fprintf(os.Stderr, "❌ Unknown zone type: %s\n", zoneType)
		fmt.Fprintf(os.Stderr, "   Use 'apex' or 'subdomain'\n\n")
		printDNSAddHelp()
		os.Exit(1)
	}

	// Parse flags
	for i := 5; i < len(os.Args); i++ {
		if os.Args[i] == "--customer" && i+1 < len(os.Args) {
			i++
			customerID = os.Args[i]
		}
	}

	// Get DNS provider
	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Display header
	fmt.Printf("\n🌐 Setting up DNS for %s\n", domain)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Create zone
	fmt.Printf("📦 Creating DNS zone...\n")
	zone, err := provider.CreateZone(ctx, dns.CreateZoneRequest{
		Name: domain,
		TTL:  86400,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create zone: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ Zone created: %s\n\n", zone.Name)

	// Success output
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ DNS zone ready!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Get nameservers
	nameservers := zone.Nameservers
	if len(nameservers) == 0 {
		nameservers = []string{
			"hydrogen.ns.hetzner.com",
			"oxygen.ns.hetzner.com",
			"helium.ns.hetzner.de",
		}
	}

	// Show type-specific instructions
	if zoneType == "apex" {
		printApexInstructions(domain, nameservers)
	} else {
		printSubdomainInstructions(domain, nameservers)
	}
}

func printApexInstructions(domain string, nameservers []string) {
	fmt.Printf("🔧 Update nameservers at your domain registrar:\n\n")
	for _, ns := range nameservers {
		fmt.Printf("   %s\n", ns)
	}

	fmt.Printf("\n🎯 What's next?\n\n")
	fmt.Printf("1. Log into your domain registrar\n")
	fmt.Printf("2. Replace existing nameservers with the ones above\n")
	fmt.Printf("3. Wait for propagation (up to 48 hours)\n\n")

	fmt.Printf("4. Create your infrastructure:\n")
	fmt.Printf("   morpheus plant\n\n")
}

func printSubdomainInstructions(domain string, nameservers []string) {
	// Extract parent domain (e.g., "experiencenet.customer.com" -> "customer.com")
	parts := splitDomain(domain)
	parent := "parent domain"
	if len(parts) >= 2 {
		parent = parts[len(parts)-2] + "." + parts[len(parts)-1]
	}

	fmt.Printf("🔧 Add NS records to the parent domain (%s):\n\n", parent)
	for _, ns := range nameservers {
		fmt.Printf("   %s  NS  %s\n", domain, ns)
	}

	fmt.Printf("\n🎯 What's next?\n\n")
	fmt.Printf("1. Log into DNS management for %s\n", parent)
	fmt.Printf("2. Add the NS records shown above\n")
	fmt.Printf("3. Wait for propagation (usually minutes)\n\n")

	fmt.Printf("4. Create your infrastructure:\n")
	fmt.Printf("   morpheus plant\n\n")
}

func splitDomain(domain string) []string {
	var parts []string
	current := ""
	for _, c := range domain {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// HandleDNSRemove handles "morpheus dns remove <domain>"
func HandleDNSRemove() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns remove <domain> [--customer ID]")
		os.Exit(1)
	}

	domain := os.Args[3]
	var customerID string

	for i := 4; i < len(os.Args); i++ {
		if os.Args[i] == "--customer" && i+1 < len(os.Args) {
			i++
			customerID = os.Args[i]
		}
	}

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("\n🗑️  Removing DNS zone: %s\n", domain)

	if err := provider.DeleteZone(ctx, domain); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to delete zone: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Zone deleted: %s\n\n", domain)
}

// HandleDNSStatus handles "morpheus dns status [domain]"
func HandleDNSStatus() {
	var customerID string
	var domain string

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--customer" && i+1 < len(os.Args) {
			i++
			customerID = os.Args[i]
		} else if !startsWithDash(os.Args[i]) && domain == "" {
			domain = os.Args[i]
		}
	}

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if domain != "" {
		// Show specific zone
		showZoneStatus(ctx, provider, domain)
	} else {
		// List all zones
		showAllZones(ctx, provider)
	}
}

func showZoneStatus(ctx context.Context, provider dns.Provider, domain string) {
	zone, err := provider.GetZone(ctx, domain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get zone: %s\n", err)
		os.Exit(1)
	}
	if zone == nil {
		fmt.Fprintf(os.Stderr, "❌ Zone not found: %s\n", domain)
		os.Exit(1)
	}

	fmt.Printf("\n🌐 DNS Zone: %s\n", zone.Name)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("📋 Zone Info:\n")
	fmt.Printf("   ID:   %s\n", zone.ID)
	fmt.Printf("   TTL:  %d seconds\n\n", zone.TTL)

	fmt.Printf("🔧 Nameservers:\n")
	for _, ns := range zone.Nameservers {
		fmt.Printf("   %s\n", ns)
	}

	// List records
	records, err := provider.ListRecords(ctx, domain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n⚠️  Failed to list records: %s\n", err)
		return
	}

	fmt.Printf("\n📝 Records (%d):\n", len(records))
	if len(records) == 0 {
		fmt.Printf("   (no records)\n")
	} else {
		for _, r := range records {
			name := r.Name
			if name == "" || name == "@" {
				name = domain
			} else {
				name = r.Name + "." + domain
			}
			fmt.Printf("   %-30s %-6s %s\n", name, r.Type, r.Value)
		}
	}
	fmt.Println()
}

func showAllZones(ctx context.Context, provider dns.Provider) {
	zones, err := provider.ListZones(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to list zones: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🌐 DNS Zones (%d)\n", len(zones))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(zones) == 0 {
		fmt.Printf("   No zones configured.\n\n")
		fmt.Printf("   Create one with:\n")
		fmt.Printf("   morpheus dns add example.com --ip 1.2.3.4\n\n")
		return
	}

	for _, z := range zones {
		fmt.Printf("   %s\n", z.Name)
	}
	fmt.Printf("\n   Use 'morpheus dns status <domain>' for details\n\n")
}

func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func printDNSAddHelp() {
	fmt.Println("Usage: morpheus dns add <type> <domain> [--customer ID]")
	fmt.Println()
	fmt.Println("Create a DNS zone in Hetzner DNS.")
	fmt.Println()
	fmt.Println("Types:")
	fmt.Println("  apex        You control the domain (update nameservers at registrar)")
	fmt.Println("  subdomain   Delegated from parent (add NS records to parent)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --customer ID    Use customer-specific DNS token")
	fmt.Println("  --help, -h       Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns add apex nimsforest.com")
	fmt.Println("  morpheus dns add subdomain experiencenet.customer.com --customer acme")
}
