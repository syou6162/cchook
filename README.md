# cchook

```
  ██████╗  ██████╗ ██╗  ██╗  ██████╗   ██████╗  ██╗  ██╗
 ██╔════╝ ██╔════╝ ██║  ██║ ██╔═══██╗ ██╔═══██╗ ██║ ██╔╝
 ██║      ██║      ███████║ ██║   ██║ ██║   ██║ █████╔╝
 ██║      ██║      ██╔══██║ ██║   ██║ ██║   ██║ ██╔═██╗
 ╚██████╗ ╚██████╗ ██║  ██║ ╚██████╔╝ ╚██████╔╝ ██║  ██╗
  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝  ╚═════╝   ╚═════╝  ╚═╝  ╚═╝
```

A smart hook manager for Claude Code that provides context-aware assistance, security validation, and tool optimization through YAML-based configuration.

## Background & Motivation

Claude Code has a powerful [hook system](https://docs.anthropic.com/ja/docs/claude-code/hooks) that allows executing custom commands at various stages of operation. However, writing hooks can become unwieldy for several reasons:

- Complex JSON configuration
  - Hooks are configured in JSON format within settings, making them hard to read and maintain
- Repetitive jq processing
  - When using multiple elements from input JSON, you need temporary files and repeated jq filters
- Single-line limitations
  - JSON strings don't support multi-line formatting like YAML, leading to very long, hard-to-read command lines

For example, a simple Stop hook that sends notifications via [ntfy](https://ntfy.sh) becomes a complex one-liner:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "transcript_path=$(jq -r '.transcript_path') && cat \"${transcript_path}\" | jq -s 'reverse | map(select(.type == \"assistant\" and .message.content[0].type == \"text\")) | .[0].message.content[0]' > /tmp/cc_ntfy.json && ntfy publish --markdown --title 'Claude Code' \"$(cat /tmp/cc_ntfy.json | jq -r '.text')\""
          }
        ]
      }
    ]
  }
}
```

**cchook** solves these problems by providing:

- YAML configuration
  - Clean, readable multi-line configuration
- Template syntax
  - Simple `{.field}` syntax for accessing JSON data with full jq query support
- Conditional logic
  - Built-in conditions for common scenarios (file extensions, command patterns, etc.)
- Better maintainability
  - Structured configuration that's easy to understand and modify

## Installation

```bash
go install github.com/syou6162/cchook@latest
```

## Building from Source

```bash
git clone https://github.com/syou6162/cchook
cd cchook
go build -o cchook
```

## Features

### 🎯 Smart Session Start
Automatically analyze your project when Claude Code starts and provide context-aware recommendations:
- Detect project type (Go, TypeScript, Python, Rust, etc.)
- Recommend appropriate tools (e.g., MCP Serena for code editing)
- Show available scripts and commands
- Warn about risky operations (main branch commits, etc.)

### 🛡️ Prompt Validation
Validate and filter user prompts before Claude processes them:
- Block accidental secret exposure (API keys, passwords)
- Add debug capabilities
- Implement custom validation rules

### 🔧 Tool Optimization
Guide better tool usage with conditional hooks:
- Suggest MCP tools for specific file types
- Auto-format code after edits
- Validate dangerous commands

## Quick Start

### 1. Configure Claude Code Hooks

Add cchook to your Claude Code hook configuration in `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cchook -event SessionStart"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cchook -event UserPromptSubmit"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cchook -event PreToolUse"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cchook -event PostToolUse"
          }
        ]
      }
    ]
  }
}
```

### 2. Create Configuration File

Create `~/.config/cchook/config.yaml` with your desired hooks:

```yaml
# Smart project analysis on session start
SessionStart:
  - matcher: startup
    actions:
      - type: command
        # This script analyzes your project and provides recommendations
        command: |
          if [ -f go.mod ]; then
            echo "📦 Go project detected!"
            echo "💡 Use MCP Serena for symbol-based editing instead of Edit/Write"
            echo "   Example: mcp__serena__replace_symbol_body"
          elif [ -f package.json ]; then
            echo "📦 Node.js/TypeScript project detected!"
            if [ -f tsconfig.json ]; then
              echo "💡 TypeScript: Use MCP Serena for better code navigation"
            fi
          fi

# Prevent accidental secret exposure
UserPromptSubmit:
  - conditions:
      - type: prompt_contains
        value: "api_key"
    actions:
      - type: output
        message: "⚠️ API key detected in prompt. Use environment variables instead!"
        exit_status: 2  # Block the prompt

# Auto-format Go files after editing
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"

