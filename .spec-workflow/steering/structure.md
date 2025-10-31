# Project Structure

## Directory Organization

cchookは**フラットな構造**を採用しています：

```
cchook/
├── .claude/              # Claude Code設定
├── .github/              # GitHub Actions ワークフロー
│   └── workflows/
├── data/                 # データファイル
├── examples/             # 使用例
├── *.go                  # ソースファイル
├── *_test.go             # テストファイル
├── go.mod / go.sum       # Go modules
├── .pre-commit-config.yaml
└── README.md / CLAUDE.md
```

**構造の特徴**:
- サブディレクトリなし、全てのGoファイルがルートに配置
- パッケージは`main`のみ（単一バイナリ）
- シンプルなCLIツールに適した構造

## File Organization

### Module Separation

機能ごとにファイルを分割：

- `main.go`: エントリーポイント、CLI引数処理
- `types.go`: 型定義（イベント、フック、アクション、条件）
- `parser.go`: JSON入力の解析
- `config.go`: YAML設定の読み込み
- `hooks.go`: フック実行ロジック
- `actions.go`: アクション実行ロジック
- `utils.go`: 条件チェックユーティリティ
- `template_jq.go`: テンプレート処理
- `errors.go`: カスタムエラー型

### File Naming Conventions

- **ソースファイル**: `[module].go`（例: `types.go`, `hooks.go`）
- **テストファイル**: `[module]_test.go`（例: `types_test.go`, `hooks_test.go`）
- **設定ファイル**: 小文字 + ハイフン（例: `.pre-commit-config.yaml`）

## Naming Conventions

### Code Naming

- **Types**: PascalCase（例: `HookEventType`, `PreToolUseInput`）
- **Constants**: PascalCase（例: `PreToolUse`, `PostToolUse`）
- **Functions (private)**: camelCase（例: `loadConfig`, `executeAction`）
- **Functions (exported)**: PascalCase（例: `GetEventType`）
- **Interfaces**: PascalCase（例: `HookInput`, `Action`）
- **Variables**: camelCase

### Struct Naming Patterns

イベント固有の構造体は一貫した命名パターン：

- Input構造体: `[Event]Input`（例: `PreToolUseInput`, `PostToolUseInput`）
- Hook構造体: `[Event]Hook`（例: `PreToolUseHook`, `PostToolUseHook`）
- Action構造体: `[Event]Action`（例: `PreToolUseAction`, `PostToolUseAction`）

## Import Patterns

### Import Order

```go
import (
    // 標準ライブラリ（アルファベット順）
    "encoding/json"
    "flag"
    "fmt"
    "os"

    // 外部依存（必要な箇所でのみ）
)
```

## Code Structure Patterns

### File Organization

ファイル内の標準的な構成：

1. パッケージ宣言
2. インポート
3. 型定義（enum、const）
4. インターフェース定義
5. 構造体定義
6. メソッド実装

### Function Organization

関数内の標準的な構成：

1. 引数/フラグのパース
2. バリデーション（エラーチェック）
3. 早期リターン（エラー処理）
4. メインロジック
5. エラーハンドリング

## Error Handling Patterns

- **Custom Error**: `ExitError` 型で終了コードと出力先を制御
- **Sentinel Errors**: `ErrConditionNotHandled` で特定の条件を識別
- **Error Return**: `(result, error)` パターン
- **Early Returns**: エラー時は早期リターン

## Testing Patterns

- **Table-driven tests**: テストケースを構造体のスライスで定義
- **Test naming**: `Test[FunctionName]_[Scenario]`
- **Integration tests**: `_Integration` サフィックス

例：

```go
func TestShouldExecutePreToolUseHook(t *testing.T) {
    tests := []struct {
        name    string
        hook    PreToolUseHook
        input   *PreToolUseInput
        want    bool
        wantErr bool
    }{
        {
            "Match with no conditions",
            PreToolUseHook{Matcher: "Write"},
            &PreToolUseInput{ToolName: "Write"},
            true,
            false,
        },
        // ...
    }
    // ...
}
```

## Code Organization Principles

1. **Single Responsibility**:
   - 各ファイルは単一の責任を持つ
   - `types.go`: 型定義のみ
   - `config.go`: 設定読み込みのみ

2. **Modularity**:
   - 機能ごとにファイルを分割
   - 再利用可能な関数をutilsに配置

3. **Testability**:
   - 各モジュールに対応するテストファイル

4. **Consistency**:
   - イベント固有の型は同じ命名パターン
   - 全ファイルで同じコード構成パターン

## Design Patterns

- **Opaque Struct Pattern**: `ConditionType` で型安全性を確保
- **Embedded Structs**: `BaseInput` を全イベント入力型に埋め込み
- **Interface Satisfaction**: 暗黙的なインターフェース実装
- **Generic Functions**: 型制約を使用

## Module Boundaries

**依存方向**:
```
実行層 (hooks.go, actions.go)
    ↓
ユーティリティ層 (utils.go, template_jq.go)
    ↓
コア型 (types.go)
```

- **Core Types** (`types.go`): 全モジュールが参照
- **Input Processing** (`parser.go`): 入力層
- **Configuration** (`config.go`): 設定層
- **Execution** (`hooks.go`, `actions.go`): 実行層
- **Utilities** (`utils.go`): 共通ユーティリティ
- **Template Engine** (`template_jq.go`): テンプレート処理

一方向の依存関係を維持。

## Comments and Documentation

- **実装詳細**: 日本語コメント（例: `// イベントタイプの妥当性検証`）
- **Exported Functions/Types**: 英語ドキュメント
- **Self-documenting Code**: 過度なコメントは避ける

例：

```go
// イベントタイプのenum定義
type HookEventType string

// イベントタイプの妥当性検証
func (e HookEventType) IsValid() bool {
```

## Configuration Standards

- **YAML**: 設定ファイル形式
- **Indentation**: 2スペース
- **Field naming**: snake_case（JSONタグ）
