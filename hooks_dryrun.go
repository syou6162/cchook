package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

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
		if hook.Matcher != "" {
			fmt.Printf("  Matcher: %s\n", hook.Matcher)
		}
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

// dryRunSubagentStartHooks performs a dry-run of SubagentStart hooks, showing what would be executed without actually running.
// Includes matcher check on agent_type.
func dryRunSubagentStartHooks(config *Config, input *SubagentStartInput, rawJSON interface{}) error {
	fmt.Println("=== SubagentStart Hooks (Dry Run) ===")

	if len(config.SubagentStart) == 0 {
		fmt.Println("No SubagentStart hooks configured")
		return nil
	}

	executed := false
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
		fmt.Printf("  Matcher: %s\n", hook.Matcher)
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
