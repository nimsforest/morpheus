package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/customer"
	"github.com/nimsforest/morpheus/pkg/dns"
	dnshetzner "github.com/nimsforest/morpheus/pkg/dns/hetzner"
	"github.com/nimsforest/morpheus/pkg/venture"
)

// HandleVenture handles the venture command and its subcommands
func HandleVenture() {
	if len(os.Args) < 3 {
		printVentureHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		handleVentureList()
	case "enable":
		handleVentureEnable()
	case "disable":
		handleVentureDisable()
	case "status":
		handleVentureStatus()
	case "help", "--help", "-h":
		printVentureHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown venture subcommand: %s\n\n", subcommand)
		printVentureHelp()
		os.Exit(1)
	}
}

// printVentureHelp prints the help message for venture commands
func printVentureHelp() {
	fmt.Println("Usage: morpheus venture <subcommand> [options]")
	fmt.Println()
	fmt.Println("Manage venture services for customers")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list                              List available venture templates")
	fmt.Println("  enable <customer-id> <venture>    Enable a venture for a customer")
	fmt.Println("    --server-ip IP                  Server IP address for DNS records")
	fmt.Println("  disable <customer-id> <venture>   Disable a venture for a customer")
	fmt.Println("    --delete-zone                   Also delete the DNS zone")
	fmt.Println("  status <customer-id> <venture>    Show venture DNS status")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus venture list")
	fmt.Println("  morpheus venture enable acme experiencenet --server-ip 1.2.3.4")
	fmt.Println("  morpheus venture disable acme experiencenet")
	fmt.Println("  morpheus venture status acme experiencenet")
}

// handleVentureList lists all available venture templates
func handleVentureList() {
	templates := venture.ListTemplates()

	fmt.Println("Available Venture Templates")
	fmt.Println("============================")
	fmt.Println()

	for _, template := range templates {
		fmt.Printf("Venture: %s\n", template.Name)
		fmt.Printf("  Description: %s\n", template.Description)
		fmt.Printf("  DNS Records:\n")
		for _, record := range template.Records {
			fmt.Printf("    - %s (%s) -> %s (TTL: %d)\n",
				record.Name, record.Type, record.Value, record.TTL)
		}
		fmt.Println()
	}

	fmt.Println("To enable a venture for a customer:")
	fmt.Println("  morpheus venture enable <customer-id> <venture-name> --server-ip <IP>")
}

