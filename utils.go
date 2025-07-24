package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ジェネリック入力パース関数
func parseInput[T HookInput](eventType HookEventType) (T, error) {
	var input T
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return input, fmt.Errorf("failed to decode %s input: %w", eventType, err)
	}
	return input, nil
}

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

// 共通変数置換関数
func replaceVariables(command string, toolInput map[string]interface{}) string {
	if filePath, ok := toolInput["file_path"].(string); ok {
		command = strings.ReplaceAll(command, "{file_path}", filePath)
	}
	return command
}

func replacePreToolUseVariables(command string, input *PreToolUseInput) string {
	return replaceVariables(command, input.ToolInput)
}

func replacePostToolUseVariables(command string, input *PostToolUseInput) string {
	return replaceVariables(command, input.ToolInput)
}

func checkPreToolUseCondition(condition PreToolUseCondition, input *PreToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		if filePath, ok := input.ToolInput["file_path"].(string); ok {
			return strings.HasSuffix(filePath, condition.Value)
		}
	case "command_contains":
		if command, ok := input.ToolInput["command"].(string); ok {
			return strings.Contains(command, condition.Value)
		}
	}
	return false
}

func checkPostToolUseCondition(condition PostToolUseCondition, input *PostToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		if filePath, ok := input.ToolInput["file_path"].(string); ok {
			return strings.HasSuffix(filePath, condition.Value)
		}
	case "command_contains":
		if command, ok := input.ToolInput["command"].(string); ok {
			return strings.Contains(command, condition.Value)
		}
	}
	return false
}

func runCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}