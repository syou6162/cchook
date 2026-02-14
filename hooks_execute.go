package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// executePostToolUseHooks executes all matching PostToolUse hooks based on matcher and condition checks.
// Collects all condition check errors and returns them aggregated.
// executeNotificationHooksJSON executes all matching Notification hooks and returns JSON output.
// Notification follows SessionStart pattern (hookSpecificOutput + additionalContext).
// Continue is always forced to true (Notification cannot block - official spec "Can block? = No").
// Returns (*NotificationOutput, error) where output is always non-nil.
func executeNotificationHooksJSON(config *Config, input *NotificationInput, rawJSON interface{}) (*NotificationOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true (forced for Notification)
	finalOutput := &NotificationOutput{
		Continue: true,
	}

	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""

	for i, hook := range config.Notification {
		// Matcher check: filter by notification_type
		if hook.Matcher != "" {
			// Warn if matcher contains unknown notification_type values
			knownTypes := map[string]bool{
				"permission_prompt":  true,
				"idle_prompt":        true,
				"auth_success":       true,
				"elicitation_dialog": true,
			}
			for _, pattern := range strings.Split(hook.Matcher, "|") {
				trimmedPattern := strings.TrimSpace(pattern)
				if trimmedPattern != "" && !knownTypes[trimmedPattern] {
					fmt.Fprintf(os.Stderr, "Warning: Notification hook matcher contains unknown notification_type: %q (known types: permission_prompt, idle_prompt, auth_success, elicitation_dialog)\n", trimmedPattern)
				}
			}

			// Filter hooks by matcher
			if !checkNotificationMatcher(hook.Matcher, input.NotificationType) {
				continue
			}
		}

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
			actionOutput, err := executor.ExecuteNotificationAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("notification hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Update finalOutput fields following merge rules

			// Continue: Always force to true (ignore action result - Notification cannot block)
			finalOutput.Continue = true

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

			// StopReason: 最後の非空値が勝ち
			if actionOutput.StopReason != "" {
				finalOutput.StopReason = actionOutput.StopReason
			}

			// SuppressOutput: 最後の値が勝ち
			finalOutput.SuppressOutput = actionOutput.SuppressOutput

			// No early return for Notification (cannot block)
		}
	}

	// Build final output
	// Always set hookEventName to "Notification"
	if hookEventName == "" {
		hookEventName = "Notification"
	}
	finalOutput.HookSpecificOutput = &NotificationHookSpecificOutput{
		HookEventName:     hookEventName,
		AdditionalContext: additionalContextBuilder.String(),
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)
	allErrors = append(allErrors, actionErrors...)

	if len(allErrors) > 0 {
		// Continue remains true (Notification cannot block)
		// Add error info to SystemMessage for graceful degradation
		errorMsg := errors.Join(allErrors...).Error()
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}
		// Return error to trigger stderr warning in main.go
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
}

