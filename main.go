package main

import (
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
		err = runHooks(config, HookEventType(*eventType))
	case "dry-run":
		err = dryRunHooks(config, HookEventType(*eventType))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}

	// ExitError の場合は特別な処理
	if err != nil {
		if exitErr, ok := err.(*ExitError); ok {
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
