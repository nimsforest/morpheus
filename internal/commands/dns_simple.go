package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/config"
	"github.com/nimsforest/morpheus/pkg/customer"
	"github.com/nimsforest/morpheus/pkg/dns"
	"github.com/nimsforest/morpheus/pkg/dns/hetzner"
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

	zoneType := os.Args[3] // "apex", "subdomain", or "gmail-mx"
	domain := os.Args[4]
	var customerID string

	// Parse flags first
	for i := 5; i < len(os.Args); i++ {
		if os.Args[i] == "--customer" && i+1 < len(os.Args) {
			i++
			customerID = os.Args[i]
		}
	}

	// Handle gmail-mx as a special case (adds MX records to existing zone)
	if zoneType == "gmail-mx" || zoneType == "gmail" {
		handleAddGmailMX(domain, customerID)
		return
	}

	// Validate zone type
	if zoneType != "apex" && zoneType != "subdomain" {
		fmt.Fprintf(os.Stderr, "❌ Unknown zone type: %s\n", zoneType)
		fmt.Fprintf(os.Stderr, "   Use 'apex', 'subdomain', or 'gmail-mx'\n\n")
		printDNSAddHelp()
		os.Exit(1)
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
	fmt.Printf("   ✓ Zone created: %s\n", zone.Name)

	// Save domain to config for plant integration (only for our own zones, not customer zones)
	if customerID == "" {
		if err := saveDomainToConfig(domain); err != nil {
			fmt.Printf("   ⚠️  Could not save domain to config: %s\n", err)
		} else {
			fmt.Printf("   ✓ Domain saved to config (plant will auto-add DNS records)\n")
		}
	}
	fmt.Println()

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

	fmt.Printf("\n📧 Using Gmail/Google Workspace for email?\n")
	fmt.Printf("   Set up complete email configuration BEFORE changing nameservers:\n")
	fmt.Printf("   morpheus dns add gmail-mx %s\n", domain)
	fmt.Printf("   (Adds MX, SPF, and DMARC records)\n\n")

	fmt.Printf("🎯 What's next?\n\n")
	fmt.Printf("1. Log into your domain registrar\n")
	fmt.Printf("2. Replace existing nameservers with the ones above\n")
	fmt.Printf("3. Wait for propagation (up to 48 hours)\n\n")

	fmt.Printf("4. Verify NS delegation:\n")
	fmt.Printf("   morpheus dns verify %s\n\n", domain)

	fmt.Printf("5. Create your infrastructure:\n")
	fmt.Printf("   morpheus plant\n")
	fmt.Printf("   (DNS records will be added automatically)\n\n")
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

	fmt.Printf("4. Verify NS delegation:\n")
	fmt.Printf("   morpheus dns verify %s\n\n", domain)

	fmt.Printf("5. Create your infrastructure:\n")
	fmt.Printf("   morpheus plant\n")
	fmt.Printf("   (DNS records will be added automatically)\n\n")
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

// saveDomainToConfig saves the DNS domain to config file
func saveDomainToConfig(domain string) error {
	configPath := config.FindConfigPath()
	if configPath == "" {
		if err := config.EnsureConfigDir(); err != nil {
			return err
		}
		configPath = config.GetDefaultConfigPath()
	}
	return config.SetConfigValue(configPath, "dns_domain", domain)
}

func printDNSAddHelp() {
	fmt.Println("Usage: morpheus dns add <type> <domain> [--customer ID]")
	fmt.Println()
	fmt.Println("Create a DNS zone or add records in Hetzner DNS.")
	fmt.Println()
	fmt.Println("Types:")
	fmt.Println("  apex        You control the domain (update nameservers at registrar)")
	fmt.Println("  subdomain   Delegated from parent (add NS records to parent)")
	fmt.Println("  gmail-mx    Complete Gmail/Google Workspace setup (MX, SPF, DMARC)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --customer ID    Use customer-specific DNS token")
	fmt.Println("  --help, -h       Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns add apex nimsforest.com")
	fmt.Println("  morpheus dns add subdomain experiencenet.customer.com --customer acme")
	fmt.Println("  morpheus dns add gmail-mx nimsforest.com")
	fmt.Println()
	fmt.Println("Note: gmail-mx adds MX records, SPF, and DMARC. DKIM requires")
	fmt.Println("      additional setup in Google Workspace Admin Console.")
}

// GmailMXRecords contains the standard Gmail/Google Workspace MX records
var GmailMXRecords = []struct {
	Priority int
	Server   string
}{
	{1, "ASPMX.L.GOOGLE.COM"},
	{5, "ALT1.ASPMX.L.GOOGLE.COM"},
	{5, "ALT2.ASPMX.L.GOOGLE.COM"},
	{10, "ALT3.ASPMX.L.GOOGLE.COM"},
	{10, "ALT4.ASPMX.L.GOOGLE.COM"},
}

// createGmailMXRRSet creates an RRSet with all Gmail MX records
func createGmailMXRRSet(ctx context.Context, provider *hetzner.Provider, domain string) error {
	// We need to create all MX records in a single RRSet via direct API call
	// since the Cloud API treats name+type as a unique RRSet
	records := make([]map[string]interface{}, len(GmailMXRecords))
	for i, mx := range GmailMXRecords {
		records[i] = map[string]interface{}{
			"value": fmt.Sprintf("%d %s", mx.Priority, mx.Server),
		}
	}

	// Create the RRSet with all MX records
	return provider.CreateRRSet(ctx, domain, "@", "MX", 3600, records)
}

// handleAddGmailMX adds Gmail/Google Workspace MX records and email authentication records
func handleAddGmailMX(domain, customerID string) {
	provider, err := getDNSProvider(customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check if zone exists
	zone, err := provider.GetZone(ctx, domain)
	if err != nil || zone == nil {
		fmt.Fprintf(os.Stderr, "❌ Zone not found: %s\n", domain)
		fmt.Fprintf(os.Stderr, "   Create the zone first with: morpheus dns add apex %s\n", domain)
		os.Exit(1)
	}

	fmt.Printf("\n📧 Setting up Gmail/Google Workspace for %s\n", domain)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	totalRecords := 0
	failedRecords := 0

	// Add MX records - all MX records must be in a single RRSet
	fmt.Printf("📮 Adding MX records:\n")
	err = createGmailMXRRSet(ctx, provider, domain)
	totalRecords++
	if err != nil {
		fmt.Printf("   ❌ %s\n", err)
		failedRecords++
	} else {
		for _, mx := range GmailMXRecords {
			fmt.Printf("   ✓ MX %s (priority %d)\n", mx.Server, mx.Priority)
		}
	}

	// Add SPF record
	fmt.Printf("\n🔐 Adding SPF record:\n")
	spfValue := "\"v=spf1 include:_spf.google.com ~all\""
	fmt.Printf("   TXT @ %s...", spfValue)
	_, err = provider.CreateRecord(ctx, dns.CreateRecordRequest{
		Domain: domain,
		Name:   "@",
		Type:   dns.RecordType("TXT"),
		Value:  spfValue,
		TTL:    3600,
	})
	totalRecords++
	if err != nil {
		fmt.Printf(" ❌ %s\n", err)
		failedRecords++
	} else {
		fmt.Printf(" ✓\n")
	}

	// Add DMARC record
	fmt.Printf("\n📊 Adding DMARC record:\n")
	dmarcValue := fmt.Sprintf("\"v=DMARC1; p=none; rua=mailto:dmarc@%s\"", domain)
	fmt.Printf("   TXT _dmarc %s...", dmarcValue)
	_, err = provider.CreateRecord(ctx, dns.CreateRecordRequest{
		Domain: domain,
		Name:   "_dmarc",
		Type:   dns.RecordType("TXT"),
		Value:  dmarcValue,
		TTL:    3600,
	})
	totalRecords++
	if err != nil {
		fmt.Printf(" ❌ %s\n", err)
		failedRecords++
	} else {
		fmt.Printf(" ✓\n")
	}

	// Summary
	fmt.Println()
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	successCount := totalRecords - failedRecords
	if failedRecords == 0 {
		fmt.Printf("✅ All %d records added successfully!\n", totalRecords)
	} else if successCount > 0 {
		fmt.Printf("⚠️  Added %d of %d records (%d failed)\n", successCount, totalRecords, failedRecords)
	} else {
		fmt.Printf("❌ Failed to add records\n")
		os.Exit(1)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// DKIM setup instructions
	fmt.Println("🔑 DKIM Setup Required:")
	fmt.Println()
	fmt.Println("DKIM requires configuration in Google Workspace Admin Console:")
	fmt.Println()
	fmt.Println("1. Go to admin.google.com")
	fmt.Println("2. Navigate to Apps → Google Workspace → Gmail → Authenticate email")
	fmt.Println("3. Click 'Generate new record' for your domain")
	fmt.Println("4. Copy the DKIM TXT record values provided by Google")
	fmt.Println("5. Add the DKIM record using:")
	fmt.Printf("   morpheus dns record create <selector>._domainkey.%s TXT \"<dkim-value>\"\n", domain)
	fmt.Println()
	fmt.Println("   Example:")
	fmt.Printf("   morpheus dns record create google._domainkey.%s TXT \"v=DKIM1; k=rsa; p=MIGfMA...\"\n", domain)
	fmt.Println()
	fmt.Println("6. Return to Google Admin Console and click 'Start authentication'")
	fmt.Println()

	// Final instructions
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 What's been configured:")
	fmt.Println()
	fmt.Println("✓ MX records    - Routes email to Gmail servers")
	fmt.Println("✓ SPF record    - Authorizes Gmail to send email for your domain")
	fmt.Println("✓ DMARC record  - Email authentication policy (set to monitoring mode)")
	fmt.Println("⚠ DKIM record   - Requires manual setup (see instructions above)")
	fmt.Println()
	fmt.Println("📧 Your email will work once DNS propagates (usually within an hour).")
	fmt.Println()
	fmt.Println("Verify records with:")
	fmt.Printf("   morpheus dns status %s\n", domain)
	fmt.Println()
	fmt.Println("Test email authentication:")
	fmt.Println("   dig TXT " + domain + " +short")
	fmt.Println("   dig TXT _dmarc." + domain + " +short")
	fmt.Println()
}

// HandleDNSVerify handles "morpheus dns verify <domain>"
// Checks if NS records point to Hetzner nameservers
func HandleDNSVerify() {
	// Check for help flag first
	for _, arg := range os.Args[3:] {
		if arg == "--help" || arg == "-h" {
			printDNSVerifyHelp()
			os.Exit(0)
		}
	}

	if len(os.Args) < 4 {
		printDNSVerifyHelp()
		os.Exit(1)
	}

	domain := os.Args[3]

	fmt.Printf("\n🔍 Verifying DNS delegation for %s\n", domain)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("Checking NS records...\n\n")

	result := dns.VerifyNSDelegation(domain, customer.HetznerNameservers)

	if result.Error != nil {
		fmt.Printf("❌ DNS lookup failed: %s\n\n", result.Error)
		fmt.Println("Possible causes:")
		fmt.Println("  - Domain does not exist")
		fmt.Println("  - NS records not yet propagated")
		fmt.Println("  - Network/DNS resolver issues")
		fmt.Println()
		fmt.Println("Try again in a few minutes, or check with:")
		fmt.Printf("  dig NS %s\n\n", domain)
		os.Exit(1)
	}

	fmt.Println("Expected nameservers:")
	for _, ns := range customer.HetznerNameservers {
		fmt.Printf("   %s\n", ns)
	}
	fmt.Println()

	fmt.Println("Actual nameservers found:")
	if len(result.ActualNS) == 0 {
		fmt.Println("   (none)")
	} else {
		for _, ns := range result.ActualNS {
			status := "⚠️"
			for _, expected := range customer.HetznerNameservers {
				if dns.NormalizeNS(ns) == dns.NormalizeNS(expected) {
					status = "✓"
					break
				}
			}
			fmt.Printf("   %s %s\n", status, ns)
		}
	}
	fmt.Println()

	if result.Delegated {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✅ NS delegation verified!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("You can now create your infrastructure:")
		fmt.Println("  morpheus plant")
		fmt.Println()
	} else if result.PartialMatch {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Partial NS delegation")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Printf("Matching:  %v\n", result.MatchingNS)
		fmt.Printf("Missing:   %v\n", result.MissingNS)
		fmt.Println()
		fmt.Println("Some nameservers are configured but not all.")
		fmt.Println("This may still work, but check your registrar settings.")
		fmt.Println()
		os.Exit(1)
	} else {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("❌ NS delegation NOT configured")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("The domain's nameservers don't point to Hetzner.")
		fmt.Println()
		fmt.Println("For apex domains, update nameservers at your registrar.")
		fmt.Println("For subdomains, add NS records to the parent domain.")
		fmt.Println()
		fmt.Println("Then wait for propagation and try again:")
		fmt.Printf("  morpheus dns verify %s\n\n", domain)
		os.Exit(1)
	}
}

func printDNSVerifyHelp() {
	fmt.Println("Usage: morpheus dns verify <domain>")
	fmt.Println()
	fmt.Println("Verify that NS delegation is configured correctly.")
	fmt.Println("Checks if the domain's nameservers point to Hetzner DNS.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns verify nimsforest.com")
	fmt.Println("  morpheus dns verify experiencenet.customer.com")
}

// HandleDNSVerifyMX handles "morpheus dns verify-mx <domain>"
// Checks if MX records are configured correctly
func HandleDNSVerifyMX() {
	// Check for help flag first
	for _, arg := range os.Args[3:] {
		if arg == "--help" || arg == "-h" {
			printDNSVerifyMXHelp()
			os.Exit(0)
		}
	}

	if len(os.Args) < 4 {
		printDNSVerifyMXHelp()
		os.Exit(1)
	}

	domain := os.Args[3]
	provider := "gmail"

	// Parse flags
	for i := 4; i < len(os.Args); i++ {
		if os.Args[i] == "--provider" && i+1 < len(os.Args) {
			i++
			provider = os.Args[i]
		}
	}

	// Validate provider
	if provider != "gmail" {
		fmt.Fprintf(os.Stderr, "❌ Unknown provider: %s\n", provider)
		fmt.Fprintf(os.Stderr, "   Supported providers: gmail\n\n")
		printDNSVerifyMXHelp()
		os.Exit(1)
	}

	fmt.Printf("\n🔍 Verifying MX records for %s\n", domain)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("Checking MX records against %s configuration...\n\n", provider)

	// Convert GmailMXRecords to dns.MXRecord format
	expectedMX := make([]dns.MXRecord, len(GmailMXRecords))
	for i, mx := range GmailMXRecords {
		expectedMX[i] = dns.MXRecord{
			Priority: mx.Priority,
			Server:   mx.Server,
		}
	}

	result := dns.VerifyMXRecords(domain, expectedMX)

	if result.Error != nil {
		fmt.Printf("❌ MX lookup failed: %s\n\n", result.Error)
		fmt.Println("Possible causes:")
		fmt.Println("  - Domain does not exist")
		fmt.Println("  - No MX records configured")
		fmt.Println("  - Network/DNS resolver issues")
		fmt.Println()
		fmt.Println("Try again in a few minutes, or check with:")
		fmt.Printf("  dig MX %s\n\n", domain)
		os.Exit(1)
	}

	fmt.Println("Expected MX records (Gmail/Google Workspace):")
	for _, mx := range expectedMX {
		fmt.Printf("   %d %s\n", mx.Priority, mx.Server)
	}
	fmt.Println()

	fmt.Println("Actual MX records found:")
	if len(result.ActualMX) == 0 {
		fmt.Println("   (none)")
	} else {
		for _, mx := range result.ActualMX {
			status := "⚠️"
			for _, expected := range expectedMX {
				if mx.Priority == expected.Priority && strings.EqualFold(mx.Server, expected.Server) {
					status = "✓"
					break
				}
			}
			fmt.Printf("   %s %d %s\n", status, mx.Priority, mx.Server)
		}
	}
	fmt.Println()

	if result.Configured {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✅ MX records verified!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Printf("All Gmail MX records are correctly configured for %s\n", domain)
		fmt.Println()
		fmt.Println("Your email should be working correctly.")
		fmt.Println("If you experience issues, check SPF and DMARC records:")
		fmt.Printf("  dig TXT %s\n", domain)
		fmt.Printf("  dig TXT _dmarc.%s\n\n", domain)
	} else if result.PartialMatch {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Partial MX configuration")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		if len(result.MatchingMX) > 0 {
			fmt.Println("Matching:")
			for _, mx := range result.MatchingMX {
				fmt.Printf("  ✓ %d %s\n", mx.Priority, mx.Server)
			}
			fmt.Println()
		}
		if len(result.MissingMX) > 0 {
			fmt.Println("Missing:")
			for _, mx := range result.MissingMX {
				fmt.Printf("  ✗ %d %s\n", mx.Priority, mx.Server)
			}
			fmt.Println()
		}
		if len(result.ExtraMX) > 0 {
			fmt.Println("Extra (not in Gmail configuration):")
			for _, mx := range result.ExtraMX {
				fmt.Printf("  • %d %s\n", mx.Priority, mx.Server)
			}
			fmt.Println()
		}
		fmt.Println("Some MX records are correct but not all.")
		fmt.Println("Update your DNS configuration with the missing records:")
		fmt.Printf("  morpheus dns add gmail-mx %s\n\n", domain)
		os.Exit(1)
	} else {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("❌ MX records NOT configured for Gmail")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("The domain's MX records don't match Gmail/Google Workspace.")
		fmt.Println()
		fmt.Println("To set up Gmail MX records, run:")
		fmt.Printf("  morpheus dns add gmail-mx %s\n", domain)
		fmt.Println()
		fmt.Println("This will configure MX, SPF, and DMARC records.")
		fmt.Println()
		fmt.Println("Then wait for DNS propagation and verify again:")
		fmt.Printf("  morpheus dns verify-mx %s\n\n", domain)
		os.Exit(1)
	}
}

func printDNSVerifyMXHelp() {
	fmt.Println("Usage: morpheus dns verify-mx <domain> [--provider PROVIDER]")
	fmt.Println()
	fmt.Println("Verify that MX records are configured correctly.")
	fmt.Println("Checks if the domain's MX records match expected values.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --provider PROVIDER   Email provider to verify against (default: gmail)")
	fmt.Println("  --help, -h            Show this help")
	fmt.Println()
	fmt.Println("Supported providers:")
	fmt.Println("  gmail    Gmail/Google Workspace MX records")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus dns verify-mx nimsforest.com")
	fmt.Println("  morpheus dns verify-mx example.com --provider gmail")
}
