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

func checkPreToolUseCondition(condition Condition, input *PreToolUseInput) bool {
	// まず汎用条件をチェック
	if checkCommonCondition(condition) {
		return true
	}
	
	// ツール固有の条件をチェック
	return checkToolCondition(condition, &input.ToolInput)
}

func checkPostToolUseCondition(condition Condition, input *PostToolUseInput) bool {
	// まず汎用条件をチェック（file_existsのみ、file_exists_recursiveは現状PostToolUseには実装されていない）
	if condition.Type == "file_exists" {
		return fileExists(condition.Value)
	}
	
	// ツール固有の条件をチェック
	return checkToolCondition(condition, &input.ToolInput)
}

func checkUserPromptSubmitCondition(condition Condition, input *UserPromptSubmitInput) bool {
	// まず汎用条件をチェック（file_existsのみ実装されている）
	if condition.Type == "file_exists" {
		return fileExists(condition.Value)
	}
	
	// プロンプト固有の条件をチェック
	return checkPromptCondition(condition, input.Prompt)
}

func checkSessionStartCondition(condition Condition, input *SessionStartInput) bool {
	// SessionStartは汎用条件のみ使用
	return checkCommonCondition(condition)
}

// 汎用条件チェック関数
func checkCommonCondition(condition Condition) bool {
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

// ツール関連の条件チェック関数
func checkToolCondition(condition Condition, toolInput *ToolInput) bool {
	switch condition.Type {
	case "file_extension":
		// ToolInput構造体から直接FilePath取得
		if toolInput.FilePath != "" {
			return strings.HasSuffix(toolInput.FilePath, condition.Value)
		}
	case "command_contains":
		// ToolInput構造体からCommand取得
		if toolInput.Command != "" {
			return strings.Contains(toolInput.Command, condition.Value)
		}
	case "command_starts_with":
		// コマンドが指定文字列で始まる
		if toolInput.Command != "" {
			return strings.HasPrefix(toolInput.Command, condition.Value)
		}
	case "url_starts_with":
		// URLが指定文字列で始まる
		if toolInput.URL != "" {
			return strings.HasPrefix(toolInput.URL, condition.Value)
		}
	}
	return false
}

// プロンプト関連の条件チェック関数
func checkPromptCondition(condition Condition, prompt string) bool {
	switch condition.Type {
	case "prompt_contains":
		// プロンプトに指定文字列が含まれる
		return strings.Contains(prompt, condition.Value)
	case "prompt_starts_with":
		// プロンプトが指定文字列で始まる
		return strings.HasPrefix(prompt, condition.Value)
	case "prompt_ends_with":
		// プロンプトが指定文字列で終わる
		return strings.HasSuffix(prompt, condition.Value)
	}
	return false
}

// Notification用の条件チェック（汎用条件のみ）
func checkNotificationCondition(condition Condition, input *NotificationInput) bool {
	return checkCommonCondition(condition)
}

// Stop用の条件チェック（汎用条件のみ）
func checkStopCondition(condition Condition, input *StopInput) bool {
	return checkCommonCondition(condition)
}

// SubagentStop用の条件チェック（汎用条件のみ）
func checkSubagentStopCondition(condition Condition, input *SubagentStopInput) bool {
	return checkCommonCondition(condition)
}

// PreCompact用の条件チェック（汎用条件のみ）
func checkPreCompactCondition(condition Condition, input *PreCompactInput) bool {
	return checkCommonCondition(condition)
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
