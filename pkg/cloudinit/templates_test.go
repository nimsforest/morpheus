package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	data := TemplateData{
		ForestID:              "test-forest",
		NimsForestInstall:     true,
		NimsForestDownloadURL: "https://example.com/nimsforest",
	}

	script, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check required components
	checks := []string{
		"#cloud-config",
		"package_update: true",
		"ufw allow 22/tcp",
		"ufw allow 4222/tcp",
		"ufw allow 6222/tcp",
		"ufw allow 8222/tcp",
		"/opt/nimsforest/bin",
		"test-forest",
		"https://example.com/nimsforest",
		"nimsforest.service",
	}

	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Errorf("Generated script missing expected content: %s", check)
		}
	}
}

func TestGenerateWithStorageBox(t *testing.T) {
	data := TemplateData{
		ForestID:              "test-forest",
		NodeID:                "test-node-1",
		NimsForestInstall:     true,
		NimsForestDownloadURL: "https://example.com/nimsforest",
		StorageBoxHost:        "u12345.your-storagebox.de",
		StorageBoxUser:        "u12345",
		StorageBoxPassword:    "secret",
	}

	script, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check StorageBox mount
	checks := []string{
		"/mnt/forest",
		"u12345.your-storagebox.de",
		"cifs",
		"registry.json",
		"jq",
		"REGISTRY_PATH=/mnt/forest/registry.json",
	}

	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Errorf("Generated script missing expected content: %s", check)
		}
	}
}

func TestGenerateWithoutNimsForest(t *testing.T) {
	data := TemplateData{
		ForestID:          "test-forest",
		NimsForestInstall: false,
	}

	script, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT contain NimsForest installation
	if strings.Contains(script, "Installing NimsForest") {
		t.Error("Script should not contain NimsForest installation when disabled")
	}

	// Should still have basic setup
	if !strings.Contains(script, "#cloud-config") {
		t.Error("Script should still be valid cloud-config")
	}
}

func TestGenerateWithoutStorageBox(t *testing.T) {
	data := TemplateData{
		ForestID:              "test-forest",
		NimsForestInstall:     true,
		NimsForestDownloadURL: "https://example.com/nimsforest",
	}

	script, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT contain StorageBox mount when not configured
	if strings.Contains(script, "Mounting StorageBox") {
		t.Error("Script should not contain StorageBox mount when not configured")
	}

	// Should NOT have REGISTRY_PATH when no StorageBox
	if strings.Contains(script, "REGISTRY_PATH=") {
		t.Error("Script should not set REGISTRY_PATH when StorageBox not configured")
	}
}
