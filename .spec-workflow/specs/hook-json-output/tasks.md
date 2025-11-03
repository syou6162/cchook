# Tasks: SessionStart Hook JSON出力対応（Phase 1）

## Task Overview

**このspecのスコープ**: Phase 1として、SessionStartフックのみをJSON出力形式に移行します。
Phase 1の成功後、Phase 2（UserPromptSubmit）、Phase 3（PreToolUse）と段階的に他のフックイベントへ拡張します。

SessionStartフックをexit statusベースからJSON出力形式に移行します。t_wada式TDDに従い、テストを先に書いてから実装を進めます。

---

- [x] 0. Action構造体へのContinueフィールド追加（types.go）
  - File: types.go
  - Action構造体に`Continue *bool`フィールドを追加
  - YAMLタグは`yaml:"continue,omitempty"`
  - Purpose: YAML設定でcontinueフィールドを明示的に指定可能にする
  - _Leverage: 既存のAction構造体、*int型のExitStatusパターン_
  - _Requirements: 要求2（YAML設定形式の更新）_
  - _Prompt: Role: Go Developer with expertise in YAML configuration and struct design | Task: Add Continue *bool field to Action struct in types.go following requirement 2 (YAML config format update). Add `Continue *bool` with yaml tag `yaml:"continue,omitempty"`. This allows users to explicitly specify `continue: true` or `continue: false` in YAML config. Use pointer type (*bool) to distinguish between unspecified (nil), explicit true, and explicit false. | Restrictions: Must not modify existing Action fields, use omitempty in yaml tag, follow existing field naming conventions (PascalCase), maintain backward compatibility (continue field is optional) | Success: Field compiles without errors, yaml tag is correct, *bool allows three-state logic (nil/true/false), existing YAML configs continue to work without continue field_
  - _Note: Codexレビュー指摘事項 - tasks.mdとdesign.mdで`action.Continue`を参照しているが、現在のAction構造体にはContinueフィールドが存在しないため、このタスクで追加する_

---

- [x] 1. SessionStartOutput構造体の定義（types.go）
  - File: types.go
  - SessionStartOutput, SessionStartHookSpecificOutput, ActionOutput構造体を追加
  - Claude Code仕様に準拠したJSONタグを設定
  - Purpose: JSON出力の型安全性を確保
  - _Leverage: 既存のBaseInput, SessionStartInput構造体パターン_
  - _Requirements: 要求1（JSON出力形式への移行）_
  - _Prompt: Role: Go Developer specializing in type systems and JSON serialization | Task: Define SessionStartOutput, SessionStartHookSpecificOutput, and ActionOutput structs in types.go following requirement 1 (JSON output format migration). Use json tags with omitempty where appropriate. SessionStartOutput contains all Claude Code common fields: Continue (bool), StopReason (string, omitempty), SuppressOutput (bool, omitempty), SystemMessage (string, omitempty), HookSpecificOutput (*SessionStartHookSpecificOutput, omitempty). SessionStartHookSpecificOutput contains HookEventName (string) and AdditionalContext (string, omitempty). ActionOutput is an internal type with Continue (bool), HookEventName (string), AdditionalContext (string), SystemMessage (string), StopReason (string), SuppressOutput (bool). | Restrictions: Must follow Claude Code JSON specification exactly including all common fields (continue, stopReason, suppressOutput, systemMessage), use pointer for HookSpecificOutput to support omitempty, ensure HookEventName is always "SessionStart" when set, maintain consistency with existing type naming conventions | Success: All structs compile without errors, JSON tags are correct with omitempty, structs match Claude Code specification exactly including all common fields, type names follow Go conventions_

---

