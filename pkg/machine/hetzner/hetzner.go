package hetzner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nimsforest/morpheus/pkg/httputil"
	"github.com/nimsforest/morpheus/pkg/machine"
)

// Provider implements the Provider interface for Hetzner Cloud
type Provider struct {
	client *hcloud.Client
}

// NewProvider creates a new Hetzner Cloud provider
func NewProvider(apiToken string) (*Provider, error) {
	// Sanitize the token by removing any invalid characters
	apiToken = sanitizeAPIToken(apiToken)

	if apiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	// Validate the token contains only valid characters
	if err := validateAPIToken(apiToken); err != nil {
		return nil, err
	}

	// Create HTTP client with proper TLS configuration and DNS resolver
	// This is essential for environments like Termux where default DNS may not work
	httpClient := httputil.CreateHTTPClient(30 * time.Second)

	client := hcloud.NewClient(
		hcloud.WithToken(apiToken),
		hcloud.WithHTTPClient(httpClient),
	)

	return &Provider{
		client: client,
	}, nil
}

// sanitizeAPIToken removes invalid characters from the API token.
// This handles common issues like:
// - Leading/trailing whitespace and newlines
// - Carriage returns (\r) from Windows-style line endings
// - BOM (Byte Order Mark) characters
// - Other non-printable control characters
func sanitizeAPIToken(token string) string {
	// First, trim any whitespace (including newlines) from the edges
	token = strings.TrimSpace(token)

	// Remove any remaining control characters and non-ASCII characters
	// that shouldn't be in an API token
	var sanitized strings.Builder
	sanitized.Grow(len(token))

	for _, r := range token {
		// Only keep printable ASCII characters (0x21-0x7E)
		// This excludes space (0x20) and DEL (0x7F) as well as control characters
		if r >= 0x21 && r <= 0x7E {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}

// validateAPIToken checks if the token contains only valid characters
// for HTTP Authorization headers. Returns an error with details if invalid.
func validateAPIToken(token string) error {
	if token == "" {
		return fmt.Errorf("API token is empty")
	}

	var invalidChars []string
	for i, r := range token {
		// Valid token characters are printable ASCII (0x21-0x7E)
		if r < 0x21 || r > 0x7E {
			// Format the character for display
			var charDesc string
			switch r {
			case '\n':
				charDesc = "newline (\\n)"
			case '\r':
				charDesc = "carriage return (\\r)"
			case '\t':
				charDesc = "tab (\\t)"
			case ' ':
				charDesc = "space"
			case 0xFEFF:
				charDesc = "BOM (byte order mark)"
			default:
				if r < 0x20 {
					charDesc = fmt.Sprintf("control character (0x%02X)", r)
				} else {
					charDesc = fmt.Sprintf("non-ASCII character (U+%04X)", r)
				}
			}
			invalidChars = append(invalidChars, fmt.Sprintf("%s at position %d", charDesc, i))
		}
	}

	if len(invalidChars) > 0 {
		return fmt.Errorf("API token contains invalid characters: %s. "+
			"Please check that the token was copied correctly without extra whitespace or special characters",
			strings.Join(invalidChars, ", "))
	}

	return nil
}

// wrapAuthError checks if the error is an authentication error and wraps it with helpful information
func wrapAuthError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if this is an unauthorized error from the Hetzner API
	if hcloud.IsError(err, hcloud.ErrorCodeUnauthorized) {
		return fmt.Errorf("%s: %w\n\n"+
			"This usually means:\n"+
			"  1. The API token is invalid, revoked, or expired\n"+
			"  2. The token was copied incorrectly (missing characters)\n"+
			"  3. The token doesn't have the required permissions\n\n"+
			"To fix this:\n"+
			"  - Go to Hetzner Cloud Console: https://console.hetzner.cloud/\n"+
			"  - Navigate to your project → Security → API Tokens\n"+
			"  - Generate a new token with 'Read & Write' permissions\n"+
			"  - Update your token: export HETZNER_API_TOKEN=\"your_new_token\"",
			operation, err)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// CreateServer provisions a new Hetzner Cloud server
func (p *Provider) CreateServer(ctx context.Context, req machine.CreateServerRequest) (*machine.Server, error) {
	// Resolve server type
	serverType, _, err := p.client.ServerType.GetByName(ctx, req.ServerType)
	if err != nil {
		return nil, wrapAuthError(err, "failed to get server type")
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", req.ServerType)
	}

	// Resolve image
	image, _, err := p.client.Image.GetByName(ctx, req.Image)
	if err != nil {
		return nil, wrapAuthError(err, "failed to get image")
	}
	if image == nil {
		return nil, fmt.Errorf("image not found: %s", req.Image)
	}

	// Resolve location
	location, _, err := p.client.Location.GetByName(ctx, req.Location)
	if err != nil {
		return nil, wrapAuthError(err, "failed to get location")
	}
	if location == nil {
		return nil, fmt.Errorf("location not found: %s", req.Location)
	}

	// Resolve SSH keys (automatically upload if not found)
	var sshKeys []*hcloud.SSHKey
	for _, keyName := range req.SSHKeys {
		key, err := p.ensureSSHKey(ctx, keyName)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure SSH key %s: %w", keyName, err)
		}
		sshKeys = append(sshKeys, key)
	}

	// Create server with IPv6 only by default (no IPv4 to save costs)
	// If EnableIPv4 is set, provision with both IPv4 and IPv6 for fallback support
	createOpts := hcloud.ServerCreateOpts{
		Name:             req.Name,
		ServerType:       serverType,
		Image:            image,
		Location:         location,
		SSHKeys:          sshKeys,
		UserData:         req.UserData,
		Labels:           req.Labels,
		StartAfterCreate: hcloud.Ptr(true),
		PublicNet: &hcloud.ServerCreatePublicNet{
			EnableIPv4: req.EnableIPv4,
			EnableIPv6: true,
		},
	}

	result, _, err := p.client.Server.Create(ctx, createOpts)
	if err != nil {
		return nil, wrapAuthError(err, "failed to create server")
	}

	return convertServer(result.Server), nil
}

// GetServer retrieves server information by ID
func (p *Provider) GetServer(ctx context.Context, serverID string) (*machine.Server, error) {
	server, _, err := p.client.Server.GetByID(ctx, parseServerID(serverID))
	if err != nil {
		return nil, wrapAuthError(err, "failed to get server")
	}
	if server == nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	return convertServer(server), nil
}

// DeleteServer removes a server
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	server, _, err := p.client.Server.GetByID(ctx, parseServerID(serverID))
	if err != nil {
		return wrapAuthError(err, "failed to get server")
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverID)
	}

	_, _, err = p.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return wrapAuthError(err, "failed to delete server")
	}

	return nil
}

