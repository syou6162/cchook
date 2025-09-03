# ずんだもんモード - 設計書

## 概要

ずんだもんモードは、UserPromptSubmitイベントに対して確率的に発動する機能です。新しい条件タイプ`random_chance`を追加し、指定された確率でアクションを実行できるようにします。この機能により、AIアシスタントの応答にランダムな要素を加え、作業の単調さを軽減します。

## Steering Document との整合性

### 技術標準 (tech.md)
- 既存のConditionTypeパターン（Opaque Struct Pattern）に従って実装
- エラーハンドリングはセンチネルエラーパターンを使用
- テストカバレッジ80%以上を維持

### プロジェクト構造 (structure.md)
- `types.go`: 新しい条件タイプの定義を追加
- `utils.go`: 確率判定ロジックの実装（既存の実装パターンに従う）
- `utils_test_random.go`: 新機能の単体テスト
- `hooks_test_random.go`: 統合テスト
- `README.md`: 使用例とドキュメントの更新

## コード再利用分析

### 既存コンポーネントの活用
- **ConditionType**: 既存のOpaque Struct Patternを使用して新しい条件タイプを定義
- **checkUserPromptSubmitCondition**: 既存の条件チェック関数に新しいケースを追加
- **processTemplate**: テンプレート処理機能をそのまま活用
- **parseUserPromptSubmitInput**: 既存のパーサーをそのまま使用

### 統合ポイント
- **UserPromptSubmitイベント処理**: 既存のフローに確率判定を追加
- **YAML設定**: 既存の設定構造を維持したまま新しい条件タイプを追加
- **エラーハンドリング**: 既存のセンチネルエラーパターンに従う

## アーキテクチャ

### モジュール設計原則
- **単一責任の原則**: 確率判定機能を単一の関数に封じ込める
- **シンプルな実装**: 過度な抽象化を避け、必要最小限の実装に留める
- **既存パターンの踏襲**: 他の条件タイプと同じ実装パターンを使用

```mermaid
graph TD
    A[UserPromptSubmitイベント] --> B[executeUserPromptSubmitHooks]
    B --> C[条件チェック]
    C --> D[checkUserPromptSubmitCondition]
    D --> E{random_chance?}
    E -->|Yes| F[rand.Intn(100)]
    F --> G{確率判定}
    G -->|成功| H[アクション実行]
    G -->|失敗| I[スキップ]
    E -->|No| J[他の条件チェック]
```

## コンポーネントとインターフェース

### 確率判定ロジック（シンプル版）
- **目的:** 指定された確率で条件を成立させる
- **実装方針:** 既存の`isPrime`関数と同レベルのシンプルさを維持
- **関数シグネチャ:**
  ```go
  func checkRandomChance(value string) (bool, error)
  ```
- **依存関係:** math/rand パッケージ（標準ライブラリ）
- **再利用:** 既存のcheckUserPromptSubmitCondition関数に統合

### 初期化処理
- **目的:** 乱数シードの適切な初期化
- **実装場所:** main.goのinit()関数
- **依存関係:** time パッケージ
- **実装:**
  ```go
  func init() {
      rand.Seed(time.Now().UnixNano())
  }
  ```

## データモデル

### 新しい条件タイプ定義
```go
// types.go に追加
var (
    ConditionRandomChance = ConditionType{"random_chance"}
)

// UnmarshalYAML に追加
case "random_chance":
    *c = ConditionRandomChance
```

### YAML設定構造（既存構造を維持）
```yaml
UserPromptSubmit:
  - conditions:
      - type: random_chance
        value: "10"  # 0-100の整数値
    actions:
      - type: output
        message: "ずんだもんモード発動なのだ！"
```

## エラーハンドリング

### エラーシナリオ

1. **無効な確率値（負の数）:**
   - **処理:** エラーを返す: "invalid random_chance value: -10 (must be 0-100)"
   - **ユーザーへの影響:** フックが実行されずにエラーメッセージが表示される

2. **無効な確率値（100超）:**
   - **処理:** エラーを返す: "invalid random_chance value: 150 (must be 0-100)"
   - **ユーザーへの影響:** フックが実行されずにエラーメッセージが表示される

3. **数値以外の値:**
   - **処理:** エラーを返す: "invalid random_chance value: abc (must be integer)"
   - **ユーザーへの影響:** フックが実行されずにエラーメッセージが表示される

## テスト戦略

