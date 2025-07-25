package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

// parseInput関数は parser.go に移動

// リフレクションを使って構造体のフィールド値を取得するヘルパー関数
func getFieldValue(data interface{}, fieldName string) string {
	if data == nil {
		return ""
	}

	value := reflect.ValueOf(data)
	// ポインタの場合は実体を取得
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return ""
	}

	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	return valueToString(field)
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

// 値を文字列に変換する関数
func valueToString(value reflect.Value) string {
	// ポインタの場合は実体を取得
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}

	if !value.IsValid() {
		return ""
	}

	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", value.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", value.Bool())
	default:
		return fmt.Sprintf("%v", value.Interface())
	}
}

func replacePreToolUseVariables(command string, input *PreToolUseInput, rawJSON interface{}) string {
	return unifiedTemplateReplace(command, rawJSON)
}

func replacePostToolUseVariables(command string, input *PostToolUseInput, rawJSON interface{}) string {
	return unifiedTemplateReplace(command, rawJSON)
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