// handleVentureEnable enables a venture for a customer
func handleVentureEnable() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Error: missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: morpheus venture enable <customer-id> <venture-name> [--server-ip IP]")
		os.Exit(1)
	}

	customerID := os.Args[3]
	ventureName := os.Args[4]

	// Parse optional flags
	var serverIP string
	for i := 5; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--server-ip", "-ip":
			if i+1 < len(os.Args) {
				serverIP = os.Args[i+1]
				i++
			} else {
				fmt.Fprintln(os.Stderr, "Error: --server-ip requires a value")
				os.Exit(1)
			}
		}
	}

	// Validate venture name
	_, err := venture.GetTemplate(ventureName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nAvailable ventures:")
		for _, name := range venture.ListVentureNames() {
			fmt.Fprintf(os.Stderr, "  - %s\n", name)
		}
		os.Exit(1)
	}

	// Load customer configuration
	cust, err := loadCustomer(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading customer: %v\n", err)
		os.Exit(1)
	}

	// Create DNS provider for customer
	dnsProvider, err := createDNSProviderForCustomer(cust)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating DNS provider: %v\n", err)
		os.Exit(1)
	}

	// Create provisioner
	provisioner := venture.NewProvisioner(dnsProvider)

	// Build venture domain
	ventureDomain := venture.GetVentureDomain(cust.Domain, ventureName)

	// Prepare variables for template expansion
	vars := make(map[string]string)
	if serverIP != "" {
		vars["ServerIP"] = serverIP
	}

	fmt.Printf("Enabling venture %s for customer %s\n", ventureName, customerID)
	fmt.Printf("Venture domain: %s\n", ventureDomain)
	fmt.Println()

	// Check if server IP is required but not provided
	template, _ := venture.GetTemplate(ventureName)
	needsServerIP := false
	for _, record := range template.Records {
		if strings.Contains(record.Value, "{{.ServerIP}}") {
			needsServerIP = true
			break
		}
	}
	if needsServerIP && serverIP == "" {
		fmt.Fprintln(os.Stderr, "Error: --server-ip is required for this venture template")
		fmt.Fprintln(os.Stderr, "The template contains A records that need a server IP address")
		os.Exit(1)
	}

	// Provision DNS records
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Provisioning DNS records...")
	result, err := provisioner.ProvisionRecords(ctx, ventureName, ventureDomain, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error provisioning DNS records: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	if result.ZoneCreated {
		fmt.Printf("Created new DNS zone: %s\n", ventureDomain)
	} else {
		fmt.Printf("Using existing DNS zone: %s\n", ventureDomain)
	}

	fmt.Println()
	fmt.Printf("Created %d DNS records:\n", len(result.Records))
	for _, record := range result.Records {
		recordName := record.Name
		if recordName == "@" {
			recordName = ventureDomain
		} else {
			recordName = record.Name + "." + ventureDomain
		}
		fmt.Printf("  %s (%s) -> %s\n", recordName, record.Type, record.Value)
	}

	// Print NS record instructions
	if len(result.Nameservers) > 0 {
		fmt.Println()
		fmt.Println("IMPORTANT: DNS Delegation Required")
		fmt.Println("===================================")
		fmt.Printf("Add the following NS records to your parent domain (%s):\n\n", cust.Domain)
		fmt.Printf("  Subdomain: %s\n", ventureName)
		fmt.Println("  Record Type: NS")
		fmt.Println("  Values:")
		for _, ns := range result.Nameservers {
			fmt.Printf("    - %s\n", ns)
		}
		fmt.Println()
		fmt.Println("This delegates DNS authority for the venture subdomain to Hetzner DNS.")
		fmt.Println("DNS propagation may take up to 48 hours.")
	}

	fmt.Println()
	fmt.Printf("Venture %s enabled successfully for customer %s\n", ventureName, customerID)
}

// handleVentureDisable disables a venture for a customer
func handleVentureDisable() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Error: missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: morpheus venture disable <customer-id> <venture-name> [--delete-zone]")
		os.Exit(1)
	}

	customerID := os.Args[3]
	ventureName := os.Args[4]

	// Parse optional flags
	deleteZone := false
	for i := 5; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--delete-zone":
			deleteZone = true
		}
	}

	// Validate venture name
	_, err := venture.GetTemplate(ventureName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load customer configuration
	cust, err := loadCustomer(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading customer: %v\n", err)
		os.Exit(1)
	}

	// Create DNS provider for customer
	dnsProvider, err := createDNSProviderForCustomer(cust)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating DNS provider: %v\n", err)
		os.Exit(1)
	}

	// Create provisioner
	provisioner := venture.NewProvisioner(dnsProvider)

	// Build venture domain
	ventureDomain := venture.GetVentureDomain(cust.Domain, ventureName)

	fmt.Printf("Disabling venture %s for customer %s\n", ventureName, customerID)
	fmt.Printf("Venture domain: %s\n", ventureDomain)
	if deleteZone {
		fmt.Println("Note: Zone will be deleted")
	}
	fmt.Println()

	// Cleanup DNS records
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Cleaning up DNS records...")
	err = provisioner.CleanupRecords(ctx, ventureName, ventureDomain, deleteZone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning up DNS records: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	if deleteZone {
		fmt.Printf("Venture %s disabled and zone deleted for customer %s\n", ventureName, customerID)
	} else {
		fmt.Printf("Venture %s disabled for customer %s\n", ventureName, customerID)
		fmt.Println("Note: DNS zone was preserved. Use --delete-zone to remove it.")
	}

	fmt.Println()
	fmt.Println("REMINDER: Remove NS Records")
	fmt.Println("============================")
	fmt.Printf("Don't forget to remove the NS records from your parent domain (%s)\n", cust.Domain)
	fmt.Printf("that point to the %s subdomain.\n", ventureName)
}

