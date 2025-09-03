# Project Structure

## Directory Organization

```
cchook/
├── *.go                    # メインソースコード（フラット構造）
├── *_test.go              # テストファイル
├── go.mod                 # Go依存関係定義
├── go.sum                 # 依存関係のチェックサム
├── README.md              # プロジェクトドキュメント
├── CLAUDE.md              # Claude Code用の指示書
├── LICENSE                # ライセンス情報
├── .gitignore             # Git無視ファイル
├── .pre-commit-config.yaml # pre-commit設定
└── .spec-workflow/        # spec-workflow生成ファイル
    ├── steering/          # プロジェクト方針ドキュメント
    └── specs/             # 機能仕様書
```

## Naming Conventions

### Files
- **ソースコード**: `snake_case.go` (例: `template_jq.go`, `utils.go`)
- **テストファイル**: `[filename]_test.go` (例: `hooks_test.go`)
- **エントリーポイント**: `main.go`

### Code
- **構造体/型**: `PascalCase` (例: `PreToolUseInput`, `HookConfig`)
- **インターフェース**: `PascalCase` (例: `Action`, `Condition`)
- **関数**: `camelCase` (例: `executeHooks`, `checkCondition`)
- **定数**: `PascalCase` (例: `ErrConditionNotHandled`)
- **変数**: `camelCase` (例: `hookConfig`, `jsonData`)
- **パッケージ**: `main` (単一パッケージ)

## Import Patterns

### Import Order
```go
import (
    // 標準ライブラリ
    "encoding/json"
    "fmt"
    "os"
    
    // 外部ライブラリ
    "github.com/itchyny/gojq"
    "gopkg.in/yaml.v3"
)
```

## Code Structure Patterns

### ファイル別の責務

- **main.go**: CLIエントリーポイント、引数処理
- **types.go**: すべての型定義、構造体、インターフェース
- **config.go**: YAML設定読み込み
- **parser.go**: JSON入力パース
- **hooks.go**: フック実行ロジック
- **actions.go**: アクション実行
- **template_jq.go**: テンプレート処理
- **utils.go**: 条件チェック、ユーティリティ関数

### 関数の構造パターン
```go
func checkCondition(condition Condition, data json.RawMessage) (bool, error) {
    // 1. 入力検証
    if condition.Type == "" {
        return false, fmt.Errorf("condition type is required")
    }
    
    // 2. 条件タイプごとの処理
    switch condition.Type {
    case FileExists:
        // 処理ロジック
    default:
        // Sentinel errorパターン
        return false, ErrConditionNotHandled
    }
    
    // 3. 結果返却
    return result, nil
}
```

## Code Organization Principles

1. **単一責任**: 各ファイルは明確な1つの責務を持つ
2. **フラット構造**: シンプルなプロジェクトなのでパッケージ分割しない
3. **テスタビリティ**: 各関数は独立してテスト可能
4. **エラーハンドリング**: `(result, error)` パターンを一貫して使用

## Module Boundaries

- **設定層**: config.go (YAML読み込みのみ)
- **入力層**: parser.go (JSON処理のみ)
- **ビジネスロジック**: hooks.go, utils.go (条件評価とフック選択)
- **実行層**: actions.go (副作用のある処理)
- **型定義**: types.go (他のすべてのファイルから参照)

## Code Size Guidelines

- **ファイルサイズ**: 最大500行
- **関数サイズ**: 最大50行（複雑な switch 文を除く）
- **ネストの深さ**: 最大3レベル
- **構造体フィールド数**: 最大10フィールド

## 新機能追加パターン

### 新しいConditionタイプの追加
1. `types.go`: ConditionTypeに新しい定数を追加
2. `utils.go`: 対応する`check*Condition`関数を追加
3. `hooks.go`: switch文にケースを追加
4. `*_test.go`: テストケースを追加

### 新しいイベントタイプの追加
1. `types.go`: 新しいInput構造体とHook構造体を定義
2. `parser.go`: パース関数を追加
3. `hooks.go`: 実行関数を追加
4. `main.go`: CLIオプションに追加

## Documentation Standards
- すべての公開関数にGoDocコメント
- 複雑なロジックにインラインコメント
- CLAUDE.mdに開発者向けガイド
- README.mdにユーザー向けドキュメント