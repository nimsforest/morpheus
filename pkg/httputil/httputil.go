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
