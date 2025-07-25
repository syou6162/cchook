package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	// 一時的な設定ファイルを作成
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {file_path}"
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "⚠️  git addの実行を検知しました"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// PostToolUse設定の検証
	if len(config.PostToolUse) != 1 {
		t.Errorf("Expected 1 PostToolUse hook, got %d", len(config.PostToolUse))
	}

	if config.PostToolUse[0].Matcher != "Write|Edit" {
		t.Errorf("Expected matcher 'Write|Edit', got '%s'", config.PostToolUse[0].Matcher)
	}

	if len(config.PostToolUse[0].Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(config.PostToolUse[0].Conditions))
	}

	if config.PostToolUse[0].Conditions[0].Type != "file_extension" {
		t.Errorf("Expected condition type 'file_extension', got '%s'", config.PostToolUse[0].Conditions[0].Type)
	}

	// PreToolUse設定の検証
	if len(config.PreToolUse) != 1 {
		t.Errorf("Expected 1 PreToolUse hook, got %d", len(config.PreToolUse))
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := loadConfig("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected 'failed to read config file' error, got: %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// 不正なYAMLを作成
	invalidYAML := `
PostToolUse:
  - matcher: "Write"
    actions:
      - type: command
        command: "gofmt -w {file_path}
    # 不正な構文: クォートが閉じていない
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	_, err := loadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected 'failed to parse config file' error, got: %v", err)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// 空のファイルでも正常に読み込まれるべき
	if config == nil {
		t.Error("Expected non-nil config for empty file")
	}

	if len(config.PostToolUse) != 0 {
		t.Errorf("Expected 0 PostToolUse hooks for empty file, got %d", len(config.PostToolUse))
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	// 環境変数をバックアップ
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	tests := []struct {
		name       string
		xdgConfig  string
		expectPath string
	}{
		{
			name:       "XDG_CONFIG_HOME set",
			xdgConfig:  "/custom/config",
			expectPath: "/custom/config/cchook/config.yaml",
		},
		{
			name:       "XDG_CONFIG_HOME empty",
			xdgConfig:  "",
			expectPath: "", // ホームディレクトリ依存のため空文字で検証
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("XDG_CONFIG_HOME", tt.xdgConfig)

			got := getDefaultConfigPath()

			if tt.expectPath != "" {
				if got != tt.expectPath {
					t.Errorf("getDefaultConfigPath() = %v, want %v", got, tt.expectPath)
				}
			} else {
				// XDG_CONFIG_HOME が空の場合、~/.config/cchook/config.yaml になることを確認
				if !strings.HasSuffix(got, "/.config/cchook/config.yaml") {
					t.Errorf("getDefaultConfigPath() = %v, expected to end with '/.config/cchook/config.yaml'", got)
				}
			}
		})
	}
}

func TestLoadConfig_DefaultPath(t *testing.T) {
	// デフォルトパスでの読み込み（ファイルが存在しない場合のエラー）
	_, err := loadConfig("")
	if err == nil {
		t.Error("Expected error when loading non-existent default config")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected 'failed to read config file' error, got: %v", err)
	}
}
