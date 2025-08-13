package ota

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileVersionProvider(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ota_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	versionFile := filepath.Join(tmpDir, "version.txt")

	// Test 1: New file with defaults
	t.Run("NewFileDefaults", func(t *testing.T) {
		provider := NewFileVersionProvider(versionFile)
		
		if version := provider.GetVersion(); version != "1.0.0" {
			t.Errorf("Expected default version 1.0.0, got %s", version)
		}
		
		if module := provider.GetModule(); module != "default" {
			t.Errorf("Expected default module 'default', got %s", module)
		}
	})

	// Test 2: Set and get version/module
	t.Run("SetAndGet", func(t *testing.T) {
		provider := NewFileVersionProvider(versionFile)
		
		// Set version and module
		if err := provider.SetVersion("2.0.0"); err != nil {
			t.Fatal(err)
		}
		
		if err := provider.SetModule("x86"); err != nil {
			t.Fatal(err)
		}
		
		// Verify values
		if version := provider.GetVersion(); version != "2.0.0" {
			t.Errorf("Expected version 2.0.0, got %s", version)
		}
		
		if module := provider.GetModule(); module != "x86" {
			t.Errorf("Expected module 'x86', got %s", module)
		}
	})

	// Test 3: Persistence across instances
	t.Run("Persistence", func(t *testing.T) {
		// Create new provider instance
		provider2 := NewFileVersionProvider(versionFile)
		
		// Should read the previously saved values
		if version := provider2.GetVersion(); version != "2.0.0" {
			t.Errorf("Expected persisted version 2.0.0, got %s", version)
		}
		
		if module := provider2.GetModule(); module != "x86" {
			t.Errorf("Expected persisted module 'x86', got %s", module)
		}
	})

	// Test 4: Backward compatibility with plain text
	t.Run("BackwardCompatibility", func(t *testing.T) {
		// Write plain text version (old format)
		plainVersionFile := filepath.Join(tmpDir, "version_plain.txt")
		if err := os.WriteFile(plainVersionFile, []byte("1.5.0"), 0644); err != nil {
			t.Fatal(err)
		}
		
		provider := NewFileVersionProvider(plainVersionFile)
		
		if version := provider.GetVersion(); version != "1.5.0" {
			t.Errorf("Expected version from plain text 1.5.0, got %s", version)
		}
		
		if module := provider.GetModule(); module != "default" {
			t.Errorf("Expected default module for plain text, got %s", module)
		}
		
		// After setting module, it should save as JSON
		if err := provider.SetModule("arm64"); err != nil {
			t.Fatal(err)
		}
		
		// Read the file and verify it's JSON now
		data, err := os.ReadFile(plainVersionFile)
		if err != nil {
			t.Fatal(err)
		}
		
		var info VersionInfo
		if err := json.Unmarshal(data, &info); err != nil {
			t.Errorf("Expected JSON format after update, got: %s", string(data))
		}
		
		if info.Version != "1.5.0" || info.Module != "arm64" {
			t.Errorf("Expected version=1.5.0, module=arm64, got version=%s, module=%s", 
				info.Version, info.Module)
		}
	})
}

// Example of creating a version.txt file with x86 module
func Example_createX86VersionFile() {
	versionFile := "version.txt"
	
	// Method 1: Use FileVersionProvider
	provider := NewFileVersionProvider(versionFile)
	provider.SetVersion("1.0.0")
	provider.SetModule("x86")
	
	// Method 2: Direct JSON write
	info := VersionInfo{
		Version: "1.0.0",
		Module:  "x86",
	}
	data, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(versionFile, data, 0644)
	
	// The file will contain:
	// {
	//   "version": "1.0.0",
	//   "module": "x86"
	// }
}