# Guide to better tool usage
PreToolUse:
  - matcher: "Edit|Write"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: output
        message: "💡 Tip: Use MCP Serena (mcp__serena__replace_symbol_body) for Go files"
```

For more advanced examples, see `examples/config_with_session_start.yaml`

## CLI Options

### Configuration File Path

By default, cchook looks for configuration files in the following order:

1. Path specified by `-config` flag
2. `$XDG_CONFIG_HOME/cchook/config.yaml` (if `XDG_CONFIG_HOME` is set)
3. `~/.config/cchook/config.yaml` (default fallback)

#### Using Custom Configuration File

You can specify a custom configuration file path using the `-config` flag:

```bash
# Use custom config file
cchook -config /path/to/my-config.yaml -event PreToolUse

# Example: Development vs Production configs
cchook -config ~/.config/cchook/dev-config.yaml -event PostToolUse
cchook -config ~/.config/cchook/prod-config.yaml -event Stop
```

#### Example Claude Code Hook with Custom Config

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cchook -config ~/.config/cchook/dev-config.yaml -event PreToolUse"
          }
        ]
      }
    ]
  }
}
```

## Configuration Examples

### 🎯 Session Start - Smart Project Analysis

Automatically analyze project type and provide context-aware guidance:

```yaml
SessionStart:
  - matcher: startup
    actions:
      - type: command
        # Full project analyzer script (see examples/project_analyzer.sh)
        command: "bash examples/project_analyzer.sh '{.session_id}'"
```

**Example output when starting in a Go project:**
```
📦 Go project detected!
🔧 Recommendations:
  • Use MCP Serena instead of Edit for symbol-based editing
  • Module: github.com/syou6162/cchook
  • Test files: 9 detected
  • Run 'go test ./...' to execute tests
⚠️ Warning: Currently on main branch - use git worktree for features
```

The analyzer detects:
- **Go projects**: Recommends MCP Serena, shows module info, test counts
- **TypeScript/Node.js**: Lists npm scripts, checks for TypeScript config
- **Python**: Detects virtual environments, test frameworks
- **Git status**: Warns about main branch work
- **Special files**: CLAUDE.md, .env files

### 🛡️ Prompt Security & Validation

Protect against accidental secret exposure and add debugging:

```yaml
UserPromptSubmit:
  # Multi-layer security checks
  - conditions:
      - type: prompt_contains
        value: "BEGIN PRIVATE KEY"
    actions:
      - type: output
        message: "🚨 Private key detected! Submission blocked for security."
        exit_status: 2

  # Smart debugging for development
  - conditions:
      - type: prompt_starts_with
        value: "TEST:"
    actions:
      - type: command
        command: |
          echo "[$(date)] TEST MODE: {.prompt}" >> test.log
          echo "Test prompt logged - proceeding with caution"
```

### File Processing

Auto-format different file types:

```yaml
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"

  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".py"
    actions:
      - type: command
        command: "black {.tool_input.file_path}"
```

Run pre-commit hooks automatically:

```yaml
PostToolUse:
  - matcher: "Write|Edit|MultiEdit"
    conditions:
      - type: file_exists
        value: ".pre-commit-config.yaml"
    actions:
      - type: command
        command: "pre-commit run --files {.tool_input.file_path}"
```

### Command Safety

Block dangerous commands:

```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_starts_with
        value: "rm -rf"
    actions:
      - type: output
        message: "🚫 Dangerous command blocked!"
        # exit_status: 2 (default - blocks execution)
```

### API Monitoring

Track external API usage:

```yaml
PreToolUse:
  - matcher: "WebFetch"
    conditions:
      - type: url_starts_with
        value: "https://api."
    actions:
      - type: output
        message: "🌐 API access: {.tool_input.url}"
        exit_status: 0
      - type: command
        command: 'echo "{.session_id}: {.tool_input.url}" >> ~/api_access.log'
```

### Notifications

Send completion notifications:

```yaml
Stop:
  - actions:
      - type: command
        command: >
          cat '{.transcript_path}' |
          jq -s 'reverse | map(select(.type == "assistant" and .message.content[0].type == "text")) | .[0].message.content[0].text' |
          xargs -I {} ntfy publish --markdown --title 'Claude Code Complete' "{}"
```

## Configuration Reference

### Event Types

- `SessionStart` (New!)
  - When Claude Code session starts
  - Matchers: `startup`, `resume`, `clear`
- `UserPromptSubmit` (New!)
  - When user submits a prompt (before Claude processes)
  - Can block submission with exit_status: 2
