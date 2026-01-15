package customer

// Customer represents a customer configuration
type Customer struct {
	ID       string        `yaml:"id" json:"id"`
	Name     string        `yaml:"name" json:"name"`
	Domain   string        `yaml:"domain" json:"domain"`     // Root domain (e.g., "customer.com")
	Ventures []string      `yaml:"ventures" json:"ventures"` // Enabled ventures
	Hetzner  HetznerConfig `yaml:"hetzner" json:"hetzner"`
}

// HetznerConfig contains Hetzner-specific configuration
type HetznerConfig struct {
	ProjectID string `yaml:"project_id" json:"project_id"`
	DNSToken  string `yaml:"dns_token" json:"dns_token"` // Can be env var reference like ${ENV_VAR}
}

// CustomerConfig holds all customer configurations
type CustomerConfig struct {
	Customers []Customer `yaml:"customers" json:"customers"`
}
