# Technology Stack

## Project Type
CLIツール - Claude Codeのフック処理を行うコマンドラインアプリケーション

## Core Technologies

### Primary Language(s)
- **Language**: Go 1.21以上
- **Runtime/Compiler**: Go標準コンパイラ
- **Language-specific tools**: go mod (依存関係管理), go test (テスト), go build (ビルド)

### Key Dependencies/Libraries
- **gopkg.in/yaml.v3**: YAML設定ファイルのパース
- **github.com/itchyny/gojq**: JQクエリエンジン（テンプレート処理用）
- **encoding/json**: JSON入力の処理（標準ライブラリ）
- **os/exec**: シェルコマンド実行（標準ライブラリ）

### Application Architecture
イベント駆動型のパイプラインアーキテクチャ：
1. **入力層**: stdin からJSONイベントを受信
2. **パース層**: イベントタイプごとの専用パーサー
3. **評価層**: 条件マッチングとフック選択
4. **実行層**: アクション実行とテンプレート処理
5. **出力層**: stdout/stderrへの結果出力

### Data Storage
- **Primary storage**: ファイルベース（YAML設定ファイル）
- **Caching**: gojqクエリのコンパイル結果をメモリキャッシュ
- **Data formats**: YAML（設定）、JSON（入力/内部処理）

### External Integrations
- **Claude Code Hooks**: stdin/stdout経由のJSONベース通信
- **Shell Commands**: os/exec経由のコマンド実行
- **Transcript Files**: Claude Codeが生成するJSONログファイルの読み取り

## Development Environment

### Build & Development Tools
- **Build System**: Go標準ツールチェーン（go build）
- **Package Management**: go mod
- **Development workflow**: コード変更 → go build → ローカルテスト

### Code Quality Tools
- **Static Analysis**: golangci-lint
- **Formatting**: gofmt（Go標準フォーマッター）
- **Testing Framework**: Go標準testingパッケージ
- **Documentation**: GoDoc形式のコメント
- **Pre-commit hooks**: pre-commitフレームワーク

### Version Control & Collaboration
- **VCS**: Git
- **Branching Strategy**: GitHub Flow
- **Code Review Process**: プルリクエストベースのレビュー

## Deployment & Distribution
- **Target Platform(s)**: macOS, Linux, Windows（Goクロスコンパイル対応）
- **Distribution Method**: go install または バイナリダウンロード
- **Installation Requirements**: Go 1.21以上（ソースからビルドの場合）
- **Update Mechanism**: 手動更新（go install -u）

## Technical Requirements & Constraints

### Performance Requirements
- イベント処理レイテンシ: 100ms以内
- メモリ使用量: 50MB以下
- 起動時間: 10ms以内
- Transcript解析: 10,000エントリを1秒以内

### Compatibility Requirements  
- **Platform Support**: macOS, Linux, Windows (AMD64, ARM64)
- **Dependency Versions**: Go 1.21以上、YAML v3
- **Standards Compliance**: POSIX準拠のシェルコマンド実行

### Security & Compliance
- **Security Requirements**: 
  - ファイルシステムアクセスは設定と transcript に限定
  - シェルコマンド実行時のインジェクション防止
  - 機密情報のログ出力防止
- **Threat Model**: 信頼できない入力からのコマンドインジェクション

### Scalability & Reliability
- **Expected Load**: 1ユーザー、秒間最大10イベント
- **Availability Requirements**: ローカルツールのため高可用性不要
- **Growth Projections**: 設定の複雑性が増加してもパフォーマンス維持

## Technical Decisions & Rationale

### Decision Log
1. **Go言語選択**: 高速起動、単一バイナリ配布、クロスプラットフォーム対応
2. **イベント駆動アーキテクチャ**: Claude Codeのフック仕様に最適、拡張性確保
3. **YAMLベース設定**: 人間が読み書きしやすく、JSONより表現力豊富
4. **gojq使用**: 純Go実装でjq互換、外部依存なし
5. **Opaque Struct Pattern**: 条件タイプの型安全性とコンパイル時チェック
6. **Sentinel Error Pattern**: 条件評価の明確なエラーハンドリング

## Known Limitations

- **JQクエリの制限**: gojqの実装制限により一部のjq機能が使用不可
- **並行処理なし**: 現在はシングルスレッド実行のみ
- **設定のホットリロードなし**: 設定変更時は再起動が必要
- **Windows対応**: テストカバレッジが不完全