package main

import (
	"fmt"
	"os"
)

func runHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, rawJSON, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return executePreToolUseHooks(config, input, rawJSON)
	case PostToolUse:
		input, rawJSON, err := parseInput[*PostToolUseInput](eventType)
		if err != nil {
			return err
		}
		return executePostToolUseHooks(config, input, rawJSON)
	case Notification:
		input, rawJSON, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		return executeNotificationHooks(config, input, rawJSON)
	case Stop:
		input, rawJSON, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		return executeStopHooks(config, input, rawJSON)
	case SubagentStop:
		input, rawJSON, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		return executeSubagentStopHooks(config, input, rawJSON)
	case PreCompact:
		input, rawJSON, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		return executePreCompactHooks(config, input, rawJSON)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func dryRunHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, rawJSON, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreToolUseHooks(config, input, rawJSON)
	case PostToolUse:
		input, rawJSON, err := parseInput[*PostToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPostToolUseHooks(config, input, rawJSON)
	case Notification:
		input, rawJSON, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		return dryRunNotificationHooks(config, input, rawJSON)
	case Stop:
		input, rawJSON, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunStopHooks(config, input, rawJSON)
	case SubagentStop:
		input, rawJSON, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSubagentStopHooks(config, input, rawJSON)
	case PreCompact:
		input, rawJSON, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreCompactHooks(config, input, rawJSON)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// イベント別のdry-run関数
func dryRunPreToolUseHooks(config *Config, input *PreToolUseInput, rawJSON interface{}) error {
	fmt.Println("=== PreToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PreToolUse {
		if shouldExecutePreToolUseHook(hook, input) {
			executed = true
			fmt.Printf("[Hook %d] Would execute:\n", i+1)
			for _, action := range hook.Actions {
				switch action.Type {
				case "command":
					cmd := unifiedTemplateReplace(action.Command, rawJSON)
					fmt.Printf("  Command: %s\n", cmd)
				case "output":
					message := unifiedTemplateReplace(action.Message, rawJSON)
					fmt.Printf("  Output: %s\n", message)
				}
			}
		}
	}
	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunPostToolUseHooks(config *Config, input *PostToolUseInput, rawJSON interface{}) error {
	fmt.Println("=== PostToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PostToolUse {
		if shouldExecutePostToolUseHook(hook, input) {
			executed = true
			fmt.Printf("[Hook %d] Would execute:\n", i+1)
			for _, action := range hook.Actions {
				switch action.Type {
				case "command":
					cmd := unifiedTemplateReplace(action.Command, rawJSON)
					fmt.Printf("  Command: %s\n", cmd)
				case "output":
					message := unifiedTemplateReplace(action.Message, rawJSON)
					fmt.Printf("  Output: %s\n", message)
				}
			}
		}
	}
	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunNotificationHooks(config *Config, input *NotificationInput, rawJSON interface{}) error {
	fmt.Println("=== Notification Hooks (Dry Run) ===")

	if len(config.Notification) == 0 {
		fmt.Println("No Notification hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.Notification {
		executed = true
		fmt.Printf("[Hook %d] Would execute:\n", i+1)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
			case "output":
				fmt.Printf("  Message: %s\n", action.Message)
			}
		}
	}

	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunStopHooks(config *Config, input *StopInput, rawJSON interface{}) error {
	fmt.Println("=== Stop Hooks (Dry Run) ===")

	if len(config.Stop) == 0 {
		fmt.Println("No Stop hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.Stop {
		executed = true
		fmt.Printf("[Hook %d] Would execute:\n", i+1)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
			case "output":
				fmt.Printf("  Message: %s\n", action.Message)
			}
		}
	}

	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunSubagentStopHooks(config *Config, input *SubagentStopInput, rawJSON interface{}) error {
	fmt.Println("=== SubagentStop Hooks (Dry Run) ===")

	if len(config.SubagentStop) == 0 {
		fmt.Println("No SubagentStop hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.SubagentStop {
		executed = true
		fmt.Printf("[Hook %d] Would execute:\n", i+1)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
			case "output":
				fmt.Printf("  Message: %s\n", action.Message)
			}
		}
	}

	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

func dryRunPreCompactHooks(config *Config, input *PreCompactInput, rawJSON interface{}) error {
	fmt.Println("=== PreCompact Hooks (Dry Run) ===")

	if len(config.PreCompact) == 0 {
		fmt.Println("No PreCompact hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.PreCompact {
		executed = true
		fmt.Printf("[Hook %d] Would execute:\n", i+1)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
			case "output":
				fmt.Printf("  Message: %s\n", action.Message)
			}
		}
	}

	if !executed {
		fmt.Println("No hooks would be executed")
	}
	return nil
}

// イベント別のフック実行関数
func executePreToolUseHooks(config *Config, input *PreToolUseInput, rawJSON interface{}) error {
	for i, hook := range config.PreToolUse {
		if shouldExecutePreToolUseHook(hook, input) {
			if err := executePreToolUseHook(hook, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PreToolUse hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executePostToolUseHooks(config *Config, input *PostToolUseInput, rawJSON interface{}) error {
	for i, hook := range config.PostToolUse {
		if shouldExecutePostToolUseHook(hook, input) {
			if err := executePostToolUseHook(hook, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PostToolUse hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeNotificationHooks(config *Config, input *NotificationInput, rawJSON interface{}) error {
	for i, hook := range config.Notification {
		for _, action := range hook.Actions {
			if err := executeNotificationAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "Notification hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeStopHooks(config *Config, input *StopInput, rawJSON interface{}) error {
	for i, hook := range config.Stop {
		for _, action := range hook.Actions {
			if err := executeStopAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "Stop hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeSubagentStopHooks(config *Config, input *SubagentStopInput, rawJSON interface{}) error {
	for i, hook := range config.SubagentStop {
		for _, action := range hook.Actions {
			if err := executeSubagentStopAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "SubagentStop hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executePreCompactHooks(config *Config, input *PreCompactInput, rawJSON interface{}) error {
	for i, hook := range config.PreCompact {
		for _, action := range hook.Actions {
			if err := executePreCompactAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PreCompact hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
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

func executePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput, rawJSON interface{}) error {
	for _, action := range hook.Actions {
		switch action.Type {
		case "command":
			cmd := unifiedTemplateReplace(action.Command, rawJSON)
			if err := runCommand(cmd); err != nil {
				return err
			}
		case "structured_output":
			if err := executeStructuredOutput(action, PreToolUse); err != nil {
				return err
			}
		}
	}
	return nil
}

func executePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput, rawJSON interface{}) error {
	for _, action := range hook.Actions {
		switch action.Type {
		case "command":
			cmd := unifiedTemplateReplace(action.Command, rawJSON)
			if err := runCommand(cmd); err != nil {
				return err
			}
		case "structured_output":
			if err := executeStructuredOutput(action, PostToolUse); err != nil {
				return err
			}
		}
	}
	return nil
}
