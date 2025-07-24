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

// jqVariablePattern は {jq: クエリ} 形式のパターンにマッチ
var jqVariablePattern = regexp.MustCompile(`\{jq:\s*([^}]+)\}`)

// executeJQQuery はgojqクエリを実行し結果を文字列として返す
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

// jqValueToString はgojqの結果値を文字列に変換
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

// replaceJQVariables はテンプレート文字列内の {jq: クエリ} を実際の値に置換
func replaceJQVariables(template string, input interface{}) string {
	return jqVariablePattern.ReplaceAllStringFunc(template, func(match string) string {
		// {jq: クエリ} からクエリ部分を抽出
		submatches := jqVariablePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // 無効なパターンの場合はそのまま返す
		}

		queryStr := strings.TrimSpace(submatches[1])
		result, err := executeJQQuery(queryStr, input)
		if err != nil {
			// エラーが発生した場合はエラーメッセージを返す
			return fmt.Sprintf("[JQ_ERROR: %s]", err.Error())
		}

		return result
	})
}

// extendedSnakeCaseReplaceVariables は従来のsnake_case変数とjq変数の両方をサポート
func extendedSnakeCaseReplaceVariables(template string, input interface{}, rawJSON interface{}) string {
	// 1. 従来のsnake_case変数を置換
	result := snakeCaseReplaceVariables(template, input)
	
	// 2. jq変数を置換（生のJSONデータを使用）
	if rawJSON != nil {
		result = replaceJQVariables(result, rawJSON)
	} else {
		// フォールバック：構造体データを使用
		result = replaceJQVariables(result, input)
	}
	
	return result
}