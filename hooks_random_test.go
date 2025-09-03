package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExecuteUserPromptSubmitHooksWithRandomChance(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		input          string
		expectedOutput string
		expectError    bool
	}{
		{
			name: "random_chance 100% should always execute",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionRandomChance, Value: "100"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Always executed"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test"}`,
			expectedOutput: "Always executed",
			expectError:    false,
		},
		{
			name: "random_chance 0% should never execute",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionRandomChance, Value: "0"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Never executed"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test"}`,
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "invalid random_chance value should continue silently",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionRandomChance, Value: "invalid"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Should not execute"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test"}`,
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "random_chance combined with other condition",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionPromptRegex, Value: "test"},
							{Type: ConditionRandomChance, Value: "100"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Both conditions met"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test message"}`,
			expectedOutput: "Both conditions met",
			expectError:    false,
		},
		{
			name: "random_chance with prompt_regex not matching",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionPromptRegex, Value: "nomatch"},
							{Type: ConditionRandomChance, Value: "100"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Should not execute"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test message"}`,
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "multiple hooks with different random_chance",
			config: Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{Type: ConditionRandomChance, Value: "0"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Never"},
						},
					},
					{
						Conditions: []Condition{
							{Type: ConditionRandomChance, Value: "100"},
						},
						Actions: []UserPromptSubmitAction{
							{Type: "output", Message: "Always"},
						},
					},
				},
			},
			input:          `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"/tmp/test.json","prompt":"test"}`,
			expectedOutput: "Always",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用のtranscriptファイルを作成
			tmpFile, err := os.CreateTemp("", "transcript*.json")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.WriteString(`{"type": "user", "sessionId": "test"}` + "\n")
			tmpFile.Close()

			// 入力のtranscript_pathを更新
			inputJSON := strings.Replace(tt.input, "/tmp/test.json", tmpFile.Name(), 1)

			// 入力をパース
			var input UserPromptSubmitInput
			var rawJSON interface{}
			err = json.Unmarshal([]byte(inputJSON), &input)
			if err == nil {
				err = json.Unmarshal([]byte(inputJSON), &rawJSON)
			}
			if err != nil && !tt.expectError {
				t.Fatal(err)
			}

			// 標準出力をキャプチャ
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// テスト実行
			if err == nil {
				err = executeUserPromptSubmitHooks(&tt.config, &input, rawJSON)
			}

			// 標準出力を復元
			w.Close()
			os.Stdout = oldStdout

			// 出力を読み取り
			output, _ := io.ReadAll(r)
			outputStr := strings.TrimSpace(string(output))

			// エラーチェック
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// 出力チェック
				if tt.expectedOutput != "" && !strings.Contains(outputStr, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got %q", tt.expectedOutput, outputStr)
				}
				if tt.expectedOutput == "" && outputStr != "" {
					t.Errorf("Expected no output, got %q", outputStr)
				}
			}
		})
	}
}

// TestRandomChanceDistributionIntegration tests random distribution in actual hook execution
func TestRandomChanceDistributionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping statistical test in short mode")
	}

	config := Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Conditions: []Condition{
					{Type: ConditionRandomChance, Value: "50"},
				},
				Actions: []UserPromptSubmitAction{
					{Type: "output", Message: "Hit"},
				},
			},
		},
	}

	// テスト用のtranscriptファイルを作成
	tmpFile, err := os.CreateTemp("", "transcript*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"type": "user", "sessionId": "test"}` + "\n")
	tmpFile.Close()

	inputJSON := `{"session_id":"test","hook_event_name":"UserPromptSubmit","transcript_path":"` + tmpFile.Name() + `","prompt":"test"}`

	hitCount := 0
	const iterations = 100

	for i := 0; i < iterations; i++ {
		// 入力をパース
		var input UserPromptSubmitInput
		var rawJSON interface{}
		err = json.Unmarshal([]byte(inputJSON), &input)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal([]byte(inputJSON), &rawJSON)
		if err != nil {
			t.Fatal(err)
		}

		// 標準出力をキャプチャ
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// テスト実行
		_ = executeUserPromptSubmitHooks(&config, &input, rawJSON)

		// 標準出力を復元
		w.Close()
		os.Stdout = oldStdout

		// 出力を読み取り
		output, _ := io.ReadAll(r)
		if strings.Contains(string(output), "Hit") {
			hitCount++
		}
	}

	// 50%の確率なので、期待値は50、許容誤差は±20%
	expectedMin := 30
	expectedMax := 70
	if hitCount < expectedMin || hitCount > expectedMax {
		t.Errorf("Expected between %d and %d hits for 50%% probability, got %d out of %d",
			expectedMin, expectedMax, hitCount, iterations)
	}
}
