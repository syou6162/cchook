package main

import (
	"fmt"
	"os"
	"os/exec"
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
		if condition.Value != "" {
			_, err := os.Stat(condition.Value)
			return err == nil
		}
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
