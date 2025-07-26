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
    ]
  }
}
```

### 2. Create Configuration File

Create `~/.config/cchook/config.yaml` with your desired hooks:

```yaml
# Auto-format Go files after Write/Edit
PostToolUse:
  - conditions:
      - type: tool_name
        value: "Write|Edit"
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"

# Guide users to use better alternatives
PreToolUse:
  - conditions:
      - type: tool_name
        value: "Bash"
      - type: command_starts_with
        value: "python"
    actions:
      - type: output
        message: "pythonは使わず`uv`を代わりに使いましょう"
  - conditions:
      - type: tool_name
        value: "WebFetch"
      - type: url_starts_with
        value: "https://github.com"
    actions:
      - type: output
        message: "WebFetchではなく、`gh`コマンド経由で情報を取得しましょう"
```

## Configuration Examples

### File Processing

Auto-format different file types:

```yaml
PostToolUse:
  - conditions:
      - type: tool_name
        value: "Write|Edit"
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {.tool_input.file_path}"
  
  - conditions:
      - type: tool_name
        value: "Write|Edit"
      - type: file_extension
        value: ".py"
    actions:
      - type: command
        command: "black {.tool_input.file_path}"
```

Run pre-commit hooks automatically:

```yaml
PostToolUse:
  - conditions:
      - type: tool_name
        value: "Write|Edit|MultiEdit"
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
  - conditions:
      - type: tool_name
        value: "Bash"
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
  - conditions:
      - type: tool_name
        value: "WebFetch"
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

### Conditions

- `tool_name`
  - Match tool name (e.g., "Write|Edit", "Bash", "WebFetch")
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

### Conditional File Processing

```yaml
PostToolUse:
  - conditions:
      - type: tool_name
        value: "Write|Edit"
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
  - conditions:
      - type: tool_name
        value: "Write|Edit"
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

```bash
go test ./...
go build -o cchook
```

## License

MIT