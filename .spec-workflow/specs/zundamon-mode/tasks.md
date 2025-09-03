# ずんだもんモード - タスクドキュメント

- [x] 1. 条件タイプの定義を追加
  - File: types.go
  - ConditionRandomChance定数を追加
  - UnmarshalYAMLメソッドに"random_chance"ケースを追加
  - Purpose: 新しい条件タイプをシステムに認識させる
  - _Requirements: 1.1_

- [x] 2. 乱数シードの初期化
  - File: main.go
  - init()関数またはmain()関数の先頭でrand.Seed()を呼び出し
  - time.Now().UnixNano()をシードとして使用
  - Purpose: 乱数生成の適切な初期化
  - _Requirements: 2.1_

- [x] 3. 確率判定関数の実装
  - File: utils.go
  - checkRandomChance(value string) (bool, error)関数を実装
  - 0-100の範囲チェックとエラーハンドリング
  - rand.Intn(100)を使用した確率判定ロジック
  - Purpose: 確率ベースの条件判定機能を提供
  - _Requirements: 1.2, 1.3_

- [x] 4. 既存の条件チェック関数への統合
  - File: utils.go
  - checkUserPromptSubmitCondition関数のswitch文にConditionRandomChanceケースを追加
  - checkRandomChance関数を呼び出す
  - Purpose: UserPromptSubmitイベントで新しい条件を使用可能にする
  - _Leverage: 既存のcheckUserPromptSubmitCondition関数_
  - _Requirements: 1.4_

- [x] 5. 単体テストの作成
  - File: utils_test_random.go (新規作成)
  - TestCheckRandomChance関数で境界値テストを実装
  - TestCheckRandomChanceDistribution関数で統計的検証を実装
  - エラーケースのテスト（負の値、100超、非数値）
  - Purpose: 確率判定ロジックの正確性を保証
  - _Requirements: 3.1_

- [x] 6. 統合テストの作成
  - File: hooks_test_random.go (新規作成)
  - TestExecuteUserPromptSubmitHooksWithRandomChance関数を実装
  - random_chance条件を含む設定でのフック実行テスト
  - 他の条件との組み合わせテスト
  - Purpose: エンドツーエンドの動作を検証
  - _Leverage: 既存のテストヘルパー関数_
  - _Requirements: 3.2_

- [x] 7. ドキュメントの更新
  - File: README.md
  - UserPromptSubmit条件セクションにrandom_chanceを追加
  - 使用例とYAML設定サンプルを追加
  - 確率値の範囲（0-100）と動作説明を記載
  - Purpose: ユーザー向けドキュメントを完備
  - _Requirements: 4.1_

- [x] 8. エンドツーエンドテストの実施
  - File: .claude/tmp/test-config-random.yaml (テスト用設定ファイル作成)
  - 確率100%と0%のケースでコマンドライン実行テスト
  - エラーケースの動作確認
  - Purpose: 実際の使用環境での動作を確認
  - _Requirements: 3.3_

## タスクの依存関係

- タスク1は全ての基盤となるため最初に実行
- タスク2は乱数生成の前提条件のため早期に実行
- タスク3と4は順番に実行（checkRandomChance関数を作成してから統合）
- タスク5と6はタスク3,4完了後に並行実行可能
- タスク7はいつでも実行可能
- タスク8は全タスク完了後に実施

## 完了基準

各タスクは以下の条件を満たした時点で完了とする：
- コードの実装が完了している
- コンパイルエラーがない
- 関連するテストが全てパスしている
- 必要に応じてドキュメントが更新されている
