package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ErrConditionNotHandled indicates that a condition type is not handled by the checking function.
var ErrConditionNotHandled = errors.New("condition not handled by this function")

// checkPreToolUseCondition checks if a condition matches for PreToolUse events.
// Returns ErrConditionNotHandled if the condition type is not applicable to this event.
func checkPreToolUseCondition(condition Condition, input *PreToolUseInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition, &input.BaseInput)
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

// checkPostToolUseCondition checks if a condition matches for PostToolUse events.
// Returns ErrConditionNotHandled if the condition type is not applicable to this event.
func checkPostToolUseCondition(condition Condition, input *PostToolUseInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition, &input.BaseInput)
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

// checkUserPromptSubmitCondition checks if a condition matches for UserPromptSubmit events.
// Supports prompt_regex and every_n_prompts conditions in addition to common conditions.
func checkUserPromptSubmitCondition(condition Condition, input *UserPromptSubmitInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition, &input.BaseInput)
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

	// every_n_prompts条件をチェック
	if condition.Type == ConditionEveryNPrompts {
		count, err := countUserPromptsFromTranscript(input.TranscriptPath, input.SessionID)
		if err != nil {
			return false, fmt.Errorf("failed to count prompts: %w", err)
		}

		n, err := strconv.Atoi(condition.Value)
		if err != nil {
			return false, fmt.Errorf("invalid value for every_n_prompts: %w", err)
		}
		if n <= 0 {
			return false, fmt.Errorf("every_n_prompts value must be positive: %d", n)
		}

		// n回ごとにtrue（1回目、n+1回目、2n+1回目...）
		return count%n == 0, nil
	}

	// どの関数も処理しなかった場合はエラー
	return false, fmt.Errorf("unknown condition type: %s", condition.Type)
}

// checkSessionStartCondition checks if a condition matches for SessionStart events.
// Only supports common conditions.
func checkSessionStartCondition(condition Condition, input *SessionStartInput) (bool, error) {
	// SessionStartは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SessionStartがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SessionStart: %s", condition.Type)
}

// checkCommonCondition checks common conditions that are applicable to all event types.
// Includes file/directory existence checks and working directory conditions.
func checkCommonCondition(condition Condition, baseInput *BaseInput) (bool, error) {
	switch condition.Type {
	case ConditionFileExists:
		// 指定ファイルが存在する
		return fileExists(condition.Value), nil
	case ConditionFileExistsRecursive:
		// ファイルが再帰的に存在するか
		return fileExistsRecursive(condition.Value), nil
	case ConditionFileNotExists:
		// 指定ファイルが存在しない
		return !fileExists(condition.Value), nil
	case ConditionFileNotExistsRecursive:
		// ファイルが再帰的に存在しない
		return !fileExistsRecursive(condition.Value), nil
	case ConditionDirExists:
		// 指定ディレクトリが存在する
		return dirExists(condition.Value), nil
	case ConditionDirExistsRecursive:
		// ディレクトリが再帰的に存在するか
		return dirExistsRecursive(condition.Value), nil
	case ConditionDirNotExists:
		// 指定ディレクトリが存在しない
		return !dirExists(condition.Value), nil
	case ConditionDirNotExistsRecursive:
		// ディレクトリが再帰的に存在しない
		return !dirExistsRecursive(condition.Value), nil
	case ConditionCwdIs:
		// cwdが完全一致
		return baseInput.Cwd == condition.Value, nil
	case ConditionCwdIsNot:
		// cwdが完全一致しない
		return baseInput.Cwd != condition.Value, nil
	case ConditionCwdContains:
		// cwdが特定の文字列を含む
		return strings.Contains(baseInput.Cwd, condition.Value), nil
	case ConditionCwdNotContains:
		// cwdが特定の文字列を含まない
		return !strings.Contains(baseInput.Cwd, condition.Value), nil
	case ConditionPermissionModeIs:
		// permission_modeが完全一致
		return baseInput.PermissionMode == condition.Value, nil
	default:
		// この関数では汎用条件のみをチェック
		// 処理できない条件タイプの場合はErrConditionNotHandledを返す
		return false, ErrConditionNotHandled
	}
}

