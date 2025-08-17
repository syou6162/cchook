package main

import (
	"testing"
)

func TestCheckCwdConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PreToolUseInput
		want      bool
		wantErr   bool
	}{
		{
			name: "cwd_is exact match",
			condition: Condition{
				Type:  ConditionCwdIs,
				Value: "/Users/yasuhisa.yoshida/work/cchook",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_is no match",
			condition: Condition{
				Type:  ConditionCwdIs,
				Value: "/Users/yasuhisa.yoshida/work/other",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cwd_is_not matches when different",
			condition: Condition{
				Type:  ConditionCwdIsNot,
				Value: "/Users/yasuhisa.yoshida/work/other",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_is_not doesn't match when same",
			condition: Condition{
				Type:  ConditionCwdIsNot,
				Value: "/Users/yasuhisa.yoshida/work/cchook",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cwd_contains matches substring",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "cchook",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_contains matches path segment",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "/work/",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_contains no match",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "other-project",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cwd_not_contains matches when not present",
			condition: Condition{
				Type:  ConditionCwdNotContains,
				Value: "other-project",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_not_contains doesn't match when present",
			condition: Condition{
				Type:  ConditionCwdNotContains,
				Value: "cchook",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/Users/yasuhisa.yoshida/work/cchook",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "cwd_contains works with empty cwd",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "test",
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					Cwd: "",
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPreToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPreToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPreToolUseCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// PostToolUseでも同じ条件が使えることを確認
func TestCheckCwdConditionsPostToolUse(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PostToolUseInput
		want      bool
		wantErr   bool
	}{
		{
			name: "cwd_contains in PostToolUse",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "project",
			},
			input: &PostToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/home/user/project/src",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_is_not in PostToolUse",
			condition: Condition{
				Type:  ConditionCwdIsNot,
				Value: "/tmp",
			},
			input: &PostToolUseInput{
				BaseInput: BaseInput{
					Cwd: "/home/user/project",
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPostToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPostToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPostToolUseCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// UserPromptSubmitでも同じ条件が使えることを確認
func TestCheckCwdConditionsUserPromptSubmit(t *testing.T) {
	// テスト用のtranscriptファイルを作成
	transcriptPath := createTestTranscript(t, "test-session", 0)

	tests := []struct {
		name      string
		condition Condition
		input     *UserPromptSubmitInput
		want      bool
		wantErr   bool
	}{
		{
			name: "cwd_contains in UserPromptSubmit",
			condition: Condition{
				Type:  ConditionCwdContains,
				Value: "github.com",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					Cwd:            "/Users/developer/go/src/github.com/user/repo",
					SessionID:      "test-session",
					TranscriptPath: transcriptPath,
				},
				Prompt: "test prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_is in UserPromptSubmit",
			condition: Condition{
				Type:  ConditionCwdIs,
				Value: "/Users/developer/go/src/github.com/user/repo",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					Cwd:            "/Users/developer/go/src/github.com/user/repo",
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

// SessionStartでも同じ条件が使えることを確認
func TestCheckCwdConditionsSessionStart(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *SessionStartInput
		want      bool
		wantErr   bool
	}{
		{
			name: "cwd_not_contains in SessionStart",
			condition: Condition{
				Type:  ConditionCwdNotContains,
				Value: "temp",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					Cwd: "/home/user/projects/important",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cwd_is_not in SessionStart",
			condition: Condition{
				Type:  ConditionCwdIsNot,
				Value: "/",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					Cwd: "/home/user",
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkSessionStartCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSessionStartCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSessionStartCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
