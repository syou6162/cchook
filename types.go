package main

// イベントタイプのenum定義
type HookEventType string

const (
	PreToolUse   HookEventType = "PreToolUse"
	PostToolUse  HookEventType = "PostToolUse"
	Notification HookEventType = "Notification"
	Stop         HookEventType = "Stop"
	SubagentStop HookEventType = "SubagentStop"
	PreCompact   HookEventType = "PreCompact"
)

// イベントタイプの妥当性検証
func (e HookEventType) IsValid() bool {
	switch e {
	case PreToolUse, PostToolUse, Notification, Stop, SubagentStop, PreCompact:
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

// Tool response structures - ドキュメントから確認できた構造
type ToolResponse struct {
	FilePath string `json:"filePath,omitempty"`
	Success  bool   `json:"success,omitempty"`
}

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
	Matcher    string                `yaml:"matcher"`
	Conditions []PreToolUseCondition `yaml:"conditions,omitempty"`
	Actions    []PreToolUseAction    `yaml:"actions"`
}

type PostToolUseHook struct {
	Matcher    string                 `yaml:"matcher"`
	Conditions []PostToolUseCondition `yaml:"conditions,omitempty"`
	Actions    []PostToolUseAction    `yaml:"actions"`
}

type NotificationHook struct {
	Actions []NotificationAction `yaml:"actions"`
}

type StopHook struct {
	Actions []StopAction `yaml:"actions"`
}

type SubagentStopHook struct {
	Actions []SubagentStopAction `yaml:"actions"`
}

type PreCompactHook struct {
	Actions []PreCompactAction `yaml:"actions"`
}

// イベントタイプ毎の条件構造体
type PreToolUseCondition struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

type PostToolUseCondition struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// 共通のベースアクション構造体
type BaseAction struct {
	Type           string  `yaml:"type"`
	Command        string  `yaml:"command,omitempty"`
	Continue       *bool   `yaml:"continue,omitempty"`
	StopReason     *string `yaml:"stop_reason,omitempty"`
	SuppressOutput *bool   `yaml:"suppress_output,omitempty"`
}

// イベントタイプ毎のアクション構造体（固有フィールドのみ追加）
type PreToolUseAction struct {
	BaseAction
	PermissionDecision       *string `yaml:"permission_decision,omitempty"`
	PermissionDecisionReason *string `yaml:"permission_reason,omitempty"`
}

type PostToolUseAction struct {
	BaseAction
	Decision *string `yaml:"decision,omitempty"`
	Reason   *string `yaml:"reason,omitempty"`
}

type NotificationAction struct {
	BaseAction
	// 固有フィールドなし（共通フィールドのみ）
}

type StopAction struct {
	BaseAction
	Decision *string `yaml:"decision,omitempty"`
	Reason   *string `yaml:"reason,omitempty"`
}

type SubagentStopAction struct {
	BaseAction
	Decision *string `yaml:"decision,omitempty"`
	Reason   *string `yaml:"reason,omitempty"`
}

type PreCompactAction struct {
	BaseAction
	// 固有フィールドなし（共通フィールドのみ）
}

// Claude Code互換の構造化JSON出力構造体

// 共通の基本出力フィールド
type BaseHookOutput struct {
	Continue       *bool   `json:"continue,omitempty"`
	StopReason     *string `json:"stopReason,omitempty"`
	SuppressOutput *bool   `json:"suppressOutput,omitempty"`
}

// PreToolUse固有の出力
type PreToolUseSpecificOutput struct {
	HookEventName            string  `json:"hookEventName"`
	PermissionDecision       *string `json:"permissionDecision,omitempty"` // "allow", "deny", "ask"
	PermissionDecisionReason *string `json:"permissionDecisionReason,omitempty"`
}

type PreToolUseOutput struct {
	BaseHookOutput
	Decision           *string                   `json:"decision,omitempty"` // 非推奨だが互換性のため
	Reason             *string                   `json:"reason,omitempty"`   // 非推奨だが互換性のため
	HookSpecificOutput *PreToolUseSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// PostToolUse固有の出力
type PostToolUseSpecificOutput struct {
	HookEventName string `json:"hookEventName"`
}

type PostToolUseOutput struct {
	BaseHookOutput
	Decision           *string                    `json:"decision,omitempty"` // "block"
	Reason             *string                    `json:"reason,omitempty"`
	HookSpecificOutput *PostToolUseSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// UserPromptSubmit固有の出力
type UserPromptSubmitSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"`
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

type UserPromptSubmitOutput struct {
	Decision           *string                         `json:"decision,omitempty"` // "block"
	Reason             *string                         `json:"reason,omitempty"`
	HookSpecificOutput *UserPromptSubmitSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// Stop/SubagentStop固有の出力
type StopSpecificOutput struct {
	HookEventName string `json:"hookEventName"`
}

type StopOutput struct {
	BaseHookOutput
	Decision           *string             `json:"decision,omitempty"` // "block"
	Reason             *string             `json:"reason,omitempty"`   // 必須（blockの場合）
	HookSpecificOutput *StopSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type SubagentStopOutput struct {
	BaseHookOutput
	Decision           *string             `json:"decision,omitempty"` // "block"
	Reason             *string             `json:"reason,omitempty"`   // 必須（blockの場合）
	HookSpecificOutput *StopSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// Notification固有の出力
type NotificationSpecificOutput struct {
	HookEventName string `json:"hookEventName"`
}

type NotificationOutput struct {
	BaseHookOutput
	HookSpecificOutput *NotificationSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// PreCompact固有の出力
type PreCompactSpecificOutput struct {
	HookEventName string `json:"hookEventName"`
}

type PreCompactOutput struct {
	BaseHookOutput
	HookSpecificOutput *PreCompactSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// 設定ファイル構造
type Config struct {
	PreToolUse   []PreToolUseHook   `yaml:"PreToolUse,omitempty"`
	PostToolUse  []PostToolUseHook  `yaml:"PostToolUse,omitempty"`
	Notification []NotificationHook `yaml:"Notification,omitempty"`
	Stop         []StopHook         `yaml:"Stop,omitempty"`
	SubagentStop []SubagentStopHook `yaml:"SubagentStop,omitempty"`
	PreCompact   []PreCompactHook   `yaml:"PreCompact,omitempty"`
}
