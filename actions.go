package main

import (
	"fmt"
)

// handleOutput processes an output action and returns ExitError if a non-zero exit status is specified.
// Exit status 2 outputs to stderr, while other non-zero statuses output to stdout with error.
func handleOutput(message string, exitStatus *int, rawJSON interface{}) error {
	processedMessage := unifiedTemplateReplace(message, rawJSON)
	status := getExitStatus(exitStatus, "output")
	if status != 0 {
		// 0以外のExitStatusはすべてExitErrorとして返す
		stderr := status == 2 // 2の場合のみstderrに出力
		return NewExitError(status, processedMessage, stderr)
	}
	fmt.Println(processedMessage)
	return nil
}

// getExitStatus returns the exit status for the given action type.
// Default for "output" actions is 2, others default to 0.
func getExitStatus(exitStatus *int, actionType string) int {
	if exitStatus != nil {
		return *exitStatus
	}
	if actionType == "output" {
		return 2
	}
	return 0
}