// handleVentureStatus shows the DNS status for a venture
func handleVentureStatus() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Error: missing required arguments")
		fmt.Fprintln(os.Stderr, "Usage: morpheus venture status <customer-id> <venture-name>")
		os.Exit(1)
	}

	customerID := os.Args[3]
	ventureName := os.Args[4]

	// Validate venture name
	template, err := venture.GetTemplate(ventureName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load customer configuration
	cust, err := loadCustomer(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading customer: %v\n", err)
		os.Exit(1)
	}

	// Create DNS provider for customer
	dnsProvider, err := createDNSProviderForCustomer(cust)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating DNS provider: %v\n", err)
		os.Exit(1)
	}

	// Build venture domain
	ventureDomain := venture.GetVentureDomain(cust.Domain, ventureName)

	fmt.Printf("Venture Status: %s\n", ventureName)
	fmt.Printf("Customer: %s\n", customerID)
	fmt.Printf("Domain: %s\n", ventureDomain)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check zone existence
	zone, err := dnsProvider.GetZone(ctx, ventureDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking zone: %v\n", err)
		os.Exit(1)
	}

	if zone == nil {
		fmt.Println("Status: NOT ENABLED")
		fmt.Println()
		fmt.Printf("No DNS zone found for %s\n", ventureDomain)
		fmt.Println()
		fmt.Println("To enable this venture:")
		fmt.Printf("  morpheus venture enable %s %s --server-ip <IP>\n", customerID, ventureName)
		return
	}

	fmt.Println("Status: ENABLED")
	fmt.Println()
	fmt.Println("Zone Information:")
	fmt.Printf("  Zone ID: %s\n", zone.ID)
	fmt.Printf("  Zone Name: %s\n", zone.Name)
	fmt.Printf("  Default TTL: %d\n", zone.TTL)
	if len(zone.Nameservers) > 0 {
		fmt.Println("  Nameservers:")
		for _, ns := range zone.Nameservers {
			fmt.Printf("    - %s\n", ns)
		}
	}

	// List current records
	records, err := dnsProvider.ListRecords(ctx, ventureDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nWarning: Could not list records: %v\n", err)
	} else {
		fmt.Println()
		fmt.Printf("Current DNS Records (%d):\n", len(records))
		for _, record := range records {
			recordName := record.Name
			if recordName == "@" {
				recordName = ventureDomain
			} else {
				recordName = record.Name + "." + ventureDomain
			}
			fmt.Printf("  %s (%s) -> %s (TTL: %d)\n",
				recordName, record.Type, record.Value, record.TTL)
		}
	}

	// Show expected records from template
	fmt.Println()
	fmt.Println("Expected Records from Template:")
	for _, record := range template.Records {
		recordName := record.Name
		if recordName == "@" {
			recordName = ventureDomain
		} else {
			recordName = record.Name + "." + ventureDomain
		}
		fmt.Printf("  %s (%s) -> %s (TTL: %d)\n",
			recordName, record.Type, record.Value, record.TTL)
	}
}

// loadCustomer loads a customer by ID from the default config path
func loadCustomer(customerID string) (*customer.Customer, error) {
	configPath := customer.GetDefaultConfigPath()

	cfg, err := customer.LoadCustomerConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load customer config from %s: %w", configPath, err)
	}

	cust, err := customer.GetCustomer(cfg, customerID)
	if err != nil {
		return nil, err
	}

	// Validate customer
	if err := customer.ValidateCustomer(cust); err != nil {
		return nil, err
	}

	return cust, nil
}

// createDNSProviderForCustomer creates a DNS provider for a specific customer
func createDNSProviderForCustomer(cust *customer.Customer) (dns.Provider, error) {
	if cust == nil {
		return nil, fmt.Errorf("customer is nil")
	}

	token := customer.ResolveToken(cust.Hetzner.APIToken)
	if token == "" {
		return nil, fmt.Errorf("no API token configured for customer %s", cust.ID)
	}

	return dnshetzner.NewProvider(token)
}
