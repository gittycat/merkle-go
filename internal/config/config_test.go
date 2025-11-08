package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `exclude:
  - "*.tmp"
  - "*.log"
  - ".git/"
  - "node_modules/"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedExclusions := []string{"*.tmp", "*.log", ".git/", "node_modules/"}
	if len(cfg.Exclude) != len(expectedExclusions) {
		t.Errorf("Expected %d exclusions, got %d", len(expectedExclusions), len(cfg.Exclude))
	}

	for i, expected := range expectedExclusions {
		if cfg.Exclude[i] != expected {
			t.Errorf("Exclusion[%d]: expected %q, got %q", i, expected, cfg.Exclude[i])
		}
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.yml")
	if err != nil {
		t.Fatalf("LoadConfig should return default config for nonexistent file, got error: %v", err)
	}

	// Should return default config with common exclusions
	if len(cfg.Exclude) == 0 {
		t.Error("Default config should have some exclusions")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yml")

	invalidYAML := `exclude:
  - "*.tmp"
    invalid indentation
  - "*.log"
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should return error for invalid YAML")
	}
}

func TestLoadConfig_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed for empty config: %v", err)
	}

	// Empty config should result in empty exclusions (not nil)
	if cfg.Exclude == nil {
		t.Error("Exclude should not be nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Exclude == nil {
		t.Error("Default config Exclude should not be nil")
	}

	// Check that common patterns are included
	expectedPatterns := []string{".git/", "node_modules/", "__pycache__/"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, exclude := range cfg.Exclude {
			if exclude == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Default config should include pattern %q", pattern)
		}
	}
}