- `PreToolUse`
  - Before tool execution (can block with exit_status: 2)
- `PostToolUse`
  - After tool execution
- `Stop`
  - When Claude Code session ends
- `SubagentStop`
  - When a subagent terminates
- `Notification`
  - System notifications
- `PreCompact`
  - Before conversation compaction

### Matcher

- `matcher`
  - Match tool name using pipe-separated patterns (e.g., "Write|Edit", "Bash", "WebFetch")
  - Empty matcher matches all tools
  - Uses the same syntax as Claude Code's built-in hook matcher field

### Conditions

#### PreToolUse/PostToolUse
- `file_extension`
  - Match file extension in `tool_input.file_path`
- `command_contains`
  - Match substring in `tool_input.command`
- `command_starts_with`
  - Match command prefix
- `file_exists`
  - Check if specified file exists
- `url_starts_with`
  - Match URL prefix (WebFetch tool)

#### UserPromptSubmit (New!)
- `prompt_contains`
  - Match substring in user prompt
- `prompt_starts_with`
  - Match prompt prefix
- `prompt_ends_with`
  - Match prompt suffix
- `file_exists`
  - Check if specified file exists

### Actions

- `command`
  - Execute shell command
- `output`
  - Print message (default `exit_status`: 2 for `PreToolUse`, 0 for others)

### Exit Status Control

- 0
  - Allow execution, output to stdout
- 2
  - Block execution (PreToolUse only), output to stderr
- Other
  - Exit with specified code

### Template Syntax

Access JSON data using `{.field}` syntax with full jq query support:

- Simple fields
  - `{.session_id}`, `{.tool_name}`, `{.hook_event_name}`
- Nested fields
  - `{.tool_input.file_path}`, `{.tool_input.url}`
- Complex queries
  - `{.transcript_path | @base64}`, `{.tool_input | keys}`
- Entire object
  - `{.}`

YAML Multi-line Support:
- `>`
  - Folded style (newlines become spaces)
- `|`
  - Literal style (preserves formatting)

## Advanced Examples

### 🔄 Complete Workflow Example

Combine multiple hooks for a comprehensive development workflow:

```yaml
# Start: Analyze project and set context
SessionStart:
  - matcher: startup
    actions:
      - type: command
        command: "bash examples/project_analyzer.sh '{.session_id}'"

# Before edits: Warn about tool choice
PreToolUse:
  - matcher: "Edit|Write"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: output
        message: "💡 Consider using mcp__serena__replace_symbol_body for Go files"

# After edits: Auto-format and test
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: |
          gofmt -w {.tool_input.file_path}
          if [[ {.tool_input.file_path} == *_test.go ]]; then
            go test -v {.tool_input.file_path}
          fi

# On completion: Send notification
Stop:
  - actions:
      - type: command
        command: |
          CHANGES=$(git diff --stat 2>/dev/null | tail -1)
          echo "Session complete: $CHANGES" | ntfy publish --title 'Claude Code'
```

### 🛠️ MCP Tool Guidance

Guide users to appropriate MCP tools based on file type:

```yaml
PreToolUse:
  # For Go files
  - matcher: "Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: output
        message: |
          📝 For Go files, consider MCP Serena:
          • mcp__serena__replace_symbol_body - Replace entire functions
          • mcp__serena__find_symbol - Find definitions
          • mcp__serena__find_referencing_symbols - Find usages

  # For TypeScript files
  - matcher: "Edit"
    conditions:
      - type: file_extension
        value: ".ts"
    actions:
      - type: output
        message: |
          📝 For TypeScript, MCP Serena offers:
          • Symbol-aware editing
          • Class/interface navigation
          • Method extraction
```

### 🚨 Security & Safety Guards

```yaml
# Prevent dangerous operations
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "rm -rf /"
    actions:
      - type: output
        message: "🚫 BLOCKED: Dangerous command detected!"
        exit_status: 2

# Check for main branch commits
  - matcher: "Bash"
    conditions:
      - type: command_starts_with
        value: "git push"
    actions:
      - type: command
        command: |
          BRANCH=$(git branch --show-current)
          if [[ "$BRANCH" == "main" || "$BRANCH" == "master" ]]; then
            echo "⚠️ Pushing to $BRANCH branch! Add --force-with-lease for safety"
            exit 2
          fi
```

## Input Format

cchook receives JSON input from Claude Code hooks via stdin. For details on the JSON structure and available fields, see the [Claude Code hook documentation](https://docs.anthropic.com/ja/docs/claude-code/hooks).

## Development

```bash
go test ./...
go build -o cchook
```

## License

MIT
