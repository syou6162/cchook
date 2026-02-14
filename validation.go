package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
)

// validateSessionStartOutput validates SessionStartOutput JSON against auto-generated schema
func validateSessionStartOutput(jsonData []byte) error {
	// Generate schema from SessionStartOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&SessionStartOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: only hookSpecificOutput (continue is always output but not required for validation)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: hookEventName must be "SessionStart"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"SessionStart"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

func validateUserPromptSubmitOutput(jsonData []byte) error {
	// Generate schema from UserPromptSubmitOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&UserPromptSubmitOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: hookSpecificOutput only (decision is optional)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: decision must be "block" only (or omitted entirely)
	if decisionProp, ok := schema.Properties.Get("decision"); ok {
		if decision := decisionProp; decision != nil {
			decision.Enum = []any{"block"}
		}
	}

	// 4. Add custom validation: hookEventName must be "UserPromptSubmit"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"UserPromptSubmit"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// validatePreToolUseOutput validates PreToolUseOutput against JSON schema (Phase 3)
func validatePreToolUseOutput(jsonData []byte) error {
	// Generate schema from PreToolUseOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&PreToolUseOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields at root level
	// - continue: optional (defaults to true per Claude Code spec)
	// - hookSpecificOutput: optional (omitempty tag, omit to delegate)
	schema.Required = nil

	// 3. Configure hookSpecificOutput
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			// hookEventName must be "PreToolUse"
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"PreToolUse"}
				}
			}
			// permissionDecision must be "allow", "deny", or "ask"
			if permissionDecisionProp, ok := hookSpecific.Properties.Get("permissionDecision"); ok {
				if permissionDecision := permissionDecisionProp; permissionDecision != nil {
					permissionDecision.Enum = []any{"allow", "deny", "ask"}
				}
			}
			// hookSpecificOutput.hookEventName and permissionDecision are required
			hookSpecific.Required = []string{"hookEventName", "permissionDecision"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, desc.String())
		}
		return fmt.Errorf("JSON schema validation failed: %s", errMsgs)
	}

	return nil
}

// validatePermissionRequestOutput validates the JSON output for PermissionRequest hooks.
// It checks both schema compliance and semantic rules (e.g., interrupt only valid with deny).
func validatePermissionRequestOutput(jsonData []byte) error {
	// Generate schema from PermissionRequestOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&PermissionRequestOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: hookSpecificOutput
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Configure hookSpecificOutput
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			// hookEventName must be "PermissionRequest"
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"PermissionRequest"}
				}
			}
			// hookSpecificOutput.hookEventName and decision are required
			hookSpecific.Required = []string{"hookEventName", "decision"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false

			// 4. Configure decision
			if decisionProp, ok := hookSpecific.Properties.Get("decision"); ok {
				if decision := decisionProp; decision != nil {
					// behavior must be "allow" or "deny"
					if behaviorProp, ok := decision.Properties.Get("behavior"); ok {
						if behavior := behaviorProp; behavior != nil {
							behavior.Enum = []any{"allow", "deny"}
						}
					}
					// decision.behavior is required
					decision.Required = []string{"behavior"}
					// decision should not allow additional properties
					decision.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
				}
			}
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, desc.String())
		}
		return fmt.Errorf("JSON schema validation failed: %s", errMsgs)
	}

	// Additional semantic validation: check behavior-specific field constraints
	var output PermissionRequestOutput
	if err := json.Unmarshal(jsonData, &output); err != nil {
		return fmt.Errorf("failed to unmarshal output for semantic validation: %w", err)
	}

	if output.HookSpecificOutput != nil && output.HookSpecificOutput.Decision != nil {
		decision := output.HookSpecificOutput.Decision
		behavior := decision.Behavior

		switch behavior {
		case "allow":
			// allow時: message と interrupt は空/false であるべき
			if decision.Message != "" {
				return fmt.Errorf("semantic validation failed: 'message' should be empty when behavior is 'allow', got: %q", decision.Message)
			}
			if decision.Interrupt {
				return fmt.Errorf("semantic validation failed: 'interrupt' should be false when behavior is 'allow'")
			}
		case "deny":
			// deny時: updatedInput は存在しないべき
			if decision.UpdatedInput != nil {
				return fmt.Errorf("semantic validation failed: 'updatedInput' should not exist when behavior is 'deny'")
			}
			// deny時: message は必須
			if decision.Message == "" {
				return fmt.Errorf("semantic validation failed: 'message' is required when behavior is 'deny'")
			}
		}
	}

	return nil
}

