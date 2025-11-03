# Tasks: UserPromptSubmit Hook JSON出力対応（Phase 2）

## Task Overview

**このspecのスコープ**: Phase 2として、UserPromptSubmitフックをJSON出力形式に移行します。
Phase 1（SessionStart）で確立したパターンを踏襲し、`decision`フィールド（allow/block）を追加します。

UserPromptSubmitフックをexit statusベースからJSON出力形式に移行します。t_wada式TDDに従い、テストを先に書いてから実装を進めます。

---

- [x] 0. ActionOutput構造体へのDecisionフィールド追加（types.go）
  - File: types.go
  - ActionOutput構造体に`Decision string`フィールドを追加
  - Purpose: Phase 1のActionOutput構造体を拡張し、UserPromptSubmit用のdecision制御を可能にする
  - _Leverage: Phase 1のActionOutput構造体_
  - _Requirements: 要求6（Phase 1構造体の再利用）_
  - _Prompt: Role: Go Developer with expertise in struct design and backward compatibility | Task: Add Decision string field to ActionOutput struct in types.go following requirement 6 (Phase 1 structure reuse). Add `Decision string` field to existing ActionOutput struct. This field will hold "allow", "block", or empty string. Ensure backward compatibility with Phase 1 (SessionStart) which doesn't use this field. | Restrictions: Must not modify existing ActionOutput fields, ensure SessionStart code continues to work (Decision will be empty string for SessionStart), maintain consistent field naming (PascalCase), do not break existing tests | Success: Field compiles without errors, ActionOutput can be used for both SessionStart and UserPromptSubmit, existing SessionStart tests still pass, field can hold "allow"/"block"/"" values_

---

- [x] 1. Action構造体へのDecisionフィールド追加（types.go）
  - File: types.go
  - Action構造体に`Decision *string`フィールドを追加
  - YAMLタグは`yaml:"decision,omitempty"`
  - Purpose: YAML設定でdecisionフィールドを明示的に指定可能にする
  - _Leverage: 既存のAction構造体、Phase 1のContinue *boolパターン_
  - _Requirements: 要求2（decisionフィールドの制御）_
  - _Prompt: Role: Go Developer with expertise in YAML configuration and struct design | Task: Add Decision *string field to Action struct in types.go following requirement 2 (decision field control). Add `Decision *string` with yaml tag `yaml:"decision,omitempty"`. This allows users to explicitly specify `decision: "allow"` or `decision: "block"` in YAML config. Use pointer type (*string) to distinguish between unspecified (nil) and explicit values. | Restrictions: Must not modify existing Action fields, use omitempty in yaml tag, follow existing field naming conventions (PascalCase), maintain backward compatibility (decision field is optional), follow Phase 1 Continue field pattern | Success: Field compiles without errors, yaml tag is correct, *string allows nil/explicit value distinction, existing YAML configs continue to work without decision field_

---

- [x] 2. UserPromptSubmitOutput構造体の定義（types.go）
  - File: types.go
  - UserPromptSubmitOutput, UserPromptSubmitHookSpecificOutput構造体を追加
  - Claude Code仕様に準拠したJSONタグを設定、decisionフィールドを含む
  - Purpose: JSON出力の型安全性を確保（Phase 1のSessionStartOutputパターンを踏襲）
  - _Leverage: Phase 1のSessionStartOutput構造体パターン_
  - _Requirements: 要求1（JSON出力形式への移行）_
  - _Prompt: Role: Go Developer specializing in type systems and JSON serialization | Task: Define UserPromptSubmitOutput and UserPromptSubmitHookSpecificOutput structs in types.go following requirement 1 (JSON output format migration), based on Phase 1 SessionStartOutput pattern. UserPromptSubmitOutput contains: Continue (bool), Decision (string), StopReason (string, omitempty), SuppressOutput (bool, omitempty), SystemMessage (string, omitempty), HookSpecificOutput (*UserPromptSubmitHookSpecificOutput, omitempty). UserPromptSubmitHookSpecificOutput contains: HookEventName (string) and AdditionalContext (string, omitempty). | Restrictions: Must follow Claude Code JSON specification exactly, use pointer for HookSpecificOutput to support omitempty, ensure HookEventName is always "UserPromptSubmit" when set, maintain consistency with Phase 1 SessionStartOutput structure, Decision field is required (no omitempty) | Success: All structs compile without errors, JSON tags are correct with omitempty, structs match Claude Code specification, type names follow Go conventions, structure mirrors Phase 1 pattern_

---

