package main

// ExitError は特定の終了ステータスでプログラムを終了したいことを示すエラー型
type ExitError struct {
	Code    int
	Message string
	Stderr  bool // true なら stderr に出力
}

// Error returns the error message of ExitError.
func (e *ExitError) Error() string {
	return e.Message
}

// NewExitError creates a new ExitError with the given exit code, message, and stderr flag.
func NewExitError(code int, message string, stderr bool) *ExitError {
	return &ExitError{
		Code:    code,
		Message: message,
		Stderr:  stderr,
	}
}
