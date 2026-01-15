package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nimsforest/morpheus/pkg/customer"
	"github.com/nimsforest/morpheus/pkg/dns"
)

// HandleCustomer handles the customer command.
func HandleCustomer() {
	if len(os.Args) < 3 {
		printCustomerHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "init":
		handleCustomerInit()
	case "list":
		handleCustomerList()
	case "verify":
		handleCustomerVerify()
	case "help", "--help", "-h":
		printCustomerHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown customer subcommand: %s\n\n", subcommand)
		printCustomerHelp()
		os.Exit(1)
	}
}

func printCustomerHelp() {
	fmt.Println("👥 Morpheus Customer - Customer Onboarding Management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  morpheus customer <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init <customer-id>       Initialize a new customer")
	fmt.Println("    --domain <domain>      Customer's domain (required)")
	fmt.Println("    --name <name>          Customer display name (optional)")
	fmt.Println()
	fmt.Println("  list                     List all configured customers")
	fmt.Println()
	fmt.Println("  verify <customer-id>     Verify NS delegation for a customer")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  morpheus customer init acme --domain acme.example.com")
	fmt.Println("  morpheus customer init acme --domain acme.example.com --name \"ACME Corp\"")
	fmt.Println("  morpheus customer list")
	fmt.Println("  morpheus customer verify acme")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Customer data is stored in: ~/.morpheus/customers.yaml")
	fmt.Println()
	fmt.Println("DNS Delegation:")
	fmt.Println("  After initializing a customer, they need to add NS records at their")
	fmt.Println("  domain registrar pointing to Hetzner nameservers:")
	for _, ns := range customer.HetznerNameservers {
		fmt.Printf("    - %s\n", ns)
	}
}

func handleCustomerInit() {
	// Parse arguments
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Error: customer-id is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: morpheus customer init <customer-id> --domain <domain> [--name <name>]")
		os.Exit(1)
	}

	customerID := os.Args[3]
	var domain, name string

	// Parse flags
	args := os.Args[4:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--domain", "-d":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --domain requires a value")
				os.Exit(1)
			}
			i++
			domain = args[i]
		case "--name", "-n":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --name requires a value")
				os.Exit(1)
			}
			i++
			name = args[i]
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", args[i])
			os.Exit(1)
		}
	}

	if domain == "" {
		fmt.Fprintln(os.Stderr, "Error: --domain is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: morpheus customer init <customer-id> --domain <domain> [--name <name>]")
		os.Exit(1)
	}

	fmt.Println("👥 Customer Initialization")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  Customer ID: %s\n", customerID)
	fmt.Printf("  Domain:      %s\n", domain)
	if name != "" {
		fmt.Printf("  Name:        %s\n", name)
	}
	fmt.Println()

	// Ask for API token
	fmt.Println("📝 Hetzner API Token Configuration")
	fmt.Println()
	fmt.Println("  You can provide either:")
	fmt.Println("    1. A direct API token")
	fmt.Println("    2. An environment variable reference (e.g., ${ACME_API_TOKEN})")
	fmt.Println()
	fmt.Print("  Enter API token or env var reference: ")

	reader := bufio.NewReader(os.Stdin)
	tokenInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
		os.Exit(1)
	}
	tokenInput = strings.TrimSpace(tokenInput)

	if tokenInput == "" {
		fmt.Println()
		fmt.Println("  ⚠️  No token provided. You can add it later by editing:")
		fmt.Printf("     %s\n", customer.GetDefaultConfigPath())
	}

	// Create customer entry
	cust := customer.Customer{
		ID:     customerID,
		Name:   name,
		Domain: domain,
		Hetzner: customer.HetznerConfig{
			APIToken: tokenInput,
		},
	}

	// Save to config file
	configPath := customer.GetDefaultConfigPath()
	if err := customer.SaveCustomer(configPath, cust); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Failed to save customer: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✅ Customer saved to: %s\n", configPath)
	fmt.Println()

	// Print NS record instructions
	fmt.Println("📋 Next Steps: DNS Delegation")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println(customer.GenerateNSInstructions(domain))
}

