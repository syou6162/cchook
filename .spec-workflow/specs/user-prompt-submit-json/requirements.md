# Requirements: UserPromptSubmit Hook JSON出力対応

## イントロダクション

cchookのJSON出力対応Phase 2として、UserPromptSubmitフックをJSON出力形式に移行します。

**Phase 2のスコープ**: UserPromptSubmitフックのみ

UserPromptSubmitは以下の特徴を持つフックです:
- **ユーザープロンプト送信前の制御**: ユーザーがプロンプトを送信する前に実行され、送信を許可またはブロックできる
- **additionalContextマッピング**: SessionStartと同様に、Claudeへの追加コンテキストを提供
- **ブロック機能**: `decision: "block"`でプロンプト送信をブロック可能（SessionStartには無い機能）
- **Phase 1パターンの踏襲**: SessionStartで確立したJSON出力パターンを再利用

**Phase 2で使用するフィールド**:
- `continue` (boolean): 処理継続制御（常にtrue、後方互換性のため）
- `decision` (string): "allow"（プロンプト送信を許可）または "block"（プロンプト送信をブロック）
- `hookSpecificOutput.hookEventName` (string): 必須、常に"UserPromptSubmit"
- `hookSpecificOutput.additionalContext` (string): Claudeへのコンテキスト提供
- `systemMessage` (string): ユーザーへの警告メッセージ

**Phase 2で未使用のフィールド**:
- `stopReason` (string): JSON Schema準拠のため構造体に定義するが、Phase 2では値を設定しない（omitemptyで出力から省略）
- `suppressOutput` (boolean): JSON Schema準拠のため構造体に定義するが、Phase 2では値を設定しない（omitemptyで出力から省略）

これらの未使用フィールドは、Phase 3（PreToolUse）以降で実装される予定です。

Phase 2完了後、Phase 3としてPreToolUseフック（最も複雑なpermissionDecision、updatedInput機能）に進みます。

## Product Visionとの整合性

product.mdで定義された以下の目標に沿います:
- **シンプルさ**: YAML設定でJSON構造を意識せずに使える
- **保守性**: イベント固有の複雑さを隠蔽
- **拡張性**: Phase 1の設計パターンを再利用し、将来の全フック対応を見据える

## 要求事項

### 要求1: JSON出力形式への移行

**ユーザーストーリー:** cchookユーザーとして、UserPromptSubmitフックでJSON形式の出力を利用したい。そうすることで、Claude Codeに情報を渡す方法がClaude Code標準仕様に準拠する。

#### 受け入れ基準

1. UserPromptSubmitフックが実行されるとき、システムはJSONを標準出力に出力すべきである
2. JSON出力が正常に生成されたとき、システムは終了コード0で終了すべきである
3. アクションが正常に完了したとき、システムはClaude Code仕様に従った有効なJSONとしてシリアライズすべきである
4. システムは`continue`フィールドを常に明示的に`true`に設定すべきである（UserPromptSubmitでは常にtrue、後方互換性のため）
5. システムは`decision`フィールドを常に明示的に出力すべきである（Claude Codeのデフォルト値"allow"に依存しない）
6. `type: output`アクションで`message`が空の場合、システムは`decision: "block"` + `systemMessage: "Action output has no message"`を設定すべきである（設定エラーとして扱う）
7. `type: command`アクションで標準出力が空の場合、システムは`decision: "allow"`で成功として扱うべきである（検証型CLIツール（fmt、linter、pre-commitなど）では問題がなければ何も出力せずexit 0で終了するため）

### 要求2: decisionフィールドの制御

**ユーザーストーリー:** cchookユーザーとして、プロンプト送信を許可またはブロックする制御を行いたい。そうすることで、特定の条件下でユーザーの操作を制限できる。

#### 受け入れ基準

1. `type: output`アクションで`decision`が指定されていない場合、システムは`decision: "allow"`を設定すべきである（通常ケースのデフォルト値。情報表示後もプロンプト送信を許可）
2. `type: output`アクションで`decision: "block"`が指定された場合、システムは`decision: "block"`を設定すべきである
3. `decision`フィールドの値は"allow"または"block"のみ許可すべきである（その他の値はエラー）
4. `decision: "block"`が設定されたとき、ユーザーにはブロックされた理由を示す`systemMessage`または`additionalContext`が提供されるべきである

### 要求3: type: commandアクションのJSON出力処理

**ユーザーストーリー:** cchookユーザーとして、外部コマンドが返すJSON出力を直接利用したい。そうすることで、複雑な判定ロジックを外部プログラムに委譲できる。

#### 受け入れ基準

