# Requirements: SessionStart Hook JSON出力対応

## イントロダクション

cchookは現在、exit codeベースでフック出力を制御していますが、Claude Code公式のJSON出力形式への移行を段階的に進めます。

**Phase 1のスコープ**: SessionStartフックのみ

SessionStartは以下の理由で最初の実装対象として最適です:
- **最もシンプル**: 情報表示のみ、ブロック機能なし
- **影響範囲が小さい**: 実際の設定で4ルール使用中
- **仕様が明確**: Claude Code公式ドキュメントで`additionalContext`へのマッピングが明記
- **失敗時の影響が軽微**: 情報表示が出ないだけ、ツール実行には影響なし

**Phase 1で使用するフィールド**:
- `continue` (boolean): セッション起動制御
- `hookSpecificOutput.hookEventName` (string): 必須、常に"SessionStart"
- `hookSpecificOutput.additionalContext` (string): Claudeへのコンテキスト提供
- `systemMessage` (string): ユーザーへの警告メッセージ

**Phase 1で未使用のフィールド**:
- `stopReason` (string): JSON Schema準拠のため構造体に定義するが、Phase 1では値を設定しない（omitemptyで出力から省略）
- `suppressOutput` (boolean): JSON Schema準拠のため構造体に定義するが、Phase 1では値を設定しない（omitemptyで出力から省略）

これらの未使用フィールドは、他のフックイベント（PreToolUse、UserPromptSubmit等）への拡張時に実装される予定です。

Phase 1の成功後、他のフックイベント（PreToolUse、UserPromptSubmit等）に段階的に拡張します。

## Product Visionとの整合性

product.mdで定義された以下の目標に沿います:
- **シンプルさ**: YAML設定でJSON構造を意識せずに使える
- **保守性**: イベント固有の複雑さを隠蔽
- **拡張性**: 将来の全フック対応を見据えた設計

## 要求事項

### 要求1: JSON出力形式への移行

**ユーザーストーリー:** cchookユーザーとして、SessionStartフックでJSON形式の出力を利用したい。そうすることで、Claude Codeに情報を渡す方法がClaude Code標準仕様に準拠する。

#### 受け入れ基準

1. SessionStartフックが実行されるとき、システムはJSONを標準出力に出力すべきである
2. JSON出力が正常に生成されたとき、システムは終了コード0で終了すべきである
3. アクションが正常に完了したとき、システムはClaude Code仕様に従った有効なJSONとしてシリアライズすべきである
4. システムは`continue`フィールドを常に明示的に出力すべきである（Claude Codeのデフォルト値`true`に依存しない）
5. `type: output`アクションで`message`が空の場合、システムは`continue: false`を設定すべきである（設定エラーとして扱う）
6. `type: command`アクションで標準出力が空の場合でも、終了コード0であれば`continue: true`を許容すべきである（検証型CLIツール（fmt、linter、pre-commitなど）では問題がなければ何も出力せずexit 0で終了するため）

### 要求2: YAML設定形式の更新

**ユーザーストーリー:** cchookユーザーとして、`continue`フィールドを明示的に指定したい。そうすることで、設定ファイルがClaude Code仕様と整合する。

#### 受け入れ基準

1. `type: output`アクションで`continue`が指定されていない場合、システムは`continue: true`を設定すべきである（通常ケースのデフォルト値。情報表示後も処理を継続）

### 要求3: type: commandアクションのJSON出力処理

**ユーザーストーリー:** cchookユーザーとして、外部コマンドが返すJSON出力を直接利用したい。そうすることで、複雑な判定ロジックを外部プログラムに委譲できる。

#### 受け入れ基準