- [x] 2. SessionStartOutput構造体のテスト作成（types_test.go）
  - File: types_test.go
  - SessionStartOutputのJSONシリアライズ/デシリアライズテストを追加
  - omitemptyフィールドの動作確認テストを追加
  - Purpose: JSON出力構造の正しさを検証
  - _Leverage: 既存のテストパターン（types_test.go）_
  - _Requirements: 要求1.3（有効なJSONシリアライズ）_
  - _Prompt: Role: Go Test Engineer with expertise in JSON serialization testing | Task: Create comprehensive tests for SessionStartOutput JSON serialization/deserialization in types_test.go following requirement 1.3 (valid JSON serialization). Test cases: (1) Full output with all Phase 1 used fields (continue, hookEventName, additionalContext, systemMessage), (2) omitempty fields are omitted when empty/nil, (3) Phase 1 unused fields (stopReason, suppressOutput) are omitted when zero values (verify they don't appear in JSON output), (4) HookEventName is always "SessionStart", (5) Round-trip serialization preserves data. Use existing test patterns from types_test.go. | Restrictions: Must use table-driven tests, verify JSON structure exactly matches Claude Code spec, test both marshaling and unmarshaling, ensure all edge cases are covered, explicitly verify stopReason and suppressOutput are omitted from JSON when zero | Success: Tests pass and verify correct JSON structure, omitempty works as expected including for Phase 1 unused fields, round-trip serialization preserves all data, test coverage includes all struct fields_

---

- [x] 2.5. JSONスキーマバリデーションのセットアップ
  - Files: testdata/schemas/session-start-output.json, go.mod, types_test.go
  - Claude Code公式JSONスキーマを定義し、バリデーション機構を追加
  - Purpose: JSON出力がClaude Code公式スキーマに準拠していることを自動検証
  - _Leverage: 既存のtypes_test.goテストパターン_
  - _Requirements: 非機能要件（スキーマ準拠の担保）_
  - _Prompt: Role: Go Developer with expertise in JSON Schema and validation | Task: Set up JSON Schema validation for SessionStartOutput following Claude Code official specification. (1) Create testdata/schemas/session-start-output.json with JSON Schema definition matching Claude Code spec: required field "hookSpecificOutput.hookEventName" (string, enum: ["SessionStart"]), optional common fields "continue" (boolean), "stopReason" (string), "suppressOutput" (boolean), "systemMessage" (string), optional hook-specific field "hookSpecificOutput.additionalContext" (string). All fields must have correct types to catch type regressions (e.g., integer stopReason should fail). (2) Add github.com/xeipuuv/gojsonschema to go.mod using "go get github.com/xeipuuv/gojsonschema". (3) Create validateSessionStartOutput function in types_test.go that loads schema from testdata/schemas/session-start-output.json and validates JSON using gojsonschema.Validate. (4) Add TestSessionStartOutputSchemaValidation in types_test.go with test cases: valid full output (all fields), valid minimal output (only hookEventName), invalid output (missing hookEventName), invalid output (wrong hookEventName value), invalid output (wrong field types for continue/suppressOutput), invalid output (wrong type for stopReason/systemMessage). Use table-driven tests. | Restrictions: Schema must match Claude Code official specification exactly (all common fields: continue, stopReason, suppressOutput, systemMessage), use file-based schema loading (not inline), handle schema loading errors gracefully, validation function should return clear error messages, follow existing test patterns | Success: Schema file is valid JSON Schema covering all Claude Code fields, validation function correctly validates/rejects outputs, all test cases pass including type regression tests, error messages are clear and actionable, schema can be reused for other hook types in future_
  - _Note: Claude Code公式スキーマはhttps://docs.claude.com/en/docs/claude-code/hooksを参照。hookEventNameは必須、continue/systemMessage/additionalContextはオプション_

---

- [x] 3. runCommandWithOutput関数の追加（utils.go）
  - File: utils.go
  - stdout/stderrをキャプチャする新しいrunCommandWithOutput関数を追加
  - 終了コード、標準出力、標準エラー出力を返す
  - Purpose: type: commandアクション用のコマンド実行とJSON出力パース
  - _Leverage: 既存のrunCommand関数_
  - _Requirements: 要求3（type: commandアクションのJSON出力処理）_
  - _Prompt: Role: Go Developer with expertise in os/exec and command execution | Task: Create runCommandWithOutput function in utils.go following requirement 3 (type: command JSON output handling). Function signature: func runCommandWithOutput(command string, useStdin bool, data interface{}) (stdout string, stderr string, exitCode int, err error). Implementation: Execute command via sh -c, capture stdout and stderr using bytes.Buffer, if useStdin is true marshal data as JSON and pipe to stdin, return captured output and exit code. | Restrictions: Must not modify existing runCommand function, handle exec.ExitError to extract exit code, ensure buffers capture all output, maintain shell execution pattern (sh -c), follow existing error handling patterns | Success: Function captures stdout and stderr correctly, exit code is properly extracted from ExitError, stdin piping works with useStdin flag, error handling is robust_

---

- [x] 4. runCommandWithOutput関数のテスト作成（utils_test.go）
  - File: utils_test.go
  - runCommandWithOutputのテストを追加（成功/失敗/stdout/stderr）
  - Purpose: コマンド実行とキャプチャの正しさを検証
  - _Leverage: 既存のrunCommandテストパターン_
  - _Requirements: 要求3.1, 3.2（コマンド実行と終了コード）_
  - _Prompt: Role: Go Test Engineer with expertise in os/exec testing | Task: Create comprehensive tests for runCommandWithOutput in utils_test.go following requirements 3.1 and 3.2 (command execution and exit codes). Test cases: (1) Successful command (exit 0) with stdout, (2) Failed command (exit non-0) with stderr, (3) Command with useStdin=true receives JSON data, (4) Empty output handling, (5) Exit code extraction from various error types. Use table-driven tests. | Restrictions: Must not rely on external commands that may not exist, use simple shell commands (echo, exit, etc.), test both success and failure paths, verify stdout/stderr capture independently | Success: Tests cover all command execution scenarios, exit codes are correctly extracted, stdin piping is verified, output capture works reliably_

---

- [x] 5. ExecuteSessionStartActionメソッドのテスト作成（executor_test.go）【TDD: テスト先行】
  - File: executor_test.go
  - ActionExecutor.ExecuteSessionStartActionの全ケースをテスト（type: output / command、正常/異常系）
  - Purpose: アクション実行ロジックの正しさを検証
  - _Leverage: 既存のActionExecutorテストパターン（stubRunnerを使用）_
  - _Requirements: 要求2, 3, 4（全アクション処理要件）_
  - _Prompt: Role: Go Test Engineer with expertise in unit testing and table-driven tests | Task: Create comprehensive tests for modified ActionExecutor.ExecuteSessionStartAction in executor_test.go covering requirements 2, 3, and 4 (all action processing). Test cases for type: output: (1) Message with continue unspecified -> continue: true, (2) Message with continue: false -> continue: false, (3) Message with template variables -> correctly expanded, (4) Empty message -> continue: false + systemMessage. Test cases for type: command: (1) Command success with valid JSON -> correctly parsed, (2) Command with hookEventName="SessionStart" -> correctly set, (3) Command with continue unspecified -> continue: false, (4) Command failure (exit != 0) -> continue: false + systemMessage with stderr, (5) Empty stdout -> continue: false + systemMessage, (6) Invalid JSON output -> continue: false + systemMessage, (7) Missing hookEventName -> continue: false + systemMessage. Use table-driven tests with clear test case names and stubRunner for mocking command execution. | Restrictions: Must use stubRunner for command execution mocking, verify all ActionOutput fields, test error messages match requirements exactly, ensure template processing is tested, maintain test isolation, follow existing executor_test.go patterns | Success: All test cases pass and cover both success and failure paths, error messages match requirements, template processing is verified, stubRunner mocking works correctly, tests are maintainable and clear_

---

- [x] 6. ExecuteSessionStartActionメソッドの改修（executor.go）【TDD: 実装】
  - File: executor.go
  - ActionExecutor.ExecuteSessionStartActionをActionOutput返却に変更（現在はerror返却）
  - type: output処理ロジック実装（messageマッピング、continue設定）
  - type: command処理ロジック実装（JSON出力パース、エラーハンドリング）
  - Purpose: 単一アクションのJSON出力生成
  - _Leverage: 既存のunifiedTemplateReplace、新しいrunCommandWithOutput_
  - _Requirements: 要求2（YAML設定形式）、要求3（type: command処理）、要求4（type: output処理）_
  - _Prompt: Role: Go Developer with expertise in JSON processing and error handling | Task: Modify ActionExecutor.ExecuteSessionStartAction method in executor.go to return (*ActionOutput, error) instead of error, implementing requirements 2 (YAML config), 3 (type: command), and 4 (type: output). For type: output: (1) Process message with unifiedTemplateReplace, (2) If message is empty return ActionOutput{Continue: false, SystemMessage: "Action output has no message"}, (3) Otherwise set Continue based on action.Continue (default true if unspecified), set HookEventName to "SessionStart", set AdditionalContext to processed message. For type: command: (1) Process command with unifiedTemplateReplace, (2) Execute with runCommandWithOutput, (3) If exit code != 0 return ActionOutput{Continue: false, SystemMessage: "Command failed with exit code X: <stderr>"}, (4) If stdout is empty return ActionOutput{Continue: false, SystemMessage: "Command produced no output"}, (5) Parse stdout as JSON, if parse fails return ActionOutput{Continue: false, SystemMessage: "Command output is not valid JSON: <output>"}, (6) If hookEventName field is missing return ActionOutput{Continue: false, SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName"}, (7) If continue field is missing set to false as fallback default, (8) Return parsed ActionOutput. IMPORTANT: Do NOT set StopReason or SuppressOutput fields (Phase 1 unused fields - leave as zero values). | Restrictions: Must handle all error cases explicitly, use appropriate error messages matching requirements exactly, maintain template processing for all string fields, Action.Continue field must already be added by Task 0, ensure ActionOutput fields are properly populated (except StopReason/SuppressOutput which remain zero), maintain ActionExecutor method signature pattern | Success: Method returns ActionOutput for both success and error cases, all requirement error messages match exactly, template processing works correctly, JSON parsing handles all edge cases, StopReason and SuppressOutput remain zero values_

---

- [x] 7. executeSessionStartHooksのテスト作成（hooks_test.go）【TDD: テスト先行】
  - File: hooks_test.go
  - executeSessionStartHooksの全ケースをテスト（単一/複数アクション、early return）
  - ActionExecutorの動作は既にexecutor_test.goで検証済みのため、hooks層の統合に集中
  - Purpose: フック実行とJSON統合の正しさを検証
  - _Leverage: 既存のexecuteSessionStartHooksテストパターン、executor_test.goで検証済みのActionExecutor_
  - _Requirements: 要求5（複数アクションの処理）_
  - _Prompt: Role: Go Test Engineer with expertise in integration testing and complex scenarios | Task: Create comprehensive tests for modified executeSessionStartHooks in hooks_test.go covering requirement 5 (multiple action processing). Test cases: (1) Single type: output action -> correct JSON output, (2) Single type: command action -> correct JSON output, (3) Multiple actions both succeed -> additionalContext concatenated with "\n", (4) First action continue: false -> early return, second action not executed, (5) Second action continue: false -> first action results preserved, (6) HookEventName preservation -> set by first action, kept even if second action has no hookSpecificOutput, (7) SystemMessage concatenation -> multiple errors concatenated with "\n", (8) Condition check error -> error collected, hook skipped, (9) Action execution error -> error collected, finalOutput reflects partial success, (10) Matcher not matching -> hook skipped. Use table-driven tests with clear scenarios. Note: ActionExecutor behavior is already tested in executor_test.go, focus on hook-level orchestration. | Restrictions: Must verify early return behavior explicitly (check action execution count), test concatenation with actual "\n" characters, verify hookEventName preservation across actions, ensure error collection works with errors.Join, validate both finalOutput and error return value, rely on executor_test.go for ActionExecutor unit tests | Success: All scenarios pass and verify correct behavior, early return is tested and works, field merging (overwrite/preserve/concatenate) is verified, error collection includes all errors, tests clearly demonstrate requirement compliance_

---

- [x] 8. executeSessionStartHooksの改修（hooks.go）【TDD: 実装】
  - File: hooks.go
  - executeSessionStartHooksをSessionStartOutput返却に変更（現在はerror返却）
  - 複数アクション実行とJSON出力統合ロジック実装
  - ActionExecutorを使用してアクションを実行
  - Early return処理（continue: falseで即座に終了）
  - Purpose: フックオーケストレーションとJSON統合
  - _Leverage: 既存のマッチャー/条件チェックロジック、ActionExecutor.ExecuteSessionStartAction_
  - _Requirements: 要求5（複数アクションの処理）_
  - _Prompt: Role: Go Developer with expertise in orchestration and state management | Task: Modify executeSessionStartHooks in hooks.go to return (*SessionStartOutput, error), implementing requirement 5 (multiple action processing). Implementation: (1) Create ActionExecutor instance with NewActionExecutor(nil), (2) Initialize finalOutput with Continue: true, (3) For each hook: check matcher and conditions (existing logic), (4) For each action: call executor.ExecuteSessionStartAction, update finalOutput with these rules: Continue is overwritten (but if false, break immediately for early return), HookEventName is set once and preserved (if unset && ActionOutput has value -> set, else keep existing), AdditionalContext is concatenated with "\n" if non-empty, SystemMessage is concatenated with "\n" if non-empty, StopReason and SuppressOutput are NOT updated (Phase 1 unused - remain zero values), (5) Collect condition/action errors with errors.Join, (6) Return finalOutput and joined errors. Preserve existing condition checking and error collection logic. | Restrictions: Must use ActionExecutor for action execution, must not modify condition checking logic, implement early return correctly (break on continue: false), preserve hookEventName once set, use strings.Builder for concatenation efficiency, maintain existing error collection pattern with errors.Join, ensure finalOutput is constructed even with errors, do not copy StopReason/SuppressOutput from ActionOutput (Phase 1 unused) | Success: Function returns SessionStartOutput for all cases, ActionExecutor is used correctly, early return works on first continue: false, field merging follows requirements exactly (overwrite/preserve/concatenate), error collection preserves all errors, existing matcher/condition logic unchanged, StopReason and SuppressOutput remain zero values_

---

- [x] 9. main.goのJSON出力処理追加
  - File: main.go
  - SessionStart処理部分を改修してJSON出力シリアライズを追加
  - executeSessionStartHooksからSessionStartOutputを受け取り、json.MarshalIndentでシリアライズ
  - 常に終了コード0で終了（continue制御はJSON内）
  - Purpose: JSON出力を標準出力に書き込む
  - _Leverage: 既存のSessionStart入力パース処理、encoding/json_
  - _Requirements: 要求1.2（終了コード0）、要求1.3（有効なJSONシリアライズ）_
  - _Prompt: Role: Go Developer with expertise in CLI applications and JSON serialization | Task: Modify SessionStart handling in main.go to serialize JSON output following requirements 1.2 (exit code 0) and 1.3 (valid JSON serialization). Implementation: (1) Call executeSessionStartHooks and receive (*SessionStartOutput, error), (2) If error is not nil, log to stderr but continue (output is still valid), (3) Marshal output with json.MarshalIndent (indent with 2 spaces), (4) If marshal fails, write error to stderr and os.Exit(1), (5) Write marshaled JSON to stdout with fmt.Println, (6) Always os.Exit(0) (JSON continue field controls behavior). Remove existing ExitError handling for SessionStart. | Restrictions: Must not modify existing input parsing logic, always exit with code 0 unless marshal fails, write JSON to stdout not stderr, handle marshal error as fatal (only case for exit 1), maintain existing command-line flag handling | Success: SessionStart always exits with code 0 on success, JSON is properly formatted and written to stdout, marshal errors are handled gracefully, existing parsing logic unchanged_

---

- [x] 10. main.goの変更テスト作成（既存テストの更新）
  - File: main_test.go (または新規作成)
  - SessionStart実行の統合テストを追加/更新
  - Purpose: End-to-endでJSON出力を検証
  - _Leverage: 既存の統合テストパターン_
  - _Requirements: 全要求事項_
  - _Prompt: Role: Go Test Engineer with expertise in integration testing and CLI testing | Task: Create or update integration tests for SessionStart in main_test.go covering all requirements. Test cases: (1) Real config file (go.mod exists) -> serena recommendation message in additionalContext, (2) .claude/tmp not exists -> creation request message in additionalContext, (3) Multiple actions -> messages concatenated with "\n", (4) Command action success -> valid JSON output, (5) Command action failure -> continue: false + systemMessage, (6) JSON output validation: continue field always present, hookEventName is "SessionStart", additionalContext contains expected messages, (7) Exit code validation: always 0 for all cases. Use actual config files from test fixtures or create minimal configs. | Restrictions: Must test with real YAML config files, verify actual JSON output structure (not just types), validate exit code is always 0, ensure tests can run independently, use temporary directories for file existence tests | Success: Integration tests cover real-world scenarios from actual config, JSON output is validated against Claude Code spec, exit code 0 is verified, tests pass reliably and demonstrate end-to-end functionality_

---

- [x] 11. 既存ExitError処理の削除
  - File: executor.go, hooks.go
  - SessionStart関連のExitError生成コードを削除
  - Purpose: JSON出力への完全移行
  - _Leverage: なし（削除作業）_
  - _Requirements: 要求1（JSON出力形式への移行）_
  - _Prompt: Role: Go Developer with expertise in refactoring and code cleanup | Task: Remove ExitError handling from SessionStart functions in executor.go and hooks.go following requirement 1 (migration to JSON output). In executor.go: Remove ExitError checks and creation in ExecuteSessionStartAction (lines 97-99). In hooks.go: Remove ExitError handling in executeSessionStartHooks (lines 841-851). Verify no ExitError code remains in SessionStart-related functions. | Restrictions: Must only remove SessionStart-related ExitError code, do not modify ExitError handling for other hook types (ExecutePreToolUseAction, ExecuteStopAction, etc.), ensure no dead code remains, verify all SessionStart paths return ActionOutput/SessionStartOutput | Success: No ExitError code remains in SessionStart functions, other hook types' ExitError handling is unchanged, code compiles without errors, no unused variables or imports remain_

---

- [x] 12. 既存テストの更新とクリーンアップ
  - File: executor_test.go, hooks_test.go
  - SessionStart関連の既存テストをJSON出力対応に更新
  - ExitError検証をActionOutput/SessionStartOutput検証に置き換え
  - Purpose: 既存テストカバレッジの維持
  - _Leverage: 既存のテスト構造、stubRunnerパターン_
  - _Requirements: 全要求事項_
  - _Prompt: Role: Go Test Engineer with expertise in test refactoring and maintenance | Task: Update existing SessionStart tests in executor_test.go and hooks_test.go to verify JSON output instead of ExitError. Replace ExitError assertions with ActionOutput/SessionStartOutput field assertions. Update test cases: (1) Where tests checked ExitError.Code, now check Continue field, (2) Where tests checked ExitError.Message, now check SystemMessage or AdditionalContext, (3) Where tests checked ExitError.Stderr, now check Continue: false + SystemMessage. In executor_test.go: Update ExecuteSessionStartAction tests to use stubRunner and verify ActionOutput. In hooks_test.go: Update executeSessionStartHooks tests to verify SessionStartOutput. Ensure test coverage is maintained or improved. Remove obsolete test cases that no longer apply. | Restrictions: Must maintain or improve test coverage, update assertions to match new return types, ensure all edge cases are still tested, verify test names reflect new behavior, do not delete tests unless truly obsolete, use stubRunner pattern in executor_test.go | Success: All existing SessionStart tests updated and passing, test coverage maintained or improved, assertions verify correct JSON output fields, test names and descriptions are clear and accurate_

---

- [x] 13. ドキュメント更新
  - File: CLAUDE.md
  - SessionStartのJSON出力形式について記載を追加/更新
  - 設定例とJSON出力例を追加
  - Purpose: 開発者向けドキュメントの更新
  - _Leverage: 既存のCLAUDE.mdの構造_
  - _Requirements: 非機能要件（ユーザビリティ）_
  - _Prompt: Role: Technical Writer with expertise in developer documentation | Task: Update CLAUDE.md to document SessionStart JSON output format following non-functional requirements (usability). Add: (1) Section explaining JSON output format for SessionStart hooks, (2) Example configuration showing type: output and type: command, (3) Example JSON output showing continue, hookSpecificOutput, systemMessage fields, (4) Note about always exiting with code 0, (5) Migration guide snippet for users upgrading from exit status approach. Use existing CLAUDE.md structure and formatting style. | Restrictions: Must follow existing documentation style, use concrete examples from actual config, keep explanations concise and clear, update "Example Configuration" section if needed, maintain existing document structure | Success: Documentation clearly explains JSON output format, examples are accurate and helpful, migration guidance is practical, formatting matches existing style, developers can understand how to use the feature_

---

## 実装順序の説明

1. **型定義とテスト** (Tasks 1-2): 型システムの基盤を確立
2. **ユーティリティとテスト** (Tasks 3-4): コマンド実行インフラを構築
3. **アクション層とテスト** (Tasks 5-6): 単一アクションのJSON生成ロジック
4. **フック層とテスト** (Tasks 7-8): 複数アクション統合ロジック
5. **統合とテスト** (Tasks 9-10): End-to-end動作確認
6. **クリーンアップ** (Tasks 11-12): 旧コード削除と既存テスト更新
7. **ドキュメント** (Task 13): 開発者向け情報提供

この順序はt_wada式TDDに従い、各機能についてテストを先に書いてから実装を進める構成になっています。
