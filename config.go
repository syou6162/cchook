package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// loadConfig loads the configuration from the specified YAML file.
// If configPath is empty, it uses the default configuration path.
func loadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return &config, nil
}

// getDefaultConfigPath returns the default configuration file path.
// It uses $XDG_CONFIG_HOME/cchook/config.yaml if XDG_CONFIG_HOME is set,
// otherwise it uses ~/.config/cchook/config.yaml.
func getDefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "cchook", "config.yaml")
}