1. `type: command`アクションが実行されるとき、システムは常に終了コード0を期待すべきである
2. コマンドが終了コード0で成功したとき、システムは標準出力からJSON出力をパースすべきである
3. パースしたJSON出力に`continue`フィールドが存在しない場合、システムは`continue: false`を設定すべきである（フォールバック時のデフォルト値）
4. パースしたJSON出力に`hookSpecificOutput.hookEventName`フィールドが存在しない場合、システムは`{"continue": false, "hookSpecificOutput": {"hookEventName": "SessionStart"}, "systemMessage": "Command output is missing required field: hookSpecificOutput.hookEventName"}`というJSON出力を生成すべきである（外部コマンドの設定エラー）
5. コマンドが終了コード0以外で終了したとき（エラー時）、システムは`{"continue": false, "hookSpecificOutput": {"hookEventName": "SessionStart"}, "systemMessage": "Command failed with exit code X: <stderr>"}`というJSON出力を生成すべきである。`systemMessage`はユーザーにのみ表示される警告メッセージであり、Claudeには見えない（Claude Code仕様）
6. コマンドの標準出力が有効なJSONでない場合（エラー時）、システムは`{"continue": false, "hookSpecificOutput": {"hookEventName": "SessionStart"}, "systemMessage": "Command output is not valid JSON: <output>"}`というJSON出力を生成すべきである。`systemMessage`はユーザーにのみ表示される
7. コマンドの標準出力が空の場合、システムは`continue: true`で成功として扱うべきである（検証型CLIツール（fmt、linter、pre-commitなど）では問題がなければ何も出力せずexit 0で終了するため）。この場合、`hookSpecificOutput.additionalContext`は空になるため、Claudeには情報が提供されない
8. コマンドが返すJSON出力にSessionStartでサポートされていないフィールド（例: `permissionDecision`, `decision`）が含まれる場合、システムは警告"Warning: Field 'xxx' is not supported for SessionStart hooks"を標準エラー出力に書き込み、該当フィールドを無視すべきである

### 要求4: type: outputアクションのJSON変換

**ユーザーストーリー:** cchookユーザーとして、既存の`type: output`アクションを適切なJSONフィールドに自動変換してほしい。そうすることで、シンプルなケースでJSON構造を手動で指定する必要がなくなる。

#### 受け入れ基準

1. システムは`hookSpecificOutput.hookEventName`フィールドに`"SessionStart"`を設定すべきである（Claude Code仕様で必須）
2. `type: output`アクションの`message`フィールドは、システムによって`hookSpecificOutput.additionalContext`にマップされるべきである（SessionStartイベントの場合）
3. `message`が空文字列の場合、システムは要求1.5に従って`continue: false`を設定すべきである（設定エラーとして扱う）。`additionalContext`フィールドはJSONから省略される（omitempty）
4. `message`にテンプレート変数が存在する場合、システムはJSONフィールドに設定する前にそれらを処理すべきである

### 要求5: 複数アクションの処理

**ユーザーストーリー:** 単一のフックに複数のアクションを持つcchookユーザーとして、明確で予測可能な動作がほしい。そうすることで、複数のチェックやコマンドを組み合わせられる。

#### 受け入れ基準

1. 単一のフック内で複数のアクションが定義されている場合、システムはそれらを順番に実行すべきである
2. 複数アクションの場合、システムは`{"continue": true}`から開始すべきである
3. 各アクションの実行後、システムは以下のルールでJSON出力を更新すべきである：
   a. アクションのJSON出力を取得
   b. 以下のルールで更新：
      - `continue`フィールド: 後のアクションの値で上書き。ただし、`continue: false`が設定された場合は後続のアクションを実行せず即座に処理を終了する
      - `hookSpecificOutput.hookEventName`: 一度設定されたら保持する（後続アクションが`hookSpecificOutput`を返さない場合でも削除しない）
      - `hookSpecificOutput.additionalContext`: 改行文字（`\n`）で連結
      - `systemMessage`: 改行文字（`\n`）で連結（エラーメッセージを保持）
      - その他のフィールド: 後のアクションの値で上書き
