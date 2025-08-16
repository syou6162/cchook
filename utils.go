package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// parseInput関数は parser.go に移動

// 共通マッチャーチェック関数
func checkMatcher(matcher string, toolName string) bool {
	if matcher == "" {
		return true
	}

	for _, pattern := range strings.Split(matcher, "|") {
		if strings.Contains(toolName, strings.TrimSpace(pattern)) {
			return true
		}
	}
	return false
}

// fileExistsRecursive は指定されたファイル名がディレクトリツリー内に存在するかを再帰的に検索する
func fileExistsRecursive(filename string) bool {
	if filename == "" {
		return false
	}
	
	found := false
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーがあっても続ける
		}
		if !info.IsDir() && filepath.Base(path) == filename {
			found = true
			return filepath.SkipAll // 見つかったら探索を終了
		}
		return nil
	})
	if err != nil {
		return false
	}
	return found
}

// fileExists は指定されたパスにファイルが存在するかをチェックする
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func checkPreToolUseCondition(condition PreToolUseCondition, input *PreToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		// ToolInput構造体から直接FilePath取得
		if input.ToolInput.FilePath != "" {
			return strings.HasSuffix(input.ToolInput.FilePath, condition.Value)
		}
	case "command_contains":
		// ToolInput構造体からCommand取得
		if input.ToolInput.Command != "" {
			return strings.Contains(input.ToolInput.Command, condition.Value)
		}
	case "command_starts_with":
		// コマンドが指定文字列で始まる
		if input.ToolInput.Command != "" {
			return strings.HasPrefix(input.ToolInput.Command, condition.Value)
		}
	case "file_exists":
		// 指定ファイルが存在する
		return fileExists(condition.Value)
	case "url_starts_with":
		// URLが指定文字列で始まる
		if input.ToolInput.URL != "" {
			return strings.HasPrefix(input.ToolInput.URL, condition.Value)
		}
	case "file_exists_recursive":
		// ファイルが再帰的に存在するか
		found := false
		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // エラーがあっても続ける
			}
			if !info.IsDir() && filepath.Base(path) == condition.Value {
				found = true
				return filepath.SkipAll // 見つかったら探索を終了
			}
			return nil
		})
		if err != nil {
			return false
		}
		return found
	}
	return false
}

func checkPostToolUseCondition(condition PostToolUseCondition, input *PostToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		// ToolInput構造体から直接FilePath取得
		if input.ToolInput.FilePath != "" {
			return strings.HasSuffix(input.ToolInput.FilePath, condition.Value)
		}
	case "command_contains":
		// ToolInput構造体からCommand取得
		if input.ToolInput.Command != "" {
			return strings.Contains(input.ToolInput.Command, condition.Value)
		}
	case "command_starts_with":
		// コマンドが指定文字列で始まる
		if input.ToolInput.Command != "" {
			return strings.HasPrefix(input.ToolInput.Command, condition.Value)
		}
	case "file_exists":
		// 指定ファイルが存在する
		if condition.Value != "" {
			_, err := os.Stat(condition.Value)
			return err == nil
		}
	case "url_starts_with":
		// URLが指定文字列で始まる
		if input.ToolInput.URL != "" {
			return strings.HasPrefix(input.ToolInput.URL, condition.Value)
		}
	}
	return false
}

func checkUserPromptSubmitCondition(condition UserPromptSubmitCondition, input *UserPromptSubmitInput) bool {
	switch condition.Type {
	case "prompt_contains":
		// プロンプトに指定文字列が含まれる
		return strings.Contains(input.Prompt, condition.Value)
	case "prompt_starts_with":
		// プロンプトが指定文字列で始まる
		return strings.HasPrefix(input.Prompt, condition.Value)
	case "prompt_ends_with":
		// プロンプトが指定文字列で終わる
		return strings.HasSuffix(input.Prompt, condition.Value)
	case "file_exists":
		// 指定ファイルが存在する
		return fileExists(condition.Value)
	}
	return false
}

func checkSessionStartCondition(condition SessionStartCondition, input *SessionStartInput) bool {
	switch condition.Type {
	case "file_exists":
		// 指定ファイルが存在する
		return fileExists(condition.Value)
	case "file_exists_recursive":
		// ファイルが再帰的に存在するか
		return fileExistsRecursive(condition.Value)
	}
	return false
}

func runCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// シェル経由でコマンドを実行
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
