package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// This file provides certificate diagnostics for debugging TLS issues

func handleDiagnoseCerts() {
	fmt.Println("🔍 Morpheus TLS Certificate Diagnostics")
	fmt.Println("========================================")
	fmt.Println()
	
	// System information
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Arch: %s\n", runtime.GOARCH)
	fmt.Printf("ANDROID_ROOT: %s\n", os.Getenv("ANDROID_ROOT"))
	fmt.Printf("TERMUX_VERSION: %s\n", os.Getenv("TERMUX_VERSION"))
	fmt.Printf("PREFIX: %s\n", os.Getenv("PREFIX"))
	fmt.Printf("SSL_CERT_FILE: %s\n", os.Getenv("SSL_CERT_FILE"))
	fmt.Println()
	
	// Check system cert pool
	fmt.Println("System Certificate Pool:")
	certPool, err := x509.SystemCertPool()
	if err != nil {
		fmt.Printf("  ❌ Failed to load: %v\n", err)
	} else {
		fmt.Println("  ✓ Loaded successfully")
	}
	fmt.Println()
	
	// Get Termux PREFIX if available
	termuxPrefix := os.Getenv("PREFIX")
	if termuxPrefix == "" {
		termuxPrefix = "/data/data/com.termux/files/usr"
	}
	
	// Check certificate file locations
	fmt.Println("Certificate File Locations:")
	certPaths := []string{
		// Termux paths
		filepath.Join(termuxPrefix, "etc/tls/certs/ca-certificates.crt"),
		filepath.Join(termuxPrefix, "etc/tls/cert.pem"),
		filepath.Join(termuxPrefix, "etc/ssl/certs/ca-certificates.crt"),
		filepath.Join(termuxPrefix, "etc/ssl/cert.pem"),
		// Android system certs
		"/system/etc/security/cacerts",
		// Standard Linux paths
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/ssl/cert.pem",
		"/usr/local/share/certs/ca-root-nss.crt",
		"/etc/pki/tls/cacert.pem",
		"/etc/certs/ca-certificates.crt",
	}
	
	foundCerts := 0
	for _, certPath := range certPaths {
		info, err := os.Stat(certPath)
		if err == nil {
			if info.IsDir() {
				fmt.Printf("  ✓ %s (directory)\n", certPath)
			} else {
				fmt.Printf("  ✓ %s (%d bytes)\n", certPath, info.Size())
			}
			foundCerts++
		} else {
			fmt.Printf("  ✗ %s (not found)\n", certPath)
		}
	}
	fmt.Printf("\nFound %d certificate locations\n\n", foundCerts)
	
	// Test TLS connection
	fmt.Println("Testing TLS Connection to GitHub:")
	testURL := "https://api.github.com"
	
	// Test with default config
	fmt.Println("  1. With system certificates...")
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    certPool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	
	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("     ❌ Failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Printf("     ✓ Success (status: %s)\n", resp.Status)
	}
	
	// Test with InsecureSkipVerify (to isolate cert issues)
	fmt.Println("  2. Without certificate verification (test only)...")
	clientNoVerify := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	
	resp, err = clientNoVerify.Get(testURL)
	if err != nil {
		fmt.Printf("     ❌ Failed: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Printf("     ✓ Success (status: %s)\n", resp.Status)
	}
	
	fmt.Println("\n📋 Recommendations:")
	
	if foundCerts == 0 {
		fmt.Println("  ⚠️  No CA certificates found!")
		fmt.Println("  Install certificates:")
		if runtime.GOOS == "android" || os.Getenv("TERMUX_VERSION") != "" {
			fmt.Println("    pkg update")
			fmt.Println("    pkg install ca-certificates-java")
			fmt.Println("    pkg install openssl")
		} else {
			fmt.Println("    • Debian/Ubuntu: apt-get install ca-certificates")
			fmt.Println("    • Fedora/RHEL: dnf install ca-certificates")
			fmt.Println("    • Alpine: apk add ca-certificates")
		}
	} else {
		fmt.Println("  ✓ Certificates appear to be installed")
		if err != nil {
			fmt.Println("  ⚠️  But TLS connection still failed")
			fmt.Println("  Try:")
			fmt.Println("    MORPHEUS_TLS_DEBUG=1 morpheus update")
		}
	}
	
	fmt.Println("\n🔧 Debug Mode:")
	fmt.Println("  MORPHEUS_TLS_DEBUG=1 morpheus update")
	fmt.Println("\n⚠️  Emergency Bypass (NOT RECOMMENDED):")
	fmt.Println("  MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update")
}
