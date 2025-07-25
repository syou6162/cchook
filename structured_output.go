package main

import (
	"encoding/json"
	"fmt"
)

func executeStructuredOutput(action interface{}, hookType HookEventType) error {
	structuredOutput, err := createStructuredOutputFromAction(action, hookType)
	if err != nil {
		return fmt.Errorf("failed to create structured output: %w", err)
	}

	outputBytes, err := json.Marshal(structuredOutput)
	if err != nil {
		return fmt.Errorf("failed to marshal structured output: %w", err)
	}

	fmt.Println(string(outputBytes))
	return nil
}

func createStructuredOutputFromAction(action interface{}, hookType HookEventType) (interface{}, error) {
	switch hookType {
	case PreToolUse:
		a := action.(PreToolUseAction)
		return createPreToolUseOutputFromAction(a), nil
	case PostToolUse:
		a := action.(PostToolUseAction)
		return createPostToolUseOutputFromAction(a), nil
	case Notification:
		a := action.(NotificationAction)
		return createNotificationOutputFromAction(a), nil
	case Stop:
		a := action.(StopAction)
		return createStopOutputFromAction(a), nil
	case SubagentStop:
		a := action.(SubagentStopAction)
		return createSubagentStopOutputFromAction(a), nil
	case PreCompact:
		a := action.(PreCompactAction)
		return createPreCompactOutputFromAction(a), nil
	default:
		return nil, fmt.Errorf("unsupported hook type: %s", hookType)
	}
}

func createPreToolUseOutputFromAction(action PreToolUseAction) *PreToolUseOutput {
	output := &PreToolUseOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
	}

	if action.PermissionDecision != nil || action.PermissionDecisionReason != nil {
		output.HookSpecificOutput = &PreToolUseSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       action.PermissionDecision,
			PermissionDecisionReason: action.PermissionDecisionReason,
		}
	}

	return output
}

func createPostToolUseOutputFromAction(action PostToolUseAction) *PostToolUseOutput {
	return &PostToolUseOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
		Decision: action.Decision,
		Reason:   action.Reason,
		HookSpecificOutput: &PostToolUseSpecificOutput{
			HookEventName: "PostToolUse",
		},
	}
}

func createNotificationOutputFromAction(action NotificationAction) *NotificationOutput {
	return &NotificationOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
		HookSpecificOutput: &NotificationSpecificOutput{
			HookEventName: "Notification",
		},
	}
}

func createStopOutputFromAction(action StopAction) *StopOutput {
	return &StopOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
		Decision: action.Decision,
		Reason:   action.Reason,
		HookSpecificOutput: &StopSpecificOutput{
			HookEventName: "Stop",
		},
	}
}

func createSubagentStopOutputFromAction(action SubagentStopAction) *SubagentStopOutput {
	return &SubagentStopOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
		Decision: action.Decision,
		Reason:   action.Reason,
		HookSpecificOutput: &StopSpecificOutput{
			HookEventName: "SubagentStop",
		},
	}
}

func createPreCompactOutputFromAction(action PreCompactAction) *PreCompactOutput {
	return &PreCompactOutput{
		BaseHookOutput: BaseHookOutput{
			Continue:       action.Continue,
			StopReason:     action.StopReason,
			SuppressOutput: action.SuppressOutput,
		},
		HookSpecificOutput: &PreCompactSpecificOutput{
			HookEventName: "PreCompact",
		},
	}
}
