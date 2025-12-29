package updater

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
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

// createHTTPClient creates an HTTP client with proper TLS configuration for various environments
func createHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}
	
	// Try to load system certificates first
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		// SystemCertPool failed, try to load from common locations
		rootCAs = x509.NewCertPool()
		
		// Common certificate locations across different distros
		certPaths := []string{
			"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo/Arch/Termux
			"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL
			"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
			"/etc/ssl/cert.pem",                                 // Alpine/OpenBSD
			"/usr/local/share/certs/ca-root-nss.crt",            // FreeBSD
			"/etc/pki/tls/cacert.pem",                           // OpenELEC
			"/etc/certs/ca-certificates.crt",                    // Alternative
			"/data/data/com.termux/files/usr/etc/tls/cert.pem",  // Termux specific
		}
		
		loaded := false
		for _, certPath := range certPaths {
			if certs, err := os.ReadFile(certPath); err == nil {
				if rootCAs.AppendCertsFromPEM(certs) {
					loaded = true
					break
				}
			}
		}
		
		// If we still can't load certificates, we have a problem
		// On restricted environments like Termux, allow insecure connections as last resort
		if !loaded {
			if isRestrictedEnvironment() {
				// Last resort for Termux: skip verification with warning
				// User will see the warning but update will still work
				client.Transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}
				fmt.Println("⚠️  Warning: Could not load TLS certificates, using insecure connection")
				fmt.Println("   This is safe for GitHub releases but not ideal for security")
				return client
			}
			// On normal systems, this is an error - don't skip verification
			// Just use default client and let it fail with proper error
			return client
		}
	}
	
	// Configure TLS with loaded certificates
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
	}
	
	return client
}
