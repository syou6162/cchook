package main

import (
	"encoding/json"
	"fmt"
)

// イベントタイプのenum定義
type HookEventType string

const (
	PreToolUse       HookEventType = "PreToolUse"
	PostToolUse      HookEventType = "PostToolUse"
	Notification     HookEventType = "Notification"
	Stop             HookEventType = "Stop"
	SubagentStop     HookEventType = "SubagentStop"
	PreCompact       HookEventType = "PreCompact"
	SessionStart     HookEventType = "SessionStart"
	SessionEnd       HookEventType = "SessionEnd"
	UserPromptSubmit HookEventType = "UserPromptSubmit"
)

// イベントタイプの妥当性検証
func (e HookEventType) IsValid() bool {
	switch e {
	case PreToolUse, PostToolUse, Notification, Stop, SubagentStop, PreCompact, SessionStart, SessionEnd, UserPromptSubmit:
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

func (p *PreToolUseInput) GetToolName() string {
	return p.ToolName
}

// Tool response structures - ツールによって配列またはオブジェクトのパターンに対応
type ToolResponse json.RawMessage

// PostToolUse用
type PostToolUseInput struct {
	BaseInput
	ToolName     string       `json:"tool_name"`
	ToolInput    ToolInput    `json:"tool_input"`
	ToolResponse ToolResponse `json:"tool_response"`
}

func (p *PostToolUseInput) GetToolName() string {
	return p.ToolName
}

// Notification用
type NotificationInput struct {
	BaseInput
	Message string `json:"message"`
}

func (n *NotificationInput) GetToolName() string {
	return ""
}

// Stop用
type StopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

func (s *StopInput) GetToolName() string {
	return ""
}

// SubagentStop用
type SubagentStopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

func (s *SubagentStopInput) GetToolName() string {
	return ""
}

// PreCompact用
type PreCompactInput struct {
	BaseInput
	Trigger            string `json:"trigger"` // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions"`
}

func (p *PreCompactInput) GetToolName() string {
	return ""
}

// SessionStart用
type SessionStartInput struct {
	BaseInput
	Source string `json:"source"` // "startup", "resume", "clear", or "compact"
}

func (s *SessionStartInput) GetToolName() string {
	return ""
}

// UserPromptSubmit用
type UserPromptSubmitInput struct {
	BaseInput
	Prompt string `json:"prompt"` // ユーザーが送信したプロンプト
}

func (u *UserPromptSubmitInput) GetToolName() string {
	return ""
}

// SessionEnd用
type SessionEndInput struct {
	BaseInput
	Reason string `json:"reason"` // "clear", "logout", "prompt_input_exit", or "other"
}

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
	Matcher    string             `yaml:"matcher"`
	Conditions []Condition        `yaml:"conditions,omitempty"`
	Actions    []PreToolUseAction `yaml:"actions"`
}

type PostToolUseHook struct {
	Matcher    string              `yaml:"matcher"`
	Conditions []Condition         `yaml:"conditions,omitempty"`
	Actions    []PostToolUseAction `yaml:"actions"`
}

type NotificationHook struct {
	Conditions []Condition          `yaml:"conditions,omitempty"`
	Actions    []NotificationAction `yaml:"actions"`
}

type StopHook struct {
	Conditions []Condition  `yaml:"conditions,omitempty"`
	Actions    []StopAction `yaml:"actions"`
}

type SubagentStopHook struct {
	Conditions []Condition          `yaml:"conditions,omitempty"`
	Actions    []SubagentStopAction `yaml:"actions"`
}

type PreCompactHook struct {
	Conditions []Condition        `yaml:"conditions,omitempty"`
	Actions    []PreCompactAction `yaml:"actions"`
}

type SessionStartHook struct {
	Matcher    string               `yaml:"matcher"` // "startup", "resume", or "clear"
	Conditions []Condition          `yaml:"conditions,omitempty"`
	Actions    []SessionStartAction `yaml:"actions"`
}

type UserPromptSubmitHook struct {
	Conditions []Condition              `yaml:"conditions,omitempty"`
	Actions    []UserPromptSubmitAction `yaml:"actions"`
}

type SessionEndHook struct {
	Conditions []Condition        `yaml:"conditions,omitempty"`
	Actions    []SessionEndAction `yaml:"actions"`
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
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	UseStdin   bool   `yaml:"use_stdin,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

// イベントタイプ毎のアクション構造体
// 全てActionを埋め込むだけだが、型として区別することで可読性を維持
type PreToolUseAction struct {
	Action `yaml:",inline"`
}

type PostToolUseAction struct {
	Action `yaml:",inline"`
}

type NotificationAction struct {
	Action `yaml:",inline"`
}

type StopAction struct {
	Action `yaml:",inline"`
}

type SubagentStopAction struct {
	Action `yaml:",inline"`
}

type PreCompactAction struct {
	Action `yaml:",inline"`
}

type SessionStartAction struct {
	Action `yaml:",inline"`
}

type UserPromptSubmitAction struct {
	Action `yaml:",inline"`
}

type SessionEndAction struct {
	Action `yaml:",inline"`
}

// 設定ファイル構造
type Config struct {
	PreToolUse       []PreToolUseHook       `yaml:"PreToolUse,omitempty"`
	PostToolUse      []PostToolUseHook      `yaml:"PostToolUse,omitempty"`
	Notification     []NotificationHook     `yaml:"Notification,omitempty"`
	Stop             []StopHook             `yaml:"Stop,omitempty"`
	SubagentStop     []SubagentStopHook     `yaml:"SubagentStop,omitempty"`
	PreCompact       []PreCompactHook       `yaml:"PreCompact,omitempty"`
	SessionStart     []SessionStartHook     `yaml:"SessionStart,omitempty"`
	SessionEnd       []SessionEndHook       `yaml:"SessionEnd,omitempty"`
	UserPromptSubmit []UserPromptSubmitHook `yaml:"UserPromptSubmit,omitempty"`
}
