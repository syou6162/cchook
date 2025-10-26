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
	case SessionStart:
		input, rawJSON, err := parseInput[*SessionStartInput](eventType)
		if err != nil {
			return err
		}
		return executeSessionStartHooks(config, input, rawJSON)
	case UserPromptSubmit:
		input, rawJSON, err := parseInput[*UserPromptSubmitInput](eventType)
		if err != nil {
			return err
		}
		return executeUserPromptSubmitHooks(config, input, rawJSON)
	case SessionEnd:
		input, rawJSON, err := parseInput[*SessionEndInput](eventType)
		if err != nil {
			return err
		}
		return executeSessionEndHooks(config, input, rawJSON)
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
	case SessionStart:
		input, rawJSON, err := parseInput[*SessionStartInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSessionStartHooks(config, input, rawJSON)
	case UserPromptSubmit:
		input, rawJSON, err := parseInput[*UserPromptSubmitInput](eventType)
		if err != nil {
			return err
		}
		return dryRunUserPromptSubmitHooks(config, input, rawJSON)
	case SessionEnd:
		input, rawJSON, err := parseInput[*SessionEndInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSessionEndHooks(config, input, rawJSON)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// イベント別のdry-run関数
func dryRunPreToolUseHooks(config *Config, input *PreToolUseInput, rawJSON interface{}) error {
	fmt.Println("=== PreToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PreToolUse {
		shouldExecute, err := shouldExecutePreToolUseHook(hook, input)
		if err != nil {
			fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
			continue
		}
		if shouldExecute {
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
		shouldExecute, err := shouldExecutePostToolUseHook(hook, input)
		if err != nil {
			fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
			continue
		}
		if shouldExecute {
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
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkNotificationCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

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
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkStopCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

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
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSubagentStopCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

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
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkPreCompactCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

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

func dryRunSessionStartHooks(config *Config, input *SessionStartInput, rawJSON interface{}) error {
	fmt.Println("=== SessionStart Hooks (Dry Run) ===")

	if len(config.SessionStart) == 0 {
		fmt.Println("No SessionStart hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.SessionStart {
		// マッチャーチェック
		if hook.Matcher != "" && hook.Matcher != input.Source {
			continue
		}

		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSessionStartCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		executed = true
		fmt.Printf("[Hook %d] Matcher: %s, Source: %s\n", i+1, hook.Matcher, input.Source)
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
		fmt.Println("No matching SessionStart hooks found")
	}
	return nil
}

func dryRunUserPromptSubmitHooks(config *Config, input *UserPromptSubmitInput, rawJSON interface{}) error {
	fmt.Println("=== UserPromptSubmit Hooks (Dry Run) ===")

	if len(config.UserPromptSubmit) == 0 {
		fmt.Println("No UserPromptSubmit hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.UserPromptSubmit {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkUserPromptSubmitCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		executed = true
		fmt.Printf("[Hook %d] Prompt: %s\n", i+1, input.Prompt)
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
		fmt.Println("No matching UserPromptSubmit hooks found")
	}
	return nil
}

// イベント別のフック実行関数
func executePreToolUseHooks(config *Config, input *PreToolUseInput, rawJSON interface{}) error {
	for i, hook := range config.PreToolUse {
		shouldExecute, err := shouldExecutePreToolUseHook(hook, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "PreToolUse hook %d condition check failed: %v\n", i, err)
			continue // 条件チェックエラーの場合はスキップして次のフックへ
		}
		if shouldExecute {
			if err := executePreToolUseHook(hook, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PreToolUse hook %d failed: %v\n", i, err)
				return err // ExitErrorの場合はすぐに返す
			}
		}
	}
	return nil
}

func executePostToolUseHooks(config *Config, input *PostToolUseInput, rawJSON interface{}) error {
	for i, hook := range config.PostToolUse {
		shouldExecute, err := shouldExecutePostToolUseHook(hook, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "PostToolUse hook %d condition check failed: %v\n", i, err)
			continue // 条件チェックエラーの場合はスキップして次のフックへ
		}
		if shouldExecute {
			if err := executePostToolUseHook(hook, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PostToolUse hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeNotificationHooks(config *Config, input *NotificationInput, rawJSON interface{}) error {
	for i, hook := range config.Notification {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkNotificationCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Notification hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

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
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkStopCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Stop hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executeStopAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "Stop hook %d failed: %v\n", i, err)
				// Stopフックはブロッキング可能なのでエラーを返す
				return err
			}
		}
	}
	return nil
}

func executeSubagentStopHooks(config *Config, input *SubagentStopInput, rawJSON interface{}) error {
	for i, hook := range config.SubagentStop {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSubagentStopCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "SubagentStop hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executeSubagentStopAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "SubagentStop hook %d failed: %v\n", i, err)
				// SubagentStopフックもブロッキング可能なのでエラーを返す
				return err
			}
		}
	}
	return nil
}

func executePreCompactHooks(config *Config, input *PreCompactInput, rawJSON interface{}) error {
	for i, hook := range config.PreCompact {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkPreCompactCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "PreCompact hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executePreCompactAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "PreCompact hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeSessionStartHooks(config *Config, input *SessionStartInput, rawJSON interface{}) error {
	for i, hook := range config.SessionStart {
		// マッチャーチェック (startup, resume, clear)
		if hook.Matcher != "" && hook.Matcher != input.Source {
			continue
		}

		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSessionStartCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "SessionStart hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executeSessionStartAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "SessionStart hook %d failed: %v\n", i, err)
			}
		}
	}
	return nil
}

func executeUserPromptSubmitHooks(config *Config, input *UserPromptSubmitInput, rawJSON interface{}) error {
	for i, hook := range config.UserPromptSubmit {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkUserPromptSubmitCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "UserPromptSubmit hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executeUserPromptSubmitAction(action, input, rawJSON); err != nil {
				// UserPromptSubmitはブロッキング可能なので、エラーを返す
				return err
			}
		}
	}
	return nil
}

func shouldExecutePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput) (bool, error) {
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false, nil
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		matched, err := checkPreToolUseCondition(condition, input)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

func shouldExecutePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput) (bool, error) {
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false, nil
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		matched, err := checkPostToolUseCondition(condition, input)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

func executePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput, rawJSON interface{}) error {
	for _, action := range hook.Actions {
		if err := executePreToolUseAction(action, input, rawJSON); err != nil {
			return err
		}
	}
	return nil
}

func executePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput, rawJSON interface{}) error {
	for _, action := range hook.Actions {
		if err := executePostToolUseAction(action, input, rawJSON); err != nil {
			return err
		}
	}
	return nil
}

func executeSessionEndHooks(config *Config, input *SessionEndInput, rawJSON interface{}) error {
	for i, hook := range config.SessionEnd {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSessionEndCondition(condition, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "SessionEnd hook %d condition check failed: %v\n", i, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		for _, action := range hook.Actions {
			if err := executeSessionEndAction(action, input, rawJSON); err != nil {
				fmt.Fprintf(os.Stderr, "SessionEnd hook %d failed: %v\n", i, err)
				// SessionEndフックはブロッキング不可能なのでエラーを返さない
				// （セッション終了時なので止める意味がない）
			}
		}
	}
	return nil
}

func dryRunSessionEndHooks(config *Config, input *SessionEndInput, rawJSON interface{}) error {
	fmt.Println("=== SessionEnd Hooks (Dry Run) ===")

	if len(config.SessionEnd) == 0 {
		fmt.Println("No SessionEnd hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.SessionEnd {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSessionEndCondition(condition, input)
			if err != nil {
				fmt.Printf("[Hook %d] Condition check error: %v\n", i+1, err)
				shouldExecute = false
				break
			}
			if !matched {
				shouldExecute = false
				break
			}
		}
		if !shouldExecute {
			continue
		}

		executed = true
		fmt.Printf("[Hook %d] Reason: %s\n", i+1, input.Reason)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
			case "output":
				msg := unifiedTemplateReplace(action.Message, rawJSON)
				fmt.Printf("  Message: %s\n", msg)
			}
		}
	}

	if !executed {
		fmt.Println("No matching SessionEnd hooks found")
	}
	return nil
}
