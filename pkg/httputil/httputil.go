// Package httputil provides HTTP client utilities with proper TLS configuration
// and DNS resolver fallback for various environments including Termux/Android.
package httputil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

// IsRestrictedEnvironment detects if we're running in a restricted environment
// like Termux/Android where certain syscalls may not be available
func IsRestrictedEnvironment() bool {
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

// CreateCustomDialer creates a custom dialer with DNS resolver fallback for Termux/minimal distros
func CreateCustomDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	// Check if we need custom DNS (Termux/Android/minimal distros)
	needsCustomDNS := IsRestrictedEnvironment()

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

// CreateHTTPClient creates an HTTP client with proper TLS configuration and DNS resolver for various environments
func CreateHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}

	// Create custom dialer (handles DNS for Termux/minimal distros)
	customDial := CreateCustomDialer()

	// For restricted environments (Termux/Android), be more aggressive with fallback
	// because SystemCertPool often returns empty/broken pools without errors
	if IsRestrictedEnvironment() {
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
	certPaths := GetCertPaths()

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
	certPaths := GetCertPaths()

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

// GetCertPaths returns common certificate file locations across different distros
func GetCertPaths() []string {
	return []string{
		// Termux-specific paths (check first for Termux)
		"/data/data/com.termux/files/usr/etc/tls/cert.pem",
		"/data/data/com.termux/files/usr/etc/ssl/certs/ca-certificates.crt",
		// Standard Linux paths
		"/etc/ssl/certs/ca-certificates.crt",     // Debian/Ubuntu/Gentoo/Arch
		"/etc/pki/tls/certs/ca-bundle.crt",       // Fedora/RHEL
		"/etc/ssl/ca-bundle.pem",                 // OpenSUSE
		"/etc/ssl/cert.pem",                      // Alpine/OpenBSD
		"/usr/local/share/certs/ca-root-nss.crt", // FreeBSD
		"/etc/pki/tls/cacert.pem",                // OpenELEC
		"/etc/certs/ca-certificates.crt",         // Alternative
		// Additional paths
		"/usr/share/ca-certificates/cacert.org/cacert.org_root.crt",
		"/etc/ca-certificates/extracted/tls-ca-bundle.pem", // Arch alternative
	}
}

// IPv6CheckResult contains the results of an IPv6 connectivity check
type IPv6CheckResult struct {
	Available bool   // Whether IPv6 is available
	Address   string // The detected IPv6 address (if available)
	Error     error  // Any error encountered during the check
}

// CheckIPv6Connectivity checks if IPv6 connectivity is available by attempting
// to connect to an IPv6-only service. This is a native Go alternative to
// running "curl -6 ifconfig.co" which doesn't work on Termux due to certificate issues.
//
// It uses multiple fallback services to ensure reliability:
// - icanhazip.com (IPv6 endpoint)
// - ifconfig.co (IPv6 endpoint)
// - api64.ipify.org (IPv6-only service)
func CheckIPv6Connectivity(ctx context.Context) IPv6CheckResult {
	// Services that return your public IP address (IPv6-only endpoints)
	services := []string{
		"https://ipv6.icanhazip.com",
		"https://api64.ipify.org",
		"https://v6.ident.me",
	}

	// Create HTTP client with proper TLS and DNS configuration
	// This handles certificate issues on Termux and other restricted environments
	client := CreateHTTPClient(10 * time.Second)

	var lastErr error
	for _, serviceURL := range services {
		// Create request with IPv6-only context
		req, err := http.NewRequestWithContext(ctx, "GET", serviceURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request for %s: %w", serviceURL, err)
			continue
		}

		// Set user agent
		req.Header.Set("User-Agent", "Morpheus-IPv6-Check/1.0")

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to connect to %s: %w", serviceURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("service %s returned status %d", serviceURL, resp.StatusCode)
			continue
		}

		// Read the response (should be the IPv6 address)
		body := make([]byte, 256)
		n, err := resp.Body.Read(body)
		if err != nil && err.Error() != "EOF" {
			lastErr = fmt.Errorf("failed to read response from %s: %w", serviceURL, err)
			continue
		}

		ipv6Address := string(body[:n])
		ipv6Address = trimWhitespace(ipv6Address)

		// Validate that we got an IPv6 address
		if isValidIPv6(ipv6Address) {
			return IPv6CheckResult{
				Available: true,
				Address:   ipv6Address,
				Error:     nil,
			}
		}

		lastErr = fmt.Errorf("service %s returned invalid IPv6 address: %s", serviceURL, ipv6Address)
	}

	// All services failed
	return IPv6CheckResult{
		Available: false,
		Address:   "",
		Error:     fmt.Errorf("IPv6 connectivity check failed for all services: %w", lastErr),
	}
}

// isValidIPv6 checks if a string is a valid IPv6 address
func isValidIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	// Check if it's IPv6 (not IPv4)
	return ip.To4() == nil && ip.To16() != nil
}

// trimWhitespace removes whitespace, newlines, and control characters from a string
func trimWhitespace(s string) string {
	var result []rune
	for _, r := range s {
		// Only keep printable non-space characters
		if r > 32 && r < 127 || r == ':' {
			result = append(result, r)
		}
	}
	return string(result)
}
