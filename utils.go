package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// parseInput関数は parser.go に移動

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

// checkNotificationMatcher checks if the notification_type matches the matcher pattern.
// Unlike checkMatcher, this uses exact matching instead of partial matching
// to prevent "idle" from matching "idle_prompt".
// Supports pipe-separated patterns for OR logic.
func checkNotificationMatcher(matcher string, notificationType string) bool {
	if matcher == "" {
		return true
	}

	for _, pattern := range strings.Split(matcher, "|") {
		if strings.TrimSpace(pattern) == notificationType {
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

// realCommandRunner is the production implementation of CommandRunner.
type realCommandRunner struct{}

// RunCommand implements CommandRunner.RunCommand
func (r *realCommandRunner) RunCommand(cmd string, useStdin bool, data any) error {
	return runCommand(cmd, useStdin, data)
}

// RunCommandWithOutput implements CommandRunner.RunCommandWithOutput
func (r *realCommandRunner) RunCommandWithOutput(cmd string, useStdin bool, data any) (stdout, stderr string, exitCode int, err error) {
	return runCommandWithOutput(cmd, useStdin, data)
}

// DefaultCommandRunner is the default implementation used in production.
var DefaultCommandRunner CommandRunner = &realCommandRunner{}

// runCommand executes a shell command with optional JSON data passed via stdin.
func runCommand(command string, useStdin bool, data any) error {
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
func runCommandWithOutput(command string, useStdin bool, data any) (stdout string, stderr string, exitCode int, err error) {
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