// WaitForServer waits until the server is in the specified state
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state machine.ServerState) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for server to reach state: %s", state)
		case <-ticker.C:
			server, err := p.GetServer(ctx, serverID)
			if err != nil {
				return err
			}

			if server.State == state {
				return nil
			}

			// Log progress
			fmt.Printf("Server %s current state: %s, waiting for: %s\n",
				serverID, server.State, state)
		}
	}
}

// ListServers lists all servers with optional filters
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*machine.Server, error) {
	opts := hcloud.ServerListOpts{}

	if len(filters) > 0 {
		opts.LabelSelector = formatLabelSelector(filters)
	}

	servers, err := p.client.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, wrapAuthError(err, "failed to list servers")
	}

	result := make([]*machine.Server, len(servers))
	for i, server := range servers {
		result[i] = convertServer(server)
	}

	return result, nil
}

// CheckLocationAvailability checks if a server type is available in a specific location
// by checking the server type's pricing information which lists all supported locations
func (p *Provider) CheckLocationAvailability(ctx context.Context, locationName, serverTypeName string) (bool, error) {
	// Get server type with pricing info
	serverType, _, err := p.client.ServerType.GetByName(ctx, serverTypeName)
	if err != nil {
		return false, wrapAuthError(err, "failed to get server type")
	}
	if serverType == nil {
		return false, fmt.Errorf("server type not found: %s", serverTypeName)
	}

	// Check if the location is in the server type's supported locations
	for _, pricing := range serverType.Pricings {
		if pricing.Location != nil && pricing.Location.Name == locationName {
			return true, nil
		}
	}

	return false, nil
}