1. `type: command`アクションが実行されるとき、システムは常に終了コード0を期待すべきである
2. コマンドが終了コード0で成功したとき、システムは標準出力からJSON出力をパースすべきである
3. パースしたJSON出力に`decision`フィールドが存在しない場合、システムは`decision: "allow"`を設定すべきである（フォールバック時のデフォルト値）
4. パースしたJSON出力の`decision`フィールドが"allow"または"block"以外の値の場合、システムは`{"continue": true, "decision": "block", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}, "systemMessage": "Invalid decision value: must be 'allow' or 'block'"}`というJSON出力を生成すべきである（設定エラーとしてブロック）
5. パースしたJSON出力に`hookSpecificOutput.hookEventName`フィールドが存在しない場合、システムは`{"continue": true, "decision": "block", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}, "systemMessage": "Command output is missing required field: hookSpecificOutput.hookEventName"}`というJSON出力を生成すべきである（外部コマンドの設定エラー）
6. パースしたJSON出力の`hookSpecificOutput.hookEventName`フィールドが"UserPromptSubmit"以外の値の場合、システムは`{"continue": true, "decision": "block", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}, "systemMessage": "Invalid hookEventName: expected 'UserPromptSubmit', got '<value>'"}`というJSON出力を生成すべきである（設定エラーとしてブロック）
7. コマンドが終了コード0以外で終了したとき（エラー時）、システムは`{"continue": true, "decision": "block", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}, "systemMessage": "Command failed with exit code X: <stderr>"}`というJSON出力を生成すべきである。`systemMessage`はユーザーにのみ表示される警告メッセージであり、Claudeには見えない（Claude Code仕様）
8. コマンドの標準出力が有効なJSONでない場合（エラー時）、システムは`{"continue": true, "decision": "block", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}, "systemMessage": "Command output is not valid JSON: <output>"}`というJSON出力を生成すべきである。`systemMessage`はユーザーにのみ表示される
9. コマンドの標準出力が空の場合、システムは`{"continue": true, "decision": "allow", "hookSpecificOutput": {"hookEventName": "UserPromptSubmit"}}`で成功として扱うべきである（検証型CLIツール（fmt、linter、pre-commitなど）では問題がなければ何も出力せずexit 0で終了するため）。この場合、`hookSpecificOutput.additionalContext`は空（omitempty）になるため、Claudeには追加情報が提供されないが、`hookEventName`は必須フィールドとして必ず設定される
10. コマンドが返すJSON出力にUserPromptSubmitでサポートされていないフィールド（例: `permissionDecision`, `updatedInput`）が含まれる場合、システムは警告"Warning: Field 'xxx' is not supported for UserPromptSubmit hooks"を標準エラー出力に書き込み、該当フィールドを無視すべきである

### 要求4: type: outputアクションのJSON変換

**ユーザーストーリー:** cchookユーザーとして、既存の`type: output`アクションを適切なJSONフィールドに自動変換してほしい。そうすることで、シンプルなケースでJSON構造を手動で指定する必要がなくなる。

#### 受け入れ基準

1. システムは`hookSpecificOutput.hookEventName`フィールドに`"UserPromptSubmit"`を設定すべきである（Claude Code仕様で必須）
2. `type: output`アクションの`message`フィールドは、システムによって`hookSpecificOutput.additionalContext`にマップされるべきである（UserPromptSubmitイベントの場合）
3. `message`が空文字列の場合、システムは要求1.6に従って`decision: "block"` + `systemMessage`を設定すべきである（設定エラーとして扱う）。`additionalContext`フィールドはJSONから省略される（omitempty）
4. `message`にテンプレート変数が存在する場合、システムはJSONフィールドに設定する前にそれらを処理すべきである

### 要求5: 複数アクションの処理

**ユーザーストーリー:** 単一のフックに複数のアクションを持つcchookユーザーとして、明確で予測可能な動作がほしい。そうすることで、複数のチェックやコマンドを組み合わせられる。

#### 受け入れ基準

1. 単一のフック内で複数のアクションが定義されている場合、システムはそれらを順番に実行すべきである
2. 複数アクションの場合、システムは`{"continue": true, "decision": "allow"}`から開始すべきである
3. 各アクションの実行後、システムは以下のルールでJSON出力を更新すべきである：
   a. アクションのJSON出力を取得
   b. 以下のルールで更新：
      - `continue`フィールド: 常にtrue（UserPromptSubmitでは固定値）
      - `decision`フィールド: 後のアクションの値で上書き。ただし、`decision: "block"`が設定された場合は後続のアクションを実行せず即座に処理を終了する（early return）
      - `hookSpecificOutput.hookEventName`: 一度設定されたら保持する（後続アクションが`hookSpecificOutput`を返さない場合でも削除しない）
      - `hookSpecificOutput.additionalContext`: 改行文字（`\n`）で連結
      - `systemMessage`: 改行文字（`\n`）で連結（エラーメッセージを保持）
      - その他のフィールド: 後のアクションの値で上書き
