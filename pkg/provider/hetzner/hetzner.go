package hetzner

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nimsforest/morpheus/pkg/provider"
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
	httpClient := createHTTPClient(30 * time.Second)

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

// CreateServer provisions a new Hetzner Cloud server
func (p *Provider) CreateServer(ctx context.Context, req provider.CreateServerRequest) (*provider.Server, error) {
	// Resolve server type
	serverType, _, err := p.client.ServerType.GetByName(ctx, req.ServerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %w", err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", req.ServerType)
	}

	// Resolve image
	image, _, err := p.client.Image.GetByName(ctx, req.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return nil, fmt.Errorf("image not found: %s", req.Image)
	}

	// Resolve location
	location, _, err := p.client.Location.GetByName(ctx, req.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
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

	// Create server
	createOpts := hcloud.ServerCreateOpts{
		Name:             req.Name,
		ServerType:       serverType,
		Image:            image,
		Location:         location,
		SSHKeys:          sshKeys,
		UserData:         req.UserData,
		Labels:           req.Labels,
		StartAfterCreate: hcloud.Ptr(true),
	}

	result, _, err := p.client.Server.Create(ctx, createOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return convertServer(result.Server), nil
}

// GetServer retrieves server information by ID
func (p *Provider) GetServer(ctx context.Context, serverID string) (*provider.Server, error) {
	server, _, err := p.client.Server.GetByID(ctx, parseServerID(serverID))
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
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
		return fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverID)
	}

	_, _, err = p.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}