- [x] 3. UserPromptSubmitOutput構造体のテスト作成（types_test.go）
  - File: types_test.go
  - UserPromptSubmitOutputのJSONシリアライズ/デシリアライズテストを追加
  - omitemptyフィールドとdecisionフィールドの動作確認テストを追加
  - Purpose: JSON出力構造の正しさを検証（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のSessionStartOutputテストパターン_
  - _Requirements: 要求1.3（有効なJSONシリアライズ）、要求2.3（decision値検証）_
  - _Prompt: Role: Go Test Engineer with expertise in JSON serialization testing | Task: Create comprehensive tests for UserPromptSubmitOutput JSON serialization/deserialization in types_test.go following requirements 1.3 and 2.3, based on Phase 1 SessionStartOutput tests. Test cases: (1) Full output with all Phase 2 used fields (continue, decision, hookEventName, additionalContext, systemMessage), (2) omitempty fields are omitted when empty/nil, (3) Phase 2 unused fields (stopReason, suppressOutput) are omitted when zero values, (4) HookEventName is always "UserPromptSubmit", (5) Decision field accepts "allow" and "block" only, (6) Round-trip serialization preserves data. Use table-driven tests following Phase 1 pattern. | Restrictions: Must use table-driven tests, verify JSON structure exactly matches Claude Code spec, test both marshaling and unmarshaling, ensure decision field validation, explicitly verify stopReason and suppressOutput are omitted from JSON when zero, follow Phase 1 test structure | Success: Tests pass and verify correct JSON structure, omitempty works as expected, decision field validation works, round-trip serialization preserves all data, test coverage includes all struct fields_

---

- [x] 4. JSONスキーマバリデーションのセットアップ
  - Files: testdata/schemas/user-prompt-submit-output.json, types_test.go
  - Claude Code公式JSONスキーマを定義し、バリデーション機構を追加
  - Purpose: JSON出力がClaude Code公式スキーマに準拠していることを自動検証（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のJSONスキーマバリデーションパターン、既存のgojsonschema依存_
  - _Requirements: 非機能要件（スキーマ準拠の担保）_
  - _Prompt: Role: Go Developer with expertise in JSON Schema and validation | Task: Set up JSON Schema validation for UserPromptSubmitOutput following Claude Code official specification, based on Phase 1 pattern. (1) Create testdata/schemas/user-prompt-submit-output.json with JSON Schema definition: required fields "decision" (string, enum: ["allow", "block"]), "hookSpecificOutput.hookEventName" (string, enum: ["UserPromptSubmit"]), optional common fields "continue" (boolean), "stopReason" (string), "suppressOutput" (boolean), "systemMessage" (string), optional hook-specific field "hookSpecificOutput.additionalContext" (string). (2) Create validateUserPromptSubmitOutput function in types_test.go using existing gojsonschema pattern from Phase 1. (3) Add TestUserPromptSubmitOutputSchemaValidation with test cases: valid full output, valid minimal output (only required fields), invalid output (missing hookEventName), invalid output (wrong hookEventName value), invalid output (invalid decision value), invalid output (wrong field types). | Restrictions: Schema must match Claude Code official specification exactly, use file-based schema loading (follow Phase 1 pattern), handle schema loading errors gracefully, reuse Phase 1 validation function pattern, follow existing test patterns | Success: Schema file is valid JSON Schema covering all Claude Code fields, validation function correctly validates/rejects outputs, all test cases pass including decision value validation, error messages are clear, follows Phase 1 pattern_

---

