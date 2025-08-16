#!/bin/bash
# プロジェクトタイプを判定して、適切なツール使用を促すスクリプト

# カラー定義
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# セッションIDを取得（引数から）
SESSION_ID="${1:-unknown}"

echo -e "${BLUE}=== プロジェクト分析中 ===${NC}"
echo ""

# プロジェクトタイプの判定と推奨設定
if [ -f "go.mod" ]; then
    echo -e "${GREEN}📦 Goプロジェクトを検出しました${NC}"
    echo ""
    echo "🔧 推奨設定:"
    echo "  • ファイル編集にはEditツールの代わりにMCP Serenaを使用してください"
    echo "  • serena MCPはGoコードのシンボル単位での編集が可能です"
    echo "  • 例: mcp__serena__replace_symbol_body でメソッドを丸ごと置換"
    echo ""

    # go.modから情報を取得
    if [ -f "go.mod" ]; then
        MODULE_NAME=$(grep "^module " go.mod | cut -d' ' -f2)
        echo "  モジュール名: $MODULE_NAME"
    fi

    # テストファイルの数を確認
    TEST_FILES=$(find . -name "*_test.go" 2>/dev/null | wc -l)
    if [ "$TEST_FILES" -gt 0 ]; then
        echo "  テストファイル: ${TEST_FILES}個検出"
        echo "  💡 ヒント: 'go test ./...' でテストを実行できます"
    fi

elif [ -f "package.json" ]; then
    echo -e "${GREEN}📦 Node.js/TypeScriptプロジェクトを検出しました${NC}"
    echo ""
    echo "🔧 推奨設定:"

    # TypeScriptプロジェクトかチェック
    if [ -f "tsconfig.json" ] || grep -q "typescript" package.json 2>/dev/null; then
        echo "  • TypeScriptプロジェクトです"
        echo "  • MCP Serenaはtypescriptもサポートしています"
        echo "  • 型チェックは 'npm run typecheck' で実行してください"
    fi

    # パッケージ情報
    if command -v jq &> /dev/null; then
        PACKAGE_NAME=$(jq -r '.name' package.json 2>/dev/null)
        echo "  パッケージ名: $PACKAGE_NAME"
    fi

    # 利用可能なスクリプトを表示
    echo ""
    echo "  利用可能なスクリプト:"
    if command -v jq &> /dev/null; then
        jq -r '.scripts | to_entries[] | "    • npm run \(.key)"' package.json 2>/dev/null | head -5
    else
        grep '".*":' package.json | head -5 | sed 's/.*"\(.*\)":.*/    • npm run \1/'
    fi

elif [ -f "requirements.txt" ] || [ -f "pyproject.toml" ] || [ -f "setup.py" ]; then
    echo -e "${GREEN}📦 Pythonプロジェクトを検出しました${NC}"
    echo ""
    echo "🔧 推奨設定:"
    echo "  • MCP Serenaはpythonもサポートしています"
    echo "  • 関数やクラス単位での編集が可能です"

    # 仮想環境の確認
    if [ -d "venv" ] || [ -d ".venv" ]; then
        echo "  • 仮想環境が検出されました"
    else
        echo "  • 💡 ヒント: 仮想環境の作成を推奨します (python -m venv venv)"
    fi

    # テストフレームワークの確認
    if [ -f "pytest.ini" ] || [ -f "setup.cfg" ] && grep -q "pytest" setup.cfg 2>/dev/null; then
        echo "  • pytestが設定されています"
        echo "  • テスト実行: pytest"
    fi

elif [ -f "Cargo.toml" ]; then
    echo -e "${GREEN}📦 Rustプロジェクトを検出しました${NC}"
    echo ""
    echo "🔧 推奨設定:"
    echo "  • cargo build でビルド"
    echo "  • cargo test でテスト実行"
    echo "  • cargo clippy でリンティング"

elif [ -f "Makefile" ]; then
    echo -e "${YELLOW}📦 Makefileを検出しました${NC}"
    echo ""
    echo "利用可能なターゲット:"
    grep "^[a-zA-Z0-9_-]*:" Makefile | grep -v "^\." | head -10 | sed 's/:.*//g' | sed 's/^/  • make /'

else
    echo -e "${YELLOW}⚠️  特定のプロジェクトタイプを検出できませんでした${NC}"
    echo ""
    echo "検出可能なプロジェクトタイプ:"
    echo "  • Go (go.mod)"
    echo "  • Node.js/TypeScript (package.json)"
    echo "  • Python (requirements.txt, pyproject.toml)"
    echo "  • Rust (Cargo.toml)"
fi

# 共通の推奨事項
echo ""
echo -e "${BLUE}=== 共通の推奨事項 ===${NC}"

# Gitリポジトリの確認
if [ -d ".git" ]; then
    BRANCH=$(git branch --show-current 2>/dev/null)
    echo "• Gitリポジトリ (現在のブランチ: ${BRANCH:-不明})"

    # mainブランチでの作業を警告
    if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
        echo -e "  ${YELLOW}⚠️  警告: main/masterブランチで作業中です${NC}"
        echo "  💡 git worktreeで作業ブランチを作成することを推奨"
    fi
fi

# CLAUDE.mdファイルの確認
if [ -f "CLAUDE.md" ]; then
    echo -e "• ${GREEN}CLAUDE.mdファイルが見つかりました${NC}"
    echo "  プロジェクト固有の指示が含まれています"
fi

# .envファイルの確認
if [ -f ".env" ] || [ -f ".env.local" ]; then
    echo -e "• ${YELLOW}環境変数ファイルが検出されました${NC}"
    echo "  秘密情報の取り扱いに注意してください"
fi

echo ""
echo "セッションID: $SESSION_ID"
echo "準備完了！作業を開始してください。"
