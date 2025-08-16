package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExecuteSessionStartHooks(t *testing.T) {
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Matcher: "startup",
				Actions: []SessionStartAction{
					{
						Type:    "output",
						Message: "Session started: {.session_id}",
					},
				},
			},
			{
				Matcher: "resume",
				Actions: []SessionStartAction{
					{
						Type:    "output",
						Message: "Session resumed",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		input          *SessionStartInput
		expectedOutput string
		shouldMatch    bool
	}{
		{
			name: "Startup matcher matches",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
			expectedOutput: "Session started: test-session-123",
			shouldMatch:    true,
		},
		{
			name: "Resume matcher matches",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-456",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "resume",
			},
			expectedOutput: "Session resumed",
			shouldMatch:    true,
		},
		{
			name: "Clear source doesn't match",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-789",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "clear",
			},
			expectedOutput: "",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// キャプチャ用バッファ
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			// フック実行
			err := executeSessionStartHooks(config, tt.input, rawJSON)

			// 出力キャプチャ
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// 出力チェック
			if tt.shouldMatch {
				if output != tt.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output, got '%s'", output)
				}
			}
		})
	}
}

func TestSessionStartParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      SessionStartInput
	}{
		{
			name: "Startup event",
			jsonInput: `{
				"session_id": "abc123",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "SessionStart",
				"source": "startup"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/transcript.json",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
		},
		{
			name: "Resume event",
			jsonInput: `{
				"session_id": "def456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "SessionStart",
				"source": "resume"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "def456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  SessionStart,
				},
				Source: "resume",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input SessionStartInput
			err := json.Unmarshal([]byte(tt.jsonInput), &input)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if input.SessionID != tt.want.SessionID {
				t.Errorf("SessionID: expected %s, got %s", tt.want.SessionID, input.SessionID)
			}
			if input.TranscriptPath != tt.want.TranscriptPath {
				t.Errorf("TranscriptPath: expected %s, got %s", tt.want.TranscriptPath, input.TranscriptPath)
			}
			if input.HookEventName != tt.want.HookEventName {
				t.Errorf("HookEventName: expected %s, got %s", tt.want.HookEventName, input.HookEventName)
			}
			if input.Source != tt.want.Source {
				t.Errorf("Source: expected %s, got %s", tt.want.Source, input.Source)
			}
		})
	}
}
