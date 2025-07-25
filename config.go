package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func loadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = getDefaultConfigPath()
		fmt.Fprintf(os.Stderr, "[Config] Using default config path: %s\n", configPath)
	} else {
		fmt.Fprintf(os.Stderr, "[Config] Using specified config path: %s\n", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] Failed to read config file '%s': %v\n", configPath, err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[Config] Successfully read config file (%d bytes)\n", len(data))

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "[Error] Failed to parse YAML config: %v\n", err)
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 設定内容の概要をログ出力
	hookCounts := map[string]int{
		"PreToolUse":   len(config.PreToolUse),
		"PostToolUse":  len(config.PostToolUse),
		"Notification": len(config.Notification),
		"Stop":         len(config.Stop),
		"SubagentStop": len(config.SubagentStop),
		"PreCompact":   len(config.PreCompact),
	}
	fmt.Fprintf(os.Stderr, "[Config] Loaded hooks: %+v\n", hookCounts)

	return &config, nil
}

func getDefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "cchook", "config.yaml")
}
