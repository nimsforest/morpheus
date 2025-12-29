package updater

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nimsforest/morpheus/pkg/updater/version"
)

const (
	githubAPIURL = "https://api.github.com/repos/nimsforest/morpheus/releases/latest"
)

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	UpdateURL      string
	ReleaseNotes   string
	Available      bool
}

// Updater handles version checking and updates
type Updater struct {
	currentVersion string
}

// NewUpdater creates a new Updater instance
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
	}
}


// CheckForUpdate checks if a new version is available using native HTTP client
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	// Create HTTP client with timeout and proper TLS configuration
	client := createHTTPClient(30 * time.Second)

	// Create request
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "morpheus-updater")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Remove 'v' prefix if present
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.currentVersion, "v")

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		UpdateURL:      release.HTMLURL,
		ReleaseNotes:   release.Body,
		Available:      version.Compare(latestVersion, currentVersion) > 0,
	}

	return info, nil
}

// PerformUpdate downloads and installs the latest version
func (u *Updater) PerformUpdate() error {
	// Get update info first to know which version to download
	updateInfo, err := u.CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.Available {
		fmt.Println("Already on the latest version!")
		return nil
	}

	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks if any
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	// Determine platform and architecture
	platform := GetPlatform()
	binaryName := fmt.Sprintf("morpheus-%s-%s", runtime.GOOS, runtime.GOARCH)

	// Construct download URL
	version := "v" + updateInfo.LatestVersion
	downloadURL := fmt.Sprintf("https://github.com/nimsforest/morpheus/releases/download/%s/%s", version, binaryName)

	fmt.Printf("📦 Downloading Morpheus %s for %s...\n", version, platform)

	// Download binary to temporary file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "morpheus-update")

	if err := downloadFile(downloadURL, tmpFile); err != nil {
		return fmt.Errorf("failed to download binary: %w\n\nFallback: You can manually download from:\n%s", err, updateInfo.UpdateURL)
	}

	// Verify downloaded file is not empty
	fileInfo, err := os.Stat(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to check downloaded file: %w", err)
	}
	if fileInfo.Size() == 0 {
		os.Remove(tmpFile)
		return fmt.Errorf("downloaded file is empty")
	}

	// Make it executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Verify the binary works (skip verification on platforms where exec may fail)
	fmt.Println("🔍 Verifying downloaded binary...")
	if !isRestrictedEnvironment() {
		verifyCmd := exec.Command(tmpFile, "version")
		if output, err := verifyCmd.CombinedOutput(); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("downloaded binary verification failed: %w\nOutput: %s", err, string(output))
		}
	} else {
		fmt.Println("⚠️  Skipping verification on restricted environment (Termux/Android)")
	}

	// Backup current version
	backupPath := execPath + ".backup"
	fmt.Printf("📋 Backing up current version to %s\n", backupPath)

	// Remove old backup if it exists
	os.Remove(backupPath)

	// Rename current binary to backup (this works even if the binary is running)
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to backup current version: %w", err)
	}

	// Replace current binary with new one using atomic rename
	fmt.Printf("✨ Installing update to %s\n", execPath)
	if err := os.Rename(tmpFile, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		os.Remove(tmpFile)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up temporary file
	os.Remove(tmpFile)

	fmt.Println("\n✅ Update completed successfully!")
	fmt.Printf("\nRun 'morpheus version' to verify the update.\n")
	fmt.Printf("Backup of previous version saved at: %s\n", backupPath)

	return nil
}

// downloadFile downloads a file from a URL to a local path using native HTTP client
func downloadFile(url, filepath string) error {
	// Create HTTP client with timeout and proper TLS configuration
	client := createHTTPClient(5 * time.Minute) // Longer timeout for binary downloads

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "morpheus-updater")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetPlatform returns the current platform string
func GetPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
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
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo/Arch
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/ssl/cert.pem",                                 // Alpine/OpenBSD
		"/usr/local/share/certs/ca-root-nss.crt",            // FreeBSD
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/certs/ca-certificates.crt",                    // Alternative
		// Additional paths
		"/usr/share/ca-certificates/cacert.org/cacert.org_root.crt",
		"/etc/ca-certificates/extracted/tls-ca-bundle.pem",  // Arch alternative
	}
}
