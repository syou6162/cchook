package main

import "errors"

// stubRunnerWithOutput is a test stub that implements CommandRunner for testing executor actions.
type stubRunnerWithOutput struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func (s *stubRunnerWithOutput) RunCommand(cmd string, useStdin bool, data any) error {
	return s.err
}

func (s *stubRunnerWithOutput) RunCommandWithOutput(cmd string, useStdin bool, data any) (stdout, stderr string, exitCode int, err error) {
	return s.stdout, s.stderr, s.exitCode, s.err
}

// stubRunnerWithMultipleOutputs は複数のコマンド実行に対して順番に異なる出力を返すstub
type stubRunnerWithMultipleOutputs struct {
	outputs []string
	index   int
}

func (s *stubRunnerWithMultipleOutputs) RunCommand(cmd string, useStdin bool, data any) error {
	return nil
}

func (s *stubRunnerWithMultipleOutputs) RunCommandWithOutput(cmd string, useStdin bool, data any) (stdout, stderr string, exitCode int, err error) {
	if s.index >= len(s.outputs) {
		return "", "", 1, errors.New("no more outputs configured")
	}
	output := s.outputs[s.index]
	s.index++
	return output, "", 0, nil
}

// Helper function to create *bool
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create *string
func stringPtr(s string) *string {
	return &s
}

// Helper function to create *int
func intPtr(i int) *int {
	return &i
}
