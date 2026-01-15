package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/dns"
)

func handleDNSRecord() {
	if len(os.Args) < 4 {
		printDNSRecordHelp()
		os.Exit(1)
	}

	subcommand := os.Args[3]
	switch subcommand {
	case "create":
		handleDNSRecordCreate()
	case "list":
		handleDNSRecordList()
	case "delete":
		handleDNSRecordDelete()
	case "help", "--help", "-h":
		printDNSRecordHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown dns record subcommand: %s\n\n", subcommand)
		printDNSRecordHelp()
		os.Exit(1)
	}
}

func printDNSRecordHelp() {
	fmt.Println("DNS Record Management - Manage DNS records via Hetzner")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus dns record <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create <fqdn> <type> <value>   Create a DNS record")
	fmt.Println("  list <zone>                    List records in a zone")
	fmt.Println("  delete <fqdn> <type>           Delete a DNS record")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --ttl <seconds>      TTL for the record (default: 300)")
	fmt.Println("  --customer <id>      Use customer-specific DNS token from customers.yaml")
	fmt.Println()
	fmt.Println("Record Types:")
	fmt.Println("  A        IPv4 address record")
	fmt.Println("  AAAA     IPv6 address record")
	fmt.Println("  CNAME    Canonical name (alias)")
	fmt.Println("  TXT      Text record")
	fmt.Println("  SRV      Service record")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns record create www.example.com A 1.2.3.4")
	fmt.Println("  morpheus dns record create mail.example.com AAAA 2001:db8::1")
	fmt.Println("  morpheus dns record create blog.example.com CNAME www.example.com")
	fmt.Println("  morpheus dns record create www.example.com A 1.2.3.4 --ttl 3600")
	fmt.Println("  morpheus dns record list example.com")
	fmt.Println("  morpheus dns record list example.com --customer acme")
	fmt.Println("  morpheus dns record delete www.example.com A")
}

// parseDNSRecordFlags parses --ttl and --customer flags from os.Args starting at startIdx
func parseDNSRecordFlags(startIdx int) (ttl int, customerID string) {
	ttl = 300 // default TTL for records

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

// parseZoneFromFQDN extracts the zone (domain) and record name from an FQDN
// For example: "www.example.com" -> zone="example.com", name="www"
// For example: "sub.host.example.com" -> zone="example.com", name="sub.host"
// For example: "example.com" -> zone="example.com", name="@"
func parseZoneFromFQDN(fqdn string) (zone, name string) {
	fqdn = strings.TrimSuffix(fqdn, ".")
	parts := strings.Split(fqdn, ".")

	if len(parts) < 2 {
		// Single label, not a valid domain
		return fqdn, "@"
	}

	if len(parts) == 2 {
		// Just the zone itself (e.g., "example.com")
		return fqdn, "@"
	}

	// Get the last two parts as the zone (e.g., "example.com")
	zone = strings.Join(parts[len(parts)-2:], ".")
	// Everything before that is the record name
	name = strings.Join(parts[:len(parts)-2], ".")

	return zone, name
}

func handleDNSRecordCreate() {
	if len(os.Args) < 7 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns record create <fqdn> <type> <value> [--ttl N] [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  morpheus dns record create www.example.com A 1.2.3.4")
		fmt.Fprintln(os.Stderr, "  morpheus dns record create mail.example.com AAAA 2001:db8::1")
		fmt.Fprintln(os.Stderr, "  morpheus dns record create blog.example.com CNAME www.example.com")
		os.Exit(1)
	}

	fqdn := os.Args[4]
	recordType := strings.ToUpper(os.Args[5])
	value := os.Args[6]
	ttl, customerID := parseDNSRecordFlags(7)

	// Validate record type
	validTypes := map[string]bool{"A": true, "AAAA": true, "CNAME": true, "TXT": true, "SRV": true, "MX": true, "NS": true}
	if !validTypes[recordType] {
		fmt.Fprintf(os.Stderr, "Invalid record type: %s\n", recordType)
		fmt.Fprintln(os.Stderr, "Valid types: A, AAAA, CNAME, TXT, SRV, MX, NS")
		os.Exit(1)
	}

	zone, name := parseZoneFromFQDN(fqdn)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Creating DNS record: %s %s %s\n", fqdn, recordType, value)
	fmt.Printf("  Zone: %s\n", zone)
	fmt.Printf("  Name: %s\n", name)

	record, err := provider.CreateRecord(ctx, dns.CreateRecordRequest{
		Domain: zone,
		Name:   name,
		Type:   dns.RecordType(recordType),
		Value:  value,
		TTL:    ttl,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create record: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Record created successfully!")
	fmt.Printf("  ID:    %s\n", record.ID)
	fmt.Printf("  FQDN:  %s\n", formatFQDN(record.Name, zone))
	fmt.Printf("  Type:  %s\n", record.Type)
	fmt.Printf("  Value: %s\n", record.Value)
	fmt.Printf("  TTL:   %d\n", record.TTL)
}

func handleDNSRecordList() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns record list <zone> [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  morpheus dns record list example.com")
		os.Exit(1)
	}

	zone := os.Args[4]
	_, customerID := parseDNSRecordFlags(5)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	records, err := provider.ListRecords(ctx, zone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list records: %s\n", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Printf("No DNS records found in zone: %s\n", zone)
		fmt.Println()
		fmt.Println("Create a record with: morpheus dns record create <fqdn> <type> <value>")
		return
	}

	fmt.Printf("DNS Records for %s\n", zone)
	fmt.Println("=" + repeatChar('=', len(zone)+16))
	fmt.Println()

	// Group records by name
	recordsByName := make(map[string][]*dns.Record)
	var names []string

	for _, record := range records {
		if _, exists := recordsByName[record.Name]; !exists {
			names = append(names, record.Name)
		}
		recordsByName[record.Name] = append(recordsByName[record.Name], record)
	}

	// Print records grouped by name
	for _, name := range names {
		recs := recordsByName[name]
		fqdn := formatFQDN(name, zone)
		fmt.Printf("  %s\n", fqdn)
		for _, rec := range recs {
			fmt.Printf("    %-6s %-40s  TTL: %d\n", rec.Type, rec.Value, rec.TTL)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d record(s)\n", len(records))
}

func handleDNSRecordDelete() {
	if len(os.Args) < 6 {
		fmt.Fprintln(os.Stderr, "Usage: morpheus dns record delete <fqdn> <type> [--customer ID]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  morpheus dns record delete www.example.com A")
		os.Exit(1)
	}

	fqdn := os.Args[4]
	recordType := strings.ToUpper(os.Args[5])
	_, customerID := parseDNSRecordFlags(6)

	zone, name := parseZoneFromFQDN(fqdn)

	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Deleting DNS record: %s %s\n", fqdn, recordType)

	err = provider.DeleteRecord(ctx, zone, name, recordType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete record: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Record deleted successfully: %s %s\n", fqdn, recordType)
}

// formatFQDN formats a record name and zone into an FQDN
func formatFQDN(name, zone string) string {
	if name == "@" || name == "" {
		return zone
	}
	return name + "." + zone
}
