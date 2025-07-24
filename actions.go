package main

import "fmt"

func executeNotificationAction(action NotificationAction, input *NotificationInput) error {
	switch action.Type {
	case "command":
		cmd := extendedSnakeCaseReplaceVariables(action.Command, input)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := extendedSnakeCaseReplaceVariables(action.Message, input)
		fmt.Println(message)
	}
	return nil
}

func executeStopAction(action StopAction, input *StopInput) error {
	switch action.Type {
	case "command":
		cmd := extendedSnakeCaseReplaceVariables(action.Command, input)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := extendedSnakeCaseReplaceVariables(action.Message, input)
		fmt.Println(message)
	}
	return nil
}

func executeSubagentStopAction(action SubagentStopAction, input *SubagentStopInput) error {
	switch action.Type {
	case "command":
		cmd := extendedSnakeCaseReplaceVariables(action.Command, input)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := extendedSnakeCaseReplaceVariables(action.Message, input)
		fmt.Println(message)
	}
	return nil
}

func executePreCompactAction(action PreCompactAction, input *PreCompactInput) error {
	switch action.Type {
	case "command":
		cmd := extendedSnakeCaseReplaceVariables(action.Command, input)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := extendedSnakeCaseReplaceVariables(action.Message, input)
		fmt.Println(message)
	}
	return nil
}