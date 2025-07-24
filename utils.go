package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
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

// JSON tagまたはフィールド名でフィールドを検索
func findFieldByNameOrJSONTag(structValue reflect.Value, name string) reflect.Value {
	structType := structValue.Type()
	
	// まず直接フィールド名で検索
	if field := structValue.FieldByName(name); field.IsValid() {
		return field
	}
	
	// JSON tagで検索
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		jsonTag := field.Tag.Get("json")
		
		// json tagをパース ("field_name,omitempty" -> "field_name")
		if jsonTag != "" {
			tagParts := strings.Split(jsonTag, ",")
			if len(tagParts) > 0 && tagParts[0] == name {
				return structValue.Field(i)
			}
		}
	}
	
	return reflect.Value{}
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
			// 構造体のフィールドを取得（JSON tagもサポート）
			fieldValue := findFieldByNameOrJSONTag(value, part)
			if !fieldValue.IsValid() {
				return "", fmt.Errorf("field '%s' not found in struct", part)
			}
			value = fieldValue
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
	return snakeCaseReplaceVariables(command, input)
}

func replacePostToolUseVariables(command string, input *PostToolUseInput) string {
	return snakeCaseReplaceVariables(command, input)
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
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}