package main

import (
	"os"
	"strings"
	"testing"
)

// TestRunNotificationHooks tests the basic notification hook execution path
func TestRunNotificationHooks(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "Notification",
		"title": "Test Notification",
		"notification_type": "idle_prompt"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		Notification: []NotificationHook{
			{
				Actions: []Action{
					{Type: "output", Message: "Received notification: {.title}"},
				},
			},
		},
	}

	output, err := RunNotificationHooks(config)
	if err != nil {
		t.Errorf("RunNotificationHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is always true for Notification (cannot block)
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	// Verify hook executed and output was captured
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "Notification" {
		t.Errorf("HookEventName = %v, want Notification", output.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "Test Notification") {
		t.Errorf("AdditionalContext does not contain notification title, got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

// TestRunStopHooks tests the basic stop hook execution path
func TestRunStopHooks(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "Stop",
		"reason": "user_requested"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		Stop: []StopHook{
			{
				Actions: []Action{
					{Type: "output", Message: "Stop requested: {.reason}"},
				},
			},
		},
	}

	output, err := RunStopHooks(config)
	if err != nil {
		t.Errorf("RunStopHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is always true by default (no blocking decision)
	if !output.Continue {
		t.Errorf("Continue = false, want true (default for Stop)")
	}

	// Verify no decision means stop proceeds
	if output.Decision != "" {
		t.Errorf("Decision = %q, want empty (allow stop)", output.Decision)
	}
}

// TestRunSubagentStopHooks tests the basic subagent stop hook execution path
func TestRunSubagentStopHooks(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "SubagentStop",
		"agent_id": "agent-123",
		"agent_type": "Explore",
		"reason": "task_complete"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		SubagentStop: []SubagentStopHook{
			{
				Actions: []Action{
					{Type: "output", Message: "Subagent stopped: {.agent_type}"},
				},
			},
		},
	}

	output, err := RunSubagentStopHooks(config)
	if err != nil {
		t.Errorf("RunSubagentStopHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is always true by default (no blocking decision)
	if !output.Continue {
		t.Errorf("Continue = false, want true (default for SubagentStop)")
	}

	// Verify no decision means stop proceeds
	if output.Decision != "" {
		t.Errorf("Decision = %q, want empty (allow stop)", output.Decision)
	}
}

// TestRunPreCompactHooks tests the basic precompact hook execution path
func TestRunPreCompactHooks(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreCompact",
		"trigger": "manual"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		PreCompact: []PreCompactHook{
			{
				Matcher: "manual",
				Actions: []Action{
					{Type: "output", Message: "Manual compaction triggered"},
				},
			},
		},
	}

	output, err := RunPreCompactHooks(config)
	if err != nil {
		t.Errorf("RunPreCompactHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is always true (PreCompact cannot block)
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	// Verify systemMessage contains output from action
	if !strings.Contains(output.SystemMessage, "Manual compaction triggered") {
		t.Errorf("SystemMessage does not contain expected message, got: %s", output.SystemMessage)
	}
}

// TestRunSessionEndHooks tests the basic session end hook execution path
func TestRunSessionEndHooks(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "SessionEnd",
		"reason": "clear"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{
					{Type: ConditionReasonIs, Value: "clear"},
				},
				Actions: []Action{
					{Type: "output", Message: "Session cleared"},
				},
			},
		},
	}

	output, err := RunSessionEndHooks(config)
	if err != nil {
		t.Errorf("RunSessionEndHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is always true (SessionEnd cannot block)
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	// Verify systemMessage contains output from action
	if !strings.Contains(output.SystemMessage, "Session cleared") {
		t.Errorf("SystemMessage does not contain expected message, got: %s", output.SystemMessage)
	}
}

// TestRunNotificationHooks_EmptyConfig tests no hooks match case
func TestRunNotificationHooks_EmptyConfig(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "Notification",
		"title": "Test",
		"notification_type": "idle_prompt"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		Notification: []NotificationHook{},
	}

	output, err := RunNotificationHooks(config)
	if err != nil {
		t.Errorf("RunNotificationHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is true even with no matching hooks
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}
}

// TestRunStopHooks_BlockingDecision tests blocking decision in Stop hook
func TestRunStopHooks_BlockingDecision(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "Stop",
		"reason": "user_requested"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	blockDecision := "block"
	blockReason := "Important work in progress"
	config := &Config{
		Stop: []StopHook{
			{
				Actions: []Action{
					{
						Type:     "output",
						Message:  "Blocking stop",
						Decision: &blockDecision,
						Reason:   &blockReason,
					},
				},
			},
		},
	}

	output, err := RunStopHooks(config)

	if err != nil {
		t.Errorf("RunStopHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify decision is "block"
	if output.Decision != "block" {
		t.Errorf("Decision = %q, want 'block'", output.Decision)
	}

	// Verify reason is set
	if output.Reason != "Important work in progress" {
		t.Errorf("Reason = %q, want 'Important work in progress'", output.Reason)
	}
}

// TestRunSessionEndHooks_NoConditionMatch tests no conditions match case
func TestRunSessionEndHooks_NoConditionMatch(t *testing.T) {
	// Backup and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Setup stdin with valid JSON
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/test",
		"hook_event_name": "SessionEnd",
		"reason": "logout"
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{
					{Type: ConditionReasonIs, Value: "clear"},
				},
				Actions: []Action{
					{Type: "output", Message: "Session cleared"},
				},
			},
		},
	}

	output, err := RunSessionEndHooks(config)
	if err != nil {
		t.Errorf("RunSessionEndHooks() error = %v", err)
	}

	if output == nil {
		t.Fatal("Expected non-nil output")
	}

	// Verify continue is true even with no matching hooks
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	// Verify systemMessage is empty (no hooks executed)
	if output.SystemMessage != "" {
		t.Errorf("SystemMessage = %q, want empty", output.SystemMessage)
	}
}
