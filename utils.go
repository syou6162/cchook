package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// parseInput関数は parser.go に移動

// ErrConditionNotHandled は、条件タイプがその関数では処理されないことを示す
var ErrConditionNotHandled = errors.New("condition not handled by this function")

// 共通マッチャーチェック関数
func checkMatcher(matcher string, toolName string) bool {
	if matcher == "" {
		return true
	}

	for _, pattern := range strings.Split(matcher, "|") {
		if strings.Contains(toolName, strings.TrimSpace(pattern)) {
			return true
		}
	}
	return false
}

// fileExistsRecursive は指定されたファイル名がディレクトリツリー内に存在するかを再帰的に検索する
func fileExistsRecursive(filename string) bool {
	if filename == "" {
		return false
	}

	found := false
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーがあっても続ける
		}
		if !info.IsDir() && filepath.Base(path) == filename {
			found = true
			return filepath.SkipAll // 見つかったら探索を終了
		}
		return nil
	})
	if err != nil {
		return false
	}
	return found
}

// fileExists は指定されたパスにファイルが存在するかをチェックする
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func checkPreToolUseCondition(condition Condition, input *PreToolUseInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// ツール固有の条件をチェック
	matched, err = checkToolCondition(condition, &input.ToolInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// どの関数も処理しなかった場合はエラー
	return false, fmt.Errorf("unknown condition type: %s", condition.Type)
}

func checkPostToolUseCondition(condition Condition, input *PostToolUseInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// ツール固有の条件をチェック
	matched, err = checkToolCondition(condition, &input.ToolInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// どの関数も処理しなかった場合はエラー
	return false, fmt.Errorf("unknown condition type: %s", condition.Type)
}

func checkUserPromptSubmitCondition(condition Condition, input *UserPromptSubmitInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// プロンプト固有の条件をチェック
	matched, err = checkPromptCondition(condition, input.Prompt)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// どの関数も処理しなかった場合はエラー
	return false, fmt.Errorf("unknown condition type: %s", condition.Type)
}

func checkSessionStartCondition(condition Condition, input *SessionStartInput) (bool, error) {
	// SessionStartは汎用条件のみ使用
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SessionStartがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SessionStart: %s", condition.Type)
}

// 汎用条件チェック関数
func checkCommonCondition(condition Condition) (bool, error) {
	switch condition.Type {
	case ConditionFileExists:
		// 指定ファイルが存在する
		return fileExists(condition.Value), nil
	case ConditionFileExistsRecursive:
		// ファイルが再帰的に存在するか
		return fileExistsRecursive(condition.Value), nil
	default:
		// この関数では汎用条件のみをチェック
		// 処理できない条件タイプの場合はErrConditionNotHandledを返す
		return false, ErrConditionNotHandled
	}
}

// ツール関連の条件チェック関数
func checkToolCondition(condition Condition, toolInput *ToolInput) (bool, error) {
	switch condition.Type {
	case ConditionFileExtension:
		// ToolInput構造体から直接FilePath取得
		if toolInput.FilePath != "" {
			return strings.HasSuffix(toolInput.FilePath, condition.Value), nil
		}
		return false, nil
	case ConditionCommandContains:
		// ToolInput構造体からCommand取得
		if toolInput.Command != "" {
			return strings.Contains(toolInput.Command, condition.Value), nil
		}
		return false, nil
	case ConditionCommandStartsWith:
		// コマンドが指定文字列で始まる
		if toolInput.Command != "" {
			return strings.HasPrefix(toolInput.Command, condition.Value), nil
		}
		return false, nil
	case ConditionURLStartsWith:
		// URLが指定文字列で始まる
		if toolInput.URL != "" {
			return strings.HasPrefix(toolInput.URL, condition.Value), nil
		}
		return false, nil
	default:
		// この関数ではツール関連条件のみをチェック
		return false, ErrConditionNotHandled
	}
}

// プロンプト関連の条件チェック関数
func checkPromptCondition(condition Condition, prompt string) (bool, error) {
	switch condition.Type {
	case ConditionPromptContains:
		// プロンプトに指定文字列が含まれる
		return strings.Contains(prompt, condition.Value), nil
	case ConditionPromptStartsWith:
		// プロンプトが指定文字列で始まる
		return strings.HasPrefix(prompt, condition.Value), nil
	case ConditionPromptEndsWith:
		// プロンプトが指定文字列で終わる
		return strings.HasSuffix(prompt, condition.Value), nil
	default:
		// この関数ではプロンプト関連条件のみをチェック
		return false, ErrConditionNotHandled
	}
}

// Notification用の条件チェック（汎用条件のみ）
func checkNotificationCondition(condition Condition, input *NotificationInput) (bool, error) {
	// Notificationは汎用条件のみ使用
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// Notificationがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for Notification: %s", condition.Type)
}

// Stop用の条件チェック（汎用条件のみ）
func checkStopCondition(condition Condition, input *StopInput) (bool, error) {
	// Stopは汎用条件のみ使用
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// Stopがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for Stop: %s", condition.Type)
}

// SubagentStop用の条件チェック（汎用条件のみ）
func checkSubagentStopCondition(condition Condition, input *SubagentStopInput) (bool, error) {
	// SubagentStopは汎用条件のみ使用
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SubagentStopがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SubagentStop: %s", condition.Type)
}

// PreCompact用の条件チェック（汎用条件のみ）
func checkPreCompactCondition(condition Condition, input *PreCompactInput) (bool, error) {
	// PreCompactは汎用条件のみ使用
	matched, err := checkCommonCondition(condition)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// PreCompactがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for PreCompact: %s", condition.Type)
}

func runCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// シェル経由でコマンドを実行
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
