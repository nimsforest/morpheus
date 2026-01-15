package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nimsforest/morpheus/pkg/customer"
	"github.com/nimsforest/morpheus/pkg/dns"
	"github.com/nimsforest/morpheus/pkg/dns/hetzner"
)

func handleDNSZone() {
	if len(os.Args) < 4 {
		printDNSZoneHelp()
		os.Exit(1)
	}

	subcommand := os.Args[3]
	switch subcommand {
	case "create":
		handleDNSZoneCreate()
	case "list":
		handleDNSZoneList()
	case "delete":
		handleDNSZoneDelete()
	case "get":
		handleDNSZoneGet()
	case "help", "--help", "-h":
		printDNSZoneHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown dns zone subcommand: %s\n\n", subcommand)
		printDNSZoneHelp()
		os.Exit(1)
	}
}

func printDNSZoneHelp() {
	fmt.Println("DNS Zone Management - Manage DNS zones via Hetzner")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus dns zone <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create <zone-name>   Create a new DNS zone")
	fmt.Println("  list                 List all DNS zones")
	fmt.Println("  get <zone-name>      Get details of a DNS zone")
	fmt.Println("  delete <zone-name>   Delete a DNS zone")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --ttl <seconds>      TTL for the zone (default: 86400)")
	fmt.Println("  --customer <id>      Use customer-specific DNS token from customers.yaml")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns zone create example.com")
	fmt.Println("  morpheus dns zone create example.com --ttl 3600")
	fmt.Println("  morpheus dns zone list")
	fmt.Println("  morpheus dns zone list --customer acme")
	fmt.Println("  morpheus dns zone get example.com")
	fmt.Println("  morpheus dns zone delete example.com")
}

// parseDNSZoneFlags parses --ttl and --customer flags from os.Args starting at startIdx
func parseDNSZoneFlags(startIdx int) (ttl int, customerID string) {
	ttl = 86400 // default TTL

	for i := startIdx; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--ttl":
			if i+1 < len(os.Args) {
				if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
					ttl = val
				}
				i++
			}
		case "--customer":
			if i+1 < len(os.Args) {
				customerID = os.Args[i+1]
				i++
			}
		}
	}
	return
}

// getDNSProvider creates a Hetzner DNS provider based on the token source
func getDNSProvider(customerID string) (*hetzner.Provider, error) {
	var token string

	if customerID != "" {
		// Load customer-specific token
		custConfig, err := customer.LoadCustomerConfig(customer.GetDefaultConfigPath())
		if err != nil {
			return nil, fmt.Errorf("failed to load customer config: %w", err)
		}

		cust, err := customer.GetCustomer(custConfig, customerID)
		if err != nil {
			return nil, err
		}

		token = customer.ResolveToken(cust.Hetzner.APIToken)
		if token == "" {
			return nil, fmt.Errorf("customer %q has no API token configured", customerID)
		}
	} else {
		// Try to load from config first
		cfg, err := LoadConfig()
		if err == nil {
			token = cfg.GetDNSToken()
		}

		// Fall back to environment variables if no config or no token in config
		if token == "" {
			// Check for dedicated DNS token first
			token = os.Getenv("HETZNER_DNS_TOKEN")
			// Fall back to Cloud API token
			if token == "" {
				token = os.Getenv("HETZNER_API_TOKEN")
			}
		}

		if token == "" {
			return nil, fmt.Errorf("no API token configured. Set HETZNER_DNS_TOKEN or HETZNER_API_TOKEN env var, or use config file")
		}
	}

	return hetzner.NewProvider(token)
}

func handleDNSZoneCreate() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns zone create <zone-name> [--ttl N] [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  morpheus dns zone create example.com")
		fmt.Fprintln(os.Stderr, "  morpheus dns zone create example.com --ttl 3600")
		os.Exit(1)
	}

	zoneName := os.Args[4]
	ttl, customerID := parseDNSZoneFlags(5)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Creating DNS zone: %s\n", zoneName)

	zone, err := provider.CreateZone(ctx, dns.CreateZoneRequest{
		Name: zoneName,
		TTL:  ttl,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create zone: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("Zone created successfully!\n")
	fmt.Printf("  ID:   %s\n", zone.ID)
	fmt.Printf("  Name: %s\n", zone.Name)
	fmt.Printf("  TTL:  %d\n", zone.TTL)
	if len(zone.Nameservers) > 0 {
		fmt.Printf("  Nameservers:\n")
		for _, ns := range zone.Nameservers {
			fmt.Printf("    - %s\n", ns)
		}
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Update your domain registrar to use the nameservers above")
	fmt.Println("  2. Add DNS records with: morpheus dns record create <fqdn> <type> <value>")
}

func handleDNSZoneList() {
	_, customerID := parseDNSZoneFlags(4)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zones, err := provider.ListZones(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list zones: %s\n", err)
		os.Exit(1)
	}

	if len(zones) == 0 {
		fmt.Println("No DNS zones found.")
		fmt.Println()
		fmt.Println("Create a zone with: morpheus dns zone create <zone-name>")
		return
	}

	fmt.Println("DNS Zones")
	fmt.Println("=========")
	fmt.Println()

	for _, zone := range zones {
		fmt.Printf("  %s\n", zone.Name)
		fmt.Printf("    ID:  %s\n", zone.ID)
		fmt.Printf("    TTL: %d\n", zone.TTL)
		if len(zone.Nameservers) > 0 {
			fmt.Printf("    NS:  %s\n", zone.Nameservers[0])
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d zone(s)\n", len(zones))
}

func handleDNSZoneGet() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns zone get <zone-name> [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  morpheus dns zone get example.com")
		os.Exit(1)
	}

	zoneName := os.Args[4]
	_, customerID := parseDNSZoneFlags(5)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zone, err := provider.GetZone(ctx, zoneName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get zone: %s\n", err)
		os.Exit(1)
	}

	if zone == nil {
		fmt.Fprintf(os.Stderr, "Zone not found: %s\n", zoneName)
		os.Exit(1)
	}

	fmt.Printf("Zone: %s\n", zone.Name)
	fmt.Println("======" + repeatChar('=', len(zone.Name)))
	fmt.Println()
	fmt.Printf("  ID:   %s\n", zone.ID)
	fmt.Printf("  TTL:  %d\n", zone.TTL)
	if len(zone.Nameservers) > 0 {
		fmt.Println("  Nameservers:")
		for _, ns := range zone.Nameservers {
			fmt.Printf("    - %s\n", ns)
		}
	}
	fmt.Println()
	fmt.Println("View records with: morpheus dns record list " + zoneName)
}

func handleDNSZoneDelete() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns zone delete <zone-name> [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  morpheus dns zone delete example.com")
		os.Exit(1)
	}

	zoneName := os.Args[4]
	_, customerID := parseDNSZoneFlags(5)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, check if the zone exists
	zone, err := provider.GetZone(ctx, zoneName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get zone: %s\n", err)
		os.Exit(1)
	}

	if zone == nil {
		fmt.Printf("Zone not found: %s\n", zoneName)
		os.Exit(0)
	}

	fmt.Printf("Deleting DNS zone: %s\n", zoneName)

	err = provider.DeleteZone(ctx, zoneName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete zone: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Zone deleted successfully: %s\n", zoneName)
}

// repeatChar returns a string with the given character repeated n times
func repeatChar(char rune, n int) string {
	result := make([]rune, n)
	for i := range result {
		result[i] = char
	}
	return string(result)
}
