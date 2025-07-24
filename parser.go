package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ジェネリック入力パース関数（基本構造のみ）
func parseInput[T HookInput](eventType HookEventType) (T, error) {
	var rawInput json.RawMessage
	var input T

	// まずJSONを取得
	if err := json.NewDecoder(os.Stdin).Decode(&rawInput); err != nil {
		return input, fmt.Errorf("failed to decode JSON input: %w", err)
	}

	// イベントタイプに応じて特別な処理を行う
	switch eventType {
	case PreToolUse:
		preInput, err := parsePreToolUseInput(rawInput)
		if err != nil {
			return input, err
		}
		if result, ok := interface{}(preInput).(T); ok {
			return result, nil
		}
		return input, fmt.Errorf("type assertion failed for PreToolUse")

	case PostToolUse:
		postInput, err := parsePostToolUseInput(rawInput)
		if err != nil {
			return input, err
		}
		if result, ok := interface{}(postInput).(T); ok {
			return result, nil
		}
		return input, fmt.Errorf("type assertion failed for PostToolUse")

	default:
		// その他のイベントタイプは従来通り
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return input, fmt.Errorf("failed to decode %s input: %w", eventType, err)
		}
		return input, nil
	}
}

// PreToolUseInputの特別なパース関数
func parsePreToolUseInput(rawInput json.RawMessage) (*PreToolUseInput, error) {
	// まず基本構造をパース
	var temp struct {
		BaseInput
		ToolName string          `json:"tool_name"`
		ToolInput json.RawMessage `json:"tool_input"`
	}

	if err := json.Unmarshal(rawInput, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse PreToolUse base structure: %w", err)
	}

	// tool_inputを適切な構造体にパース
	toolInput, err := parseToolInputByName(temp.ToolName, temp.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool_input for %s: %w", temp.ToolName, err)
	}

	return &PreToolUseInput{
		BaseInput: temp.BaseInput,
		ToolName:  temp.ToolName,
		ToolInput: toolInput,
	}, nil
}

// PostToolUseInputの特別なパース関数
func parsePostToolUseInput(rawInput json.RawMessage) (*PostToolUseInput, error) {
	// まず基本構造をパース
	var temp struct {
		BaseInput
		ToolName     string           `json:"tool_name"`
		ToolInput    json.RawMessage  `json:"tool_input"`
		ToolResponse json.RawMessage  `json:"tool_response"`
	}

	if err := json.Unmarshal(rawInput, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse PostToolUse base structure: %w", err)
	}

	// tool_inputを適切な構造体にパース
	toolInput, err := parseToolInputByName(temp.ToolName, temp.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool_input for %s: %w", temp.ToolName, err)
	}

	// tool_responseをパース
	var toolResponse ToolResponse
	if err := json.Unmarshal(temp.ToolResponse, &toolResponse); err != nil {
		return nil, fmt.Errorf("failed to parse tool_response: %w", err)
	}

	return &PostToolUseInput{
		BaseInput:    temp.BaseInput,
		ToolName:     temp.ToolName,
		ToolInput:    toolInput,
		ToolResponse: toolResponse,
	}, nil
}

// ツール名に基づいてtool_inputを適切な構造体にパースする関数
// 全ツール共通構造と仮定
func parseToolInputByName(toolName string, rawToolInput json.RawMessage) (ToolInput, error) {
	var input ToolInput
	if err := json.Unmarshal(rawToolInput, &input); err != nil {
		return ToolInput{}, fmt.Errorf("failed to parse tool input: %w", err)
	}
	return input, nil
}