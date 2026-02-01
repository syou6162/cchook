package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// parseInput関数は parser.go に移動

// ErrConditionNotHandled indicates that a condition type is not handled by the checking function.
var ErrConditionNotHandled = errors.New("condition not handled by this function")

// checkMatcher checks if the tool name matches the matcher pattern.
// Supports pipe-separated patterns with partial matching.
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

// existsRecursive recursively searches for a file or directory by name.
// If isDir is true, it searches for directories; otherwise, it searches for files.
func existsRecursive(name string, isDir bool) bool {
	if name == "" {
		return false
	}

	found := false
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーがあっても続ける
		}
		if info.IsDir() == isDir && filepath.Base(path) == name {
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

// fileExistsRecursive recursively searches for a file by name in the directory tree.
func fileExistsRecursive(filename string) bool {
	return existsRecursive(filename, false)
}

// fileExists checks if a file exists at the specified path.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// dirExists checks if a directory exists at the specified path.
func dirExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// dirExistsRecursive recursively searches for a directory by name in the directory tree.
func dirExistsRecursive(dirname string) bool {
	return existsRecursive(dirname, true)
}

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
	default:
		// この関数では汎用条件のみをチェック
		// 処理できない条件タイプの場合はErrConditionNotHandledを返す
		return false, ErrConditionNotHandled
	}
}

// findGitRepository finds the Git repository containing the given path.
func findGitRepository(path string) (*git.Repository, error) {
	dir := filepath.Dir(path)
	for {
		repo, err := git.PlainOpen(dir)
		if err == nil {
			return repo, nil
		}

		// 親ディレクトリへ
		parent := filepath.Dir(dir)
		if parent == dir {
			// ルートディレクトリに到達
			return nil, fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

// isFileTrackedInRepo checks if a file is tracked in the given Git repository
func isFileTrackedInRepo(repo *git.Repository, absPath string) (bool, error) {
	// リポジトリのルートディレクトリを取得
	wt, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	// リポジトリルートからの相対パスを計算
	relPath, err := filepath.Rel(wt.Filesystem.Root(), absPath)
	if err != nil {
		return false, err
	}

	// Gitのindexは常にスラッシュを使う
	relPath = filepath.ToSlash(relPath)

	// インデックスをチェック
	idx, err := repo.Storer.Index()
	if err != nil {
		return false, err
	}

	_, err = idx.Entry(relPath)
	return err == nil, nil
}

// isGitTracked checks if a file is tracked by Git.
func isGitTracked(filePath string) (bool, error) {
	// 絶対パスに変換
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false, err
	}

	// リポジトリを探す
	repo, err := findGitRepository(absPath)
	if err != nil {
		// Gitリポジトリではない
		return false, nil
	}

	// ファイルがトラックされているかチェック
	return isFileTrackedInRepo(repo, absPath)
}

// checkGitTrackedFileOperation checks if a command is trying to operate on Git-tracked files
func checkGitTrackedFileOperation(command string, blockedOps string) (bool, error) {
	if command == "" {
		return false, nil
	}

	// プロセス置換 <() や >() を事前チェック
	// shell.Fieldsはプロセス置換を展開できずpanicするため
	if containsProcessSubstitution(command) {
		return false, ErrProcessSubstitutionDetected
	}

	// コマンドラインをパース（環境変数展開あり）
	args, err := shell.Fields(command, nil)
	if err != nil {
		// パースエラーの場合は条件にマッチしないとする
		return false, nil
	}

	if len(args) == 0 {
		return false, nil
	}

	// ブロック対象のコマンドリストを作成
	blockedOpsList := strings.Split(blockedOps, "|")
	cmdName := args[0]

	// コマンドがブロック対象かチェック
	isBlockedCmd := false
	for _, op := range blockedOpsList {
		if cmdName == strings.TrimSpace(op) {
			isBlockedCmd = true
			break
		}
	}

	if !isBlockedCmd {
		return false, nil
	}

	// rmとmvのオプションを解析してファイルパスを抽出
	var filePaths []string
	skipNext := false

	for i := 1; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}

		arg := args[i]

		// オプションの処理
		if strings.HasPrefix(arg, "-") {
			// 一部のオプションは次の引数を取る
			if cmdName == "mv" && (arg == "-t" || arg == "--target-directory") {
				skipNext = true
			}
			continue
		}

		// mvコマンドの場合、最後の引数は移動先なので除外
		if cmdName == "mv" && i == len(args)-1 && len(filePaths) > 0 {
			continue
		}

		// ファイルパスとして扱う
		filePaths = append(filePaths, arg)
	}

	// 各ファイルがGit管理下にあるかチェック
	for _, filePath := range filePaths {
		tracked, err := isGitTracked(filePath)
		if err != nil {
			// エラーは無視して続行
			continue
		}
		if tracked {
			// 1つでもGit管理下のファイルがあれば条件にマッチ
			return true, nil
		}
	}

	return false, nil
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
	defer file.Close()

	decoder := json.NewDecoder(file)
	count := 0

	for {
		var entry map[string]interface{}
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

// realCommandRunner is the production implementation of CommandRunner.
type realCommandRunner struct{}

// RunCommand implements CommandRunner.RunCommand
func (r *realCommandRunner) RunCommand(cmd string, useStdin bool, data interface{}) error {
	return runCommand(cmd, useStdin, data)
}

// RunCommandWithOutput implements CommandRunner.RunCommandWithOutput
func (r *realCommandRunner) RunCommandWithOutput(cmd string, useStdin bool, data interface{}) (stdout, stderr string, exitCode int, err error) {
	return runCommandWithOutput(cmd, useStdin, data)
}

// DefaultCommandRunner is the default implementation used in production.
var DefaultCommandRunner CommandRunner = &realCommandRunner{}

// runCommand executes a shell command with optional JSON data passed via stdin.
func runCommand(command string, useStdin bool, data interface{}) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// シェル経由でコマンドを実行
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// useStdinがtrueの場合、dataをJSON形式でstdinに渡す
	if useStdin && data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for stdin: %w", err)
		}
		cmd.Stdin = bytes.NewReader(jsonData)
	}

	return cmd.Run()
}

