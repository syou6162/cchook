package main

func executeNotificationAction(action NotificationAction, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		message := unifiedTemplateReplace(action.Message, rawJSON)
		fmt.Println(message)
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
		fmt.Println(message)
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
		fmt.Println(message)
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
		fmt.Println(message)
	}
	return nil
}
