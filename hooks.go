package main

import (
	"fmt"
	"os"
)

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

// 未実装イベント用のジェネリック関数（実行用）
func executeUnimplementedHooks() error {
	// 未実装イベントでは何もしない（将来の実装用）
	return nil
}

func executeNotificationHooks(config *Config, input *NotificationInput) error {
	return executeUnimplementedHooks()
}

func executeStopHooks(config *Config, input *StopInput) error {
	return executeUnimplementedHooks()
}

func executeSubagentStopHooks(config *Config, input *SubagentStopInput) error {
	return executeUnimplementedHooks()
}

func executePreCompactHooks(config *Config, input *PreCompactInput) error {
	return executeUnimplementedHooks()
}

func shouldExecutePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput) bool {
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false
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
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		if !checkPostToolUseCondition(condition, input) {
			return false
		}
	}

	return true
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