// runCommandWithOutput executes a command and captures stdout, stderr, and exit code
func runCommandWithOutput(command string, useStdin bool, data interface{}) (stdout string, stderr string, exitCode int, err error) {
	if strings.TrimSpace(command) == "" {
		return "", "", 1, fmt.Errorf("empty command")
	}

	// シェル経由でコマンドを実行
	cmd := exec.Command("sh", "-c", command)

	// stdout/stderrをキャプチャするためのバッファ
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// useStdinがtrueの場合、dataをJSON形式でstdinに渡す
	if useStdin && data != nil {
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return "", "", 1, fmt.Errorf("failed to marshal JSON for stdin: %w", marshalErr)
		}
		cmd.Stdin = bytes.NewReader(jsonData)
	}

	// コマンド実行
	runErr := cmd.Run()

	// 出力を文字列として取得
	stdout = outBuf.String()
	stderr = errBuf.String()

	// 終了コードを抽出
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// exec.ExitError以外のエラー（コマンドが見つからない等）
			exitCode = 1
		}
		err = runErr
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode, err
}

// validateSessionStartOutput validates SessionStartOutput JSON against auto-generated schema
func validateSessionStartOutput(jsonData []byte) error {
	// Generate schema from SessionStartOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&SessionStartOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: only hookSpecificOutput (continue is always output but not required for validation)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: hookEventName must be "SessionStart"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []interface{}{"SessionStart"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

func validateUserPromptSubmitOutput(jsonData []byte) error {
	// Generate schema from UserPromptSubmitOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&UserPromptSubmitOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level (requirement: additionalProperties: true)
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. Set required fields: hookSpecificOutput only (decision is optional)
	schema.Required = []string{"hookSpecificOutput"}

	// 3. Add custom validation: decision must be "block" only (or omitted entirely)
	if decisionProp, ok := schema.Properties.Get("decision"); ok {
		if decision := decisionProp; decision != nil {
			decision.Enum = []interface{}{"block"}
		}
	}

	// 4. Add custom validation: hookEventName must be "UserPromptSubmit"
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []interface{}{"UserPromptSubmit"}
				}
			}
			// hookSpecificOutput.hookEventName is required
			hookSpecific.Required = []string{"hookEventName"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, validationErr := range result.Errors() {
			errMsgs = append(errMsgs, validationErr.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// validatePreToolUseOutput validates PreToolUseOutput against JSON schema (Phase 3)
func validatePreToolUseOutput(jsonData []byte) error {
	// Generate schema from PreToolUseOutput struct
	reflector := jsonschema.Reflector{
		DoNotReference: true, // Inline all definitions
	}
	schema := reflector.Reflect(&PreToolUseOutput{})

	// Customize schema to match requirements
	// 1. Allow additional properties at root level
	schema.AdditionalProperties = nil // nil means allow any additional properties

	// 2. No required fields at root level
	// - continue: optional (defaults to true per Claude Code spec)
	// - hookSpecificOutput: optional (omitempty tag, omit to delegate)
	schema.Required = nil

	// 3. Configure hookSpecificOutput
	if hookSpecificProp, ok := schema.Properties.Get("hookSpecificOutput"); ok {
		if hookSpecific := hookSpecificProp; hookSpecific != nil {
			// hookEventName must be "PreToolUse"
			if hookEventNameProp, ok := hookSpecific.Properties.Get("hookEventName"); ok {
				if hookEventName := hookEventNameProp; hookEventName != nil {
					hookEventName.Enum = []interface{}{"PreToolUse"}
				}
			}
			// permissionDecision must be "allow", "deny", or "ask"
			if permissionDecisionProp, ok := hookSpecific.Properties.Get("permissionDecision"); ok {
				if permissionDecision := permissionDecisionProp; permissionDecision != nil {
					permissionDecision.Enum = []interface{}{"allow", "deny", "ask"}
				}
			}
			// hookSpecificOutput.hookEventName and permissionDecision are required
			hookSpecific.Required = []string{"hookEventName", "permissionDecision"}
			// hookSpecificOutput should not allow additional properties
			hookSpecific.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}} // false
		}
	}

	// Convert schema to map for gojsonschema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	schemaLoader := gojsonschema.NewGoLoader(schemaMap)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, desc.String())
		}
		return fmt.Errorf("JSON schema validation failed: %s", errMsgs)
	}

	return nil
}

// containsProcessSubstitution checks if the command contains process substitution (<() or >()).
// Uses syntax.NewParser() to parse the command and syntax.Walk to detect *syntax.ProcSubst nodes.
// Falls back to string-level detection if parsing fails.
func containsProcessSubstitution(command string) bool {
	if command == "" {
		return false
	}

	// ASTベースでプロセス置換を検出
	p := syntax.NewParser()
	f, err := p.Parse(strings.NewReader(command), "")
	if err != nil {
		// パースエラーの場合は文字列レベルでフォールバック検出
		// 安全側に倒すため、疑わしい場合はtrueを返す
		return strings.Contains(command, "<(") || strings.Contains(command, ">(")
	}

	found := false
	syntax.Walk(f, func(node syntax.Node) bool {
		if found {
			return false // 既に見つかっていたら子ノードをスキップ
		}
		if _, ok := node.(*syntax.ProcSubst); ok {
			found = true
			return false // 子ノードをスキップ（注: 兄弟ノードの探索は継続される）
		}
		return true // 継続
	})

	return found
}
