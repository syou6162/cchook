package main

import (
	"fmt"
	"testing"
)

func TestExecuteJQQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		input   interface{}
		want    string
		wantErr bool
	}{
		// 基本的なクエリ
		{
			"simple field access",
			".name",
			map[string]interface{}{"name": "test", "age": 30},
			"test",
			false,
		},
		{
			"nested field access",
			".user.name",
			map[string]interface{}{"user": map[string]interface{}{"name": "alice"}},
			"alice",
			false,
		},
		{
			"array access",
			".[0]",
			[]interface{}{"first", "second"},
			"first",
			false,
		},
		{
			"array length",
			"length",
			[]interface{}{"a", "b", "c"},
			"3",
			false,
		},

		// データ型の変換
		{
			"number to string",
			".count",
			map[string]interface{}{"count": 42},
			"42",
			false,
		},
		{
			"boolean true",
			".active",
			map[string]interface{}{"active": true},
			"true",
			false,
		},
		{
			"boolean false",
			".active",
			map[string]interface{}{"active": false},
			"false",
			false,
		},
		{
			"null value",
			".missing",
			map[string]interface{}{"existing": "value"},
			"",
			false,
		},

		// 複雑なクエリ
		{
			"array filter",
			"map(select(.age > 20))",
			[]interface{}{
				map[string]interface{}{"name": "alice", "age": 25},
				map[string]interface{}{"name": "bob", "age": 15},
				map[string]interface{}{"name": "charlie", "age": 30},
			},
			"[{\"age\":25,\"name\":\"alice\"},{\"age\":30,\"name\":\"charlie\"}]",
			false,
		},
		{
			"string manipulation",
			".name | ascii_upcase",
			map[string]interface{}{"name": "hello"},
			"HELLO",
			false,
		},
		{
			"array reverse",
			"reverse",
			[]interface{}{"a", "b", "c"},
			"[\"c\",\"b\",\"a\"]",
			false,
		},

		// エラーケース
		{
			"invalid syntax",
			".invalid.[",
			map[string]interface{}{"test": "value"},
			"",
			true,
		},
		{
			"non-existent path with error",
			".missing.deeply.nested",
			map[string]interface{}{"test": "value"},
			"",
			false, // gojqではnullを返すのでエラーにならない
		},

		// 境界値テスト
		{
			"empty object",
			".",
			map[string]interface{}{},
			"{}",
			false,
		},
		{
			"empty array",
			".",
			[]interface{}{},
			"[]",
			false,
		},
		{
			"complex nested structure",
			".data[0].items | length",
			map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"items": []interface{}{"a", "b", "c"},
					},
				},
			},
			"3",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeJQQuery(tt.query, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeJQQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("executeJQQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJqValueToString(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"string value", "hello", "hello"},
		{"nil value", nil, ""},
		{"true boolean", true, "true"},
		{"false boolean", false, "false"},
		{"integer", 42, "42"},
		{"float", 3.14, "3.14"},
		{"array", []interface{}{"a", "b"}, "[\"a\",\"b\"]"},
		{"object", map[string]interface{}{"key": "value"}, "{\"key\":\"value\"}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jqValueToString(tt.value); got != tt.want {
				t.Errorf("jqValueToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnifiedTemplateReplace(t *testing.T) {
	sampleData := map[string]interface{}{
		"name":   "test-user",
		"age":    25,
		"active": true,
		"tags":   []interface{}{"dev", "golang"},
		"profile": map[string]interface{}{
			"email": "test@example.com",
			"city":  "Tokyo",
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		// 基本的な置換
		{
			"simple field",
			"Hello {.name}",
			"Hello test-user",
		},
		{
			"multiple fields",
			"User: {.name}, Age: {.age}",
			"User: test-user, Age: 25",
		},
		{
			"nested field",
			"Email: {.profile.email}",
			"Email: test@example.com",
		},
		{
			"boolean field",
			"Active: {.active}",
			"Active: true",
		},
		{
			"array access",
			"First tag: {.tags[0]}",
			"First tag: dev",
		},
		{
			"array length",
			"Tag count: {.tags | length}",
			"Tag count: 2",
		},

		// 複雑なクエリ
		{
			"string transformation",
			"Name: {.name | ascii_upcase}",
			"Name: TEST-USER",
		},
		{
			"conditional logic",
			"Status: {if .active then \"enabled\" else \"disabled\" end}",
			"Status: enabled",
		},
		{
			"array join",
			"Tags: {.tags | join(\", \")}",
			"Tags: dev, golang",
		},

		// エラーケース
		{
			"invalid query",
			"Error: {.invalid.[}",
			"Error: [JQ_ERROR: invalid jq query '.invalid.[': unexpected EOF]",
		},
		{
			"non-existent field",
			"Missing: {.nonexistent}",
			"Missing: ",
		},

		// 特殊ケース
		{
			"no templates",
			"Plain text without templates",
			"Plain text without templates",
		},
		{
			"empty template",
			"Empty: {}",
			"Empty: {}",
		},
		{
			"multiple templates",
			"{.name} is {.age} years old and lives in {.profile.city}",
			"test-user is 25 years old and lives in Tokyo",
		},
		{
			"template in middle",
			"User {.name} has {.tags | length} tags",
			"User test-user has 2 tags",
		},

		// 境界値テスト
		{
			"braces without query",
			"Just {} braces",
			"Just {} braces",
		},
		{
			"simple object construction",
			"User: {.name}, Email: {.profile.email}",
			"User: test-user, Email: test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unifiedTemplateReplace(tt.template, sampleData)
			if got != tt.want {
				t.Errorf("unifiedTemplateReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnifiedTemplateReplace_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			"nil data",
			"Value: {.}",
			nil,
			"Value: ",
		},
		{
			"empty string data",
			"Value: {.}",
			"",
			"Value: ",
		},
		{
			"number data",
			"Value: {.}",
			42,
			"Value: 42",
		},
		{
			"array data",
			"First: {.[0]}, Length: {length}",
			[]interface{}{"a", "b", "c"},
			"First: a, Length: 3",
		},
		{
			"simple nested access",
			"Name: {.name}",
			map[string]interface{}{"name": "test"},
			"Name: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unifiedTemplateReplace(tt.template, tt.data)
			if got != tt.want {
				t.Errorf("unifiedTemplateReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJQQueryCache(t *testing.T) {
	// キャッシュの動作確認
	query := ".test"
	data := map[string]interface{}{"test": "value"}

	// 初回実行
	result1, err1 := executeJQQuery(query, data)
	if err1 != nil {
		t.Fatalf("First execution failed: %v", err1)
	}

	// キャッシュから実行
	result2, err2 := executeJQQuery(query, data)
	if err2 != nil {
		t.Fatalf("Second execution failed: %v", err2)
	}

	// 結果が同じであることを確認
	if result1 != result2 {
		t.Errorf("Cached query result differs: %v != %v", result1, result2)
	}

	// キャッシュにクエリが保存されていることを確認
	jqCacheMutex.RLock()
	_, exists := jqQueryCache[query]
	jqCacheMutex.RUnlock()

	if !exists {
		t.Error("Query was not cached")
	}
}

func TestExecuteJQQuery_Performance(t *testing.T) {
	// パフォーマンステスト用の大きなデータ
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	query := ".field_500"

	// 複数回実行してキャッシュの効果を確認
	for i := 0; i < 10; i++ {
		result, err := executeJQQuery(query, largeData)
		if err != nil {
			t.Fatalf("Execution %d failed: %v", i, err)
		}
		if result != "value_500" {
			t.Errorf("Execution %d: got %v, want value_500", i, result)
		}
	}
}

func TestExecuteJQQuery_ConcurrentAccess(t *testing.T) {
	// 並行アクセステスト
	data := map[string]interface{}{"test": "value"}
	query := ".test"

	// 複数のgoroutineで同時実行
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			result, err := executeJQQuery(query, data)
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
				return
			}
			if result != "value" {
				t.Errorf("Goroutine %d: got %v, want value", id, result)
			}
		}(i)
	}

	// すべてのgoroutineの完了を待つ
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUnifiedTemplateReplace_RealWorldExamples(t *testing.T) {
	// 実際のClaude Code Transcriptデータのサンプル
	transcriptData := map[string]interface{}{
		"session_id":      "abc123",
		"transcript_path": "/tmp/transcript.json",
		"hook_event_name": "PostToolUse",
		"tool_name":       "Write",
		"tool_input": map[string]interface{}{
			"file_path": "main.go",
			"content":   "package main",
		},
		"tool_response": map[string]interface{}{
			"success": true,
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			"hook event message",
			"Event: {.hook_event_name} for tool {.tool_name}",
			"Event: PostToolUse for tool Write",
		},
		{
			"file path access",
			"Processing file: {.tool_input.file_path}",
			"Processing file: main.go",
		},
		{
			"conditional success message",
			"Status: {if .tool_response.success then \"OK\" else \"FAILED\" end}",
			"Status: OK",
		},
		{
			"gofmt command template",
			"gofmt -w {.tool_input.file_path}",
			"gofmt -w main.go",
		},
		{
			"notification message",
			"Tool {.tool_name} completed for session {.session_id}",
			"Tool Write completed for session abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unifiedTemplateReplace(tt.template, transcriptData)
			if got != tt.want {
				t.Errorf("unifiedTemplateReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}
