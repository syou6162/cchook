package main

import (
	"errors"
	"fmt"
	"strings"
)

// runHooks parses input and executes hooks for the specified event type.
func runHooks(config *Config, eventType HookEventType) error {
	switch eventType {
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
func executePostToolUseHooks(config *Config, input *PostToolUseInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

	for i, hook := range config.PostToolUse {
		shouldExecute, err := shouldExecutePostToolUseHook(hook, input)
		if err != nil {
			conditionErrors = append(conditionErrors,
				fmt.Errorf("hook[PostToolUse][%d]: %w", i, err))
			continue
		}
		if shouldExecute {
			if err := executePostToolUseHook(executor, hook, input, rawJSON); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("PostToolUse hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("PostToolUse hook %d failed: %w", i, err)
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

// executeStopHooks executes all matching Stop hooks based on condition checks.
// Returns an error to block the stop operation if any hook fails.
func executeStopHooks(config *Config, input *StopInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

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
			if err := executor.ExecuteStopAction(action, input, rawJSON); err != nil {
				// Stopフックはブロッキング可能なのでエラーを返す
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("Stop hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("Stop hook %d failed: %w", i, err)
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

// executeSubagentStopHooks executes all matching SubagentStop hooks based on condition checks.
// Returns an error to block the subagent stop operation if any hook fails.
func executeSubagentStopHooks(config *Config, input *SubagentStopInput, rawJSON interface{}) error {
	executor := NewActionExecutor(nil)
	var conditionErrors []error

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
			if err := executor.ExecuteSubagentStopAction(action, input, rawJSON); err != nil {
				// SubagentStopフックもブロッキング可能なのでエラーを返す
				if exitErr, ok := err.(*ExitError); ok {
					actionErr := &ExitError{
						Code:    exitErr.Code,
						Message: fmt.Sprintf("SubagentStop hook %d failed: %s", i, exitErr.Message),
						Stderr:  exitErr.Stderr,
					}
					if len(conditionErrors) > 0 {
						return errors.Join(append(conditionErrors, actionErr)...)
					}
					return actionErr
				}
				actionErr := fmt.Errorf("SubagentStop hook %d failed: %w", i, err)
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

		// PermissionDecision: last value wins
		// If permissionDecision changes, reset permissionDecisionReason to avoid contradictions
		previousDecision := output.PermissionDecision
		output.PermissionDecision = actionOutput.PermissionDecision
		if previousDecision != output.PermissionDecision {
			reasonBuilder.Reset()
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
func executePostToolUseHook(executor *ActionExecutor, hook PostToolUseHook, input *PostToolUseInput, rawJSON interface{}) error {
	for _, action := range hook.Actions {
		if err := executor.ExecutePostToolUseAction(action, input, rawJSON); err != nil {
			return err
		}
	}
	return nil
}

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
			if err := executor.ExecuteSessionEndAction(action, input, rawJSON); err != nil {
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
