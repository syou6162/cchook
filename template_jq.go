package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
)

// JQクエリのキャッシュ（パフォーマンス向上のため）
var (
	jqQueryCache = make(map[string]*gojq.Query)
	jqCacheMutex sync.RWMutex
)

// 削除: 古い {jq: } パターンは不要

// executeJQQuery executes a gojq query against the input and returns the result as a string.
// It caches compiled queries for performance. Returns an error if the query is invalid or execution fails.
func executeJQQuery(queryStr string, input interface{}) (string, error) {
	// クエリをキャッシュから取得または作成
	jqCacheMutex.RLock()
	query, exists := jqQueryCache[queryStr]
	jqCacheMutex.RUnlock()

	if !exists {
		// クエリをパースしてキャッシュに保存
		var err error
		query, err = gojq.Parse(queryStr)
		if err != nil {
			return "", fmt.Errorf("invalid jq query '%s': %w", queryStr, err)
		}

		jqCacheMutex.Lock()
		jqQueryCache[queryStr] = query
		jqCacheMutex.Unlock()
	}

	// 入力データをJSONとしてマーシャル/アンマーシャルして、gojq互換の型に変換
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input to JSON: %w", err)
	}

	var gojqInput interface{}
	if err := json.Unmarshal(inputJSON, &gojqInput); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON for gojq: %w", err)
	}

	// クエリを実行
	iter := query.Run(gojqInput)
	var results []interface{}

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", fmt.Errorf("jq query execution error: %w", err)
		}
		results = append(results, v)
	}

	// 結果を文字列に変換
	switch len(results) {
	case 0:
		return "", nil
	case 1:
		return jqValueToString(results[0]), nil
	default:
		// 複数の結果がある場合は配列として返す
		resultJSON, err := json.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("failed to marshal jq results: %w", err)
		}
		return string(resultJSON), nil
	}
}

// jqValueToString converts a gojq result value to a string representation.
// Handles strings, booleans, null, numbers, and objects/arrays (as JSON).
func jqValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		// 数値やオブジェクトの場合はJSONとして出力
		if result, err := json.Marshal(v); err == nil {
			return string(result)
		}
		return fmt.Sprintf("%v", v)
	}
}

// 削除: 古い {jq: } システムは不要

// unifiedTemplateReplace replaces all {query} patterns in the template with JQ query results.
// Patterns are detected using {}, and the content is treated as a JQ query executed against rawJSON.
func unifiedTemplateReplace(template string, rawJSON interface{}) string {
	// パターン: { で始まり } で終わる任意の内容
	pattern := regexp.MustCompile(`\{([^}]+)\}`)

	return pattern.ReplaceAllStringFunc(template, func(match string) string {
		jqQuery := strings.TrimSpace(match[1 : len(match)-1]) // {} を除去

		// 常にJQクエリとして処理
		result, err := executeJQQuery(jqQuery, rawJSON)
		if err != nil {
			return fmt.Sprintf("[JQ_ERROR: %s]", err.Error())
		}
		return result
	})
}
