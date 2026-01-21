package main

import (
	"encoding/json"
	"fmt"
)

// CommandRunner is an interface for executing shell commands.
// This interface allows for dependency injection in tests.
type CommandRunner interface {
	// RunCommand executes a shell command with optional stdin data.
	RunCommand(cmd string, useStdin bool, data interface{}) error
	// RunCommandWithOutput executes a shell command and returns stdout, stderr, exit code, and error.
	RunCommandWithOutput(cmd string, useStdin bool, data interface{}) (stdout, stderr string, exitCode int, err error)
}

// イベントタイプのenum定義
type HookEventType string

const (
	PreToolUse        HookEventType = "PreToolUse"
	PostToolUse       HookEventType = "PostToolUse"
	PermissionRequest HookEventType = "PermissionRequest"
	Notification      HookEventType = "Notification"
	Stop              HookEventType = "Stop"
	SubagentStop      HookEventType = "SubagentStop"
	PreCompact        HookEventType = "PreCompact"
	SessionStart      HookEventType = "SessionStart"
	SessionEnd        HookEventType = "SessionEnd"
	UserPromptSubmit  HookEventType = "UserPromptSubmit"
)

// IsValid validates whether the HookEventType is a recognized event type.
func (e HookEventType) IsValid() bool {
	switch e {
	case PreToolUse, PostToolUse, PermissionRequest, Notification, Stop, SubagentStop, PreCompact, SessionStart, SessionEnd, UserPromptSubmit:
		return true
	default:
		return false
	}
}

// 共通インターフェース
type HookInput interface {
	GetEventType() HookEventType
	GetToolName() string
}

// 共通フィールド
type BaseInput struct {
	SessionID      string        `json:"session_id"`
	TranscriptPath string        `json:"transcript_path"`
	Cwd            string        `json:"cwd,omitempty"`
	PermissionMode string        `json:"permission_mode,omitempty"`
	HookEventName  HookEventType `json:"hook_event_name"`
}

// GetEventType returns the hook event type from the base input.
func (b BaseInput) GetEventType() HookEventType {
	return b.HookEventName
}

// Tool input structures - 全ツール共通構造と仮定
type ToolInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	Command  string `json:"command"`
	URL      string `json:"url"`    // WebFetch用
	Prompt   string `json:"prompt"` // WebFetch用
}

// PreToolUse用
type PreToolUseInput struct {
	BaseInput
	ToolName  string    `json:"tool_name"`
	ToolInput ToolInput `json:"tool_input"`
}

// GetToolName returns the tool name from the PreToolUse input.
func (p *PreToolUseInput) GetToolName() string {
	return p.ToolName
}

// PermissionRequest用 - PreToolUseと同じ入力スキーマ
type PermissionRequestInput = PreToolUseInput

// Tool response structures - ツールによって配列またはオブジェクトのパターンに対応
type ToolResponse json.RawMessage

// PostToolUse用
type PostToolUseInput struct {
	BaseInput
	ToolName     string       `json:"tool_name"`
	ToolInput    ToolInput    `json:"tool_input"`
	ToolResponse ToolResponse `json:"tool_response"`
}

// GetToolName returns the tool name from the PostToolUse input.
func (p *PostToolUseInput) GetToolName() string {
	return p.ToolName
}

// Notification用
type NotificationInput struct {
	BaseInput
	Message string `json:"message"`
}

// GetToolName returns an empty string as Notification events have no associated tool.
func (n *NotificationInput) GetToolName() string {
	return ""
}

// Stop用
type StopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

// GetToolName returns an empty string as Stop events have no associated tool.
func (s *StopInput) GetToolName() string {
	return ""
}

// SubagentStop用
type SubagentStopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

// GetToolName returns an empty string as SubagentStop events have no associated tool.
func (s *SubagentStopInput) GetToolName() string {
	return ""
}

// PreCompact用
type PreCompactInput struct {
	BaseInput
	Trigger            string `json:"trigger"` // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions"`
}

// GetToolName returns an empty string as PreCompact events have no associated tool.
func (p *PreCompactInput) GetToolName() string {
	return ""
}