4. いずれかのアクションがエラー（コマンド実行失敗や無効なJSON出力）を返した場合、システムは要求3の基準5-7に従ってエラー用のJSON出力（`continue: false` + `systemMessage`）を生成すべきである
5. 全てのアクションが完了した後、システムは最終的なJSON出力を返すべきである
6. `additionalContext`および`systemMessage`の連結時、各値の間に改行文字（`\n`）を挿入すべきである（Claude Code仕様: "Multiple hooks' `additionalContext` values are concatenated."に準拠）

## 非機能要件

### コードアーキテクチャとモジュール性

- **単一責任原則**:
  - `types.go`: SessionStartのJSON出力構造を定義
  - `executor.go`: SessionStart用のJSON出力生成ロジック（ActionExecutor.ExecuteSessionStartAction）
  - `main.go`: JSONをシリアライズして標準出力に書き込む

- **モジュラー設計**:
  - JSON出力構築はSessionStart専用関数に分離
  - 将来の他イベント対応を見据えた拡張可能な設計
  - テンプレート処理は新しいJSON出力構造で引き続き機能

- **依存関係管理**:
  - JSONスキーマバリデーション用に`github.com/xeipuuv/gojsonschema`を追加（実行時依存として使用）
  - JSON出力のシリアライズには標準ライブラリの`encoding/json`を使用

- **明確なインターフェース**:
  - SessionStart出力構造が実装する`SessionStartOutput`型を定義
  - YAML設定パースとJSON出力生成の間の明確な分離を維持

### パフォーマンス

- **シリアライズオーバーヘッド**: JSONマーシャリングはフック実行時間に< 1ms追加すべき（現在の30ms起動時間と比較して無視できる）
- **メモリ使用量**: 出力構造は現在のメモリフットプリントを5%以上増加させるべきではない（SessionStartのみ対応のため）

### セキュリティ

- **入力バリデーション**: 全てのYAMLフィールドはインジェクション攻撃を防ぐため使用前にバリデートされなければならない
- **コマンド実行**: 既存の`use_stdin`保護はシェルコマンドの安全性のために維持される

### 信頼性

- **エラーハンドリング**: 全てのJSONシリアライズエラーはキャッチされ明確に報告されなければならない
- **グレースフルデグラデーション**: JSONシリアライズが失敗した場合、プレーンテキストエラーを標準エラー出力に出力しコード1で終了
- **テストカバレッジ**: SessionStart関連のテストカバレッジを維持または改善

### スキーマ準拠性

- **JSON Schema準拠**: SessionStartのJSON出力はClaude Code公式スキーマに完全準拠しなければならない（https://docs.claude.com/en/docs/claude-code/hooks）
- **自動検証**: JSON出力構造はJSON Schemaバリデーションによって自動検証されなければならない（`github.com/invopop/jsonschema`を使って`SessionStartOutput`構造体から自動生成、単一の情報源を維持）
- **必須フィールド検証**: `hookSpecificOutput`フィールドの存在を保証し、その中の`hookEventName`が常に存在し値が`"SessionStart"`であることを保証
- **型安全性**: 全フィールドの型（boolean, string等）が公式スキーマと一致することを保証（構造体タグから自動生成）
- **将来の拡張性**: スキーマ自動生成パターンは他のフックタイプ（UserPromptSubmit, PreToolUse等）にも適用可能であること

### ユーザビリティ

- **ドキュメント**: SessionStart用の新しい設定形式と例でCLAUDE.mdを更新
- **移行ガイド**: SessionStartの既存設定から新しいJSON形式への移行例を提供

## 将来の拡張

Phase 1（SessionStart）の成功後、以下の順序で他のフックイベントに拡張:

**Phase 2**: UserPromptSubmit
- SessionStartと同じパターン（additionalContextマッピング）
- ブロック機能の追加（`decision: "block"`）

**Phase 3**: PreToolUse
- 最も複雑なブロック判定
- `permissionDecision`（allow/deny/ask）の実装
- `updatedInput`機能

**Phase 4以降**: PostToolUse, Stop, SubagentStop, Notification, PreCompact, SessionEnd
