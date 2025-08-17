# cchook

```
  ██████╗  ██████╗ ██╗  ██╗  ██████╗   ██████╗  ██╗  ██╗
 ██╔════╝ ██╔════╝ ██║  ██║ ██╔═══██╗ ██╔═══██╗ ██║ ██╔╝
 ██║      ██║      ███████║ ██║   ██║ ██║   ██║ █████╔╝
 ██║      ██║      ██╔══██║ ██║   ██║ ██║   ██║ ██╔═██╗
 ╚██████╗ ╚██████╗ ██║  ██║ ╚██████╔╝ ╚██████╔╝ ██║  ██╗
  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝  ╚═════╝   ╚═════╝  ╚═╝  ╚═╝
```

A CLI tool for executing hooks at various stages of Claude Code operations.

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

## Features

- **YAML Configuration**: Write clean, maintainable hook configurations
- **Template Engine**: Use `{.field}` syntax with full jq query support
- **Conditional Execution**: Execute actions based on file types, commands, or prompts
- **Error Handling**: Robust error handling for unknown condition types
- **Dry-Run Mode**: Test configurations before deployment
- **Performance**: Cached jq query compilation for efficient template processing

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

## Quick Start

### 1. Configure Claude Code Hooks

Add cchook to your Claude Code hook configuration in `.claude/settings.json`:

```json
{
  "hooks": {
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
    ],
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
    ]
  }
}
```

### 2. Create Configuration File

Create `~/.config/cchook/config.yaml` with your desired hooks:

```yaml
# Auto-format Go files after Write/Edit
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"

# Guide users to use better alternatives
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_starts_with
        value: "python"
    actions:
      - type: output
        message: "pythonは使わず`uv`を代わりに使いましょう"
  - matcher: "WebFetch"
    conditions:
      - type: url_starts_with
        value: "https://github.com"
    actions:
      - type: output
        message: "WebFetchではなく、`gh`コマンド経由で情報を取得しましょう"
```

## CLI Options

### Flags

- `-event` (required): Specify the event type (PreToolUse, PostToolUse, SessionStart, etc.)
- `-config`: Path to configuration file (default: `~/.config/cchook/config.yaml`)
- `-command`: Override configuration with a single command (useful for dry-run testing)

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

#### Dry-Run Testing

Test your configuration without making actual changes:

```bash
# Test with a simple echo command
echo '{"session_id":"test","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"test.go"}}' | \
  cchook -event PreToolUse -command "echo 'Would process: {.tool_name} on {.tool_input.file_path}'"
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

Conditional processing based on project type:

```yaml
PreToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".py"
      - type: file_exists_recursive
        value: "pyproject.toml"
    actions:
      - type: output
        message: "📝 Python project detected with pyproject.toml"
        exit_status: 0
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

  # Protect Git-tracked files from accidental deletion/move
  - matcher: "Bash"
    conditions:
      - type: git_tracked_file_operation
        value: "rm|mv"  # Check both rm and mv commands
    actions:
      - type: output
        message: |
          ⚠️  Error: Attempting to operate on Git-tracked files
          Use 'git rm' or 'git mv' instead for Git-tracked files
          Command attempted: {.tool_input.command}
        exit_status: 2  # Block execution (exit code 2 blocks in PreToolUse)
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

### Session Management

Initialize session with custom setup:

```yaml
SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "echo 'Session {.session_id} started at $(date)' >> ~/claude-sessions.log"
      - type: output
        message: "🚀 Claude Code session initialized"
        exit_status: 0

  # Project-specific initialization
  - matcher: "startup"
    conditions:
      - type: file_exists
        value: "go.mod"
    actions:
      - type: output
        message: "Go project detected - remember to run tests"
        exit_status: 0

  - matcher: "startup"
    conditions:
      - type: file_exists_recursive
        value: "pyproject.toml"
    actions:
      - type: output
        message: "Python project detected - using uv for package management"
        exit_status: 0
```

### User Prompt Filtering

Guide users based on their prompts using regex patterns:

```yaml
UserPromptSubmit:
  # Add context/warnings (exit_status: 0 - adds to context)
  - conditions:
      - type: prompt_regex
        value: "delete|削除|remove"
    actions:
      - type: output
        message: "⚠️ 削除操作を実行する前に、必ずバックアップを取ってください"
        exit_status: 0  # Adds message to context, prompt continues

  # Block dangerous prompts (exit_status: 2 - blocks prompt)
  - conditions:
      - type: prompt_regex
        value: "rm -rf /"
    actions:
      - type: output
        message: "🚫 危険なコマンドが含まれています。このプロンプトはブロックされました。"
        exit_status: 2  # Blocks prompt processing, shows message to user

  # Add helpful context
  - conditions:
      - type: prompt_regex
        value: "^(python|pip|conda)"
    actions:
      - type: output
        message: "💡 Pythonの代わりに`uv`を使用することをお勧めします"
        exit_status: 0

  # Add context for questions
  - conditions:
      - type: prompt_regex
        value: "\\?$"
    actions:
      - type: output
        message: "📚 質問を検知しました。ドキュメントを確認することをお勧めします"
        exit_status: 0
```

## Configuration Reference

### Event Types

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
- `SessionStart`
  - When Claude Code session starts
  - Supports conditions like `file_exists` and `file_exists_recursive`
- `UserPromptSubmit`
  - When user submits a prompt

### Matcher

- `matcher`
  - Match tool name using pipe-separated patterns (e.g., "Write|Edit", "Bash", "WebFetch")
  - Empty matcher matches all tools
  - Uses the same syntax as Claude Code's built-in hook matcher field

### Conditions

All conditions return proper error messages for unknown condition types, ensuring clear feedback when misconfigured.

#### Common Conditions (All Events)
- `file_exists`
  - Check if specified file exists
- `file_exists_recursive`
  - Check if file exists recursively in directory tree

#### PreToolUse & PostToolUse
- All common conditions, plus:
- `file_extension`
  - Match file extension in `tool_input.file_path`
- `command_contains`
  - Match substring in `tool_input.command`
- `command_starts_with`
  - Match command prefix
- `url_starts_with`
  - Match URL prefix (WebFetch tool)
- `git_tracked_file_operation`
  - Check if command (rm, mv, etc.) operates on Git-tracked files
  - Value specifies commands to check (e.g., `"rm"`, `"mv"`, `"rm|mv"`)

#### UserPromptSubmit
- All common conditions, plus:
- `prompt_regex`
  - Match user prompt with regular expression
  - Supports OR conditions: `"help|助けて|サポート"`
  - Supports anchors: `"^prefix"` (starts with), `"suffix$"` (ends with)
  - Supports complex patterns: `"^(DEBUG|INFO|WARN|ERROR):"`

#### Other Events (SessionStart, Stop, Notification, SubagentStop, PreCompact)
- Support common conditions only (`file_exists`, `file_exists_recursive`)

### Actions

- `command`
  - Execute shell command
- `output`
  - Print message
  - Default `exit_status`:
    - 0 for SessionStart, UserPromptSubmit (non-blocking events)
    - 2 for PreToolUse, PostToolUse, Stop, SubagentStop, Notification, PreCompact

### Exit Status Control

- 0
  - Success, allow execution, output to stdout
- 2
  - Block execution (PreToolUse), output to stderr
  - Claude will process the stderr message
- Other (1, 3, etc.)
  - Non-blocking error, stderr shown to user
  - Execution continues normally

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

### Conditional File Processing

```yaml
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".py"
      - type: file_exists
        value: "pyproject.toml"
    actions:
      - type: command
        command: "ruff format {.tool_input.file_path}"
      - type: command
        command: "ruff check --fix {.tool_input.file_path}"
```

### Multi-Step Workflows

```yaml
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"
      - type: command
        command: "go vet {.tool_input.file_path}"
      - type: output
        message: "✅ Go file formatted and vetted: {.tool_input.file_path}"
        exit_status: 0
```

### Complex Notifications

```yaml
Stop:
  - actions:
      - type: command
        command: |
          LAST_MSG=$(cat '{.transcript_path}' | jq -s 'reverse | map(select(.type == "assistant" and .message.content[0].type == "text")) | .[0].message.content[0].text' | head -c 100)
          ntfy publish --markdown --title 'Claude Code Session Complete' --tags 'checkmark' "$LAST_MSG..."
```

## Input Format

cchook receives JSON input from Claude Code hooks via stdin. For details on the JSON structure and available fields, see the [Claude Code hook documentation](https://docs.anthropic.com/ja/docs/claude-code/hooks).

## Development

### Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building

```bash
# Build binary
go build -o cchook

# Install locally
go install
```

### Linting

```bash
# Using pre-commit hooks
pre-commit run --all-files

# Direct golangci-lint
golangci-lint run
```

## License

MIT
