package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// parseInput parses JSON input from stdin and returns both structured data and raw JSON.
// It handles special processing for PreToolUse and PostToolUse events that have complex tool_input fields.
func parseInput[T HookInput](eventType HookEventType) (T, interface{}, error) {
	var rawInput json.RawMessage
	var input T

	// まずJSONを取得
	if err := json.NewDecoder(os.Stdin).Decode(&rawInput); err != nil {
		return input, nil, fmt.Errorf("failed to decode JSON input: %w", err)
	}

	// 生のJSONをinterface{}に変換（JQ用）
	var rawJSON interface{}
	if err := json.Unmarshal(rawInput, &rawJSON); err != nil {
		return input, nil, fmt.Errorf("failed to parse raw JSON: %w", err)
	}

	// イベントタイプに応じて特別な処理を行う
	switch eventType {
	case PreToolUse:
		preInput, err := parsePreToolUseInput(rawInput)
		if err != nil {
			return input, nil, err
		}
		if result, ok := interface{}(preInput).(T); ok {
			return result, rawJSON, nil
		}
		return input, nil, fmt.Errorf("type assertion failed for PreToolUse")

	case PostToolUse:
		postInput, err := parsePostToolUseInput(rawInput)
		if err != nil {
			return input, nil, err
		}
		if result, ok := interface{}(postInput).(T); ok {
			return result, rawJSON, nil
		}
		return input, nil, fmt.Errorf("type assertion failed for PostToolUse")

	default:
		// その他のイベントタイプは従来通り
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return input, nil, fmt.Errorf("failed to decode %s input: %w", eventType, err)
		}
		return input, rawJSON, nil
	}
}

// parsePreToolUseInput parses PreToolUse event input with special handling for tool_input field.
// It first parses the base structure, then parses tool_input according to the tool name.
func parsePreToolUseInput(rawInput json.RawMessage) (*PreToolUseInput, error) {
	// まず基本構造をパース
	var temp struct {
		BaseInput
		ToolName  string          `json:"tool_name"`
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

// parsePostToolUseInput parses PostToolUse event input with special handling for tool_input and tool_response fields.
// It parses tool_input according to the tool name and preserves tool_response as RawMessage.
func parsePostToolUseInput(rawInput json.RawMessage) (*PostToolUseInput, error) {
	// まず基本構造をパース
	var temp struct {
		BaseInput
		ToolName     string          `json:"tool_name"`
		ToolInput    json.RawMessage `json:"tool_input"`
		ToolResponse json.RawMessage `json:"tool_response"`
	}

	if err := json.Unmarshal(rawInput, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse PostToolUse base structure: %w", err)
	}

	// tool_inputを適切な構造体にパース
	toolInput, err := parseToolInputByName(temp.ToolName, temp.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool_input for %s: %w", temp.ToolName, err)
	}

	// tool_responseをそのままRawMessageとして保持
	toolResponse := ToolResponse(temp.ToolResponse)

	return &PostToolUseInput{
		BaseInput:    temp.BaseInput,
		ToolName:     temp.ToolName,
		ToolInput:    toolInput,
		ToolResponse: toolResponse,
	}, nil
}

// parseToolInputByName parses tool_input JSON into a ToolInput struct based on the tool name.
// Currently assumes a common structure for all tools.
func parseToolInputByName(toolName string, rawToolInput json.RawMessage) (ToolInput, error) {
	var input ToolInput
	if err := json.Unmarshal(rawToolInput, &input); err != nil {
		return ToolInput{}, fmt.Errorf("failed to parse tool input: %w", err)
	}
	return input, nil
}