// GetAvailableLocations returns a list of locations where the server type is available
func (p *Provider) GetAvailableLocations(ctx context.Context, serverTypeName string) ([]string, error) {
	serverType, _, err := p.client.ServerType.GetByName(ctx, serverTypeName)
	if err != nil {
		return nil, wrapAuthError(err, "failed to get server type")
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", serverTypeName)
	}

	var locations []string
	for _, pricing := range serverType.Pricings {
		if pricing.Location != nil {
			locations = append(locations, pricing.Location.Name)
		}
	}

	return locations, nil
}

// ValidateServerType checks if a server type exists in Hetzner's API
func (p *Provider) ValidateServerType(ctx context.Context, serverTypeName string) (bool, error) {
	serverType, _, err := p.client.ServerType.GetByName(ctx, serverTypeName)
	if err != nil {
		return false, wrapAuthError(err, "failed to validate server type")
	}
	return serverType != nil, nil
}

// ListAvailableServerTypes returns all server types available from Hetzner
func (p *Provider) ListAvailableServerTypes(ctx context.Context) ([]string, error) {
	serverTypes, err := p.client.ServerType.All(ctx)
	if err != nil {
		return nil, wrapAuthError(err, "failed to list server types")
	}

	var names []string
	for _, st := range serverTypes {
		names = append(names, st.Name)
	}
	return names, nil
}

// GetServerTypeInfo returns detailed information about a server type
func (p *Provider) GetServerTypeInfo(ctx context.Context, serverTypeName string) (*ServerTypeInfo, error) {
	serverType, _, err := p.client.ServerType.GetByName(ctx, serverTypeName)
	if err != nil {
		return nil, wrapAuthError(err, "failed to get server type info")
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", serverTypeName)
	}

	var locations []string
	for _, pricing := range serverType.Pricings {
		if pricing.Location != nil {
			locations = append(locations, pricing.Location.Name)
		}
	}

	return &ServerTypeInfo{
		Name:         serverType.Name,
		Description:  serverType.Description,
		Cores:        serverType.Cores,
		Memory:       serverType.Memory,
		Disk:         serverType.Disk,
		Architecture: string(serverType.Architecture),
		Locations:    locations,
	}, nil
}

// ServerTypeInfo contains information about a Hetzner server type
type ServerTypeInfo struct {
	Name         string
	Description  string
	Cores        int
	Memory       float32
	Disk         int
	Architecture string
	Locations    []string
}

// FilterLocationsByServerType filters the given locations to only include those
// where the specified server type is available
func (p *Provider) FilterLocationsByServerType(ctx context.Context, locations []string, serverTypeName string) ([]string, []string, error) {
	availableLocations, err := p.GetAvailableLocations(ctx, serverTypeName)
	if err != nil {
		return nil, nil, err
	}

	// Build a set of available locations for fast lookup
	availableSet := make(map[string]bool)
	for _, loc := range availableLocations {
		availableSet[loc] = true
	}

	var supported, unsupported []string
	for _, loc := range locations {
		if availableSet[loc] {
			supported = append(supported, loc)
		} else {
			unsupported = append(unsupported, loc)
		}
	}

	return supported, unsupported, nil
}

// CheckSSHKeyExists checks if an SSH key with the given name exists in Hetzner Cloud.
// Returns true if the key exists, false otherwise.
func (p *Provider) CheckSSHKeyExists(ctx context.Context, keyName string) (bool, error) {
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return false, wrapAuthError(err, "failed to query SSH key")
	}
	return key != nil, nil
}

// SSHKeyInfo contains information about an SSH key from Hetzner Cloud
type SSHKeyInfo struct {
	Name        string
	Fingerprint string
	PublicKey   string
}