// checkToolCondition checks tool-specific conditions like file_extension, command_contains, and url_starts_with.
// Returns ErrConditionNotHandled if the condition type is not a tool condition.
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
	case ConditionGitTrackedFileOperation:
		// Git管理ファイルに対する操作をチェック
		if toolInput.Command != "" {
			return checkGitTrackedFileOperation(toolInput.Command, condition.Value)
		}
		return false, nil
	default:
		// この関数ではツール関連条件のみをチェック
		return false, ErrConditionNotHandled
	}
}

// countUserPromptsFromTranscript counts user prompts in the transcript file for the specified session.
// Returns the count including the current prompt.
func countUserPromptsFromTranscript(transcriptPath, sessionID string) (int, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open transcript: %w", err)
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	count := 0

	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			// JSONのパースエラーは継続（壊れた行をスキップ）
			continue
		}

		// type: "user" かつ同じセッションIDのメッセージをカウント
		if entryType, ok := entry["type"].(string); ok && entryType == "user" {
			if sid, ok := entry["sessionId"].(string); ok && sid == sessionID {
				count++
			}
		}
	}

	// 現在の発話を含める
	return count + 1, nil
}

// checkPromptCondition checks prompt-specific conditions like prompt_regex.
// Returns ErrConditionNotHandled if the condition type is not a prompt condition.
func checkPromptCondition(condition Condition, prompt string) (bool, error) {
	switch condition.Type {
	case ConditionPromptRegex:
		// プロンプトが正規表現パターンにマッチする
		// 例: "keyword" (部分一致), "^prefix" (前方一致), "suffix$" (後方一致), "a|b|c" (OR条件)
		re, err := regexp.Compile(condition.Value)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.MatchString(prompt), nil
	default:
		// この関数ではプロンプト関連条件のみをチェック
		return false, ErrConditionNotHandled
	}
}

// checkNotificationCondition checks if a condition matches for Notification events.
// Only supports common conditions.
func checkNotificationCondition(condition Condition, input *NotificationInput) (bool, error) {
	// Notificationは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// Notificationがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for Notification: %s", condition.Type)
}

// checkStopCondition checks if a condition matches for Stop events.
// Only supports common conditions.
func checkStopCondition(condition Condition, input *StopInput) (bool, error) {
	// Stopは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// Stopがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for Stop: %s", condition.Type)
}

// checkSubagentStopCondition checks if a condition matches for SubagentStop events.
// Only supports common conditions.
func checkSubagentStopCondition(condition Condition, input *SubagentStopInput) (bool, error) {
	// SubagentStopは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SubagentStopがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SubagentStop: %s", condition.Type)
}

// checkSubagentStartCondition checks if a condition matches for SubagentStart events.
// Only supports common conditions.
func checkSubagentStartCondition(condition Condition, input *SubagentStartInput) (bool, error) {
	// SubagentStartは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SubagentStartがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SubagentStart: %s", condition.Type)
}

// checkSessionEndCondition checks if a condition matches for SessionEnd events.
// Supports common conditions and reason_is condition.
func checkSessionEndCondition(condition Condition, input *SessionEndInput) (bool, error) {
	// reason_is condition
	if condition.Type == ConditionReasonIs {
		return input.Reason == condition.Value, nil
	}

	// SessionEndは汎用条件も使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// SessionEndがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for SessionEnd: %s", condition.Type)
}

// checkPreCompactCondition checks if a condition matches for PreCompact events.
// Only supports common conditions.
func checkPreCompactCondition(condition Condition, input *PreCompactInput) (bool, error) {
	// PreCompactは汎用条件のみ使用
	matched, err := checkCommonCondition(condition, &input.BaseInput)
	if err == nil {
		return matched, nil // 処理された
	}
	if !errors.Is(err, ErrConditionNotHandled) {
		return false, err // 本当のエラー
	}

	// PreCompactがサポートしない条件タイプの場合はエラー
	return false, fmt.Errorf("unknown condition type for PreCompact: %s", condition.Type)
}

// checkPermissionRequestCondition checks if a condition matches for PermissionRequest events.
// Returns ErrConditionNotHandled if the condition type is not applicable to this event.
func checkPermissionRequestCondition(condition Condition, input *PermissionRequestInput) (bool, error) {
	// まず汎用条件をチェック
	matched, err := checkCommonCondition(condition, &input.BaseInput)
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
