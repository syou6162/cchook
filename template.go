package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// pathGetter は構造体の値から特定のフィールドを取得する関数型
type pathGetter func(reflect.Value) (interface{}, error)

// グローバルなパステーブル（起動時に一度だけ初期化）
var (
	preToolUsePathTable  map[string]pathGetter
	postToolUsePathTable map[string]pathGetter
	notificationPathTable map[string]pathGetter
	stopPathTable        map[string]pathGetter
	subagentStopPathTable map[string]pathGetter
	preCompactPathTable  map[string]pathGetter
)

// 初期化時にパステーブルを構築
func init() {
	preToolUsePathTable = buildPathTable(reflect.TypeOf(PreToolUseInput{}), "")
	postToolUsePathTable = buildPathTable(reflect.TypeOf(PostToolUseInput{}), "")
	notificationPathTable = buildPathTable(reflect.TypeOf(NotificationInput{}), "")
	stopPathTable = buildPathTable(reflect.TypeOf(StopInput{}), "")
	subagentStopPathTable = buildPathTable(reflect.TypeOf(SubagentStopInput{}), "")
	preCompactPathTable = buildPathTable(reflect.TypeOf(PreCompactInput{}), "")
}

// buildPathTable は型からJSON tagベースのパスマッピングテーブルを構築
func buildPathTable(t reflect.Type, prefix string) map[string]pathGetter {
	table := make(map[string]pathGetter)
	
	// ポインタ型の場合は実体の型を取得
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return table
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// 非公開フィールドはスキップ
		if !field.IsExported() {
			continue
		}
		
		// JSON tagを取得
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		
		// JSON tag名を解析（",omitempty" などを除去）
		tagParts := strings.Split(jsonTag, ",")
		fieldName := tagParts[0]
		if fieldName == "" {
			// JSON tagがない場合はフィールド名をsnake_caseに変換
			fieldName = camelToSnake(field.Name)
		}
		
		path := prefix + fieldName
		fieldIndex := i // クロージャで使用するためにコピー
		
		// ネストした構造体の場合は再帰的に処理
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		
		// 埋め込み構造体（Anonymous field）の場合は、プレフィックスなしで追加
		if field.Anonymous && fieldType.Kind() == reflect.Struct {
			nestedTable := buildPathTable(fieldType, prefix) // プレフィックスなし
			for nestedPath, nestedGetter := range nestedTable {
				table[nestedPath] = func(rv reflect.Value) (interface{}, error) {
					fieldValue := rv.Field(fieldIndex)
					// ポインタの場合は実体を取得
					if fieldValue.Kind() == reflect.Ptr {
						if fieldValue.IsNil() {
							return nil, fmt.Errorf("nil pointer at path: %s", nestedPath)
						}
						fieldValue = fieldValue.Elem()
					}
					return nestedGetter(fieldValue)
				}
			}
		} else if fieldType.Kind() == reflect.Struct && !isBasicType(fieldType) {
			// 通常のネストした構造体のフィールドを追加
			nestedTable := buildPathTable(fieldType, path+".")
			for nestedPath, nestedGetter := range nestedTable {
				table[nestedPath] = func(rv reflect.Value) (interface{}, error) {
					fieldValue := rv.Field(fieldIndex)
					// ポインタの場合は実体を取得
					if fieldValue.Kind() == reflect.Ptr {
						if fieldValue.IsNil() {
							return nil, fmt.Errorf("nil pointer at path: %s", nestedPath)
						}
						fieldValue = fieldValue.Elem()
					}
					return nestedGetter(fieldValue)
				}
			}
		}
		
		// 現在のフィールド自体のgetter
		table[path] = func(rv reflect.Value) (interface{}, error) {
			fieldValue := rv.Field(fieldIndex)
			if !fieldValue.IsValid() {
				return nil, fmt.Errorf("invalid field at path: %s", path)
			}
			
			// ポインタの場合は実体を取得
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					return nil, fmt.Errorf("nil pointer at path: %s", path)
				}
				fieldValue = fieldValue.Elem()
			}
			
			return fieldValue.Interface(), nil
		}
	}
	
	return table
}

