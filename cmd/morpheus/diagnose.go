package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// This file provides diagnostics for update issues

func handleDiagnose() {
	fmt.Println("🔍 Morpheus Update Diagnostics")
	fmt.Println("==============================")
	fmt.Println()
	
	// System information
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Arch: %s\n", runtime.GOARCH)
	isAndroid := runtime.GOOS == "android" || os.Getenv("ANDROID_ROOT") != "" || os.Getenv("TERMUX_VERSION") != ""
	fmt.Printf("Termux/Android detected: %v\n", isAndroid)
	fmt.Println()
	
	if isAndroid {
		// Check curl on Termux/Android
		fmt.Println("📋 Termux/Android Requirements:")
		curlPath, err := exec.LookPath("curl")
		if err != nil {
			fmt.Println("  ❌ curl is NOT installed")
			fmt.Println()
			fmt.Println("💡 To fix:")
			fmt.Println("  pkg install curl")
			fmt.Println()
			fmt.Println("After installing curl, morpheus update will work.")
		} else {
			fmt.Printf("  ✓ curl is installed at: %s\n", curlPath)
			
			// Test curl
			cmd := exec.Command(curlPath, "--version")
			output, err := cmd.CombinedOutput()
			if err == nil {
				fmt.Printf("  ✓ curl version: %s\n", string(output[:50]))
			}
			
			fmt.Println()
			fmt.Println("✅ Everything looks good!")
			fmt.Println("   You can run: morpheus update")
		}
	} else {
		// Non-Android system
		fmt.Println("📋 System Status:")
		fmt.Println("  ✓ Running on a standard system (not Termux/Android)")
		fmt.Println("  ✓ Updates use the built-in HTTP client")
		fmt.Println()
		fmt.Println("✅ Everything should work fine.")
		fmt.Println("   You can run: morpheus update")
	}
}
