package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckUserPromptSubmitCondition_IsPrimeTurn(t *testing.T) {
	// テスト用のtranscriptファイルを作成
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.json")

	// 3つのユーザープロンプトを含むtranscriptを作成（現在は4つ目になる）
	transcriptContent := `{"type": "user", "sessionId": "test-session"}
{"type": "assistant", "sessionId": "test-session"}
{"type": "user", "sessionId": "test-session"}
{"type": "assistant", "sessionId": "test-session"}
{"type": "user", "sessionId": "test-session"}
{"type": "assistant", "sessionId": "test-session"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	tests := []struct {
		name      string
		condition Condition
		input     *UserPromptSubmitInput
		want      bool
		wantErr   bool
	}{
		{
			name: "Prime turn (4th turn is not prime) - should not match when value=true",
			condition: Condition{
				Type:  ConditionIsPrimeTurn,
				Value: "true",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Non-prime turn (4th turn) - should match when value=false",
			condition: Condition{
				Type:  ConditionIsPrimeTurn,
				Value: "false",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkUserPromptSubmitCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkUserPromptSubmitCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkUserPromptSubmitCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckUserPromptSubmitCondition_IsPrimeTurn_PrimeNumber(t *testing.T) {
	// テスト用のtranscriptファイルを作成
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.json")

	// 1つのユーザープロンプトを含むtranscriptを作成（現在は2つ目になる - 素数）
	transcriptContent := `{"type": "user", "sessionId": "test-session"}
{"type": "assistant", "sessionId": "test-session"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	tests := []struct {
		name      string
		condition Condition
		input     *UserPromptSubmitInput
		want      bool
		wantErr   bool
	}{
		{
			name: "Prime turn (2nd turn is prime) - should match when value=true",
			condition: Condition{
				Type:  ConditionIsPrimeTurn,
				Value: "true",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Prime turn (2nd turn) - should not match when value=false",
			condition: Condition{
				Type:  ConditionIsPrimeTurn,
				Value: "false",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Prime turn with default value (empty) - should match",
			condition: Condition{
				Type:  ConditionIsPrimeTurn,
				Value: "",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkUserPromptSubmitCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkUserPromptSubmitCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkUserPromptSubmitCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