// camelToSnake はCamelCaseをsnake_caseに変換
func camelToSnake(s string) string {
	var result strings.Builder
	runes := []rune(s)
	
	for i, r := range runes {
		isUpper := 'A' <= r && r <= 'Z'
		
		if i > 0 && isUpper {
			// 前の文字が小文字の場合、またはこの文字が大文字で次の文字が小文字の場合
			prevIsLower := i > 0 && 'a' <= runes[i-1] && runes[i-1] <= 'z'
			nextIsLower := i < len(runes)-1 && 'a' <= runes[i+1] && runes[i+1] <= 'z'
			
			if prevIsLower || nextIsLower {
				result.WriteByte('_')
			}
		}
		
		// 大文字を小文字に変換
		if isUpper {
			result.WriteByte(byte(r - 'A' + 'a'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isBasicType は基本型かどうかを判定
func isBasicType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 reflect.Float32, reflect.Float64, reflect.Bool:
		return true
	default:
		return false
	}
}

// templateLookup は指定されたパスの値を高速ルックアップで取得
func templateLookup(data interface{}, path string) (interface{}, error) {
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	
	var table map[string]pathGetter
	
	// データ型に応じてテーブルを選択
	switch data.(type) {
	case *PreToolUseInput, PreToolUseInput:
		table = preToolUsePathTable
	case *PostToolUseInput, PostToolUseInput:
		table = postToolUsePathTable
	case *NotificationInput, NotificationInput:
		table = notificationPathTable
	case *StopInput, StopInput:
		table = stopPathTable
	case *SubagentStopInput, SubagentStopInput:
		table = subagentStopPathTable
	case *PreCompactInput, PreCompactInput:
		table = preCompactPathTable
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}
	
	getter, exists := table[path]
	if !exists {
		// パステーブルにない場合、動的にネストアクセスを試行
		return dynamicNestedLookup(rv, path)
	}
	
	return getter(rv)
}

// dynamicNestedLookup は実行時にネストした構造体へのアクセスを処理
func dynamicNestedLookup(rv reflect.Value, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	currentValue := rv
	
	for _, part := range parts {
		// 現在の値がポインタの場合は実体を取得
		for currentValue.Kind() == reflect.Ptr || currentValue.Kind() == reflect.Interface {
			if currentValue.IsNil() {
				return nil, fmt.Errorf("nil value encountered in path: %s", path)
			}
			currentValue = currentValue.Elem()
		}
		
		switch currentValue.Kind() {
		case reflect.Struct:
			// JSON tagまたはフィールド名で検索
			fieldValue := findFieldByNameOrJSONTag(currentValue, part)
			if !fieldValue.IsValid() {
				return nil, fmt.Errorf("field '%s' not found in struct", part)
			}
			currentValue = fieldValue
			
		case reflect.Map:
			// マップのキーを取得
			mapValue := currentValue.MapIndex(reflect.ValueOf(part))
			if !mapValue.IsValid() {
				return nil, fmt.Errorf("key '%s' not found in map", part)
			}
			currentValue = mapValue
			
		default:
			return nil, fmt.Errorf("cannot access field '%s' in type %s", part, currentValue.Kind())
		}
	}
	
	return currentValue.Interface(), nil
}

// snakeCaseReplaceVariables は新しいsnake_case対応のテンプレート変数置換
func snakeCaseReplaceVariables(template string, data interface{}) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		path := match[1 : len(match)-1] // {} を除去
		
		value, err := templateLookup(data, path)
		if err != nil {
			// エラーの場合は元のプレースホルダーを保持
			return match
		}
		
		return fmt.Sprintf("%v", value)
	})
}