### 単体テスト（utils_test_random.go）
- **checkRandomChance関数のテスト:**
  - 境界値テスト（0%, 100%）
  - エラー処理のテスト（負の値、100超、非数値）
  - 通常の確率値のテスト（統計的な検証）

### 統合テスト（hooks_test_random.go）
- **executeUserPromptSubmitHooksのテスト:**
  - random_chance条件を含む設定でのフック実行
  - 他の条件との組み合わせテスト
  - 複数のrandom_chance条件の独立性テスト

### エンドツーエンドテスト
- **実際のコマンドラインでのテスト:**
  ```bash
  # テスト用設定ファイルで確率100%に設定
  echo '{"session_id":"test","hook_event_name":"UserPromptSubmit","prompt":"テスト"}' | ./cchook -event UserPromptSubmit -config test-config.yaml
  ```

## 実装の詳細

### 確率判定の実装（シンプル版）
```go
// utils.go
func checkRandomChance(value string) (bool, error) {
    probability, err := strconv.Atoi(value)
    if err != nil {
        return false, fmt.Errorf("invalid random_chance value: %s (must be integer)", value)
    }
    
    if probability < 0 || probability > 100 {
        return false, fmt.Errorf("invalid random_chance value: %d (must be 0-100)", probability)
    }
    
    if probability == 0 {
        return false, nil
    }
    if probability == 100 {
        return true, nil
    }
    
    // 0-99の乱数を生成し、probability未満なら条件成立
    randomValue := rand.Intn(100)
    return randomValue < probability, nil
}
```

### 既存関数への統合
```go
// utils.go の checkUserPromptSubmitCondition に追加
case ConditionRandomChance:
    return checkRandomChance(condition.Value)
```

### テスト用のヘルパー関数
```go
// utils_test_random.go
func TestCheckRandomChance(t *testing.T) {
    tests := []struct {
        name    string
        value   string
        wantErr bool
    }{
        {"valid 0%", "0", false},
        {"valid 100%", "100", false},
        {"valid 50%", "50", false},
        {"invalid negative", "-1", true},
        {"invalid over 100", "101", true},
        {"invalid non-integer", "abc", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := checkRandomChance(tt.value)
            if (err != nil) != tt.wantErr {
                t.Errorf("checkRandomChance() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

// 統計的な検証
func TestCheckRandomChanceDistribution(t *testing.T) {
    // 50%の確率で1000回試行
    trueCount := 0
    for i := 0; i < 1000; i++ {
        result, _ := checkRandomChance("50")
        if result {
            trueCount++
        }
    }
    
    // 期待値は500、許容誤差は±10%
    if trueCount < 400 || trueCount > 600 {
        t.Errorf("Expected ~500 true results, got %d", trueCount)
    }
}
```

## パフォーマンス考慮事項

1. **乱数生成のコスト:** math/randは高速で、パフォーマンスへの影響は最小限
2. **条件評価の順序:** random_chanceが他の条件より先に評価されても問題なし
3. **キャッシング:** 不要（各イベントで独立した判定が必要）

## セキュリティ考慮事項

1. **暗号学的に安全な乱数は不要:** 単なるUX改善機能のため
2. **設定値の検証:** 0-100の範囲外の値は確実にエラーとする
3. **DoS攻撃への耐性:** 乱数生成は軽量で、大量のリクエストでも問題なし

## 実装の簡潔さ

本設計では、以下の理由から**最小限の実装**を選択しました：

1. **既存パターンとの一貫性**: `isPrime`のようなシンプルな条件判定関数と同じ実装レベル
2. **テストの容易さ**: 統計的なテストで十分に検証可能
3. **保守性**: インターフェースや抽象化層を増やさず、コードの可読性を維持
4. **YAGNI原則**: 現時点で必要な機能のみを実装し、将来の拡張は必要になった時に対応

これにより、全体の実装は以下のシンプルな構成になります：
- types.go: 2行追加（定数定義とUnmarshalYAMLのケース）
- utils.go: 20行程度の関数1つと、switch文への1ケース追加
- main.go: 1行追加（rand.Seed）
- テストファイル: 2つの新規ファイル

## 将来の拡張性

シンプルな実装でも、以下の拡張は容易に対応可能：

1. **他のイベントタイプへの適用:** 同じ関数を他のイベントタイプでも使用可能
2. **確率の組み合わせ:** 複数のrandom_chance条件は現状でもAND条件として動作
3. **テスト用のシード固定:** 環境変数でシードを指定可能にする（必要時に追加）