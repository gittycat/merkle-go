package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `skip = [
  "*.tmp",
  "*.log",
  ".git/",
  "node_modules/",
]
output_file = "output/custom-tree.json"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedSkip := []string{"*.tmp", "*.log", ".git/", "node_modules/"}
	if len(cfg.Skip) != len(expectedSkip) {
		t.Errorf("Expected %d skip patterns, got %d", len(expectedSkip), len(cfg.Skip))
	}

	for i, expected := range expectedSkip {
		if cfg.Skip[i] != expected {
			t.Errorf("Skip[%d]: expected %q, got %q", i, expected, cfg.Skip[i])
		}
	}

	if cfg.OutputFile != "output/custom-tree.json" {
		t.Errorf("Expected output_file %q, got %q", "output/custom-tree.json", cfg.OutputFile)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("LoadConfig should return default config for nonexistent file, got error: %v", err)
	}

	// Should return default config with common exclusions
	if len(cfg.Skip) == 0 {
		t.Error("Default config should have some skip patterns")
	}

	// Default output file should be empty (handled in main.go)
	if cfg.OutputFile != "" {
		t.Errorf("Expected default output_file to be empty, got %q", cfg.OutputFile)
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	invalidTOML := `skip = [
  "*.tmp"
  invalid syntax
  "*.log"
]`

	if err := os.WriteFile(configPath, []byte(invalidTOML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should return error for invalid TOML")
	}
}

func TestLoadConfig_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.toml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed for empty config: %v", err)
	}

	// Empty config should result in empty skip patterns (not nil)
	if cfg.Skip == nil {
		t.Error("Skip should not be nil")
	}

	// Output file should be empty (handled in main.go)
	if cfg.OutputFile != "" {
		t.Errorf("Expected output_file to be empty, got %q", cfg.OutputFile)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Skip == nil {
		t.Error("Default config Skip should not be nil")
	}

	// Check that common patterns are included
	expectedPatterns := []string{".git/", "node_modules/", "__pycache__/"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, skip := range cfg.Skip {
			if skip == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Default config should include pattern %q", pattern)
		}
	}

	// Default output file should be empty (handled in main.go)
	if cfg.OutputFile != "" {
		t.Errorf("Expected default output_file to be empty, got %q", cfg.OutputFile)
	}
}