// validateSubagentStopOutput validates SubagentStop hook JSON output against schema and semantic rules.
// SubagentStop uses top-level decision/reason (no hookSpecificOutput), same as Stop.
// Semantic rule: decision == "block" requires reason to be non-empty.
func validateSubagentStopOutput(jsonData []byte) error {
	// Generate schema from SubagentStopOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&SubagentStopOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields (decision and reason are both optional at schema level)
	schema.Required = nil

	// 3. Add custom validation: decision must be "block" only (or omitted entirely)
	if decisionProp, ok := schema.Properties.Get("decision"); ok {
		if decision := decisionProp; decision != nil {
			decision.Enum = []any{"block"}
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Semantic validation: decision == "block" requires reason to be non-empty
	var output SubagentStopOutput
	if err := json.Unmarshal(jsonData, &output); err != nil {
		return fmt.Errorf("failed to unmarshal output for semantic validation: %w", err)
	}

	if output.Decision == "block" && output.Reason == "" {
		return fmt.Errorf("semantic validation failed: 'reason' is required when decision is 'block'")
	}

	return nil
}

// validateStopOutput validates Stop hook JSON output against schema and semantic rules.
// Stop uses top-level decision/reason (no hookSpecificOutput).
// Semantic rule: decision == "block" requires reason to be non-empty.
func validateStopOutput(jsonData []byte) error {
	// Generate schema from StopOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&StopOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields (decision and reason are both optional at schema level)
	schema.Required = nil

	// 3. Add custom validation: decision must be "block" only (or omitted entirely)
	if decisionProp, ok := schema.Properties.Get("decision"); ok {
		if decision := decisionProp; decision != nil {
			decision.Enum = []any{"block"}
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Semantic validation: decision == "block" requires reason to be non-empty
	var output StopOutput
	if err := json.Unmarshal(jsonData, &output); err != nil {
		return fmt.Errorf("failed to unmarshal output for semantic validation: %w", err)
	}

	if output.Decision == "block" && output.Reason == "" {
		return fmt.Errorf("semantic validation failed: 'reason' is required when decision is 'block'")
	}

	return nil
}

// validateNotificationOutput validates NotificationOutput JSON against auto-generated schema
func validateNotificationOutput(jsonData []byte) error {
	// Generate schema from NotificationOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&NotificationOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: only hookSpecificOutput (continue is always output but not required for validation)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: hookEventName must be "Notification"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"Notification"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Additional validation: check for unsupported fields (decision, reason)
	// Notification does NOT support decision/reason fields (those are for Stop/SubagentStop/PostToolUse)
	var rawOutput map[string]any
	if err := json.Unmarshal(jsonData, &rawOutput); err != nil {
		return fmt.Errorf("failed to unmarshal for unsupported field check: %w", err)
	}

	unsupportedFields := []string{"decision", "reason"}
	for _, field := range unsupportedFields {
		if _, exists := rawOutput[field]; exists {
			return fmt.Errorf("unsupported field for Notification: %s", field)
		}
	}

	return nil
}

// validateSubagentStartOutput validates the JSON output for SubagentStart hooks
// against both schema and semantic requirements.
func validateSubagentStartOutput(jsonData []byte) error {
	// Generate schema from SubagentStartOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&SubagentStartOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: only hookSpecificOutput (continue is always output but not required for validation)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: hookEventName must be "SubagentStart"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"SubagentStart"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Additional validation: check for unsupported fields (decision, reason)
	// SubagentStart does NOT support decision/reason fields (those are for Stop/SubagentStop/PostToolUse)
	var rawOutput map[string]any
	if err := json.Unmarshal(jsonData, &rawOutput); err != nil {
		return fmt.Errorf("failed to unmarshal for unsupported field check: %w", err)
	}

	unsupportedFields := []string{"decision", "reason"}
	for _, field := range unsupportedFields {
		if _, exists := rawOutput[field]; exists {
			return fmt.Errorf("unsupported field for SubagentStart: %s", field)
		}
	}

	return nil
}

// validatePostToolUseOutput validates the JSON output for PostToolUse hooks
// against both schema and semantic requirements.
func validatePostToolUseOutput(jsonData []byte) error {
	// Generate schema from PostToolUseOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&PostToolUseOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields (decision and reason are both optional at schema level)
	schema.Required = nil

	// 3. Add custom validation: decision must be "block" only (or omitted entirely)
	if decisionProp, ok := schema.Properties.Get("decision"); ok {
		if decision := decisionProp; decision != nil {
			decision.Enum = []any{"block"}
		}
	}

	// 4. Customize hookSpecificOutput: hookEventName must be "PostToolUse"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []any{"PostToolUse"}
				}
			}
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Semantic validation: decision == "block" requires reason to be non-empty
	var output PostToolUseOutput
	if err := json.Unmarshal(jsonData, &output); err != nil {
		return fmt.Errorf("failed to unmarshal output for semantic validation: %w", err)
	}

	if output.Decision == "block" && output.Reason == "" {
		return fmt.Errorf("semantic validation failed: 'reason' is required when decision is 'block'")
	}

	return nil
}

// validateSessionEndOutput validates SessionEnd hook JSON output against schema.
// SessionEnd uses Common JSON Fields only (no decision/reason/hookSpecificOutput).
// No semantic validation required (all fields optional, no interdependencies).
func validateSessionEndOutput(jsonData []byte) error {
	// Generate schema from SessionEndOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&SessionEndOutput{})

	// Customize schema to match SessionEnd requirements:
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields (all fields optional)
	schema.Required = nil

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// No semantic validation required for SessionEnd
	return nil
}

func validatePreCompactOutput(jsonData []byte) error {
	// Generate schema from PreCompactOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&PreCompactOutput{})

	// Customize schema to match PreCompact requirements:
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields (all fields optional)
	schema.Required = nil

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// No semantic validation required for PreCompact
	return nil
}
