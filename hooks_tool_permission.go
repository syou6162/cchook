package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

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

// executePreToolUseHooksJSON executes all matching PreToolUse hooks and returns JSON output.
// This function implements Phase 3 JSON output functionality for PreToolUse hooks.
func executePreToolUseHooksJSON(config *Config, input *PreToolUseInput, rawJSON any) (*PreToolUseOutput, error) {
	executor := NewActionExecutor(nil)
	var conditionErrors []error
	var actionErrors []error

	// Initialize finalOutput with Continue: true (always)
	finalOutput := &PreToolUseOutput{
		Continue: true,
	}

	var reasonBuilder strings.Builder
	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	hookEventName := ""
	permissionDecision := "" // Empty = delegate to Claude Code's permission system
	var updatedInput map[string]any
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
	// Set HookSpecificOutput when any hook-specific field is set or errors occurred
	if permissionDecision != "" || additionalContextBuilder.Len() > 0 || updatedInput != nil || len(allErrors) > 0 {
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
			AdditionalContext:        additionalContextBuilder.String(),
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

// executePreToolUseHook executes all actions for a single PreToolUse hook and returns JSON output.
// This function implements Phase 3 JSON output functionality for PreToolUse hooks.
func executePreToolUseHook(executor *ActionExecutor, hook PreToolUseHook, input *PreToolUseInput, rawJSON any) (*ActionOutput, error) {
	// Initialize output with Continue: true (always true for PreToolUse)
	// permissionDecision starts empty and will be set by actions or remain empty to delegate
	output := &ActionOutput{
		Continue:           true,
		PermissionDecision: "",
		HookEventName:      "PreToolUse",
	}

	var reasonBuilder strings.Builder
	var additionalContextBuilder strings.Builder
	var systemMessageBuilder strings.Builder
	var updatedInput map[string]any

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
		additionalContextBuilder.Len() == 0 &&
		systemMessageBuilder.Len() == 0 &&
		updatedInput == nil &&
		output.StopReason == "" &&
		!output.SuppressOutput {
		return nil, nil
	}

	// Build final output
	output.PermissionDecisionReason = reasonBuilder.String()
	output.AdditionalContext = additionalContextBuilder.String()
	output.SystemMessage = systemMessageBuilder.String()
	output.UpdatedInput = updatedInput

	return output, nil
}

// executePostToolUseHooksJSON executes all matching PostToolUse hooks and returns JSON output.
// Implements merging rules for multiple hook outputs.
func executePostToolUseHooksJSON(config *Config, input *PostToolUseInput, rawJSON any) (*PostToolUseOutput, error) {
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

			// UpdatedMCPToolOutput: 最後の非nil値が勝ち
			if actionOutput.UpdatedMCPToolOutput != nil {
				finalOutput.UpdatedMCPToolOutput = actionOutput.UpdatedMCPToolOutput
			}

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

// executePermissionRequestHooksJSON executes PermissionRequest hooks and returns JSON output
func executePermissionRequestHooksJSON(config *Config, input *PermissionRequestInput, rawJSON any) (*PermissionRequestOutput, error) {
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
	var updatedInput map[string]any
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
func executePermissionRequestHook(executor *ActionExecutor, hook PermissionRequestHook, input *PermissionRequestInput, rawJSON any) (*ActionOutput, error) {
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
