package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// イベントタイプのenum定義
type HookEventType string

const (
	PreToolUse    HookEventType = "PreToolUse"
	PostToolUse   HookEventType = "PostToolUse"
	Notification  HookEventType = "Notification"
	Stop          HookEventType = "Stop"
	SubagentStop  HookEventType = "SubagentStop"
	PreCompact    HookEventType = "PreCompact"
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
	HookEventName  HookEventType `json:"hook_event_name"`
}

func (b BaseInput) GetEventType() HookEventType {
	return b.HookEventName
}

// PreToolUse用
type PreToolUseInput struct {
	BaseInput
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

func (p *PreToolUseInput) GetToolName() string {
	return p.ToolName
}

// PostToolUse用
type PostToolUseInput struct {
	BaseInput
	ToolName     string                 `json:"tool_name"`
	ToolInput    map[string]interface{} `json:"tool_input"`
	ToolResponse map[string]interface{} `json:"tool_response"`
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

// イベントタイプ毎のアクション構造体
type PreToolUseAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type PostToolUseAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type NotificationAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type StopAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type SubagentStopAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type PreCompactAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	Message string `yaml:"message,omitempty"`
}

// 設定ファイル構造
type Config struct {
	PreToolUse    []PreToolUseHook    `yaml:"PreToolUse,omitempty"`
	PostToolUse   []PostToolUseHook   `yaml:"PostToolUse,omitempty"`
	Notification  []NotificationHook  `yaml:"Notification,omitempty"`
	Stop          []StopHook          `yaml:"Stop,omitempty"`
	SubagentStop  []SubagentStopHook  `yaml:"SubagentStop,omitempty"`
	PreCompact    []PreCompactHook    `yaml:"PreCompact,omitempty"`
}

func main() {
	configPath := flag.String("config", "", "Path to config file")
	command := flag.String("command", "run", "Command to execute (run, dry-run)")
	eventType := flag.String("event", "", "Event type for run/dry-run command")
	flag.Parse()

	if (*command == "run" || *command == "dry-run") && *eventType == "" {
		fmt.Fprintf(os.Stderr, "Error: event type is required for %s command\n", *command)
		os.Exit(1)
	}

	// イベントタイプの妥当性検証
	if *command == "run" || *command == "dry-run" {
		eventType := HookEventType(*eventType)
		if !eventType.IsValid() {
			fmt.Fprintf(os.Stderr, "Error: invalid event type '%s'. Valid types: PreToolUse, PostToolUse, Notification, Stop, SubagentStop, PreCompact\n", string(eventType))
			os.Exit(1)
		}
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch *command {
	case "run":
		err = runHooks(config, HookEventType(*eventType))
	case "dry-run":
		err = dryRunHooks(config, HookEventType(*eventType))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func getDefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "cchook", "config.yaml")
}

// ジェネリック入力パース関数
func parseInput[T HookInput](eventType HookEventType) (T, error) {
	var input T
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return input, fmt.Errorf("failed to decode %s input: %w", eventType, err)
	}
	return input, nil
}

func runHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return executePreToolUseHooks(config, input)
	case PostToolUse:
		input, err := parseInput[*PostToolUseInput](eventType)
		if err != nil {
			return err
		}
		return executePostToolUseHooks(config, input)
	case Notification:
		input, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		return executeNotificationHooks(config, input)
	case Stop:
		input, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		return executeStopHooks(config, input)
	case SubagentStop:
		input, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		return executeSubagentStopHooks(config, input)
	case PreCompact:
		input, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		return executePreCompactHooks(config, input)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func dryRunHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreToolUseHooks(config, input)
	case PostToolUse:
		input, err := parseInput[*PostToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPostToolUseHooks(config, input)
	case Notification:
		input, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		return dryRunNotificationHooks(config, input)
	case Stop:
		input, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunStopHooks(config, input)
	case SubagentStop:
		input, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSubagentStopHooks(config, input)
	case PreCompact:
		input, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreCompactHooks(config, input)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// イベント別のdry-run関数
func dryRunPreToolUseHooks(config *Config, input *PreToolUseInput) error {
	fmt.Println("=== PreToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PreToolUse {
		if shouldExecutePreToolUseHook(hook, input) {
			executed = true
			fmt.Printf("[Hook %d] Would execute:\n", i+1)
			for _, action := range hook.Actions {
				switch action.Type {
				case "command":
					cmd := replacePreToolUseVariables(action.Command, input)
					fmt.Printf("  Command: %s\n", cmd)
				case "output":
					fmt.Printf("  Message: %s\n", action.Message)
				}
			}
		}
	}
	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunPostToolUseHooks(config *Config, input *PostToolUseInput) error {
	fmt.Println("=== PostToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PostToolUse {
		if shouldExecutePostToolUseHook(hook, input) {
			executed = true
			fmt.Printf("[Hook %d] Would execute:\n", i+1)
			for _, action := range hook.Actions {
				switch action.Type {
				case "command":
					cmd := replacePostToolUseVariables(action.Command, input)
					fmt.Printf("  Command: %s\n", cmd)
				case "output":
					fmt.Printf("  Message: %s\n", action.Message)
				}
			}
		}
	}
	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

// 未実装イベント用のジェネリック関数
func dryRunUnimplementedHooks(eventType HookEventType) error {
	fmt.Printf("=== %s Hooks (Dry Run) ===\n", eventType)
	fmt.Println("No hooks implemented yet")
	return nil
}

func dryRunNotificationHooks(config *Config, input *NotificationInput) error {
	return dryRunUnimplementedHooks(Notification)
}

func dryRunStopHooks(config *Config, input *StopInput) error {
	return dryRunUnimplementedHooks(Stop)
}

func dryRunSubagentStopHooks(config *Config, input *SubagentStopInput) error {
	return dryRunUnimplementedHooks(SubagentStop)
}

func dryRunPreCompactHooks(config *Config, input *PreCompactInput) error {
	return dryRunUnimplementedHooks(PreCompact)
}

// イベント別のフック実行関数
func executePreToolUseHooks(config *Config, input *PreToolUseInput) error {
	for i, hook := range config.PreToolUse {
		if shouldExecutePreToolUseHook(hook, input) {
			if err := executePreToolUseHook(hook, input); err != nil {
				fmt.Fprintf(os.Stderr, "PreToolUse hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executePostToolUseHooks(config *Config, input *PostToolUseInput) error {
	for i, hook := range config.PostToolUse {
		if shouldExecutePostToolUseHook(hook, input) {
			if err := executePostToolUseHook(hook, input); err != nil {
				fmt.Fprintf(os.Stderr, "PostToolUse hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeNotificationHooks(config *Config, input *NotificationInput) error {
	// Notificationフック実行（後で実装）
	return nil
}

func executeStopHooks(config *Config, input *StopInput) error {
	// Stopフック実行（後で実装）
	return nil
}

func executeSubagentStopHooks(config *Config, input *SubagentStopInput) error {
	// SubagentStopフック実行（後で実装）
	return nil
}

func executePreCompactHooks(config *Config, input *PreCompactInput) error {
	// PreCompactフック実行（後で実装）
	return nil
}

func shouldExecutePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput) bool {
	// マッチャーチェック（正規表現風のパターンマッチング）
	if hook.Matcher != "" {
		matched := false
		for _, pattern := range strings.Split(hook.Matcher, "|") {
			if strings.Contains(input.ToolName, strings.TrimSpace(pattern)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		if !checkPreToolUseCondition(condition, input) {
			return false
		}
	}

	return true
}

func shouldExecutePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput) bool {
	// マッチャーチェック（正規表現風のパターンマッチング）
	if hook.Matcher != "" {
		matched := false
		for _, pattern := range strings.Split(hook.Matcher, "|") {
			if strings.Contains(input.ToolName, strings.TrimSpace(pattern)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		if !checkPostToolUseCondition(condition, input) {
			return false
		}
	}

	return true
}

func checkPreToolUseCondition(condition PreToolUseCondition, input *PreToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		if filePath, ok := input.ToolInput["file_path"].(string); ok {
			return strings.HasSuffix(filePath, condition.Value)
		}
	case "command_contains":
		if command, ok := input.ToolInput["command"].(string); ok {
			return strings.Contains(command, condition.Value)
		}
	}
	return false
}

func checkPostToolUseCondition(condition PostToolUseCondition, input *PostToolUseInput) bool {
	switch condition.Type {
	case "file_extension":
		if filePath, ok := input.ToolInput["file_path"].(string); ok {
			return strings.HasSuffix(filePath, condition.Value)
		}
	case "command_contains":
		if command, ok := input.ToolInput["command"].(string); ok {
			return strings.Contains(command, condition.Value)
		}
	}
	return false
}

func executePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput) error {
	for _, action := range hook.Actions {
		switch action.Type {
		case "command":
			cmd := replacePreToolUseVariables(action.Command, input)
			if err := runCommand(cmd); err != nil {
				return err
			}
		case "output":
			fmt.Println(action.Message)
		}
	}
	return nil
}

func executePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput) error {
	for _, action := range hook.Actions {
		switch action.Type {
		case "command":
			cmd := replacePostToolUseVariables(action.Command, input)
			if err := runCommand(cmd); err != nil {
				return err
			}
		case "output":
			fmt.Println(action.Message)
		}
	}
	return nil
}

func replacePreToolUseVariables(command string, input *PreToolUseInput) string {
	if filePath, ok := input.ToolInput["file_path"].(string); ok {
		command = strings.ReplaceAll(command, "{file_path}", filePath)
	}
	return command
}

func replacePostToolUseVariables(command string, input *PostToolUseInput) string {
	if filePath, ok := input.ToolInput["file_path"].(string); ok {
		command = strings.ReplaceAll(command, "{file_path}", filePath)
	}
	return command
}

func runCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}