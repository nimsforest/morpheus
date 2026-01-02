// Package sshutil provides utility functions for SSH-related formatting.
package sshutil

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FormatSSHCommand returns a properly formatted SSH command for display to users.
// IPv6 addresses do NOT need brackets for the ssh command.
// Example: ssh root@2001:db8::1
func FormatSSHCommand(user, ip string) string {
	return fmt.Sprintf("ssh %s@%s", user, ip)
}

// FormatSSHCommandWithIdentity returns a formatted SSH command with explicit identity file.
// This helps users who have multiple SSH keys or when the default key isn't automatically used.
// Example: ssh -i ~/.ssh/id_ed25519 root@2001:db8::1
func FormatSSHCommandWithIdentity(user, ip, identityFile string) string {
	if identityFile == "" {
		return FormatSSHCommand(user, ip)
	}
	return fmt.Sprintf("ssh -i %s %s@%s", identityFile, user, ip)
}

// DetectSSHPrivateKeyPath attempts to find the SSH private key that corresponds
// to the public key that was uploaded to the cloud provider.
// It checks common SSH key locations and returns the path to the private key.
// Returns empty string if no suitable key is found.
func DetectSSHPrivateKeyPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Check common SSH key locations in order of preference
	// (ed25519 is recommended, then ecdsa, then rsa)
	keyPaths := []string{
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_rsa"),
	}

	for _, keyPath := range keyPaths {
		// Check if both private key and public key exist
		if _, err := os.Stat(keyPath); err == nil {
			pubKeyPath := keyPath + ".pub"
			if _, err := os.Stat(pubKeyPath); err == nil {
				// Both files exist, return the private key path
				// Use ~ shorthand for display
				return "~/.ssh/" + filepath.Base(keyPath)
			}
		}
	}

	return ""
}

// GetSSHPrivateKeyForPublicKey returns the private key path for a given public key path.
// If publicKeyPath ends with .pub, returns the path without .pub extension.
// Otherwise returns an empty string.
func GetSSHPrivateKeyForPublicKey(publicKeyPath string) string {
	if publicKeyPath == "" {
		return ""
	}

	// Expand ~ to home directory for checking
	expandedPath := publicKeyPath
	if strings.HasPrefix(publicKeyPath, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			expandedPath = filepath.Join(homeDir, publicKeyPath[2:])
		}
	}

	// If it's a .pub file, get the private key path
	if strings.HasSuffix(expandedPath, ".pub") {
		privateKeyPath := strings.TrimSuffix(expandedPath, ".pub")
		if _, err := os.Stat(privateKeyPath); err == nil {
			// Return with ~ shorthand if it's in home directory
			if homeDir, err := os.UserHomeDir(); err == nil {
				if strings.HasPrefix(privateKeyPath, homeDir) {
					return "~" + privateKeyPath[len(homeDir):]
				}
			}
			return privateKeyPath
		}
	}

	return ""
}

// FormatSSHAddress returns a properly formatted address for TCP connections.
// IPv6 addresses need brackets when combined with a port.
// Example: [2001:db8::1]:22
func FormatSSHAddress(ip string, port int) string {
	// Check if this looks like an IPv6 address (contains colons and no brackets already)
	if strings.Contains(ip, ":") && !strings.HasPrefix(ip, "[") {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	// IPv4 or already bracketed
	return fmt.Sprintf("%s:%d", ip, port)
}

// IsIPv6 returns true if the given IP address appears to be IPv6.
func IsIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}

// CalculateSSHKeyFingerprint calculates the MD5 fingerprint of an SSH public key.
// The fingerprint format matches what Hetzner Cloud and other providers display.
// Example output: "ab:cd:ef:12:34:56:78:90:ab:cd:ef:12:34:56:78:90"
func CalculateSSHKeyFingerprint(publicKey string) (string, error) {
	// SSH public key format: <type> <base64-data> [comment]
	// Example: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG... user@host
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid SSH public key format: expected at least 2 parts")
	}

	// Decode the base64 key data
	keyData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode SSH key data: %w", err)
	}

	// Calculate MD5 hash
	hash := md5.Sum(keyData)

	// Format as colon-separated hex bytes
	fingerprint := make([]string, len(hash))
	for i, b := range hash {
		fingerprint[i] = fmt.Sprintf("%02x", b)
	}

	return strings.Join(fingerprint, ":"), nil
}

// ReadAndCalculateFingerprint reads an SSH public key file and calculates its fingerprint.
// Returns the fingerprint and the full public key content.
func ReadAndCalculateFingerprint(keyPath string) (fingerprint, publicKey string, err error) {
	// Expand ~ if present
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("failed to get home directory: %w", err)
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read key file: %w", err)
	}

	publicKey = strings.TrimSpace(string(data))
	fingerprint, err = CalculateSSHKeyFingerprint(publicKey)
	if err != nil {
		return "", "", err
	}

	return fingerprint, publicKey, nil
}
