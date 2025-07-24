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

// 共通フィールド
type BaseInput struct {
	SessionID      string        `json:"session_id"`
	TranscriptPath string        `json:"transcript_path"`
	HookEventName  HookEventType `json:"hook_event_name"`
}

// PreToolUse用
type PreToolUseInput struct {
	BaseInput
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// PostToolUse用
type PostToolUseInput struct {
	BaseInput
	ToolName     string                 `json:"tool_name"`
	ToolInput    map[string]interface{} `json:"tool_input"`
	ToolResponse map[string]interface{} `json:"tool_response"`
}

// Notification用
type NotificationInput struct {
	BaseInput
	Message string `json:"message"`
}

// Stop用
type StopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

// SubagentStop用
type SubagentStopInput struct {
	BaseInput
	StopHookActive bool `json:"stop_hook_active"`
}

// PreCompact用
type PreCompactInput struct {
	BaseInput
	Trigger            string `json:"trigger"` // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions"`
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
	command := flag.String("command", "run", "Command to execute (run, exec, test, dry-run)")
	eventType := flag.String("event", "", "Event type for run command")
	flag.Parse()

	if *command == "run" && *eventType == "" {
		fmt.Fprintf(os.Stderr, "Error: event type is required for run command\n")
		os.Exit(1)
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch *command {
	case "run":
		err = runHooks(config, *eventType)
	case "dry-run":
		err = dryRunHooks(config)
	case "test":
		err = testHooks(config)
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

func runHooks(config *Config, eventType string) error {
	input, err := readStdinJSON()
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	for _, hook := range config.Hooks {
		if shouldExecuteHook(hook, eventType, input) {
			if err := executeHook(hook, input); err != nil {
				fmt.Fprintf(os.Stderr, "Hook %s failed: %v\n", hook.Name, err)
			}
		}
	}

	return nil
}

func dryRunHooks(config *Config) error {
	input, err := readStdinJSON()
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	fmt.Println("Hooks that would be executed:")
	for _, hook := range config.Hooks {
		if shouldExecuteHook(hook, input.Event, input) {
			fmt.Printf("- %s: %s\n", hook.Name, hook.Description)
			for _, action := range hook.Actions {
				if action.Type == "command" {
					cmd := replaceVariables(action.Command, input)
					fmt.Printf("  Command: %s\n", cmd)
				}
			}
		}
	}

	return nil
}

func testHooks(config *Config) error {
	mockInput := &ClaudeInput{
		Event:    "PostToolUse",
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
		},
	}

	fmt.Println("Testing hooks with mock data:")
	for _, hook := range config.Hooks {
		if shouldExecuteHook(hook, mockInput.Event, mockInput) {
			fmt.Printf("Testing hook: %s\n", hook.Name)
			if err := executeHook(hook, mockInput); err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				fmt.Printf("  Success\n")
			}
		}
	}

	return nil
}

func readStdinJSON() (*ClaudeInput, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	var input ClaudeInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}

	return &input, nil
}

func shouldExecuteHook(hook Hook, eventType string, input *ClaudeInput) bool {
	eventMatch := false
	for _, event := range hook.Events {
		if event == eventType {
			eventMatch = true
			break
		}
	}
	if !eventMatch {
		return false
	}

	if hook.Matcher != "" && !strings.Contains(hook.Matcher, input.ToolName) {
		return false
	}

	for _, condition := range hook.Conditions {
		if !checkCondition(condition, input) {
			return false
		}
	}

	return true
}

func checkCondition(condition Condition, input *ClaudeInput) bool {
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

func executeHook(hook Hook, input *ClaudeInput) error {
	for _, action := range hook.Actions {
		switch action.Type {
		case "command":
			cmd := replaceVariables(action.Command, input)
			if err := runCommand(cmd); err != nil {
				return err
			}
		case "output":
			fmt.Println(action.Message)
		}
	}
	return nil
}

func replaceVariables(command string, input *ClaudeInput) string {
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