# Technology Stack

## Project Type

cchookは、Claude Codeのフック設定を管理する**CLIツール**です。標準入力からJSONを受け取り、YAML設定に基づいて条件チェックとアクション実行を行います。

## Core Technologies

### Primary Language(s)
- **Language**: Go 1.24.5（go.modで確認）
- **Runtime/Compiler**: Go compiler (gc)
- **Language-specific tools**:
  - `go mod`: 依存関係管理
  - `go build`: バイナリビルド
  - `go test`: テスト実行
  - `go install`: インストール

### Key Dependencies/Libraries

以下はgo.modから確認した依存関係：

- **github.com/itchyny/gojq v0.12.17**:
  - テンプレートエンジンのバックエンド
  - jqクエリのパースと実行

- **gopkg.in/yaml.v3 v3.0.1**:
  - YAML設定ファイルのパース

- **github.com/go-git/go-git/v5 v5.16.2**:
  - Git操作
  - `git_tracked_file_operation` 条件のためのファイル追跡チェック

### Application Architecture

**イベント駆動アーキテクチャ**:

1. **入力処理**: stdin からJSON入力を受け取り、イベントタイプに応じてパース
2. **設定読み込み**: YAML設定ファイルから該当イベントのフック定義を取得
3. **マッチング**: ツール名のマッチャーと条件の評価
4. **アクション実行**: 条件を満たした場合、設定されたアクション（コマンド実行、出力）を実行
5. **終了制御**: ExitErrorを使った終了ステータスとstdout/stderrの制御

**モジュラー設計**（実際のファイル構成）:
- `main.go`: エントリーポイント、CLI引数処理
- `types.go`: 型定義（イベント、フック、アクション、条件、CommandRunnerインターフェース）
- `parser.go`: JSON入力の解析
- `config.go`: YAML設定の読み込み
- `hooks.go`: フック実行ロジック
- `actions.go`: アクション実行ロジック（依存性注入対応）
- `utils.go`: 条件チェックユーティリティ、CommandRunner実装
- `template_jq.go`: テンプレート処理
- `errors.go`: カスタムエラー型

**依存性注入パターン**:
- `CommandRunner`インターフェースによる抽象化
  - コマンド実行ロジックをインターフェース化
  - 本番環境: `realCommandRunner`による実際のシェルコマンド実行
  - テスト環境: `stubRunner`によるモック実装
- `commandRunner`パッケージ変数による実行時の切り替え
  - デフォルトで`DefaultCommandRunner`を使用
  - テストでは`stubRunner`に置き換え可能

### Data Storage

- **Primary storage**: ファイルシステム（YAML設定ファイル）
- **Data formats**:
  - 入力: JSON（Claude Codeから）
  - 設定: YAML
  - 内部処理: Go構造体

### External Integrations

- **Claude Code**:
  - Protocol: stdin/stdoutを通じたJSON入出力
  - Integration: Claude Codeのフックシステムと統合

- **Shell Commands**:
  - アクション実行時にシェルコマンドを呼び出し
  - `use_stdin: true`で全JSONデータをコマンドの標準入力に渡す機能をサポート
  - シェルエスケープ問題を回避し、改行・クォート・特殊文字を含む複雑なデータを安全に処理

- **Git**:
  - go-gitライブラリを通じた読み取り専用のGit操作

## Development Environment

### Build & Development Tools

- **Build System**:
  - `go build`: バイナリのビルド
  - Makefileは存在しない（確認済み）

- **Package Management**:
  - `go mod`: Go modules による依存関係管理
  - `go mod download`: 依存関係のダウンロード
  - `go mod tidy`: 不要な依存の削除

- **Development workflow**:
  - ドライランモード（`-command` フラグ）で設定のテスト

### Code Quality Tools

以下は.pre-commit-config.yamlから確認：

- **Static Analysis**:
  - `golangci-lint` v2.3.0: Go linter

- **Formatting**:
  - `gofmt`: 標準のGoフォーマッター

