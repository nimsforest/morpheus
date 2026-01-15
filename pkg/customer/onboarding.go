package customer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// HetznerNameservers are the Hetzner DNS nameservers
var HetznerNameservers = []string{
	"hydrogen.ns.hetzner.com",
	"oxygen.ns.hetzner.com",
	"helium.ns.hetzner.de",
}

// GenerateNSInstructions returns instructions for customer to add NS records
func GenerateNSInstructions(subdomain string) string {
	var sb strings.Builder

	sb.WriteString("To delegate DNS to Hetzner, add the following NS records at your domain registrar:\n")
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Domain: %s\n", subdomain))
	sb.WriteString("\n")
	sb.WriteString("  NS Records to add:\n")
	for _, ns := range HetznerNameservers {
		sb.WriteString(fmt.Sprintf("    %s  NS  %s\n", subdomain, ns))
	}
	sb.WriteString("\n")
	sb.WriteString("  Note: DNS propagation may take up to 48 hours, but usually completes within 1-4 hours.\n")
	sb.WriteString("  You can verify delegation with: morpheus customer verify <customer-id>\n")

	return sb.String()
}

// SaveCustomer saves or updates a customer in the config file
// If configPath doesn't exist, it will be created along with its parent directory
func SaveCustomer(configPath string, cust Customer) error {
	// Validate the customer
	if err := ValidateCustomer(&cust); err != nil {
		return fmt.Errorf("invalid customer: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing config or create new one
	var cfg CustomerConfig
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	// Update existing customer or add new one
	found := false
	for i := range cfg.Customers {
		if cfg.Customers[i].ID == cust.ID {
			cfg.Customers[i] = cust
			found = true
			break
		}
	}
	if !found {
		cfg.Customers = append(cfg.Customers, cust)
	}

	// Marshal and write
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// DeleteCustomer removes a customer from the config file
func DeleteCustomer(configPath string, customerID string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var cfg CustomerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Find and remove customer
	found := false
	newCustomers := make([]Customer, 0, len(cfg.Customers))
	for _, c := range cfg.Customers {
		if c.ID == customerID {
			found = true
		} else {
			newCustomers = append(newCustomers, c)
		}
	}

	if !found {
		return fmt.Errorf("customer %q not found", customerID)
	}

	cfg.Customers = newCustomers

	// Marshal and write
	data, err = yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// EnsureCustomerConfigDir ensures the customer config directory exists
func EnsureCustomerConfigDir() error {
	configPath := GetDefaultConfigPath()
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0755)
}

// GenerateOnboardingChecklist returns a checklist of steps for onboarding a customer
func GenerateOnboardingChecklist(cust Customer) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Onboarding Checklist for %s (%s)\n", cust.ID, cust.Domain))
	sb.WriteString(strings.Repeat("-", 50) + "\n")
	sb.WriteString("\n")
	sb.WriteString("[ ] 1. Get Hetzner DNS API token from customer\n")
	sb.WriteString("[ ] 2. Configure customer in Morpheus (morpheus customer init)\n")
	sb.WriteString("[ ] 3. Customer adds NS records at their registrar\n")
	sb.WriteString("[ ] 4. Verify NS delegation (morpheus customer verify)\n")
	sb.WriteString("[ ] 5. Create DNS zone in Hetzner DNS\n")
	sb.WriteString("[ ] 6. Test DNS record creation\n")

	return sb.String()
}

// FormatCustomerInfo returns a formatted string with customer information
func FormatCustomerInfo(cust *Customer) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Customer: %s\n", cust.ID))
	if cust.Name != "" {
		sb.WriteString(fmt.Sprintf("  Name:     %s\n", cust.Name))
	}
	sb.WriteString(fmt.Sprintf("  Domain:   %s\n", cust.Domain))

	if cust.Hetzner.ProjectID != "" {
		sb.WriteString(fmt.Sprintf("  Project:  %s\n", cust.Hetzner.ProjectID))
	}

	if cust.Hetzner.DNSToken != "" {
		// Show token status but mask the actual value
		if strings.HasPrefix(cust.Hetzner.DNSToken, "${") {
			sb.WriteString(fmt.Sprintf("  DNS Token: %s (env var reference)\n", cust.Hetzner.DNSToken))
		} else {
			masked := maskToken(cust.Hetzner.DNSToken)
			sb.WriteString(fmt.Sprintf("  DNS Token: %s\n", masked))
		}
	}

	if len(cust.Ventures) > 0 {
		sb.WriteString(fmt.Sprintf("  Ventures: %s\n", strings.Join(cust.Ventures, ", ")))
	}

	return sb.String()
}

// maskToken masks a token for display, showing only first and last few characters
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
