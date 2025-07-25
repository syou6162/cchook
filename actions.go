package main

import (
	"fmt"
	"os"
)

func executeNotificationAction(action NotificationAction, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

func executeStopAction(action StopAction, input *StopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

func executeSubagentStopAction(action SubagentStopAction, input *SubagentStopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

func executePreCompactAction(action PreCompactAction, input *PreCompactInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

func executePreToolUseAction(action PreToolUseAction, input *PreToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

func executePostToolUseAction(action PostToolUseAction, input *PostToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		exitStatus := getExitStatus(action.ExitStatus, "output")
		if exitStatus == 2 {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			os.Exit(2)
		} else {
			fmt.Println(message)
		}
	}
	return nil
}

// getExitStatus returns the exit status for the given action type
// Default for "output" actions is 2, others default to 0
func getExitStatus(exitStatus *int, actionType string) int {
	if exitStatus != nil {
		return *exitStatus
	}
	if actionType == "output" {
		return 2
	}
	return 0
}
