package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	command := flag.String("command", "run", "Command to execute (run, dry-run)")
	eventType := flag.String("event", "", "Event type for run/dry-run command")
	flag.Parse()

	if (*command == "run" || *command == "dry-run") && *eventType == "" {
		fmt.Fprintf(os.Stderr, "Error: event type is required for %s command\n", *command)
		os.Exit(1)
	}

	// イベントタイプの妥当性検証
	if *command == "run" || *command == "dry-run" {
		eventType := HookEventType(*eventType)
		if !eventType.IsValid() {
			fmt.Fprintf(os.Stderr, "Error: invalid event type '%s'. Valid types: PreToolUse, PostToolUse, Notification, Stop, SubagentStop, PreCompact, SessionStart, SessionEnd, UserPromptSubmit\n", string(eventType))
			os.Exit(1)
		}
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch *command {
	case "run":
		if HookEventType(*eventType) == SessionStart {
			// SessionStart special handling with JSON output
			output, err := RunSessionStartHooks(config)
			if err != nil {
				// Log error to stderr
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				// Ensure output has continue field and hookSpecificOutput even on error (requirement 1.4)
				if output == nil {
					output = &SessionStartOutput{
						Continue:      false,
						SystemMessage: fmt.Sprintf("Failed to process SessionStart: %v", err),
						HookSpecificOutput: &SessionStartHookSpecificOutput{
							HookEventName: "SessionStart",
						},
					}
				}
			}

			// Marshal JSON with 2-space indent
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				// Marshal failure should not be fatal - output minimal valid JSON and exit 0
				fmt.Fprintf(os.Stderr, "Warning: Error marshaling JSON: %v\n", err)
				// Fallback to minimal valid output
				fallbackOutput := SessionStartOutput{
					Continue: false,
					HookSpecificOutput: &SessionStartHookSpecificOutput{
						HookEventName: "SessionStart",
					},
					SystemMessage: fmt.Sprintf("Failed to marshal output: %v", err),
				}
				jsonBytes, _ = json.MarshalIndent(fallbackOutput, "", "  ")
			}

			// Validate final JSON output against schema (non-functional requirement)
			if err := validateSessionStartOutput(jsonBytes); err != nil {
				// Validation failure should not be fatal - log warning and continue
				fmt.Fprintf(os.Stderr, "Warning: Final JSON output validation failed: %v\n", err)
			}

			// Output JSON to stdout
			fmt.Println(string(jsonBytes))
			// Always exit 0 for SessionStart (continue field controls behavior)
			os.Exit(0)
		}

		if HookEventType(*eventType) == UserPromptSubmit {
			// UserPromptSubmit special handling with JSON output
			output, err := RunUserPromptSubmitHooks(config)
			if err != nil {
				// Log error to stderr
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				// Ensure output has decision field and hookSpecificOutput even on error
				if output == nil {
					output = &UserPromptSubmitOutput{
						Continue:      true,
						Decision:      "block",
						SystemMessage: fmt.Sprintf("Failed to process UserPromptSubmit: %v", err),
						HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
							HookEventName: "UserPromptSubmit",
						},
					}
				}
			}

			// Marshal JSON with 2-space indent
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				// Marshal failure should not be fatal - output minimal valid JSON and exit 0
				fmt.Fprintf(os.Stderr, "Warning: Error marshaling JSON: %v\n", err)
				// Fallback to minimal valid output
				fallbackOutput := UserPromptSubmitOutput{
					Continue: true,
					Decision: "block",
					HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
						HookEventName: "UserPromptSubmit",
					},
					SystemMessage: fmt.Sprintf("Failed to marshal output: %v", err),
				}
				jsonBytes, _ = json.MarshalIndent(fallbackOutput, "", "  ")
			}

			// Validate final JSON output against schema (non-functional requirement)
			if err := validateUserPromptSubmitOutput(jsonBytes); err != nil {
				// Validation failure should not be fatal - log warning and continue
				fmt.Fprintf(os.Stderr, "Warning: Final JSON output validation failed: %v\n", err)
			}

			// Output JSON to stdout
			fmt.Println(string(jsonBytes))
			// Always exit 0 for UserPromptSubmit (decision field controls behavior)
			os.Exit(0)
		}

		if HookEventType(*eventType) == PreToolUse {
			// PreToolUse special handling with JSON output
			output, err := RunPreToolUseHooks(config)
			if err != nil {
				// Log error to stderr
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				// Ensure output has hookSpecificOutput even on error
				if output == nil {
					output = &PreToolUseOutput{
						Continue:      true,
						SystemMessage: fmt.Sprintf("Failed to process PreToolUse: %v", err),
						HookSpecificOutput: &PreToolUseHookSpecificOutput{
							HookEventName:      "PreToolUse",
							PermissionDecision: "deny",
						},
					}
				}
			}

			// Marshal JSON with 2-space indent
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				// Marshal failure should not be fatal - output minimal valid JSON and exit 0
				fmt.Fprintf(os.Stderr, "Warning: Error marshaling JSON: %v\n", err)
				// Fallback to minimal valid output
				fallbackOutput := PreToolUseOutput{
					Continue: true,
					HookSpecificOutput: &PreToolUseHookSpecificOutput{
						HookEventName:      "PreToolUse",
						PermissionDecision: "deny",
					},
					SystemMessage: fmt.Sprintf("Failed to marshal output: %v", err),
				}
				jsonBytes, _ = json.MarshalIndent(fallbackOutput, "", "  ")
			}

			// Validate final JSON output against schema (non-functional requirement)
			if err := validatePreToolUseOutput(jsonBytes); err != nil {
				// Validation failure should not be fatal - log warning and continue
				fmt.Fprintf(os.Stderr, "Warning: Final JSON output validation failed: %v\n", err)
			}

			// Output JSON to stdout
			fmt.Println(string(jsonBytes))
			// Always exit 0 for PreToolUse (permissionDecision field controls behavior)
			os.Exit(0)
		}

		if HookEventType(*eventType) == Stop {
			// Stop special handling with JSON output
			output, err := RunStopHooks(config)
			if err != nil {
				// Log error to stderr
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				if output == nil {
					output = &StopOutput{
						Continue:      true,
						Decision:      "block",
						Reason:        fmt.Sprintf("Failed to process Stop: %v", err),
						SystemMessage: fmt.Sprintf("Failed to process Stop: %v", err),
					}
				} else {
					// fail-safe: エラー時はdecisionを"block"に強制
					output.Decision = "block"
					if output.Reason == "" {
						output.Reason = fmt.Sprintf("Failed to process Stop: %v", err)
					}
					errMsg := fmt.Sprintf("Failed to process Stop: %v", err)
					if output.SystemMessage != "" {
						output.SystemMessage += "\n" + errMsg
					} else {
						output.SystemMessage = errMsg
					}
				}
			}

			// Marshal JSON with 2-space indent
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				// Marshal failure should not be fatal - output minimal valid JSON and exit 0
				fmt.Fprintf(os.Stderr, "Warning: Error marshaling JSON: %v\n", err)
				fallbackOutput := StopOutput{
					Continue:      true,
					Decision:      "block",
					Reason:        fmt.Sprintf("Failed to marshal output: %v", err),
					SystemMessage: fmt.Sprintf("Failed to marshal output: %v", err),
				}
				jsonBytes, _ = json.MarshalIndent(fallbackOutput, "", "  ")
			}

			// Validate final JSON output against schema (non-functional requirement)
			if err := validateStopOutput(jsonBytes); err != nil {
				// Validation failure should not be fatal - log warning and continue
				fmt.Fprintf(os.Stderr, "Warning: Final JSON output validation failed: %v\n", err)
			}

			// Output JSON to stdout
			fmt.Println(string(jsonBytes))
			// Always exit 0 for Stop (decision field controls behavior)
			os.Exit(0)
		}

		if HookEventType(*eventType) == PermissionRequest {
			// PermissionRequest special handling with JSON output
			err := RunPermissionRequestHooks(config)
			// Always exit 0 (error handling is done inside RunPermissionRequestHooks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			os.Exit(0)
		}
		err = runHooks(config, HookEventType(*eventType))
	case "dry-run":
		err = dryRunHooks(config, HookEventType(*eventType))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}

	// ExitError の場合は特別な処理
	if err != nil {
		var exitErr *ExitError
		// errors.Joinでラップされた場合でもExitErrorを取り出せるようにerrors.Asを使用
		if errors.As(err, &exitErr) {
			// ExitError の場合は適切な出力先に出力して指定のコードで終了
			// err.Error()を使ってラップされた全メッセージを出力
			if exitErr.Stderr {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			} else {
				fmt.Println(err.Error())
			}
			os.Exit(exitErr.Code)
		} else {
			// 通常のエラーの場合
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
