package main

func executeNotificationAction(action NotificationAction, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "structured_output":
		if err := executeStructuredOutput(action, Notification); err != nil {
			return err
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
	case "structured_output":
		if err := executeStructuredOutput(action, Stop); err != nil {
			return err
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
	case "structured_output":
		if err := executeStructuredOutput(action, SubagentStop); err != nil {
			return err
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
	case "structured_output":
		if err := executeStructuredOutput(action, PreCompact); err != nil {
			return err
		}
	}
	return nil
}
