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
	UserPromptSubmit HookEventType = "UserPromptSubmit"
)

// イベントタイプの妥当性検証
func (e HookEventType) IsValid() bool {
	switch e {
	case PreToolUse, PostToolUse, Notification, Stop, SubagentStop, PreCompact, SessionStart, UserPromptSubmit:
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
	Source string `json:"source"` // "startup", "resume", or "clear"
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

// Hook共通インターフェース
type Hook interface {
	GetMatcher() string
	HasConditions() bool
	GetEventType() HookEventType
}

// Action共通インターフェース
type Action interface {
	GetType() string
	GetCommand() string
	GetMessage() string
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
	ConditionFileExists          = ConditionType{"file_exists"}
	ConditionFileExistsRecursive = ConditionType{"file_exists_recursive"}

	// Tool-related conditions (PreToolUse/PostToolUse)
	ConditionFileExtension     = ConditionType{"file_extension"}
	ConditionCommandContains   = ConditionType{"command_contains"}
	ConditionCommandStartsWith = ConditionType{"command_starts_with"}
	ConditionURLStartsWith     = ConditionType{"url_starts_with"}

	// Prompt-related conditions (UserPromptSubmit)
	ConditionPromptContains   = ConditionType{"prompt_contains"}
	ConditionPromptStartsWith = ConditionType{"prompt_starts_with"}
	ConditionPromptEndsWith   = ConditionType{"prompt_ends_with"}
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
	case "file_extension":
		*c = ConditionFileExtension
	case "command_contains":
		*c = ConditionCommandContains
	case "command_starts_with":
		*c = ConditionCommandStartsWith
	case "url_starts_with":
		*c = ConditionURLStartsWith
	case "prompt_contains":
		*c = ConditionPromptContains
	case "prompt_starts_with":
		*c = ConditionPromptStartsWith
	case "prompt_ends_with":
		*c = ConditionPromptEndsWith
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

// イベントタイプ毎のアクション構造体
type PreToolUseAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type PostToolUseAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type NotificationAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type StopAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type SubagentStopAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type PreCompactAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type SessionStartAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
}

type UserPromptSubmitAction struct {
	Type       string `yaml:"type"`
	Command    string `yaml:"command,omitempty"`
	Message    string `yaml:"message,omitempty"`
	ExitStatus *int   `yaml:"exit_status,omitempty"`
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
	UserPromptSubmit []UserPromptSubmitHook `yaml:"UserPromptSubmit,omitempty"`
}