// GetSSHKeyInfo retrieves detailed information about an SSH key from Hetzner Cloud.
// Returns nil if the key doesn't exist.
func (p *Provider) GetSSHKeyInfo(ctx context.Context, keyName string) (*SSHKeyInfo, error) {
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return nil, wrapAuthError(err, "failed to query SSH key")
	}
	if key == nil {
		return nil, nil
	}
	return &SSHKeyInfo{
		Name:        key.Name,
		Fingerprint: key.Fingerprint,
		PublicKey:   key.PublicKey,
	}, nil
}

// DeleteSSHKey deletes an SSH key from Hetzner Cloud by name.
// Returns nil if the key was deleted or didn't exist.
func (p *Provider) DeleteSSHKey(ctx context.Context, keyName string) error {
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return wrapAuthError(err, "failed to query SSH key")
	}
	if key == nil {
		// Key doesn't exist, nothing to delete
		return nil
	}
	_, err = p.client.SSHKey.Delete(ctx, key)
	if err != nil {
		return wrapAuthError(err, "failed to delete SSH key")
	}
	return nil
}

// ensureSSHKey checks if an SSH key exists in Hetzner Cloud by name.
// If not found, it attempts to read from common SSH key locations and upload it.
// Returns the SSH key from Hetzner Cloud.
func (p *Provider) ensureSSHKey(ctx context.Context, keyName string) (*hcloud.SSHKey, error) {
	// First, check if the key already exists in Hetzner
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return nil, wrapAuthError(err, "failed to query SSH key")
	}
	if key != nil {
		// Key already exists
		fmt.Printf("      ✓ SSH key '%s' found in Hetzner (ID: %d)\n", keyName, key.ID)
		return key, nil
	}

	// Key doesn't exist, try to upload it
	fmt.Printf("SSH key '%s' not found in Hetzner Cloud, attempting to upload...\n", keyName)

	// Try to read the public key from common locations
	publicKeyContent, err := readSSHPublicKey(keyName, "")
	if err != nil {
		return nil, fmt.Errorf("SSH key '%s' not found in Hetzner Cloud and could not read local key: %w", keyName, err)
	}

	// Upload the key to Hetzner
	opts := hcloud.SSHKeyCreateOpts{
		Name:      keyName,
		PublicKey: publicKeyContent,
	}

	key, _, err = p.client.SSHKey.Create(ctx, opts)
	if err != nil {
		return nil, wrapAuthError(err, "failed to upload SSH key")
	}

	fmt.Printf("✓ Successfully uploaded SSH key '%s' to Hetzner Cloud\n", keyName)
	return key, nil
}

// EnsureSSHKeyWithPath ensures an SSH key exists in Hetzner Cloud, with optional custom path.
// This is useful when you want to specify a specific SSH key file path.
func (p *Provider) EnsureSSHKeyWithPath(ctx context.Context, keyName, keyPath string) (*hcloud.SSHKey, error) {
	// First, check if the key already exists in Hetzner
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return nil, wrapAuthError(err, "failed to query SSH key")
	}
	if key != nil {
		// Key already exists
		return key, nil
	}

	// Key doesn't exist, try to upload it
	fmt.Printf("SSH key '%s' not found in Hetzner Cloud, attempting to upload...\n", keyName)

	// Try to read the public key from specified path or common locations
	publicKeyContent, err := readSSHPublicKey(keyName, keyPath)
	if err != nil {
		return nil, fmt.Errorf("SSH key '%s' not found in Hetzner Cloud and could not read local key: %w", keyName, err)
	}

	// Upload the key to Hetzner
	opts := hcloud.SSHKeyCreateOpts{
		Name:      keyName,
		PublicKey: publicKeyContent,
	}

	key, _, err = p.client.SSHKey.Create(ctx, opts)
	if err != nil {
		return nil, wrapAuthError(err, "failed to upload SSH key")
	}

	fmt.Printf("✓ Successfully uploaded SSH key '%s' to Hetzner Cloud\n", keyName)
	return key, nil
}

