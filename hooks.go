package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// runHooks parses input and executes hooks for the specified event type.
func runHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PermissionRequest:
		return RunPermissionRequestHooks(config)
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
		_, err = executeStopHooks(config, input, rawJSON)
		return err
	case SubagentStop:
		input, rawJSON, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		_, err = executeSubagentStopHooks(config, input, rawJSON)
		return err
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
		// TODO: Task 9 will handle JSON serialization and output
		_, err = executeSessionStartHooks(config, input, rawJSON)
		return err
	case UserPromptSubmit:
		input, rawJSON, err := parseInput[*UserPromptSubmitInput](eventType)
		if err != nil {
			return err
		}
		// TODO: Task 9 will handle JSON serialization and output
		_, err = executeUserPromptSubmitHooks(config, input, rawJSON)
		return err
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

// RunSessionStartHooks executes SessionStart hooks and returns JSON output.
// This is a special exported function for SessionStart event handling in main.go.
func RunSessionStartHooks(config *Config) (*SessionStartOutput, error) {
	input, rawJSON, err := parseInput[*SessionStartInput](SessionStart)
	if err != nil {
		return nil, err
	}
	return executeSessionStartHooks(config, input, rawJSON)
}

// RunUserPromptSubmitHooks is a wrapper function that parses UserPromptSubmit input and executes hooks.
// This function is called from main.go to handle UserPromptSubmit events with JSON output.
func RunUserPromptSubmitHooks(config *Config) (*UserPromptSubmitOutput, error) {
	input, rawJSON, err := parseInput[*UserPromptSubmitInput](UserPromptSubmit)
	if err != nil {
		return nil, err
	}
	return executeUserPromptSubmitHooks(config, input, rawJSON)
}

// RunPreToolUseHooks parses input from stdin and executes PreToolUse hooks.
// Returns PreToolUseOutput for JSON serialization.
func RunPreToolUseHooks(config *Config) (*PreToolUseOutput, error) {
	input, rawJSON, err := parseInput[*PreToolUseInput](PreToolUse)
	if err != nil {
		return nil, err
	}
	return executePreToolUseHooksJSON(config, input, rawJSON)
}