// SessionStart用
type SessionStartInput struct {
	BaseInput
	Source string `json:"source"` // "startup", "resume", "clear", or "compact"
}

// GetToolName returns an empty string as SessionStart events have no associated tool.
func (s *SessionStartInput) GetToolName() string {
	return ""
}

// SessionStart JSON出力用の構造体

// SessionStartOutput はSessionStartフックのJSON出力全体を表す（Claude Code共通フィールド含む）
type SessionStartOutput struct {
	Continue           bool                            `json:"continue"`
	StopReason         string                          `json:"stopReason,omitempty"`
	SuppressOutput     bool                            `json:"suppressOutput,omitempty"`
	SystemMessage      string                          `json:"systemMessage,omitempty"`
	HookSpecificOutput *SessionStartHookSpecificOutput `json:"hookSpecificOutput"`
}

// SessionStartHookSpecificOutput はSessionStart固有の出力フィールド
type SessionStartHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// ActionOutput はアクション実行結果を表す内部型（JSONには直接出力されない）
type ActionOutput struct {
	Continue                 bool
	Decision                 string                 // "block" or "" (internal: empty string will be omitted from JSON via omitempty; UserPromptSubmit only, empty for SessionStart)
	PermissionDecision       string                 // "allow", "deny", or "ask" (PreToolUse only, empty for SessionStart/UserPromptSubmit)
	PermissionDecisionReason string                 // Reason for permission decision (PreToolUse only)
	UpdatedInput             map[string]interface{} // Updated tool input parameters (PreToolUse only)
	Behavior                 string                 // "allow" or "deny" (PermissionRequest only)
	Message                  string                 // Deny message (PermissionRequest only)
	Interrupt                bool                   // Interrupt flag for deny (PermissionRequest only)
	StopReason               string
	SuppressOutput           bool
	SystemMessage            string
	HookEventName            string
	AdditionalContext        string
}

// PreToolUseOutput represents the complete JSON output structure for PreToolUse hooks
// following Claude Code JSON specification for Phase 3
type PreToolUseOutput struct {
	Continue           bool                          `json:"continue"`
	StopReason         string                        `json:"stopReason,omitempty"`
	SuppressOutput     bool                          `json:"suppressOutput,omitempty"`
	SystemMessage      string                        `json:"systemMessage,omitempty"`
	HookSpecificOutput *PreToolUseHookSpecificOutput `json:"hookSpecificOutput"` // Required for PreToolUse (not omitempty)
}