// readSSHPublicKey attempts to read an SSH public key from common locations.
// If customPath is provided and non-empty, it tries that first.
// Otherwise, it tries the following in order:
// 1. {customPath} (if provided)
// 2. ~/.ssh/{keyName}.pub
// 3. ~/.ssh/{keyName} (if it's already a .pub file path)
// 4. ~/.ssh/id_ed25519.pub
// 5. ~/.ssh/id_rsa.pub
func readSSHPublicKey(keyName, customPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)

	// Build list of paths to try
	var paths []string

	// If custom path is provided, try it first
	if customPath != "" {
		// Expand ~ if present
		if strings.HasPrefix(customPath, "~/") {
			customPath = strings.Replace(customPath, "~", homeDir, 1)
		}
		paths = append(paths, customPath)
	}

	// Add common locations
	paths = append(paths,
		fmt.Sprintf("%s/%s.pub", sshDir, keyName),
		fmt.Sprintf("%s/%s", sshDir, keyName),
	)

	// If keyName doesn't look like a default key, also try common defaults
	if keyName != "id_ed25519" && keyName != "id_rsa" && keyName != "id_ecdsa" {
		paths = append(paths,
			fmt.Sprintf("%s/id_ed25519.pub", sshDir),
			fmt.Sprintf("%s/id_rsa.pub", sshDir),
		)
	}

	var lastErr error
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			// Successfully read the file
			publicKey := strings.TrimSpace(string(content))
			if publicKey != "" && isValidSSHPublicKey(publicKey) {
				fmt.Printf("  Found SSH public key at: %s\n", path)
				return publicKey, nil
			}
			lastErr = fmt.Errorf("file exists but doesn't contain valid SSH public key: %s", path)
			continue
		}
		lastErr = err
	}

	return "", fmt.Errorf("could not find SSH public key in any of the expected locations: %w", lastErr)
}

// isValidSSHPublicKey performs basic validation on SSH public key format
func isValidSSHPublicKey(key string) bool {
	// SSH public keys typically start with ssh-rsa, ssh-ed25519, ecdsa-sha2-, etc.
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2-"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// Helper functions

func convertServer(server *hcloud.Server) *machine.Server {
	var publicIPv4, publicIPv6 string
	if server.PublicNet.IPv4.IP != nil {
		publicIPv4 = server.PublicNet.IPv4.IP.String()
	}
	if server.PublicNet.IPv6.IP != nil {
		// Hetzner returns the /64 network address (e.g., 2a01:4f8:c17:1234::)
		// The server's actual address is ::1 within that network
		ipv6Base := server.PublicNet.IPv6.IP.String()
		if strings.HasSuffix(ipv6Base, "::") {
			publicIPv6 = ipv6Base + "1"
		} else {
			publicIPv6 = ipv6Base
		}
	}

	return &machine.Server{
		ID:         fmt.Sprintf("%d", server.ID),
		Name:       server.Name,
		PublicIPv4: publicIPv4,
		PublicIPv6: publicIPv6,
		Location:   server.Datacenter.Location.Name,
		State:      convertServerState(server.Status),
		Labels:     server.Labels,
		CreatedAt:  server.Created.Format(time.RFC3339),
	}
}

func convertServerState(status hcloud.ServerStatus) machine.ServerState {
	switch status {
	case hcloud.ServerStatusStarting:
		return machine.ServerStateStarting
	case hcloud.ServerStatusRunning:
		return machine.ServerStateRunning
	case hcloud.ServerStatusStopping, hcloud.ServerStatusOff:
		return machine.ServerStateStopped
	case hcloud.ServerStatusDeleting:
		return machine.ServerStateDeleting
	default:
		return machine.ServerStateUnknown
	}
}

func parseServerID(id string) int64 {
	var serverID int64
	fmt.Sscanf(id, "%d", &serverID)
	return serverID
}

func formatLabelSelector(filters map[string]string) string {
	selector := ""
	first := true
	for key, value := range filters {
		if !first {
			selector += ","
		}
		selector += fmt.Sprintf("%s=%s", key, value)
		first = false
	}
	return selector
}
