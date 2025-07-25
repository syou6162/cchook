package main

import (
	"encoding/json"
	"fmt"
)

// 強化されたoutput処理：JSONかテキストかを自動判別して適切に処理する

func processEnhancedOutput(message string, hookType HookEventType, rawJSON interface{}) error {
	// テンプレート変数の置換
	processedMessage := unifiedTemplateReplace(message, rawJSON)

	// JSON形式かどうかを判定
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(processedMessage), &jsonData); err == nil {
		// JSON形式の場合：バリデーション後に構造化出力として処理
		if err := validateJSONStructure(jsonData, hookType); err != nil {
			return fmt.Errorf("JSON validation failed: %w", err)
		}
		return outputStructuredJSON(jsonData, hookType)
	}

	// 通常のテキスト出力
	fmt.Println(processedMessage)
	return nil
}

// JSON構造のバリデーション
func validateJSONStructure(data map[string]interface{}, hookType HookEventType) error {
	// 共通フィールドのバリデーション
	if cont, exists := data["continue"]; exists {
		if _, ok := cont.(bool); !ok {
			return fmt.Errorf("'continue' must be a boolean")
		}
	}

	if suppress, exists := data["suppressOutput"]; exists {
		if _, ok := suppress.(bool); !ok {
			return fmt.Errorf("'suppressOutput' must be a boolean")
		}
	}

	// フックタイプ別のバリデーション
	switch hookType {
	case PreToolUse:
		return validatePreToolUseOutput(data)
	case PostToolUse:
		return validatePostToolUseOutput(data)
	case Stop, SubagentStop:
		return validateStopOutput(data)
	default:
		// その他のフックタイプは共通バリデーションのみ
		return nil
	}
}

// PreToolUse固有のバリデーション
func validatePreToolUseOutput(data map[string]interface{}) error {
	if hookSpecific, exists := data["hookSpecificOutput"]; exists {
		specific, ok := hookSpecific.(map[string]interface{})
		if !ok {
			return fmt.Errorf("'hookSpecificOutput' must be an object")
		}

		if permission, exists := specific["permissionDecision"]; exists {
			permStr, ok := permission.(string)
			if !ok {
				return fmt.Errorf("'permissionDecision' must be a string")
			}
			if permStr != "allow" && permStr != "deny" && permStr != "ask" {
				return fmt.Errorf("'permissionDecision' must be 'allow', 'deny', or 'ask'")
			}
		}
	}
	return nil
}

// PostToolUse固有のバリデーション
func validatePostToolUseOutput(data map[string]interface{}) error {
	if decision, exists := data["decision"]; exists {
		decStr, ok := decision.(string)
		if !ok {
			return fmt.Errorf("'decision' must be a string")
		}
		if decStr != "block" {
			return fmt.Errorf("'decision' must be 'block'")
		}
	}
	return nil
}

// Stop/SubagentStop固有のバリデーション
func validateStopOutput(data map[string]interface{}) error {
	if decision, exists := data["decision"]; exists {
		decStr, ok := decision.(string)
		if !ok {
			return fmt.Errorf("'decision' must be a string")
		}
		if decStr != "block" {
			return fmt.Errorf("'decision' must be 'block'")
		}

		// blockの場合、reasonは必須
		if _, reasonExists := data["reason"]; !reasonExists {
			return fmt.Errorf("'reason' is required when 'decision' is 'block'")
		}
	}
	return nil
}

func outputStructuredJSON(data map[string]interface{}, hookType HookEventType) error {
	// Claude Code互換の構造化JSON出力を生成
	structuredOutput, err := createStructuredOutput(data, hookType)
	if err != nil {
		return fmt.Errorf("failed to create structured output: %w", err)
	}

	// 構造化JSON出力
	outputBytes, err := json.Marshal(structuredOutput)
	if err != nil {
		return fmt.Errorf("failed to marshal structured output: %w", err)
	}

	fmt.Println(string(outputBytes))
	return nil
}