// PreToolUseHookSpecificOutput represents the hookSpecificOutput field for PreToolUse hooks
type PreToolUseHookSpecificOutput struct {
	HookEventName            string                 `json:"hookEventName"`      // Always "PreToolUse"
	PermissionDecision       string                 `json:"permissionDecision"` // Required: "allow", "deny", or "ask"
	PermissionDecisionReason string                 `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             map[string]interface{} `json:"updatedInput,omitempty"`
}

// PermissionRequestOutput represents the complete JSON output structure for PermissionRequest hooks
// following Claude Code JSON specification for Phase 5
type PermissionRequestOutput struct {
	Continue           bool                                 `json:"continue"`
	StopReason         string                               `json:"stopReason,omitempty"`
	SuppressOutput     bool                                 `json:"suppressOutput,omitempty"`
	SystemMessage      string                               `json:"systemMessage,omitempty"`
	HookSpecificOutput *PermissionRequestHookSpecificOutput `json:"hookSpecificOutput"` // Required
}

// PermissionRequestHookSpecificOutput represents the hookSpecificOutput field for PermissionRequest hooks
type PermissionRequestHookSpecificOutput struct {
	HookEventName string                     `json:"hookEventName"` // Always "PermissionRequest"
	Decision      *PermissionRequestDecision `json:"decision"`      // Required
}

// PermissionRequestDecision represents the decision object within PermissionRequest hookSpecificOutput
type PermissionRequestDecision struct {
	Behavior     string                 `json:"behavior"`               // Required: "allow" or "deny"
	UpdatedInput map[string]interface{} `json:"updatedInput,omitempty"` // Optional: allow時のみ
	Message      string                 `json:"message,omitempty"`      // Optional: deny時のみ
	Interrupt    bool                   `json:"interrupt,omitempty"`    // Optional: deny時のみ、デフォルトfalse
}

// UserPromptSubmit用
type UserPromptSubmitInput struct {
	BaseInput
	Prompt string `json:"prompt"` // ユーザーが送信したプロンプト
}

// GetToolName returns an empty string as UserPromptSubmit events have no associated tool.
func (u *UserPromptSubmitInput) GetToolName() string {
	return ""
}

// UserPromptSubmitOutput はUserPromptSubmitフックのJSON出力全体を表す（Claude Code共通フィールド含む）
type UserPromptSubmitOutput struct {
	Continue           bool                                `json:"continue"`
	Decision           string                              `json:"decision,omitempty"` // "block" only; omit field to allow prompt
	StopReason         string                              `json:"stopReason,omitempty"`
	SuppressOutput     bool                                `json:"suppressOutput,omitempty"`
	SystemMessage      string                              `json:"systemMessage,omitempty"`
	HookSpecificOutput *UserPromptSubmitHookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// UserPromptSubmitHookSpecificOutput はUserPromptSubmit固有の出力フィールド
type UserPromptSubmitHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// SessionEnd用
type SessionEndInput struct {
	BaseInput
	Reason string `json:"reason"` // "clear", "logout", "prompt_input_exit", or "other"
}

// GetToolName returns an empty string as SessionEnd events have no associated tool.
func (s *SessionEndInput) GetToolName() string {
	return ""
}

// Hook共通インターフェース
type Hook interface {
	GetMatcher() string
	HasConditions() bool
	GetEventType() HookEventType
}

// イベントタイプ毎の設定構造体
type PreToolUseHook struct {
	Matcher    string      `yaml:"matcher"`
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type PostToolUseHook struct {
	Matcher    string      `yaml:"matcher"`
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type PermissionRequestHook struct {
	Matcher    string      `yaml:"matcher"`
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type NotificationHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type StopHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type SubagentStopHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type PreCompactHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type SessionStartHook struct {
	Matcher    string      `yaml:"matcher"` // "startup", "resume", or "clear"
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type UserPromptSubmitHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

type SessionEndHook struct {
	Conditions []Condition `yaml:"conditions,omitempty"`
	Actions    []Action    `yaml:"actions"`
}

// 共通の条件構造体
// ConditionType represents the type of condition to check (opaque struct)
type ConditionType struct{ v string }

// String returns the string representation of the condition type
func (c ConditionType) String() string {
	return c.v
}

// Predefined valid condition types (singletons)
var (
	// Common conditions (all events)
	ConditionFileExists             = ConditionType{"file_exists"}
	ConditionFileExistsRecursive    = ConditionType{"file_exists_recursive"}
	ConditionFileNotExists          = ConditionType{"file_not_exists"}
	ConditionFileNotExistsRecursive = ConditionType{"file_not_exists_recursive"}
	ConditionDirExists              = ConditionType{"dir_exists"}
	ConditionDirExistsRecursive     = ConditionType{"dir_exists_recursive"}
	ConditionDirNotExists           = ConditionType{"dir_not_exists"}
	ConditionDirNotExistsRecursive  = ConditionType{"dir_not_exists_recursive"}

	// Tool-related conditions (PreToolUse/PostToolUse)
	ConditionFileExtension     = ConditionType{"file_extension"}
	ConditionCommandContains   = ConditionType{"command_contains"}
	ConditionCommandStartsWith = ConditionType{"command_starts_with"}
	ConditionURLStartsWith     = ConditionType{"url_starts_with"}

	// Prompt-related conditions (UserPromptSubmit)
	ConditionPromptRegex   = ConditionType{"prompt_regex"}
	ConditionEveryNPrompts = ConditionType{"every_n_prompts"}

	// Reason-related conditions (SessionEnd)
	ConditionReasonIs = ConditionType{"reason_is"}

	// Git-related conditions (PreToolUse for Bash commands)
	ConditionGitTrackedFileOperation = ConditionType{"git_tracked_file_operation"}
	ConditionCwdIs                   = ConditionType{"cwd_is"}
	ConditionCwdIsNot                = ConditionType{"cwd_is_not"}
	ConditionCwdContains             = ConditionType{"cwd_contains"}
	ConditionCwdNotContains          = ConditionType{"cwd_not_contains"}
)

// UnmarshalYAML implements yaml.Unmarshaler for ConditionType
func (c *ConditionType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	switch s {
	case "file_exists":
		*c = ConditionFileExists
	case "file_exists_recursive":
		*c = ConditionFileExistsRecursive
	case "file_not_exists":
		*c = ConditionFileNotExists
	case "file_not_exists_recursive":
		*c = ConditionFileNotExistsRecursive
	case "dir_exists":
		*c = ConditionDirExists
	case "dir_exists_recursive":
		*c = ConditionDirExistsRecursive
	case "dir_not_exists":
		*c = ConditionDirNotExists
	case "dir_not_exists_recursive":
		*c = ConditionDirNotExistsRecursive
	case "file_extension":
		*c = ConditionFileExtension
	case "command_contains":
		*c = ConditionCommandContains
	case "command_starts_with":
		*c = ConditionCommandStartsWith
	case "url_starts_with":
		*c = ConditionURLStartsWith
	case "prompt_regex":
		*c = ConditionPromptRegex
	case "every_n_prompts":
		*c = ConditionEveryNPrompts
	case "reason_is":
		*c = ConditionReasonIs
	case "git_tracked_file_operation":
		*c = ConditionGitTrackedFileOperation
	case "cwd_is":
		*c = ConditionCwdIs
	case "cwd_is_not":
		*c = ConditionCwdIsNot
	case "cwd_contains":
		*c = ConditionCwdContains
	case "cwd_not_contains":
		*c = ConditionCwdNotContains
	default:
		return fmt.Errorf("invalid condition type: %s", s)
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler for ConditionType
func (c ConditionType) MarshalYAML() (interface{}, error) {
	return c.v, nil
}

type Condition struct {
	Type  ConditionType `yaml:"type"`
	Value string        `yaml:"value"`
}

// Action - 全てのイベントタイプで共通のアクション構造体
type Action struct {
	Type               string  `yaml:"type"`
	Command            string  `yaml:"command,omitempty"`
	Message            string  `yaml:"message,omitempty"`
	UseStdin           bool    `yaml:"use_stdin,omitempty"`
	ExitStatus         *int    `yaml:"exit_status,omitempty"`
	Continue           *bool   `yaml:"continue,omitempty"`
	Decision           *string `yaml:"decision,omitempty"`            // "block" only, or omit field entirely (internal: empty string will be omitted from JSON; UserPromptSubmit only)
	PermissionDecision *string `yaml:"permission_decision,omitempty"` // "allow", "deny", or "ask" (PreToolUse only)
	Behavior           *string `yaml:"behavior,omitempty"`            // "allow" or "deny" (PermissionRequest only)
	Interrupt          *bool   `yaml:"interrupt,omitempty"`           // deny時のみ (PermissionRequest only)
}

// 設定ファイル構造
type Config struct {
	PreToolUse        []PreToolUseHook        `yaml:"PreToolUse,omitempty"`
	PostToolUse       []PostToolUseHook       `yaml:"PostToolUse,omitempty"`
	PermissionRequest []PermissionRequestHook `yaml:"PermissionRequest,omitempty"`
	Notification      []NotificationHook      `yaml:"Notification,omitempty"`
	Stop              []StopHook              `yaml:"Stop,omitempty"`
	SubagentStop      []SubagentStopHook      `yaml:"SubagentStop,omitempty"`
	PreCompact        []PreCompactHook        `yaml:"PreCompact,omitempty"`
	SessionStart      []SessionStartHook      `yaml:"SessionStart,omitempty"`
	SessionEnd        []SessionEndHook        `yaml:"SessionEnd,omitempty"`
	UserPromptSubmit  []UserPromptSubmitHook  `yaml:"UserPromptSubmit,omitempty"`
}
