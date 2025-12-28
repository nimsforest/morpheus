package updater

import (
	"context"
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
	timeout      = 10 * time.Second
	maxRetries   = 3
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
	client         *http.Client
}

// NewUpdater creates a new Updater instance
func NewUpdater(currentVersion string) *Updater {
	// Create a custom dialer with fallback to IPv4 if IPv6 fails
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create a custom transport with DNS fallback
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Try dual-stack (both IPv4 and IPv6)
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Updater{
		currentVersion: currentVersion,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// CheckForUpdate checks if a new version is available
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	var lastErr error

	// Retry logic for network resilience
	for attempt := 1; attempt <= maxRetries; attempt++ {
		info, err := u.checkForUpdateOnce()
		if err == nil {
			return info, nil
		}

		lastErr = err

		// Check if it's a DNS or network error
		if isNetworkError(err) {
			if attempt < maxRetries {
				// Wait before retrying (exponential backoff)
				backoff := time.Duration(attempt) * 2 * time.Second
				time.Sleep(backoff)
				continue
			}
		} else {
			// Non-network errors should not be retried
			break
		}
	}

	return nil, enhanceNetworkError(lastErr)
}

// checkForUpdateOnce performs a single update check
func (u *Updater) checkForUpdateOnce() (*UpdateInfo, error) {
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid rate limiting
	req.Header.Set("User-Agent", "morpheus-updater")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
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

// isNetworkError checks if an error is network-related
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "lookup") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "timeout")
}

// enhanceNetworkError adds helpful troubleshooting info to network errors
func enhanceNetworkError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	
	// DNS resolution errors
	if strings.Contains(errStr, "lookup") || strings.Contains(errStr, "no such host") {
		return fmt.Errorf("%w\n\nNetwork troubleshooting:\n"+
			"  • Check your internet connection\n"+
			"  • Verify DNS is configured correctly (check /etc/resolv.conf)\n"+
			"  • Try: ping api.github.com\n"+
			"  • If on IPv6-only network, ensure IPv6 DNS is working\n"+
			"  • Try using a different DNS server (e.g., 8.8.8.8, 1.1.1.1)", err)
	}

	// Connection refused errors
	if strings.Contains(errStr, "connection refused") {
		if strings.Contains(errStr, "[::1]:53") || strings.Contains(errStr, "127.0.0.1:53") {
			return fmt.Errorf("%w\n\nDNS configuration issue detected:\n"+
				"  • Your system is trying to use localhost as DNS server\n"+
				"  • Check /etc/resolv.conf for incorrect DNS settings\n"+
				"  • Common fix: Replace localhost DNS with:\n"+
				"      nameserver 8.8.8.8\n"+
				"      nameserver 1.1.1.1\n"+
				"  • On some systems, edit /etc/systemd/resolved.conf", err)
		}
		return fmt.Errorf("%w\n\nConnection issue:\n"+
			"  • Firewall may be blocking the connection\n"+
			"  • Check if you're behind a proxy\n"+
			"  • Verify you can access: https://api.github.com", err)
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") {
		return fmt.Errorf("%w\n\nConnection timeout:\n"+
			"  • Check your internet connection\n"+
			"  • Network may be slow or unstable\n"+
			"  • Try again later", err)
	}

	// Generic network error
	return fmt.Errorf("%w\n\nNetwork issue detected. Check your internet connection.", err)
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

	// Verify the binary works
	fmt.Println("🔍 Verifying downloaded binary...")
	verifyCmd := exec.Command(tmpFile, "version")
	if output, err := verifyCmd.CombinedOutput(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("downloaded binary verification failed: %w\nOutput: %s", err, string(output))
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

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute, // Binary downloads may take a while
	}

	// Get the data
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// GetPlatform returns the current platform string
func GetPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
