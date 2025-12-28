package updater

import (
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
		client: &http.Client{
			Timeout: timeout,
		},
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

	// Get temporary directory
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "morpheus-update")

	// Clone the repository and build
	fmt.Println("📦 Cloning latest version from GitHub...")
	repoDir := filepath.Join(tmpDir, "morpheus-repo")

	// Clean up old clone if exists
	os.RemoveAll(repoDir)

	// Clone repository
	cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/nimsforest/morpheus.git", repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Build the binary
	fmt.Println("🔨 Building latest version...")
	cmd = exec.Command("go", "build", "-o", tmpFile, "./cmd/morpheus")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	// Make it executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Backup current version
	backupPath := execPath + ".backup"
	fmt.Printf("📋 Backing up current version to %s\n", backupPath)
	
	// Remove old backup if it exists
	os.Remove(backupPath)
	
	// Rename current binary to backup (this works even if the binary is running)
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current version: %w", err)
	}

	// Replace current binary with new one using atomic rename
	fmt.Printf("✨ Installing update to %s\n", execPath)
	if err := os.Rename(tmpFile, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up
	os.Remove(tmpFile)
	os.RemoveAll(repoDir)

	fmt.Println("\n✅ Update completed successfully!")
	fmt.Printf("\nRun 'morpheus version' to verify the update.\n")
	fmt.Printf("Backup of previous version saved at: %s\n", backupPath)

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// GetPlatform returns the current platform string
func GetPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