// WaitForServer waits until the server is in the specified state
func (p *Provider) WaitForServer(ctx context.Context, serverID string, state provider.ServerState) error {
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
func (p *Provider) ListServers(ctx context.Context, filters map[string]string) ([]*provider.Server, error) {
	opts := hcloud.ServerListOpts{}

	if len(filters) > 0 {
		opts.LabelSelector = formatLabelSelector(filters)
	}

	servers, err := p.client.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	result := make([]*provider.Server, len(servers))
	for i, server := range servers {
		result[i] = convertServer(server)
	}

	return result, nil
}

// ensureSSHKey checks if an SSH key exists in Hetzner Cloud by name.
// If not found, it attempts to read from common SSH key locations and upload it.
// Returns the SSH key from Hetzner Cloud.
func (p *Provider) ensureSSHKey(ctx context.Context, keyName string) (*hcloud.SSHKey, error) {
	// First, check if the key already exists in Hetzner
	key, _, err := p.client.SSHKey.GetByName(ctx, keyName)
	if err != nil {
		return nil, fmt.Errorf("failed to query SSH key: %w", err)
	}
	if key != nil {
		// Key already exists
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
		return nil, fmt.Errorf("failed to upload SSH key: %w", err)
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
		return nil, fmt.Errorf("failed to query SSH key: %w", err)
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
		return nil, fmt.Errorf("failed to upload SSH key: %w", err)
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

func convertServer(server *hcloud.Server) *provider.Server {
	var publicIPv4, publicIPv6 string
	if server.PublicNet.IPv4.IP != nil {
		publicIPv4 = server.PublicNet.IPv4.IP.String()
	}
	if server.PublicNet.IPv6.IP != nil {
		publicIPv6 = server.PublicNet.IPv6.IP.String()
	}

	return &provider.Server{
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

func convertServerState(status hcloud.ServerStatus) provider.ServerState {
	switch status {
	case hcloud.ServerStatusStarting:
		return provider.ServerStateStarting
	case hcloud.ServerStatusRunning:
		return provider.ServerStateRunning
	case hcloud.ServerStatusStopping, hcloud.ServerStatusOff:
		return provider.ServerStateStopped
	case hcloud.ServerStatusDeleting:
		return provider.ServerStateDeleting
	default:
		return provider.ServerStateUnknown
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

// isRestrictedEnvironment detects if we're running in a restricted environment
// like Termux/Android where certain syscalls may not be available
func isRestrictedEnvironment() bool {
	// Check for Termux environment
	if os.Getenv("TERMUX_VERSION") != "" {
		return true
	}

	// Check if running on Android (Termux reports as linux but with android characteristics)
	if runtime.GOOS == "linux" {
		// Check for /system/bin/app_process which is Android-specific
		if _, err := os.Stat("/system/bin/app_process"); err == nil {
			return true
		}
		// Check for Termux directories
		if _, err := os.Stat("/data/data/com.termux"); err == nil {
			return true
		}
	}

	return false
}

// createCustomDialer creates a custom dialer with DNS resolver fallback for Termux/minimal distros
func createCustomDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	// Check if we need custom DNS (Termux/Android/minimal distros)
	needsCustomDNS := isRestrictedEnvironment()

	// Base dialer
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if !needsCustomDNS {
		// Use standard dialer for normal environments
		return dialer.DialContext
	}

	// Custom resolver using public DNS servers (Google 8.8.8.8, Cloudflare 1.1.1.1)
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			// Try Google DNS first
			conn, err := d.DialContext(ctx, "udp", "8.8.8.8:53")
			if err != nil {
				// Fallback to Cloudflare DNS
				conn, err = d.DialContext(ctx, "udp", "1.1.1.1:53")
			}
			if err != nil {
				// Last fallback to Quad9
				conn, err = d.DialContext(ctx, "udp", "9.9.9.9:53")
			}
			return conn, err
		},
	}

	// Return custom dial function that uses the custom resolver
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Resolve hostname using custom resolver
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed for %s: %w", host, err)
		}

		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses found for %s", host)
		}

		// Try each resolved IP
		var lastErr error
		for _, ip := range ips {
			resolvedAddr := net.JoinHostPort(ip.String(), port)
			conn, err := dialer.DialContext(ctx, network, resolvedAddr)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

// createHTTPClient creates an HTTP client with proper TLS configuration and DNS resolver for various environments
func createHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}

	// Create custom dialer (handles DNS for Termux/minimal distros)
	customDial := createCustomDialer()

	// For restricted environments (Termux/Android), be more aggressive with fallback
	// because SystemCertPool often returns empty/broken pools without errors
	if isRestrictedEnvironment() {
		return createHTTPClientForRestrictedEnv(client, customDial)
	}

	// For normal systems, try the standard approach
	rootCAs, err := x509.SystemCertPool()
	if err == nil && rootCAs != nil {
		// System cert pool loaded successfully
		client.Transport = &http.Transport{
			DialContext: customDial,
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}
		return client
	}

	// SystemCertPool failed, try manual loading from known paths
	rootCAs = x509.NewCertPool()
	certPaths := getCertPaths()

	for _, certPath := range certPaths {
		if certs, err := os.ReadFile(certPath); err == nil {
			rootCAs.AppendCertsFromPEM(certs)
		}
	}

	client.Transport = &http.Transport{
		DialContext: customDial,
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
	}
	return client
}

// createHTTPClientForRestrictedEnv creates an HTTP client optimized for Termux/Android
// where certificate handling is often problematic
func createHTTPClientForRestrictedEnv(client *http.Client, customDial func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Client {
	// Try to load certificates from known Termux/Linux paths
	rootCAs := x509.NewCertPool()
	certPaths := getCertPaths()

	loaded := false
	for _, certPath := range certPaths {
		if certs, err := os.ReadFile(certPath); err == nil {
			if rootCAs.AppendCertsFromPEM(certs) {
				loaded = true
			}
		}
	}

	// Also try system cert pool and merge
	if sysCAs, err := x509.SystemCertPool(); err == nil && sysCAs != nil {
		// We can't merge pools directly, but if system pool works, use it as base
		// and our manually loaded certs as supplement
		rootCAs = sysCAs
		// Re-add manual certs to system pool
		for _, certPath := range certPaths {
			if certs, err := os.ReadFile(certPath); err == nil {
				rootCAs.AppendCertsFromPEM(certs)
				loaded = true
			}
		}
	}

	if loaded {
		// We loaded some certificates, try using them
		client.Transport = &http.Transport{
			DialContext: customDial,
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}
		return client
	}

	// No certificates loaded - use insecure fallback with warning
	// This is the last resort for Termux without ca-certificates installed
	client.Transport = &http.Transport{
		DialContext: customDial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	fmt.Println("⚠️  Warning: Could not load TLS certificates, using insecure connection")
	fmt.Println("   To fix on Termux: pkg install ca-certificates")
	return client
}

// getCertPaths returns common certificate file locations across different distros
func getCertPaths() []string {
	return []string{
		// Termux-specific paths (check first for Termux)
		"/data/data/com.termux/files/usr/etc/tls/cert.pem",
		"/data/data/com.termux/files/usr/etc/ssl/certs/ca-certificates.crt",
		// Standard Linux paths
		"/etc/ssl/certs/ca-certificates.crt",               // Debian/Ubuntu/Gentoo/Arch
		"/etc/pki/tls/certs/ca-bundle.crt",                 // Fedora/RHEL
		"/etc/ssl/ca-bundle.pem",                           // OpenSUSE
		"/etc/ssl/cert.pem",                                // Alpine/OpenBSD
		"/usr/local/share/certs/ca-root-nss.crt",           // FreeBSD
		"/etc/pki/tls/cacert.pem",                          // OpenELEC
		"/etc/certs/ca-certificates.crt",                   // Alternative
		// Additional paths
		"/usr/share/ca-certificates/cacert.org/cacert.org_root.crt",
		"/etc/ca-certificates/extracted/tls-ca-bundle.pem", // Arch alternative
	}
}
