package customer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadCustomerConfig loads customer configuration from a YAML file
func LoadCustomerConfig(path string) (*CustomerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read customer config file: %w", err)
	}

	var config CustomerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse customer config: %w", err)
	}

	return &config, nil
}

// GetCustomer returns a customer by ID from the configuration
func GetCustomer(cfg *CustomerConfig, id string) (*Customer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("customer config is nil")
	}

	for i := range cfg.Customers {
		if cfg.Customers[i].ID == id {
			return &cfg.Customers[i], nil
		}
	}

	// Build list of available customer IDs for helpful error message
	var available []string
	for _, c := range cfg.Customers {
		available = append(available, c.ID)
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("customer %q not found: no customers configured", id)
	}

	return nil, fmt.Errorf("customer %q not found, available customers: %s", id, strings.Join(available, ", "))
}

// ResolveToken resolves a token value, expanding environment variable references
// If the token starts with ${, it's treated as an environment variable reference
// e.g., ${ACME_DNS_TOKEN} -> os.Getenv("ACME_DNS_TOKEN")
func ResolveToken(token string) string {
	token = strings.TrimSpace(token)

	// Check if it's an environment variable reference
	if strings.HasPrefix(token, "${") && strings.HasSuffix(token, "}") {
		envVar := token[2 : len(token)-1]
		return strings.TrimSpace(os.Getenv(envVar))
	}

	return token
}

// GetDefaultConfigPath returns the default customer configuration file path
func GetDefaultConfigPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, ".morpheus", "customers.yaml")
}

// ListCustomers returns all customer IDs in the configuration
func ListCustomers(cfg *CustomerConfig) []string {
	if cfg == nil {
		return nil
	}

	ids := make([]string, len(cfg.Customers))
	for i, c := range cfg.Customers {
		ids[i] = c.ID
	}
	return ids
}

// ValidateCustomer checks if a customer configuration is valid
func ValidateCustomer(cust *Customer) error {
	if cust == nil {
		return fmt.Errorf("customer is nil")
	}

	if cust.ID == "" {
		return fmt.Errorf("customer ID is required")
	}

	if cust.Domain == "" {
		return fmt.Errorf("customer %q: domain is required", cust.ID)
	}

	return nil
}

// ValidateCustomerConfig validates the entire customer configuration
func ValidateCustomerConfig(cfg *CustomerConfig) error {
	if cfg == nil {
		return fmt.Errorf("customer config is nil")
	}

	seenIDs := make(map[string]bool)
	for _, c := range cfg.Customers {
		if err := ValidateCustomer(&c); err != nil {
			return err
		}

		if seenIDs[c.ID] {
			return fmt.Errorf("duplicate customer ID: %s", c.ID)
		}
		seenIDs[c.ID] = true
	}

	return nil
}