// Claude Code互換の構造化出力を生成
func createStructuredOutput(data map[string]interface{}, hookType HookEventType) (interface{}, error) {
	switch hookType {
	case PreToolUse:
		return createPreToolUseOutput(data), nil
	case PostToolUse:
		return createPostToolUseOutput(data), nil
	case Notification:
		return createNotificationOutput(data), nil
	case Stop:
		return createStopOutput(data), nil
	case SubagentStop:
		return createSubagentStopOutput(data), nil
	case PreCompact:
		return createPreCompactOutput(data), nil
	default:
		return nil, fmt.Errorf("unsupported hook type: %s", hookType)
	}
}

// PreToolUse用の構造化出力を生成
func createPreToolUseOutput(data map[string]interface{}) *PreToolUseOutput {
	output := &PreToolUseOutput{
		BaseHookOutput: BaseHookOutput{},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	// 非推奨フィールドの設定（互換性のため）
	if decision, ok := data["decision"].(string); ok {
		output.Decision = &decision
	}
	if reason, ok := data["reason"].(string); ok {
		output.Reason = &reason
	}

	// PreToolUse固有フィールドの設定
	if hookSpecific, ok := data["hookSpecificOutput"].(map[string]interface{}); ok {
		specificOutput := &PreToolUseSpecificOutput{
			HookEventName: "PreToolUse",
		}

		if permission, ok := hookSpecific["permissionDecision"].(string); ok {
			specificOutput.PermissionDecision = &permission
		}
		if reason, ok := hookSpecific["permissionDecisionReason"].(string); ok {
			specificOutput.PermissionDecisionReason = &reason
		}

		output.HookSpecificOutput = specificOutput
	}

	return output
}

// PostToolUse用の構造化出力を生成
func createPostToolUseOutput(data map[string]interface{}) *PostToolUseOutput {
	output := &PostToolUseOutput{
		BaseHookOutput: BaseHookOutput{},
		HookSpecificOutput: &PostToolUseSpecificOutput{
			HookEventName: "PostToolUse",
		},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	// PostToolUse固有フィールドの設定
	if decision, ok := data["decision"].(string); ok {
		output.Decision = &decision
	}
	if reason, ok := data["reason"].(string); ok {
		output.Reason = &reason
	}

	return output
}

// Notification用の構造化出力を生成
func createNotificationOutput(data map[string]interface{}) *NotificationOutput {
	output := &NotificationOutput{
		BaseHookOutput: BaseHookOutput{},
		HookSpecificOutput: &NotificationSpecificOutput{
			HookEventName: "Notification",
		},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	return output
}

// Stop用の構造化出力を生成
func createStopOutput(data map[string]interface{}) *StopOutput {
	output := &StopOutput{
		BaseHookOutput: BaseHookOutput{},
		HookSpecificOutput: &StopSpecificOutput{
			HookEventName: "Stop",
		},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	// Stop固有フィールドの設定
	if decision, ok := data["decision"].(string); ok {
		output.Decision = &decision
	}
	if reason, ok := data["reason"].(string); ok {
		output.Reason = &reason
	}

	return output
}

// SubagentStop用の構造化出力を生成
func createSubagentStopOutput(data map[string]interface{}) *SubagentStopOutput {
	output := &SubagentStopOutput{
		BaseHookOutput: BaseHookOutput{},
		HookSpecificOutput: &StopSpecificOutput{
			HookEventName: "SubagentStop",
		},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	// SubagentStop固有フィールドの設定
	if decision, ok := data["decision"].(string); ok {
		output.Decision = &decision
	}
	if reason, ok := data["reason"].(string); ok {
		output.Reason = &reason
	}

	return output
}

// PreCompact用の構造化出力を生成
func createPreCompactOutput(data map[string]interface{}) *PreCompactOutput {
	output := &PreCompactOutput{
		BaseHookOutput: BaseHookOutput{},
		HookSpecificOutput: &PreCompactSpecificOutput{
			HookEventName: "PreCompact",
		},
	}

	// 共通フィールドの設定
	if cont, ok := data["continue"].(bool); ok {
		output.Continue = &cont
	}
	if stopReason, ok := data["stopReason"].(string); ok {
		output.StopReason = &stopReason
	}
	if suppress, ok := data["suppressOutput"].(bool); ok {
		output.SuppressOutput = &suppress
	}

	return output
}