4. いずれかのアクションがエラー（コマンド実行失敗や無効なJSON出力）を返した場合、システムは要求3の基準5-7に従ってエラー用のJSON出力（`decision: "block"` + `systemMessage`）を生成すべきである
5. 全てのアクションが完了した後、システムは最終的なJSON出力を返すべきである
6. `additionalContext`および`systemMessage`の連結時、各値の間に改行文字（`\n`）を挿入すべきである（Claude Code仕様: "Multiple hooks' `additionalContext` values are concatenated."に準拠）

### 要求6: Phase 1構造体の再利用

**ユーザーストーリー:** cchook開発者として、Phase 1で確立したJSON出力構造を可能な限り再利用したい。そうすることで、コードの保守性と一貫性が向上する。

#### 受け入れ基準

1. `ActionOutput`構造体をPhase 1から再利用すべきである（SessionStart、UserPromptSubmit共通の内部型）
2. `ActionOutput`構造体に`Decision`フィールド（string型）を追加すべきである
3. 既存の`ExecuteSessionStartAction`メソッドは変更せず、そのまま維持すべきである
4. 新規の`ExecuteUserPromptSubmitAction`メソッドを追加すべきである（`ExecuteSessionStartAction`と同様のシグネチャ）
5. 既存のテンプレート処理、エラーハンドリング、コマンド実行ロジックを再利用すべきである

## 非機能要件

### コードアーキテクチャとモジュール性

- **単一責任原則**:
  - `types.go`: UserPromptSubmitのJSON出力構造を追加定義（Phase 1の構造体を拡張）
  - `executor.go`: UserPromptSubmit用のJSON出力生成ロジック（ActionExecutor.ExecuteUserPromptSubmitAction）
  - `hooks.go`: UserPromptSubmit用のフック実行関数（executeUserPromptSubmitHooks）
  - `main.go`: UserPromptSubmitケースを追加してJSONシリアライズ

- **モジュラー設計**:
  - Phase 1のActionOutput構造体を拡張して再利用
  - テンプレート処理、コマンド実行、エラーハンドリングはPhase 1のロジックを再利用
  - イベント固有のロジックのみを新規実装

- **依存関係管理**:
  - Phase 1で導入した依存関係をそのまま使用（新規依存なし）
  - JSON出力のシリアライズには標準ライブラリの`encoding/json`を使用

- **明確なインターフェース**:
  - UserPromptSubmit出力構造が実装する`UserPromptSubmitOutput`型を定義
  - Phase 1と同様のJSON出力生成フローを維持

### パフォーマンス

- **シリアライズオーバーヘッド**: JSONマーシャリングはフック実行時間に< 1ms追加すべき（Phase 1と同等）
- **メモリ使用量**: 出力構造は現在のメモリフットプリントを5%以上増加させるべきではない（Phase 1と同等の制約）

### セキュリティ

- **入力バリデーション**: 全てのYAMLフィールドはインジェクション攻撃を防ぐため使用前にバリデートされなければならない（Phase 1と同様）
- **コマンド実行**: 既存の`use_stdin`保護はシェルコマンドの安全性のために維持される（Phase 1と同様）

### 信頼性

- **エラーハンドリング**: 全てのJSONシリアライズエラーはキャッチされ明確に報告されなければならない（Phase 1と同様）
- **グレースフルデグラデーション**: JSONシリアライズが失敗した場合、プレーンテキストエラーを標準エラー出力に出力しコード1で終了（Phase 1と同様）
- **テストカバレッジ**: UserPromptSubmit関連のテストカバレッジをPhase 1と同等以上に維持

### スキーマ準拠性

- **JSON Schema準拠**: UserPromptSubmitのJSON出力はClaude Code公式スキーマに完全準拠しなければならない（https://docs.claude.com/en/docs/claude-code/hooks）
- **自動検証**: JSON出力構造はJSON Schemaバリデーションによって自動検証されなければならない（Phase 1と同じパターン）
- **必須フィールド検証**: `hookSpecificOutput`フィールドの存在を保証し、その中の`hookEventName`が常に存在し値が`"UserPromptSubmit"`であることを保証
- **型安全性**: 全フィールドの型（boolean, string等）が公式スキーマと一致することを保証
- **決定値の検証**: `decision`フィールドの値が"allow"または"block"のみであることを保証

### ユーザビリティ

- **ドキュメント**: UserPromptSubmit用の新しい設定形式と例でCLAUDE.mdを更新
- **移行ガイド**: UserPromptSubmitの既存設定から新しいJSON形式への移行例を提供
- **Phase 1との一貫性**: SessionStartと同じ設定パターンを維持し、学習コストを最小化

## 将来の拡張

Phase 2（UserPromptSubmit）の成功後、Phase 3として以下に進む:

**Phase 3**: PreToolUse
- 最も複雑なブロック判定
- `permissionDecision`（allow/deny/ask）の実装
- `updatedInput`機能（ツール入力の動的変更）
- SessionStart/UserPromptSubmitで確立したパターンを拡張

**Phase 4以降**: PostToolUse, Stop, SubagentStop, Notification, PreCompact, SessionEnd