- [x] 5. ExecuteUserPromptSubmitActionメソッドのテスト作成（executor_test.go）【TDD: テスト先行】
  - File: executor_test.go
  - ActionExecutor.ExecuteUserPromptSubmitActionの全ケースをテスト（type: output / command、正常/異常系）
  - Purpose: アクション実行ロジックの正しさを検証（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のExecuteSessionStartActionテストパターン、既存のstubRunner_
  - _Requirements: 要求2, 3, 4（全アクション処理要件）_
  - _Prompt: Role: Go Test Engineer with expertise in unit testing and table-driven tests | Task: Create comprehensive tests for ActionExecutor.ExecuteUserPromptSubmitAction in executor_test.go covering requirements 2, 3, and 4, based on Phase 1 ExecuteSessionStartAction tests. Test cases for type: output: (1) Message with decision unspecified -> decision: "allow", (2) Message with decision: "block" -> decision: "block", (3) Message with invalid decision value -> decision: "block" + systemMessage, (4) Message with template variables -> correctly expanded, (5) Empty message -> decision: "block" + systemMessage. Test cases for type: command: (1) Command success with valid JSON -> correctly parsed, (2) Command with hookEventName="UserPromptSubmit" -> correctly set, (3) Command with decision unspecified -> decision: "allow", (4) Command with decision: "block" -> decision: "block", (5) Command failure (exit != 0) -> decision: "block" + systemMessage with stderr, hookEventName set to "UserPromptSubmit", (6) Empty stdout -> decision: "allow", hookEventName: "UserPromptSubmit" (validation tool case - hookEventName must always be set), (7) Invalid JSON output -> decision: "block" + systemMessage, hookEventName set to "UserPromptSubmit", (8) Missing hookEventName -> decision: "block" + systemMessage, hookEventName set to "UserPromptSubmit", (9) Invalid hookEventName value -> decision: "block" + systemMessage, hookEventName set to "UserPromptSubmit", (10) Invalid decision value -> decision: "block" + systemMessage (not warning - treat as blocking error). Use stubRunner for mocking. | Restrictions: Must use stubRunner for command execution mocking, verify all ActionOutput fields including Decision and HookEventName, test error messages match requirements exactly (no warnings for invalid values - must block), ensure continue is always true for UserPromptSubmit, ensure hookEventName is ALWAYS set to "UserPromptSubmit" even in error cases, follow Phase 1 test pattern | Success: All test cases pass and cover both success and failure paths, decision field logic is verified, hookEventName is always "UserPromptSubmit" in all cases, error messages match requirements, invalid values cause blocking errors (not warnings), stubRunner mocking works correctly, tests follow Phase 1 pattern_

---

- [x] 6. ExecuteUserPromptSubmitActionメソッドの実装（executor.go）【TDD: 実装】
  - File: executor.go
  - ActionExecutor.ExecuteUserPromptSubmitActionを追加（Phase 1のExecuteSessionStartActionは維持）
  - type: output/command処理ロジック実装、decisionフィールド制御を含む
  - Purpose: 単一アクションのJSON出力生成（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のExecuteSessionStartAction、unifiedTemplateReplace、runCommandWithOutput_
  - _Requirements: 要求2（decision制御）、要求3（type: command処理）、要求4（type: output処理）_
  - _Prompt: Role: Go Developer with expertise in JSON processing and error handling | Task: Implement ActionExecutor.ExecuteUserPromptSubmitAction method in executor.go following requirements 2, 3, and 4, based on Phase 1 ExecuteSessionStartAction pattern. Method signature: func (e *ActionExecutor) ExecuteUserPromptSubmitAction(action Action, input *UserPromptSubmitInput, rawJSON interface{}) (*ActionOutput, error). For type: output: (1) Process message with unifiedTemplateReplace, (2) If message is empty return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Action output has no message"}, (3) If action.Decision is set, validate it ("allow" or "block"), if invalid return error ActionOutput with HookEventName set, (4) Otherwise set Continue to true, Decision based on action.Decision (default "allow" if unspecified), HookEventName to "UserPromptSubmit", AdditionalContext to processed message. For type: command: (1) Process command with unifiedTemplateReplace, (2) Execute with runCommandWithOutput, (3) If exit code != 0 return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Command failed with exit code X: <stderr>"}, (4) If stdout is empty return ActionOutput{Continue: true, Decision: "allow", HookEventName: "UserPromptSubmit"} (validation tool case - hookEventName must always be set), (5) Parse stdout as JSON, if parse fails return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Command output is not valid JSON: <output>"}, (6) If hookEventName field is missing return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName"}, (7) If hookEventName is not "UserPromptSubmit" return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Invalid hookEventName: expected 'UserPromptSubmit', got '<value>'"} (configuration error - must block), (8) If decision field is missing set to "allow" as fallback default, validate decision value ("allow" or "block"), if invalid return ActionOutput{Continue: true, Decision: "block", HookEventName: "UserPromptSubmit", SystemMessage: "Invalid decision value: must be 'allow' or 'block'"} (configuration error - must block, not warn), (9) Return parsed ActionOutput with Continue always true. IMPORTANT: Do NOT set StopReason or SuppressOutput fields (Phase 2 unused fields - leave as zero values). Follow Phase 1 pattern. | Restrictions: Must handle all error cases explicitly, use appropriate error messages matching requirements exactly, maintain template processing for all string fields, Action.Decision field was added by Task 1, ensure ActionOutput fields are properly populated (except StopReason/SuppressOutput which remain zero), maintain ActionExecutor method signature pattern, Continue is always true for UserPromptSubmit, hookEventName must ALWAYS be set to "UserPromptSubmit" in all cases (including error cases), invalid decision/hookEventName values must block with systemMessage (not warn), follow Phase 1 structure | Success: Method returns ActionOutput for both success and error cases, all requirement error messages match exactly, template processing works correctly, JSON parsing handles all edge cases, decision field logic is correct, Continue is always true, hookEventName is always "UserPromptSubmit" in all cases, invalid values cause blocking errors (not warnings), follows Phase 1 pattern_

