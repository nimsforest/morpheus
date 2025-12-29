package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	
	// Test with curl (if available)
	fmt.Println("  3. Using curl command (fallback)...")
	curlPath, curlErr := exec.LookPath("curl")
	if curlErr != nil {
		fmt.Printf("     ⚠️  curl not found: %v\n", curlErr)
	} else {
		cmd := exec.Command(curlPath, "-s", "-I", "-L", testURL)
		output, curlErr := cmd.CombinedOutput()
		if curlErr != nil {
			fmt.Printf("     ❌ Failed: %v\n", curlErr)
		} else {
			// Check if we got a 200 OK response
			if strings.Contains(string(output), "200 OK") || strings.Contains(string(output), "HTTP/2 200") {
				fmt.Printf("     ✓ Success (curl is available and working)\n")
			} else {
				fmt.Printf("     ⚠️  Unexpected response\n")
			}
		}
	}
	
	fmt.Println("\n📋 Recommendations:")
	
	// Check if curl is available for fallback
	_, curlAvailable := exec.LookPath("curl")
	
	if foundCerts == 0 {
		fmt.Println("  ⚠️  No CA certificates found!")
		fmt.Println("  Install certificates:")
		if runtime.GOOS == "android" || os.Getenv("TERMUX_VERSION") != "" {
			fmt.Println("    pkg update")
			fmt.Println("    pkg install ca-certificates-java openssl")
			if curlAvailable != nil {
				fmt.Println("    pkg install curl")
			}
		} else {
			fmt.Println("    • Debian/Ubuntu: apt-get install ca-certificates")
			fmt.Println("    • Fedora/RHEL: dnf install ca-certificates")
			fmt.Println("    • Alpine: apk add ca-certificates")
			if curlAvailable != nil {
				fmt.Println("    • Install curl for fallback support")
			}
		}
	} else {
		fmt.Println("  ✓ Certificates appear to be installed")
		if err != nil {
			fmt.Println("  ⚠️  But TLS connection still failed")
			if curlAvailable == nil {
				fmt.Println("  ✓ curl is available and will be used as fallback")
			} else {
				fmt.Println("  ⚠️  curl is not installed (install for fallback support)")
			}
			fmt.Println("  Try:")
			fmt.Println("    MORPHEUS_TLS_DEBUG=1 morpheus update")
		}
	}
	
	fmt.Println("\n🔧 Debug Mode:")
	fmt.Println("  MORPHEUS_TLS_DEBUG=1 morpheus update")
	fmt.Println("\n⚠️  Emergency Bypass (NOT RECOMMENDED):")
	fmt.Println("  MORPHEUS_SKIP_TLS_VERIFY=1 morpheus update")
}