// executeSubagentStartHooksJSON executes all matching SubagentStart hooks and returns JSON output.
// Similar to Notification, SubagentStart uses hookSpecificOutput with additionalContext.
// Includes matcher check on agent_type.
func executeSubagentStartHooksJSON(config *Config, input *SubagentStartInput, rawJSON interface{}) (*SubagentStartOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true (forced for SubagentStart)
	finalOutput := &SubagentStartOutput{
		Continue: true,
	}

	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""

	for i, hook := range config.SubagentStart {
		// Matcher check (agent type filter)
		if !checkMatcher(hook.Matcher, input.AgentType) {
			continue
		}

		// 条件チェック
		shouldExecute := true
		for _, condition := range hook.Conditions {
			matched, err := checkSubagentStartCondition(condition, input)
			if err != nil {
				conditionErrors = append(conditionErrors,
					fmt.Errorf("hook[SubagentStart][%d]: %w", i, err))
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
			actionOutput, err := executor.ExecuteSubagentStartAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("SubagentStart hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
			}

			// Update finalOutput fields following merge rules

			// Continue: Always force to true (ignore action result - SubagentStart cannot block)
			finalOutput.Continue = true

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

			// StopReason: 最後の非空値が勝ち
			if actionOutput.StopReason != "" {
				finalOutput.StopReason = actionOutput.StopReason
			}

			// SuppressOutput: 最後の値が勝ち
			finalOutput.SuppressOutput = actionOutput.SuppressOutput

			// No early return for SubagentStart (cannot block)
		}
	}

	// Build final output
	// Always set hookEventName to "SubagentStart"
	if hookEventName == "" {
		hookEventName = "SubagentStart"
	}
	finalOutput.HookSpecificOutput = &SubagentStartHookSpecificOutput{
		HookEventName:     hookEventName,
		AdditionalContext: additionalContextBuilder.String(),
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)
	allErrors = append(allErrors, actionErrors...)

	if len(allErrors) > 0 {
		// Continue remains true (SubagentStart cannot block)
		// Add error info to SystemMessage for graceful degradation
		errorMsg := errors.Join(allErrors...).Error()
		if finalOutput.SystemMessage != "" {
			finalOutput.SystemMessage += "\n" + errorMsg
		} else {
			finalOutput.SystemMessage = errorMsg
		}
		// Return error to trigger stderr warning in main.go
		return finalOutput, errors.Join(allErrors...)
	}

	return finalOutput, nil
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
func executePreCompactHooksJSON(config *Config, input *PreCompactInput, rawJSON interface{}) (*PreCompactOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput: Continue always true (compaction cannot be blocked)
	finalOutput := &PreCompactOutput{
		Continue: true,
	}

	var systemMessageBuilder strings.Builder

	for i, hook := range config.PreCompact {
		// Warn about invalid matcher values (early detection of configuration mistakes)
		if hook.Matcher != "" && hook.Matcher != "manual" && hook.Matcher != "auto" {
			fmt.Fprintf(os.Stderr, "Warning: PreCompact hook %d has invalid matcher value %q (expected: \"manual\", \"auto\", or empty)\n", i, hook.Matcher)
		}

		// マッチャーチェック (manual/auto)
		if hook.Matcher != "" && hook.Matcher != input.Trigger {
			continue
		}

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
			actionOutput, err := executor.ExecutePreCompactAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("pre compact hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
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
		}
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// アクションエラーがある場合はfail-safe: systemMessageに追加
	if len(actionErrors) > 0 {
		errorMsg := errors.Join(actionErrors...).Error()
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

// executePostToolUseHook executes all actions for a single PostToolUse hook.
// Note: This is a temporary implementation for compatibility.
// Will be replaced with executePostToolUseHooksJSON in Step 4.

// executeSessionEndHooks executes all matching SessionEnd hooks based on condition checks.
// Returns errors to inform users of failures, though session end cannot be blocked.
// executeSessionEndHooksJSON executes all matching SessionEnd hooks and returns SessionEndOutput.
// SessionEnd always returns continue=true (fail-safe: session end cannot be blocked).
// Errors are reported via systemMessage field, not by blocking execution.
func executeSessionEndHooksJSON(config *Config, input *SessionEndInput, rawJSON interface{}) (*SessionEndOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput: Continue always true (session end cannot be blocked)
	finalOutput := &SessionEndOutput{
		Continue: true,
	}

	var systemMessageBuilder strings.Builder

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
			actionOutput, err := executor.ExecuteSessionEndAction(action, input, rawJSON)
			if err != nil {
				actionErrors = append(actionErrors, fmt.Errorf("session end hook %d action failed: %w", i, err))
				continue
			}

			if actionOutput == nil {
				continue
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
		}
	}

	finalOutput.SystemMessage = systemMessageBuilder.String()

	// Collect all errors
	var allErrors []error
	allErrors = append(allErrors, conditionErrors...)

	// アクションエラーがある場合はfail-safe: systemMessageに追加
	if len(actionErrors) > 0 {
		errorMsg := errors.Join(actionErrors...).Error()
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
