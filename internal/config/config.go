package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Skip       []string `toml:"skip"`
	OutputFile string   `toml:"output_file"`
}

func DefaultConfig() *Config {
	return &Config{
		Skip: []string{
			".git/",
			".svn/",
			"node_modules/",
			"vendor/",
			"__pycache__/",
			"*.o",
			"*.so",
			"*.exe",
			"bin/",
			"dist/",
			"*.tmp",
			"*.swp",
			"*.log",
			".DS_Store",
			"Thumbs.db",
		},
		OutputFile: "",
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config TOML: %w", err)
	}

	// Initialize Skip slice if nil (for empty configs)
	if cfg.Skip == nil {
		cfg.Skip = []string{}
	}

	// OutputFile can be empty - will default to ./output/<root-hash>.json in main

	return &cfg, nil
}