func handleCustomerList() {
	fmt.Println("👥 Configured Customers")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	configPath := customer.GetDefaultConfigPath()
	cfg, err := customer.LoadCustomerConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No customers configured yet.")
			fmt.Println()
			fmt.Println("  Add a customer with:")
			fmt.Println("    morpheus customer init <customer-id> --domain <domain>")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "❌ Failed to load customer config: %s\n", err)
		os.Exit(1)
	}

	if len(cfg.Customers) == 0 {
		fmt.Println("  No customers configured yet.")
		fmt.Println()
		fmt.Println("  Add a customer with:")
		fmt.Println("    morpheus customer init <customer-id> --domain <domain>")
		os.Exit(0)
	}

	for i, cust := range cfg.Customers {
		if i > 0 {
			fmt.Println()
		}
		fmt.Print(customer.FormatCustomerInfo(&cust))
	}

	fmt.Println()
	fmt.Printf("Total: %d customer(s)\n", len(cfg.Customers))
	fmt.Printf("Config file: %s\n", configPath)
}

func handleCustomerVerify() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Error: customer-id is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: morpheus customer verify <customer-id>")
		os.Exit(1)
	}

	customerID := os.Args[3]

	// Load customer config
	configPath := customer.GetDefaultConfigPath()
	cfg, err := customer.LoadCustomerConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load customer config: %s\n", err)
		os.Exit(1)
	}

	cust, err := customer.GetCustomer(cfg, customerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %s\n", err)
		os.Exit(1)
	}

	fmt.Println("🔍 DNS Delegation Verification")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  Customer: %s\n", cust.ID)
	fmt.Printf("  Domain:   %s\n", cust.Domain)
	fmt.Println()
	fmt.Println("  Checking NS records...")
	fmt.Println()

	// Verify NS delegation
	result := dns.VerifyNSDelegation(cust.Domain, customer.HetznerNameservers)

	if result.Error != nil {
		fmt.Printf("  ❌ DNS lookup failed: %s\n", result.Error)
		fmt.Println()
		fmt.Println("  Possible causes:")
		fmt.Println("    - Domain does not exist")
		fmt.Println("    - NS records not yet propagated")
		fmt.Println("    - Network/DNS resolver issues")
		fmt.Println()
		fmt.Println("  Try again in a few minutes, or check with:")
		fmt.Printf("    dig NS %s\n", cust.Domain)
		os.Exit(1)
	}

	fmt.Println("  Expected nameservers:")
	for _, ns := range customer.HetznerNameservers {
		fmt.Printf("    - %s\n", ns)
	}
	fmt.Println()

	fmt.Println("  Actual nameservers found:")
	if len(result.ActualNS) == 0 {
		fmt.Println("    (none)")
	} else {
		for _, ns := range result.ActualNS {
			// Check if this NS matches expected
			matched := false
			for _, expected := range customer.HetznerNameservers {
				if strings.EqualFold(strings.TrimSuffix(ns, "."), strings.TrimSuffix(expected, ".")) {
					matched = true
					break
				}
			}
			if matched {
				fmt.Printf("    - %s ✓\n", ns)
			} else {
				fmt.Printf("    - %s\n", ns)
			}
		}
	}
	fmt.Println()

	if result.Delegated {
		fmt.Println("  ✅ DNS delegation is correctly configured!")
		fmt.Println()
		fmt.Println("  The domain is ready for DNS management through Hetzner.")
		fmt.Println("  Next steps:")
		fmt.Println("    1. Create the DNS zone in Hetzner (if not already done)")
		fmt.Println("    2. Start managing DNS records for this customer")
		os.Exit(0)
	} else if result.PartialMatch {
		fmt.Println("  ⚠️  Partial NS delegation detected")
		fmt.Println()
		fmt.Println("  Some nameservers match, but not all:")
		fmt.Printf("    Matching: %s\n", strings.Join(result.MatchingNS, ", "))
		fmt.Printf("    Missing:  %s\n", strings.Join(result.MissingNS, ", "))
		fmt.Println()
		fmt.Println("  Please ensure ALL required NS records are added at the registrar.")
		os.Exit(1)
	} else {
		fmt.Println("  ❌ DNS delegation NOT configured for Hetzner")
		fmt.Println()
		fmt.Println("  The domain is not pointing to Hetzner nameservers.")
		fmt.Println()
		fmt.Println("  To fix this, add NS records at your domain registrar:")
		fmt.Println()
		for _, ns := range customer.HetznerNameservers {
			fmt.Printf("    %s  NS  %s\n", cust.Domain, ns)
		}
		fmt.Println()
		fmt.Println("  Note: DNS propagation can take up to 48 hours.")
		fmt.Println("        Run this command again after making changes.")
		os.Exit(1)
	}
}
