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
	return &Updater{
		currentVersion: currentVersion,
		client:         createHTTPClient(),
	}
}

// createHTTPClient creates an HTTP client with custom DNS resolver for Termux/Android
func createHTTPClient() *http.Client {
	// Check if we're running on Android/Termux by looking for common indicators
	isAndroid := runtime.GOOS == "android" || os.Getenv("ANDROID_ROOT") != "" || os.Getenv("TERMUX_VERSION") != ""
	
	// On Android/Termux, use a custom DNS resolver to bypass broken system DNS
	if isAndroid {
		// Create a custom dialer with Google DNS (8.8.8.8) and Cloudflare DNS (1.1.1.1)
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		
		// Custom resolver that uses public DNS servers
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// Use Google DNS (8.8.8.8:53) and Cloudflare DNS (1.1.1.1:53) as fallback
				d := net.Dialer{
					Timeout: 10 * time.Second,
				}
				
				// Try Google DNS first
				conn, err := d.DialContext(ctx, "udp", "8.8.8.8:53")
				if err != nil {
					// Fallback to Cloudflare DNS
					conn, err = d.DialContext(ctx, "udp", "1.1.1.1:53")
				}
				return conn, err
			},
		}
		
		// Create custom transport with the custom resolver
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Resolve the hostname using our custom resolver
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				
				ips, err := resolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				
				if len(ips) == 0 {
					return nil, fmt.Errorf("no IP addresses found for %s", host)
				}
				
				// Use the first IP address
				resolvedAddr := net.JoinHostPort(ips[0].String(), port)
				return dialer.DialContext(ctx, network, resolvedAddr)
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		
		return &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	}
	
	// For other platforms, use the default HTTP client
	return &http.Client{
		Timeout: timeout,
	}
}

// CheckForUpdate checks if a new version is available
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
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

	// Create HTTP client with timeout and custom DNS resolver for Termux
	client := createHTTPClient()
	client.Timeout = 5 * time.Minute // Binary downloads may take a while

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
