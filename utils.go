package main

import (
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
	"mvdan.cc/sh/v3/shell"
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
// isGitTracked checks if a file is tracked by Git
// findGitRepository finds the Git repository containing the given path
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

// プロンプト関連の条件チェック関数
// countUserPromptsFromTranscript はtranscriptファイルから指定セッションのユーザープロンプトをカウントする
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
				// isMetaがtrueの場合は除外（コマンド実行など）
				if isMeta, exists := entry["isMeta"].(bool); exists && isMeta {
					continue
				}
				count++
			}
		}
	}

	// 現在の発話を含める
	return count + 1, nil
}

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
