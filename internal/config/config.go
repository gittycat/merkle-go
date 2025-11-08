package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Exclude []string `yaml:"exclude"`
}

func DefaultConfig() *Config {
	return &Config{
		Exclude: []string{
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Initialize Exclude slice if nil (for empty configs)
	if cfg.Exclude == nil {
		cfg.Exclude = []string{}
	}

	return &cfg, nil
}