// dryRunHooks parses input and performs a dry-run of hooks for the specified event type.
func dryRunHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, rawJSON, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreToolUseHooks(config, input, rawJSON)
	case PermissionRequest:
		input, rawJSON, err := parseInput[*PermissionRequestInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPermissionRequestHooks(config, input, rawJSON)
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

// dryRunPreToolUseHooks performs a dry-run of PreToolUse hooks, showing what would be executed without actually running.
func dryRunPreToolUseHooks(config *Config, input *PreToolUseInput, rawJSON interface{}) error {
	fmt.Println("=== PreToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PreToolUse {
		shouldExecute, err := shouldExecutePreToolUseHook(hook, input)
		if err != nil {
			if errors.Is(err, ErrProcessSubstitutionDetected) {
				fmt.Printf("[Hook %d] Process substitution detected - would deny\n", i+1)
				continue
			}
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
					if action.UseStdin {
						fmt.Printf("  UseStdin: true\n")
					}
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

// dryRunPostToolUseHooks performs a dry-run of PostToolUse hooks, showing what would be executed without actually running.
func dryRunPostToolUseHooks(config *Config, input *PostToolUseInput, rawJSON interface{}) error {
	fmt.Println("=== PostToolUse Hooks (Dry Run) ===")
	executed := false
	for i, hook := range config.PostToolUse {
		shouldExecute, err := shouldExecutePostToolUseHook(hook, input)
		if err != nil {
			if errors.Is(err, ErrProcessSubstitutionDetected) {
				fmt.Printf("[Hook %d] Process substitution detected - would warn\n", i+1)
				continue
			}
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
					if action.UseStdin {
						fmt.Printf("  UseStdin: true\n")
					}
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

// dryRunNotificationHooks performs a dry-run of Notification hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// dryRunStopHooks performs a dry-run of Stop hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// dryRunSubagentStopHooks performs a dry-run of SubagentStop hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// dryRunPreCompactHooks performs a dry-run of PreCompact hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// dryRunSessionStartHooks performs a dry-run of SessionStart hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// dryRunUserPromptSubmitHooks performs a dry-run of UserPromptSubmit hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// executePostToolUseHooks executes all matching PostToolUse hooks based on matcher and condition checks.
// Collects all condition check errors and returns them aggregated.
// executeNotificationHooks executes all matching Notification hooks based on condition checks.
func executeNotificationHooks(config *Config, input *NotificationInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

	for i, hook := range config.Notification {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkNotificationCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[Notification][%d]: %w", i, err))
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
			if err := executor.ExecuteNotificationAction(action, input, rawJSON); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("Notification hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("Notification hook %d failed: %w", i, err)
				if len(conditionErrors) > 0 {
					return errors.Join(append(conditionErrors, actionErr)...)
				}
				return actionErr
			}
		}
	}

	if len(conditionErrors) > 0 {
		return errors.Join(conditionErrors...)
	}
	return nil
}

// executeStopHooks executes all matching Stop hooks and returns JSON output.
// Stop uses top-level decision pattern (no hookSpecificOutput).
// Returns (*StopOutput, error) where output is always non-nil.
func executeStopHooks(config *Config, input *StopInput, rawJSON interface{}) (*StopOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput: Continue always true, Decision "" (allow stop)
	finalOutput := &StopOutput{
		Continue: true,
		Decision: "",
	}

	var systemMessageBuilder strings.Builder

	for i, hook := range config.Stop {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkStopCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[Stop][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecuteStopAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("stop hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Decision: 後勝ち。decision変更時はReasonリセット
			prevDecision := finalOutput.Decision
			finalOutput.Decision = actionOutput.Decision

			// Reason: decision変更時はリセット、同一decision内では改行連結
			if actionOutput.Decision != prevDecision {
				finalOutput.Reason = actionOutput.Reason
			} else if actionOutput.Reason != "" {
				if finalOutput.Reason != "" {
					finalOutput.Reason += "\n" + actionOutput.Reason
				} else {
					finalOutput.Reason = actionOutput.Reason
				}
			}

			// SystemMessage: 改行連結
			if actionOutput.SystemMessage != "" {
				if systemMessageBuilder.Len() > 0 {
					systemMessageBuilder.WriteString("\n")
				}
				systemMessageBuilder.WriteString(actionOutput.SystemMessage)
			}

			// StopReason: 最後の非空値が勝ち
			if actionOutput.StopReason != "" {
				finalOutput.StopReason = actionOutput.StopReason
			}

			// SuppressOutput: 最後の値が勝ち
			finalOutput.SuppressOutput = actionOutput.SuppressOutput

			// Early return on decision: "block"
			if actionOutput.Decision == "block" {
				break
			}
		}

		// Early return if decision is "block" (across hooks)
		if finalOutput.Decision == "block" {
			break
		}
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// アクションエラーがある場合はfail-safe: decision "block"
	if len(actionErrors) > 0 {
		finalOutput.Decision = "block"

		errorMsg := errors.Join(actionErrors...).Error()
		if finalOutput.Reason == "" {
			finalOutput.Reason = errorMsg
		}
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		allErrors = append(allErrors, actionErrors...)
	}

	if len(allErrors) > 0 {
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// executePostToolUseHooksJSON executes all matching PostToolUse hooks and returns JSON output.
// Implements merging rules for multiple hook outputs.
func executePostToolUseHooksJSON(config *Config, input *PostToolUseInput, rawJSON interface{}) (*PostToolUseOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput: Continue always true, Decision "" (allow tool result)
	finalOutput := &PostToolUseOutput{
		Continue: true,
		Decision: "",
	}

	var systemMessageBuilder strings.Builder
	var additionalContextBuilder strings.Builder
	var hookEventName string

	for i, hook := range config.PostToolUse {
		// マッチャーチェック
		if !checkMatcher(hook.Matcher, input.ToolName) {
			continue
		}

		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkPostToolUseCondition(condition, input)
			if err != nil {
				// プロセス置換検出の場合は警告をstderrに出力してフック継続
				if errors.Is(err, ErrProcessSubstitutionDetected) {
					fmt.Fprintln(os.Stderr, "⚠️ プロセス置換 (<() または >()) が検出されました。")
					fmt.Fprintln(os.Stderr, "この構文はサポートされていません。一時ファイルを使用するなど、プロセス置換を使わない方法で実行してください。")
					shouldExecute = false
					break
				}
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[PostToolUse][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecutePostToolUseAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("PostToolUse hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Decision: 後勝ち。decision変更時はReasonリセット
			prevDecision := finalOutput.Decision
			finalOutput.Decision = actionOutput.Decision

			// Reason: decision変更時はリセット、同一decision内では改行連結
			if actionOutput.Decision != prevDecision {
				finalOutput.Reason = actionOutput.Reason
			} else if actionOutput.Reason != "" {
				if finalOutput.Reason != "" {
					finalOutput.Reason += "\n" + actionOutput.Reason
				} else {
					finalOutput.Reason = actionOutput.Reason
				}
			}

			// SystemMessage: 改行連結
			if actionOutput.SystemMessage != "" {
				if systemMessageBuilder.Len() > 0 {
					systemMessageBuilder.WriteString("\n")
				}
				systemMessageBuilder.WriteString(actionOutput.SystemMessage)
			}

			// HookEventName: 初回設定のみ（set once）
			if hookEventName == "" && actionOutput.HookEventName != "" {
				hookEventName = actionOutput.HookEventName
			}

			// AdditionalContext: 改行連結
			if actionOutput.AdditionalContext != "" {
				if additionalContextBuilder.Len() > 0 {
					additionalContextBuilder.WriteString("\n")
				}
				additionalContextBuilder.WriteString(actionOutput.AdditionalContext)
			}

			// StopReason: 最後の非空値が勝ち
			if actionOutput.StopReason != "" {
				finalOutput.StopReason = actionOutput.StopReason
			}

			// SuppressOutput: 最後の値が勝ち
			finalOutput.SuppressOutput = actionOutput.SuppressOutput

		}

	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// HookSpecificOutputの構築（hookEventNameまたはadditionalContextがある場合のみ）
	additionalContext := additionalContextBuilder.String()
	if hookEventName != "" || additionalContext != "" {
		finalOutput.HookSpecificOutput = &PostToolUseHookSpecificOutput{
			HookEventName:     hookEventName,
			AdditionalContext: additionalContext,
		}
	}

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// Fail-safe: conditionErrorsまたはactionErrorsのいずれかがあればdecision="block"
	// PermissionRequestパターン（L1720-1728）踏襲
	if len(conditionErrors) > 0 || len(actionErrors) > 0 {
		finalOutput.Decision = "block"

		var errorMsg string
		if len(conditionErrors) > 0 && len(actionErrors) > 0 {
			errorMsg = errors.Join(append(conditionErrors, actionErrors...)...).Error()
		} else if len(conditionErrors) > 0 {
			errorMsg = errors.Join(conditionErrors...).Error()
		} else {
			errorMsg = errors.Join(actionErrors...).Error()
		}

		if finalOutput.Reason == "" {
			finalOutput.Reason = errorMsg
		}
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		allErrors = append(allErrors, actionErrors...)
	}

	if len(allErrors) > 0 {
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// RunStopHooks parses input from stdin and executes Stop hooks.
// Returns StopOutput for JSON serialization.
func RunStopHooks(config *Config) (*StopOutput, error) {
	input, rawJSON, err := parseInput[*StopInput](Stop)
	if err != nil {
		return nil, err
	}
	return executeStopHooks(config, input, rawJSON)
}

// RunSubagentStopHooks parses input from stdin and executes SubagentStop hooks.
// Returns SubagentStopOutput for JSON serialization.
func RunSubagentStopHooks(config *Config) (*SubagentStopOutput, error) {
	input, rawJSON, err := parseInput[*SubagentStopInput](SubagentStop)
	if err != nil {
		return nil, err
	}
	return executeSubagentStopHooks(config, input, rawJSON)
}

// RunPostToolUseHooks parses input from stdin and executes PostToolUse hooks.
// Returns PostToolUseOutput for JSON serialization.
func RunPostToolUseHooks(config *Config) (*PostToolUseOutput, error) {
	input, rawJSON, err := parseInput[*PostToolUseInput](PostToolUse)
	if err != nil {
		return nil, err
	}
	return executePostToolUseHooksJSON(config, input, rawJSON)
}

// executeSubagentStopHooks executes all matching SubagentStop hooks based on condition checks.
// Returns an error to block the subagent stop operation if any hook fails.
func executeSubagentStopHooks(config *Config, input *SubagentStopInput, rawJSON interface{}) (*SubagentStopOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput: Continue always true, Decision "" (allow subagent stop)
	finalOutput := &SubagentStopOutput{
		Continue: true,
		Decision: "",
	}

	var systemMessageBuilder strings.Builder

	for i, hook := range config.SubagentStop {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSubagentStopCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[SubagentStop][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecuteSubagentStopAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("subagent stop hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Decision: 後勝ち。decision変更時はReasonリセット
			prevDecision := finalOutput.Decision
			finalOutput.Decision = actionOutput.Decision

			// Reason: decision変更時はリセット、同一decision内では改行連結
			if actionOutput.Decision != prevDecision {
				finalOutput.Reason = actionOutput.Reason
			} else if actionOutput.Reason != "" {
				if finalOutput.Reason != "" {
					finalOutput.Reason += "\n" + actionOutput.Reason
				} else {
					finalOutput.Reason = actionOutput.Reason
				}
			}

			// SystemMessage: 改行連結
			if actionOutput.SystemMessage != "" {
				if systemMessageBuilder.Len() > 0 {
					systemMessageBuilder.WriteString("\n")
				}
				systemMessageBuilder.WriteString(actionOutput.SystemMessage)
			}

			// StopReason: 最後の非空値が勝ち
			if actionOutput.StopReason != "" {
				finalOutput.StopReason = actionOutput.StopReason
			}

			// SuppressOutput: 最後の値が勝ち
			finalOutput.SuppressOutput = actionOutput.SuppressOutput

			// Early return on decision: "block"
			if actionOutput.Decision == "block" {
				break
			}
		}

		// Early return if decision is "block" (across hooks)
		if finalOutput.Decision == "block" {
			break
		}
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// アクションエラーがある場合はfail-safe: decision "block"
	if len(actionErrors) > 0 {
		finalOutput.Decision = "block"

		errorMsg := errors.Join(actionErrors...).Error()
		if finalOutput.Reason == "" {
			finalOutput.Reason = errorMsg
		}
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		allErrors = append(allErrors, actionErrors...)
	}

	if len(allErrors) > 0 {
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// executePreCompactHooks executes all matching PreCompact hooks based on condition checks.
func executePreCompactHooks(config *Config, input *PreCompactInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

	for i, hook := range config.PreCompact {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkPreCompactCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[PreCompact][%d]: %w", i, err))
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
			if err := executor.ExecutePreCompactAction(action, input, rawJSON); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("PreCompact hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("PreCompact hook %d failed: %w", i, err)
				if len(conditionErrors) > 0 {
					return errors.Join(append(conditionErrors, actionErr)...)
				}
				return actionErr
			}
		}
	}

	if len(conditionErrors) > 0 {
		return errors.Join(conditionErrors...)
	}
	return nil
}

// executeSessionStartHooks executes all matching SessionStart hooks based on matcher and condition checks.
// Returns SessionStartOutput for JSON serialization.
func executeSessionStartHooks(config *Config, input *SessionStartInput, rawJSON interface{}) (*SessionStartOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true
	finalOutput := &SessionStartOutput{
		Continue: true,
	}

	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""

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
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[SessionStart][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecuteSessionStartAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("SessionStart hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Update finalOutput fields following merge rules

			// Continue: overwrite
			finalOutput.Continue = actionOutput.Continue

			// HookEventName: set once and preserve
			if hookEventName == "" && actionOutput.HookEventName != "" {
				hookEventName = actionOutput.HookEventName
			}

			// AdditionalContext: concatenate with "\n"
			if actionOutput.AdditionalContext != "" {
				if additionalContextBuilder.Len() > 0 {
					additionalContextBuilder.WriteString("\n")
				}
				additionalContextBuilder.WriteString(actionOutput.AdditionalContext)
			}

			// SystemMessage: concatenate with "\n"
			if actionOutput.SystemMessage != "" {
				if systemMessageBuilder.Len() > 0 {
					systemMessageBuilder.WriteString("\n")
				}
				systemMessageBuilder.WriteString(actionOutput.SystemMessage)
			}

			// Phase 1: Do NOT update StopReason or SuppressOutput (remain zero values)

			// Early return check AFTER collecting this action's data
			if !actionOutput.Continue {
				break
			}
		}

		// Early return if continue is false
		if !finalOutput.Continue {
			break
		}
	}

	// Build final output
	// Always set hookEventName to "SessionStart" (requirement 4.1)
	if hookEventName == "" {
		hookEventName = "SessionStart"
	}
	finalOutput.HookSpecificOutput = &SessionStartHookSpecificOutput{
		HookEventName:     hookEventName,
		AdditionalContext: additionalContextBuilder.String(),
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)
	allErrors = append(allErrors, actionErrors...)

	if len(allErrors) > 0 {
		// Safe side default: エラー時は必ず continue: false を設定
		finalOutput.Continue = false

		// エラー内容をSystemMessageに追加（グレースフルデグラデーション）
		// Codex指摘: JSON出力にエラー情報を含めることで、ユーザーに原因を伝える
		errorMsg := errors.Join(allErrors...).Error()
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// executeUserPromptSubmitHooks executes all matching UserPromptSubmit hooks and returns JSON output.
// This implements Phase 2 JSON output functionality for UserPromptSubmit hooks.
func executeUserPromptSubmitHooks(config *Config, input *UserPromptSubmitInput, rawJSON interface{}) (*UserPromptSubmitOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true, Decision: "" (omit to allow)
	finalOutput := &UserPromptSubmitOutput{
		Continue: true,
		Decision: "", // Empty string will be omitted from JSON (omitempty), allowing prompt
	}

	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""

	for i, hook := range config.UserPromptSubmit {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkUserPromptSubmitCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[UserPromptSubmit][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecuteUserPromptSubmitAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("UserPromptSubmit hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Update finalOutput fields following merge rules

			// Continue: always true (do not overwrite from actionOutput)
			// finalOutput.Continue remains true

			// Decision: overwrite (last one wins)
			finalOutput.Decision = actionOutput.Decision

			// HookEventName: set once and preserve
			if hookEventName == "" && actionOutput.HookEventName != "" {
				hookEventName = actionOutput.HookEventName
			}

			// AdditionalContext: concatenate with "\n"
			if actionOutput.AdditionalContext != "" {
				if additionalContextBuilder.Len() > 0 {
					additionalContextBuilder.WriteString("\n")
				}
				additionalContextBuilder.WriteString(actionOutput.AdditionalContext)
			}

			// SystemMessage: concatenate with "\n"
			if actionOutput.SystemMessage != "" {
				if systemMessageBuilder.Len() > 0 {
					systemMessageBuilder.WriteString("\n")
				}
				systemMessageBuilder.WriteString(actionOutput.SystemMessage)
			}

			// Phase 2: Do NOT update StopReason or SuppressOutput (remain zero values)

			// Early return check AFTER collecting this action's data
			if actionOutput.Decision == "block" {
				break
			}
		}

		// Early return if decision is "block"
		if finalOutput.Decision == "block" {
			break
		}
	}

	// Build final output
	// Always set hookEventName to "UserPromptSubmit"
	if hookEventName == "" {
		hookEventName = "UserPromptSubmit"
	}
	finalOutput.HookSpecificOutput = &UserPromptSubmitHookSpecificOutput{
		HookEventName:     hookEventName,
		AdditionalContext: additionalContextBuilder.String(),
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// アクションエラーがある場合のみdecision: "block"を設定
	// 仕様: 条件チェックエラーはフックスキップ、プロンプト送信は可能（decisionフィールド省略）
	if len(actionErrors) > 0 {
		finalOutput.Decision = "block"

		// アクションエラーのみSystemMessageに追加
		errorMsg := errors.Join(actionErrors...).Error()
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		allErrors = append(allErrors, actionErrors...)
	}

	// エラーがあれば返す（条件エラーのみでもエラーは返す）
	if len(allErrors) > 0 {
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// executePreToolUseHooksJSON executes all matching PreToolUse hooks and returns JSON output.
// This function implements Phase 3 JSON output functionality for PreToolUse hooks.
func executePreToolUseHooksJSON(config *Config, input *PreToolUseInput, rawJSON interface{}) (*PreToolUseOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true (always)
	finalOutput := &PreToolUseOutput{
		Continue: true,
	}

	var reasonBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""
	permissionDecision := "" // Empty = delegate to Claude Code's permission system
	var updatedInput map[string]interface{}
	stopReason := ""
	suppressOutput := false

	for i, hook := range config.PreToolUse {
		// Matcher and condition checks
		shouldExecute, err := shouldExecutePreToolUseHook(hook, input)
		if err != nil {
			// プロセス置換検出の場合はdenyとして処理し、以降のフックを処理しない
			if errors.Is(err, ErrProcessSubstitutionDetected) {
				permissionDecision = "deny"
				if reasonBuilder.Len() > 0 {
					reasonBuilder.WriteString("\n")
				}
				reasonBuilder.WriteString("⚠️ プロセス置換 (<() または >()) が検出されました。\nこの構文はサポートされていません。一時ファイルを使用するなど、プロセス置換を使わない方法で実行してください。")
				if hookEventName == "" {
					hookEventName = "PreToolUse"
				}
				break // denyを確定させるため、以降のフックを処理しない
			}
			conditionErrors = append(conditionErrors,
				fmt.Errorf("hook[PreToolUse][%d]: %w", i, err))
			continue // Skip this hook but continue checking others
		}

		if !shouldExecute {
			continue
		}

		// Execute hook actions
		actionOutput, err := executePreToolUseHook(executor, hook, input, rawJSON)
		if err != nil {
			actionErrors = append(actionErrors, fmt.Errorf("PreToolUse hook %d action failed: %w", i, err))
			continue
		}

		if actionOutput == nil {
			continue
		}

		// Update finalOutput fields following merge rules

		// Continue: always true (do not overwrite from actionOutput)
		// finalOutput.Continue remains true

		// PermissionDecision: last non-empty value wins
		// Empty permissionDecision means "no opinion" and should not overwrite previous values
		// If permissionDecision changes, reset permissionDecisionReason to avoid contradictions
		if actionOutput.PermissionDecision != "" {
			previousDecision := permissionDecision
			permissionDecision = actionOutput.PermissionDecision
			if previousDecision != permissionDecision {
				reasonBuilder.Reset()
			}
		}

		// HookEventName: set once and preserve
		if hookEventName == "" && actionOutput.HookEventName != "" {
			hookEventName = actionOutput.HookEventName
		}

		// PermissionDecisionReason: concatenate with "\n" if decision unchanged, otherwise replace
		if actionOutput.PermissionDecisionReason != "" {
			if reasonBuilder.Len() > 0 {
				reasonBuilder.WriteString("\n")
			}
			reasonBuilder.WriteString(actionOutput.PermissionDecisionReason)
		}

		// SystemMessage: concatenate with "\n"
		if actionOutput.SystemMessage != "" {
			if systemMessageBuilder.Len() > 0 {
				systemMessageBuilder.WriteString("\n")
			}
			systemMessageBuilder.WriteString(actionOutput.SystemMessage)
		}

		// UpdatedInput: last non-nil value wins
		if actionOutput.UpdatedInput != nil {
			updatedInput = actionOutput.UpdatedInput
		}

		// StopReason: last non-empty value wins
		if actionOutput.StopReason != "" {
			stopReason = actionOutput.StopReason
		}

		// SuppressOutput: last value wins
		suppressOutput = actionOutput.SuppressOutput

		// Early return check AFTER collecting this action's data
		if actionOutput.PermissionDecision == "deny" {
			break
		}
	}

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)
	allErrors = append(allErrors, actionErrors...)

	// Build final output
	// Set HookSpecificOutput only when permissionDecision is set or errors occurred
	if permissionDecision != "" || len(allErrors) > 0 {
		// Always set hookEventName to "PreToolUse"
		if hookEventName == "" {
			hookEventName = "PreToolUse"
		}

		// For errors, override permissionDecision to "deny" (fail-safe)
		if len(allErrors) > 0 {
			permissionDecision = "deny"
			// Clear previous permission decision reason to avoid inconsistency
			reasonBuilder.Reset()
		}

		finalOutput.HookSpecificOutput = &PreToolUseHookSpecificOutput{
			HookEventName:            hookEventName,
			PermissionDecision:       permissionDecision,
			PermissionDecisionReason: reasonBuilder.String(),
			UpdatedInput:             updatedInput,
		}
	}
	// Otherwise, leave HookSpecificOutput as nil to delegate to Claude Code's permission system

	finalOutput.SystemMessage = systemMessageBuilder.String()
	finalOutput.StopReason = stopReason
	finalOutput.SuppressOutput = suppressOutput

	if len(allErrors) > 0 {
		// Requirement 6.4: On error, include error in systemMessage

		// Build error message
		var errorMessages []string
		for _, err := range allErrors {
			errorMessages = append(errorMessages, err.Error())
		}
		errorMsg := strings.Join(errorMessages, "\n")

		// Append to systemMessage (preserve existing messages if any)
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}

		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// shouldExecutePreToolUseHook checks if a PreToolUse hook should be executed based on matcher and conditions.
func shouldExecutePreToolUseHook(hook PreToolUseHook, input *PreToolUseInput) (bool, error) {
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false, nil
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		matched, err := checkPreToolUseCondition(condition, input)
		if err != nil {
			// プロセス置換検出の場合は条件マッチとして扱う
			if errors.Is(err, ErrProcessSubstitutionDetected) {
				return true, err
			}
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

// shouldExecutePostToolUseHook checks if a PostToolUse hook should be executed based on matcher and conditions.
func shouldExecutePostToolUseHook(hook PostToolUseHook, input *PostToolUseInput) (bool, error) {
	// マッチャーチェック
	if !checkMatcher(hook.Matcher, input.ToolName) {
		return false, nil
	}

	// 条件チェック
	for _, condition := range hook.Conditions {
		matched, err := checkPostToolUseCondition(condition, input)
		if err != nil {
			// プロセス置換検出の場合は条件マッチとして扱う
			if errors.Is(err, ErrProcessSubstitutionDetected) {
				return true, err
			}
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

// executePreToolUseHook executes all actions for a single PreToolUse hook and returns JSON output.
// This function implements Phase 3 JSON output functionality for PreToolUse hooks.
func executePreToolUseHook(executor *ActionExecutor, hook PreToolUseHook, input *PreToolUseInput, rawJSON interface{}) (*ActionOutput, error) {
	// Initialize output with Continue: true (always true for PreToolUse)
	// permissionDecision starts empty and will be set by actions or remain empty to delegate
	output := &ActionOutput{
		Continue:           true,
		PermissionDecision: "",
		HookEventName:      "PreToolUse",
	}

	var reasonBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	var updatedInput map[string]interface{}

	for _, action := range hook.Actions {
		actionOutput, err := executor.ExecutePreToolUseAction(action, input, rawJSON)
		if err != nil {
			return nil, err
		}

		if actionOutput == nil {
			continue
		}

		// Update output fields following merge rules

		// PermissionDecision: last non-empty value wins
		// Empty permissionDecision means "no opinion" and should not overwrite previous values
		// If permissionDecision changes, reset permissionDecisionReason to avoid contradictions
		if actionOutput.PermissionDecision != "" {
			previousDecision := output.PermissionDecision
			output.PermissionDecision = actionOutput.PermissionDecision
			if previousDecision != output.PermissionDecision {
				reasonBuilder.Reset()
			}
		}

		// PermissionDecisionReason: concatenate with "\n" if decision unchanged, otherwise replace
		if actionOutput.PermissionDecisionReason != "" {
			if reasonBuilder.Len() > 0 {
				reasonBuilder.WriteString("\n")
			}
			reasonBuilder.WriteString(actionOutput.PermissionDecisionReason)
		}

		// SystemMessage: concatenate with "\n"
		if actionOutput.SystemMessage != "" {
			if systemMessageBuilder.Len() > 0 {
				systemMessageBuilder.WriteString("\n")
			}
			systemMessageBuilder.WriteString(actionOutput.SystemMessage)
		}

		// UpdatedInput: last value wins (only non-nil)
		if actionOutput.UpdatedInput != nil {
			updatedInput = actionOutput.UpdatedInput
		}

		// StopReason: last non-empty value wins
		if actionOutput.StopReason != "" {
			output.StopReason = actionOutput.StopReason
		}

		// SuppressOutput: last value wins
		output.SuppressOutput = actionOutput.SuppressOutput

		// Early return check for permissionDecision: deny
		if actionOutput.PermissionDecision == "deny" {
			break
		}
	}

	// If no actions produced any output, return nil to delegate completely
	// This prevents empty ActionOutput from overwriting previous hooks' decisions
	if output.PermissionDecision == "" &&
		reasonBuilder.Len() == 0 &&
		systemMessageBuilder.Len() == 0 &&
		updatedInput == nil &&
		output.StopReason == "" &&
		!output.SuppressOutput {
		return nil, nil
	}

	// Build final output
	output.PermissionDecisionReason = reasonBuilder.String()
	output.SystemMessage = systemMessageBuilder.String()
	output.UpdatedInput = updatedInput

	return output, nil
}

// executePostToolUseHook executes all actions for a single PostToolUse hook.
// Note: This is a temporary implementation for compatibility.
// Will be replaced with executePostToolUseHooksJSON in Step 4.

// executeSessionEndHooks executes all matching SessionEnd hooks based on condition checks.
// Returns errors to inform users of failures, though session end cannot be blocked.
func executeSessionEndHooks(config *Config, input *SessionEndInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

	for i, hook := range config.SessionEnd {
		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSessionEndCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[SessionEnd][%d]: %w", i, err))
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
			_, err := executor.ExecuteSessionEndAction(action, input, rawJSON)
			if err != nil {
				// SessionEndフックはセッション終了をブロックできないが、
				// エラーをユーザーに通知するため返す
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("SessionEnd hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("SessionEnd hook %d failed: %w", i, err)
				if len(conditionErrors) > 0 {
					return errors.Join(append(conditionErrors, actionErr)...)
				}
				return actionErr
			}
		}
	}

	if len(conditionErrors) > 0 {
		return errors.Join(conditionErrors...)
	}
	return nil
}

// dryRunSessionEndHooks performs a dry-run of SessionEnd hooks, showing what would be executed without actually running.
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
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
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

// executePermissionRequestHooksJSON executes PermissionRequest hooks and returns JSON output
func executePermissionRequestHooksJSON(config *Config, input *PermissionRequestInput, rawJSON interface{}) (*PermissionRequestOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true (always)
	finalOutput := &PermissionRequestOutput{
		Continue: true,
	}

	var messageBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := "PermissionRequest"
	behavior := "allow" // Default: allow when no hooks match
	var updatedInput map[string]interface{}
	interrupt := false
	stopReason := ""
	suppressOutput := false
	matchedAny := false // Track if any hook matched

	for i, hook := range config.PermissionRequest {
		// Matcher and condition checks
		shouldExecute, err := shouldExecutePermissionRequestHook(hook, input)
		if err != nil {
			conditionErrors = append(conditionErrors,
				fmt.Errorf("hook[PermissionRequest][%d]: %w", i, err))
			continue // Skip this hook but continue checking others
		}

		if !shouldExecute {
			continue
		}

		matchedAny = true // Mark that at least one hook matched

		// Execute hook actions
		actionOutput, err := executePermissionRequestHook(executor, hook, input, rawJSON)
		if err != nil {
			actionErrors = append(actionErrors, fmt.Errorf("PermissionRequest hook %d action failed: %w", i, err))
			continue
		}

		if actionOutput == nil {
			continue
		}

		// Update finalOutput fields following merge rules

		// Continue: last value wins
		finalOutput.Continue = actionOutput.Continue

		// Behavior: last value wins
		previousBehavior := behavior
		behavior = actionOutput.Behavior

		// Message: concatenate with newline
		if actionOutput.Message != "" {
			if messageBuilder.Len() > 0 {
				messageBuilder.WriteString("\n")
			}
			messageBuilder.WriteString(actionOutput.Message)
		}

		// Interrupt: last value wins
		interrupt = actionOutput.Interrupt

		// UpdatedInput: last non-null value wins (top-level merge, not deep merge)
		if actionOutput.UpdatedInput != nil {
			updatedInput = actionOutput.UpdatedInput
		}

		// Clear incompatible fields when behavior changes (across multiple hooks)
		// This must happen AFTER merging fields from actionOutput
		if previousBehavior != behavior {
			switch behavior {
			case "allow":
				// allow時: decision内のmessage/interruptをクリア（公式仕様: allow時はdecision.message/interrupt不可）
				// Note: systemMessageはトップレベルのフィールドでdecisionとは独立なので残す
				messageBuilder.Reset()
				interrupt = false
			case "deny":
				// deny時: updatedInputをクリア (公式仕様: deny時はupdatedInput不可)
				updatedInput = nil
			}
		}

		// HookEventName: fixed as "PermissionRequest" (initialized above, no update needed)

		// SystemMessage: concatenate with newline
		if actionOutput.SystemMessage != "" {
			if systemMessageBuilder.Len() > 0 {
				systemMessageBuilder.WriteString("\n")
			}
			systemMessageBuilder.WriteString(actionOutput.SystemMessage)
		}

		// StopReason: last value wins
		if actionOutput.StopReason != "" {
			stopReason = actionOutput.StopReason
		}

		// SuppressOutput: last value wins
		suppressOutput = actionOutput.SuppressOutput

		// Early return on continue=false
		if !actionOutput.Continue {
			break
		}
	}

	// Build hookSpecificOutput
	// Ensure message is set when behavior is deny (required by spec)
	message := messageBuilder.String()
	if behavior == "deny" && message == "" {
		if matchedAny {
			message = "Permission denied"
		} else {
			message = "Permission denied (no hooks matched)"
		}
	}

	finalOutput.HookSpecificOutput = &PermissionRequestHookSpecificOutput{
		HookEventName: hookEventName,
		Decision: &PermissionRequestDecision{
			Behavior:     behavior,
			UpdatedInput: updatedInput,
			Message:      message,
			Interrupt:    interrupt,
		},
	}

	// Set top-level fields
	finalOutput.SystemMessage = systemMessageBuilder.String()
	finalOutput.StopReason = stopReason
	finalOutput.SuppressOutput = suppressOutput

	// Fail-safe: force deny on errors
	if len(conditionErrors) > 0 || len(actionErrors) > 0 {
		behavior = "deny"
		updatedInput = nil // deny時はupdatedInputをクリア
		interrupt = false  // エラー時はinterruptも明示的にfalseにリセット
		// deny時はmessageが必須なので、エラー概要を設定
		var errMsg string
		if len(conditionErrors) > 0 {
			errMsg = "Hook execution failed: condition errors occurred"
		} else {
			errMsg = "Hook execution failed: action errors occurred"
		}
		finalOutput.HookSpecificOutput.Decision.Behavior = behavior
		finalOutput.HookSpecificOutput.Decision.UpdatedInput = updatedInput
		finalOutput.HookSpecificOutput.Decision.Message = errMsg
		finalOutput.HookSpecificOutput.Decision.Interrupt = interrupt
		finalOutput.SystemMessage = errMsg
		// Force continue=true on fail-safe to prevent blocking subsequent hooks
		finalOutput.Continue = true
	}

	// Validate final output against JSON schema
	finalOutputJSON, err := json.Marshal(finalOutput)
	if err == nil {
		if validationErr := validatePermissionRequestOutput(finalOutputJSON); validationErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Final output validation failed: %v\n", validationErr)
		}
	}

	// Validation errors
	if len(conditionErrors) > 0 {
		return finalOutput, fmt.Errorf("condition evaluation errors: %v", conditionErrors)
	}
	if len(actionErrors) > 0 {
		return finalOutput, fmt.Errorf("action execution errors: %v", actionErrors)
	}

	return finalOutput, nil
}

// executePermissionRequestHook executes all actions in a single hook and merges their outputs
func executePermissionRequestHook(executor *ActionExecutor, hook PermissionRequestHook, input *PermissionRequestInput, rawJSON interface{}) (*ActionOutput, error) {
	var mergedOutput *ActionOutput

	for _, action := range hook.Actions {
		actionOutput, err := executor.ExecutePermissionRequestAction(action, input, rawJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to execute action: %w", err)
		}

		if mergedOutput == nil {
			mergedOutput = actionOutput
			// Early return on continue=false (first action)
			if !actionOutput.Continue {
				break
			}
			continue
		}

		// Merge actionOutput into mergedOutput following PermissionRequest merge rules

		// Continue: last value wins
		mergedOutput.Continue = actionOutput.Continue

		// Behavior: last value wins
		previousBehavior := mergedOutput.Behavior
		mergedOutput.Behavior = actionOutput.Behavior

		// Message: concatenate with newline
		if actionOutput.Message != "" {
			if mergedOutput.Message != "" {
				mergedOutput.Message += "\n" + actionOutput.Message
			} else {
				mergedOutput.Message = actionOutput.Message
			}
		}

		// Interrupt: last value wins
		mergedOutput.Interrupt = actionOutput.Interrupt

		// UpdatedInput: last non-null value wins
		if actionOutput.UpdatedInput != nil {
			mergedOutput.UpdatedInput = actionOutput.UpdatedInput
		}

		// Clear fields incompatible with behavior change (公式仕様準拠)
		// This must happen AFTER setting all fields from actionOutput
		if previousBehavior != mergedOutput.Behavior {
			switch mergedOutput.Behavior {
			case "deny":
				// deny時: updatedInputをクリア (公式仕様: deny時はupdatedInput不可)
				mergedOutput.UpdatedInput = nil
			case "allow":
				// allow時: decision内のmessage/interruptをクリア（公式仕様: allow時はdecision.message/interrupt不可）
				// Note: systemMessageはトップレベルのフィールドでdecisionとは独立なので残す
				mergedOutput.Message = ""
				mergedOutput.Interrupt = false
			}
		}

		// HookEventName: set once by first action
		if mergedOutput.HookEventName == "" && actionOutput.HookEventName != "" {
			mergedOutput.HookEventName = actionOutput.HookEventName
		}

		// SystemMessage: concatenate with newline
		if actionOutput.SystemMessage != "" {
			if mergedOutput.SystemMessage != "" {
				mergedOutput.SystemMessage += "\n" + actionOutput.SystemMessage
			} else {
				mergedOutput.SystemMessage = actionOutput.SystemMessage
			}
		}

		// StopReason: last value wins
		if actionOutput.StopReason != "" {
			mergedOutput.StopReason = actionOutput.StopReason
		}

		// SuppressOutput: last value wins
		mergedOutput.SuppressOutput = actionOutput.SuppressOutput

		// Early return on continue=false
		if !actionOutput.Continue {
			break
		}
	}

	return mergedOutput, nil
}

// shouldExecutePermissionRequestHook checks if a hook should be executed based on matcher and conditions
func shouldExecutePermissionRequestHook(hook PermissionRequestHook, input *PermissionRequestInput) (bool, error) {
	// Check matcher (tool name partial match)
	if hook.Matcher != "" {
		matchers := strings.Split(hook.Matcher, "|")
		matched := false
		for _, m := range matchers {
			if strings.Contains(input.ToolName, strings.TrimSpace(m)) {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	// Check conditions
	for _, condition := range hook.Conditions {
		matched, err := checkPermissionRequestCondition(condition, input)
		if err != nil {
			return false, fmt.Errorf("condition check failed: %w", err)
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}

// RunPermissionRequestHooks runs PermissionRequest hooks and outputs JSON
func RunPermissionRequestHooks(config *Config) error {
	// Read JSON input from stdin
	input, rawJSON, err := parseInput[*PermissionRequestInput](PermissionRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse input: %v\n", err)
		// Fail-safe: output deny decision
		errMsg := fmt.Sprintf("Failed to parse input: %v", err)
		output := &PermissionRequestOutput{
			Continue: true,
			HookSpecificOutput: &PermissionRequestHookSpecificOutput{
				HookEventName: "PermissionRequest",
				Decision: &PermissionRequestDecision{
					Behavior: "deny",
					Message:  errMsg,
				},
			},
			SystemMessage: errMsg,
		}
		jsonOutput, _ := json.Marshal(output)
		fmt.Println(string(jsonOutput))
		return nil // Always exit 0
	}

	// Execute hooks
	output, err := executePermissionRequestHooksJSON(config, input, rawJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: hook execution errors: %v\n", err)
		// Continue with output (fail-safe already set in executePermissionRequestHooksJSON)
	}

	// Output JSON
	jsonOutput, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal output: %v\n", err)
		// Fail-safe: output deny decision
		errMsg := fmt.Sprintf("Failed to marshal output: %v", err)
		fallbackOutput := &PermissionRequestOutput{
			Continue: true,
			HookSpecificOutput: &PermissionRequestHookSpecificOutput{
				HookEventName: "PermissionRequest",
				Decision: &PermissionRequestDecision{
					Behavior: "deny",
					Message:  errMsg,
				},
			},
			SystemMessage: errMsg,
		}
		jsonOutput, _ = json.Marshal(fallbackOutput)
	}

	fmt.Println(string(jsonOutput))
	return nil // Always exit 0
}

// dryRunPermissionRequestHooks prints what would be executed for PermissionRequest hooks
func dryRunPermissionRequestHooks(config *Config, input *PermissionRequestInput, rawJSON interface{}) error {
	fmt.Println("\n=== PermissionRequest Hooks ===")
	if len(config.PermissionRequest) == 0 {
		fmt.Println("No PermissionRequest hooks configured")
		return nil
	}

	executed := false
	for i, hook := range config.PermissionRequest {
		shouldExecute, err := shouldExecutePermissionRequestHook(hook, input)
		if err != nil {
			fmt.Printf("[Hook %d] Error checking conditions: %v\n", i+1, err)
			continue
		}
		if !shouldExecute {
			continue
		}

		executed = true
		fmt.Printf("[Hook %d] Tool: %s\n", i+1, input.ToolName)
		for _, action := range hook.Actions {
			switch action.Type {
			case "command":
				cmd := unifiedTemplateReplace(action.Command, rawJSON)
				fmt.Printf("  Command: %s\n", cmd)
				if action.UseStdin {
					fmt.Printf("  UseStdin: true\n")
				}
			case "output":
				msg := unifiedTemplateReplace(action.Message, rawJSON)
				fmt.Printf("  Message: %s\n", msg)
				if action.Behavior != nil {
					fmt.Printf("  Behavior: %s\n", *action.Behavior)
				}
				if action.Interrupt != nil && *action.Interrupt {
					fmt.Printf("  Interrupt: true\n")
				}
			}
		}
	}

	if !executed {
		fmt.Println("No matching PermissionRequest hooks found")
	}
	return nil
}