---

- [x] 7. executeUserPromptSubmitHooksのテスト作成（hooks_test.go）【TDD: テスト先行】
  - File: hooks_test.go
  - executeUserPromptSubmitHooksの全ケースをテスト（単一/複数アクション、early return）
  - Purpose: フック実行とJSON統合の正しさを検証（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のexecuteSessionStartHooksテストパターン_
  - _Requirements: 要求5（複数アクションの処理）_
  - _Prompt: Role: Go Test Engineer with expertise in integration testing and complex scenarios | Task: Create comprehensive tests for executeUserPromptSubmitHooks in hooks_test.go covering requirement 5, based on Phase 1 executeSessionStartHooks tests. Test cases: (1) Single type: output action -> correct JSON output, (2) Single type: command action -> correct JSON output, (3) Multiple actions both succeed -> additionalContext concatenated with "\n", (4) First action decision: "block" -> early return, second action not executed, (5) Second action decision: "block" -> first action results preserved, (6) HookEventName preservation -> set by first action, kept even if second action has no hookSpecificOutput, (7) SystemMessage concatenation -> multiple errors concatenated with "\n", (8) Condition check error -> error collected, hook skipped, (9) Action execution error -> error collected, finalOutput reflects partial success, (10) Matcher not matching -> hook skipped, (11) Continue field always true -> verified in all cases. Use table-driven tests. Follow Phase 1 pattern. | Restrictions: Must verify early return behavior explicitly (check action execution count), test concatenation with actual "\n" characters, verify hookEventName preservation across actions, ensure error collection works with errors.Join, validate both finalOutput and error return value, verify Continue is always true, follow Phase 1 test structure | Success: All scenarios pass and verify correct behavior, early return is tested and works, field merging (overwrite/preserve/concatenate) is verified, error collection includes all errors, Continue is always true, tests follow Phase 1 pattern_

---

- [x] 8. executeUserPromptSubmitHooksの実装（hooks.go）【TDD: 実装】
  - File: hooks.go
  - executeUserPromptSubmitHooksを追加（Phase 1のexecuteSessionStartHooksは維持）
  - 複数アクション実行とJSON出力統合ロジック実装、decision: "block"でearly return
  - Purpose: フックオーケストレーションとJSON統合（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のexecuteSessionStartHooks、既存のマッチャー/条件チェックロジック、ActionExecutor_
  - _Requirements: 要求5（複数アクションの処理）_
  - _Prompt: Role: Go Developer with expertise in orchestration and state management | Task: Implement executeUserPromptSubmitHooks in hooks.go following requirement 5, based on Phase 1 executeSessionStartHooks pattern. Function signature: func executeUserPromptSubmitHooks(config *Config, input *UserPromptSubmitInput, rawJSON interface{}) (*UserPromptSubmitOutput, error). Implementation: (1) Create ActionExecutor instance with NewActionExecutor(nil), (2) Initialize finalOutput with Continue: true, Decision: "allow", (3) For each hook: check matcher and conditions (existing logic), (4) For each action: call executor.ExecuteUserPromptSubmitAction, update finalOutput with these rules: Continue is always true (do not overwrite), Decision is overwritten (but if "block", break immediately for early return), HookEventName is set once and preserved (if unset && ActionOutput has value -> set, else keep existing), AdditionalContext is concatenated with "\n" if non-empty, SystemMessage is concatenated with "\n" if non-empty, StopReason and SuppressOutput are NOT updated (Phase 2 unused - remain zero values), (5) Collect condition/action errors with errors.Join, (6) Return finalOutput and joined errors. Follow Phase 1 pattern. | Restrictions: Must use ActionExecutor for action execution, must not modify condition checking logic, implement early return correctly (break on decision: "block"), preserve hookEventName once set, use strings.Builder for concatenation efficiency, maintain existing error collection pattern with errors.Join, ensure finalOutput is constructed even with errors, Continue is always true (never change it), do not copy StopReason/SuppressOutput from ActionOutput (Phase 2 unused), follow Phase 1 structure | Success: Function returns UserPromptSubmitOutput for all cases, ActionExecutor is used correctly, early return works on first decision: "block", field merging follows requirements exactly (overwrite/preserve/concatenate), error collection preserves all errors, Continue is always true, follows Phase 1 pattern_

---