- **Testing Framework**:
  - Go標準の`testing`パッケージ
  - Table-driven tests パターンを採用（hooks_test.goで確認）
  - カバレッジ: 49.7%（実測値）
  - **2層テストアーキテクチャ**:
    - **ユニットテスト** (*_test.go):
      - 外部依存なし、高速実行
      - `stubRunner`によるコマンド実行のモック化
      - `go test ./...`で実行
    - **統合テスト** (*_integration_test.go):
      - 実際のシェルコマンド実行（cat, jq等が必要）
      - `//go:build integration`タグで分離
      - `go test -tags=integration ./...`で実行

- **Documentation**:
  - README.md
  - CLAUDE.md（プロジェクト固有の開発ガイドライン）

- **Pre-commit hooks**:
  - end-of-file-fixer
  - trailing-whitespace
  - check-json
  - detect-private-key
  - debug-statements
  - actionlint v1.7.7

### Version Control & Collaboration

- **VCS**: Git（確認済み）
- **Code Review Process**:
  - GitHub Pull Requests（.github/workflowsの存在から推測）
  - pre-commitフックによる自動チェック

## Deployment & Distribution

- **Distribution Method**（READMEから確認）:
  - `go install github.com/syou6162/cchook@latest`
  - ソースからのビルド

- **Installation Requirements**:
  - Go 1.24.5 以降（ソースビルドの場合）

- **Configuration File Path**（config.goから確認）:
  - XDG_CONFIG_HOME準拠
  - デフォルト: `~/.config/cchook/config.yaml`
  - カスタムパス: `-config` フラグで指定可能

## Technical Requirements & Constraints

### Performance Characteristics

以下は実測値（darwin/arm64環境）:

- **起動時間**: 約0.03秒（30ms）
- **メモリ使用量**: 約9.8MB（maximum resident set size）
- **バイナリサイズ**: 10MB
- **テストカバレッジ**: 49.7%

### Compatibility Requirements

- **Platform Support**（確認済み）:
  - Go 1.24.5以降
  - 現在のバイナリ: darwin/arm64 Mach-O 64-bit executable

- **Dependency Versions**（go.modから）:
  - gojq v0.12.17
  - yaml.v3 v3.0.1
  - go-git/v5 v5.16.2

- **Standards Compliance**（コードから確認）:
  - XDG Base Directory Specification（config.goで実装確認）

## CI/CD

以下は.github/workflowsから確認：

- **GitHub Actions**:
  - ubuntu-latest
  - Go 1.25.3
  - `go test -v ./...`（go-test.yml）
  - `go build -v ./...`（go-build.yml）

## Technical Decisions & Rationale

### Decision Log

1. **Go言語の採用**:
   - 単一バイナリ配布
   - 高速な起動時間（実測30ms）
   - 標準ライブラリが充実

2. **gojqの採用**:
   - jq互換のクエリ言語
   - Goネイティブ実装で外部依存なし

3. **YAML設定**:
   - 人間が読み書きしやすい
   - コメントとマルチライン対応

4. **イベント駆動アーキテクチャ**:
   - Claude Codeのイベントモデルに対応
   - イベントごとに異なる条件とアクションを柔軟に定義

5. **XDG Base Directory準拠**:
   - Linux/Unixの標準的な設定ファイル配置
   - config.goで実装確認済み

6. **依存性注入パターンの導入**:
   - コマンド実行をインターフェース化してテスト可能性を向上
   - `CommandRunner`インターフェースによる抽象化
   - 本番環境とテスト環境で実装を切り替え可能

7. **2層テストアーキテクチャ**:
   - ユニットテストと統合テストを明確に分離
   - Build tagsによる統合テストの制御
   - CI/CD環境での柔軟なテスト実行
   - テスト実行速度の向上（ユニットテストは外部依存なし）

## Known Limitations

確認できている制約：

- **テストカバレッジ**: 現在49.7%（実測値）
