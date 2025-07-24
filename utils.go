package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
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

// ネストしたフィールドにアクセスする関数
func resolveNestedField(data interface{}, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	parts := strings.Split(path, ".")
	value := reflect.ValueOf(data)

	for _, part := range parts {
		// ポインタの場合は実体を取得
		for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
			if value.IsNil() {
				return "", fmt.Errorf("nil value encountered in path: %s", path)
			}
			value = value.Elem()
		}

		switch value.Kind() {
		case reflect.Struct:
			// 構造体のフィールドを取得
			value = value.FieldByName(part)
			if !value.IsValid() {
				return "", fmt.Errorf("field '%s' not found in struct", part)
			}
		case reflect.Map:
			// マップのキーを取得
			mapValue := value.MapIndex(reflect.ValueOf(part))
			if !mapValue.IsValid() {
				return "", fmt.Errorf("key '%s' not found in map", part)
			}
			value = mapValue
		default:
			return "", fmt.Errorf("cannot access field '%s' in type %s", part, value.Kind())
		}
	}

	// 最終的な値を文字列に変換
	return valueToString(value), nil
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

// 新しい変数置換関数
func replaceVariables(template string, data interface{}) string {
	// {path.to.field} 形式のプレースホルダを検出する正規表現
	re := regexp.MustCompile(`\{([^}]+)\}`)
	
	return re.ReplaceAllStringFunc(template, func(match string) string {
		// {} を取り除いてパスを取得
		path := match[1 : len(match)-1]
		
		// ネストしたフィールドの値を取得
		value, err := resolveNestedField(data, path)
		if err != nil {
			// エラーの場合は元のプレースホルダを返す
			return match
		}
		
		return value
	})
}

func replacePreToolUseVariables(command string, input *PreToolUseInput) string {
	return replaceVariables(command, input)
}

func replacePostToolUseVariables(command string, input *PostToolUseInput) string {
	return replaceVariables(command, input)
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