- [x] 9. main.goのUserPromptSubmit処理追加
  - File: main.go
  - UserPromptSubmit処理部分を追加してJSON出力シリアライズを実装
  - executeUserPromptSubmitHooksからUserPromptSubmitOutputを受け取り、json.MarshalIndentでシリアライズ
  - 常に終了コード0で終了（decision制御はJSON内）
  - Purpose: JSON出力を標準出力に書き込む（Phase 1パターンを踏襲）
  - _Leverage: Phase 1のSessionStart処理、既存のUserPromptSubmit入力パース処理、encoding/json_
  - _Requirements: 要求1.2（終了コード0）、要求1.3（有効なJSONシリアライズ）_
  - _Prompt: Role: Go Developer with expertise in CLI applications and JSON serialization | Task: Add UserPromptSubmit handling to main.go following requirements 1.2 and 1.3, based on Phase 1 SessionStart pattern. Implementation: Add case for UserPromptSubmit event type: (1) Use existing UserPromptSubmit parsing logic, (2) Call executeUserPromptSubmitHooks and receive (*UserPromptSubmitOutput, error), (3) If error is not nil, log to stderr but continue (output is still valid), (4) Marshal output with json.MarshalIndent (indent with 2 spaces), (5) If marshal fails, write error to stderr and os.Exit(1), (6) Write marshaled JSON to stdout with fmt.Println, (7) Always os.Exit(0) (JSON decision field controls behavior). Do not modify existing SessionStart handling. Follow Phase 1 pattern. | Restrictions: Must not modify existing input parsing logic, must not modify SessionStart handling, always exit with code 0 unless marshal fails, write JSON to stdout not stderr, handle marshal error as fatal (only case for exit 1), maintain existing command-line flag handling, follow Phase 1 structure | Success: UserPromptSubmit always exits with code 0 on success, JSON is properly formatted and written to stdout, marshal errors are handled gracefully, existing parsing logic unchanged, SessionStart handling unchanged, follows Phase 1 pattern_

---

- [x] 10. main.goの変更テスト作成（または既存テストの更新）
  - File: user_prompt_submit_integration_test.go
  - UserPromptSubmit実行の統合テストを追加/更新
  - Purpose: End-to-endでJSON出力を検証（Phase 1パターンを踏襲）
  - _Leverage: Phase 1の統合テストパターン_
  - _Requirements: 全要求事項_

---

- [x] 11. ドキュメント更新
  - File: CLAUDE.md
  - UserPromptSubmitのJSON出力形式について記載を追加/更新
  - 設定例とJSON出力例を追加（Phase 1と一貫性を保つ）
  - Purpose: 開発者向けドキュメントの更新
  - _Leverage: Phase 1のSessionStart設定例、既存のCLAUDE.mdの構造_
  - _Requirements: 非機能要件（ユーザビリティ）_
  - _Prompt: Role: Technical Writer with expertise in developer documentation | Task: Update CLAUDE.md to document UserPromptSubmit JSON output format following non-functional requirements (usability), maintaining consistency with Phase 1 SessionStart documentation. Add: (1) Section explaining JSON output format for UserPromptSubmit hooks (similar to SessionStart section), (2) Example configuration showing type: output with decision field and type: command, (3) Example JSON output showing continue (always true), decision, hookSpecificOutput, systemMessage fields, (4) Note about always exiting with code 0, (5) Note about decision: "block" for early return. Update existing "SessionStart JSON Output" section if needed to show consistency. Use existing CLAUDE.md structure and formatting style. Follow Phase 1 documentation pattern. | Restrictions: Must follow existing documentation style, use concrete examples from actual config, keep explanations concise and clear, maintain consistency with Phase 1 SessionStart documentation, update relevant sections without breaking existing content, maintain existing document structure | Success: Documentation clearly explains UserPromptSubmit JSON output format, examples are accurate and helpful, consistent with Phase 1 documentation, formatting matches existing style, developers can understand how to use the feature, decision field usage is clear_

---

## 実装順序の説明

1. **型定義とテスト** (Tasks 0-4): 型システムの基盤を確立、Phase 1構造体を拡張
2. **アクション層とテスト** (Tasks 5-6): 単一アクションのJSON生成ロジック（Phase 1パターン踏襲）
3. **フック層とテスト** (Tasks 7-8): 複数アクション統合ロジック（Phase 1パターン踏襲）
4. **統合とテスト** (Tasks 9-10): End-to-end動作確認
5. **ドキュメント** (Task 11): 開発者向け情報提供

この順序はt_wada式TDDに従い、各機能についてテストを先に書いてから実装を進める構成になっています。また、Phase 1のパターンを最大限再利用することで、一貫性のある実装を実現します。
