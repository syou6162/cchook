# cchook

A CLI tool for executing hooks at various stages of Claude Code operations.

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

## Usage

Configure as a Claude Code hook in your settings.json:

**`.claude/settings.json`**:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
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
        "matcher": "Write|Edit", 
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

**Manual execution** (for testing):
```bash
cchook -event PostToolUse < input.json
cchook -command dry-run -event PreToolUse < input.json  
```

## Configuration

Create a YAML configuration file at `~/.config/cchook/config.yaml`:

```yaml
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {tool_input.file_path}"
      - type: output
        message: "Formatted {tool_input.file_path}"

PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "Consider using semantic commit workflow"
```

## Input Format

Expects JSON input via stdin:

```json
{
  "session_id": "abc123",
  "transcript_path": "/tmp/transcript",
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "main.go",
    "content": "package main"
  }
}
```

## Templates

### Basic Templates

Use `{path.to.field}` to access input data:

- `{session_id}` - session_id field
- `{tool_name}` - tool_name field  
- `{tool_input.file_path}` - file_path in tool_input
- `{tool_input.content}` - content in tool_input

### JQ Templates (Advanced)

Use `{jq: query}` for complex JSON processing with jq-compatible queries:

```yaml
Stop:
  - actions:
      - type: output
        message: "Last assistant message: {jq: .data | reverse | map(select(.type == \"assistant\")) | .[0].content}"
      - type: command
        command: 'ntfy publish --title "Claude Code Session" "{jq: .transcript_path}"'

Notification:  
  - actions:
      - type: command
        command: 'ntfy publish --title "Claude Code" "{jq: .message | @base64}"'
```

**JQ Features:**
- Full jq query language support via [gojq](https://github.com/itchyny/gojq)
- Array manipulation: `reverse`, `map`, `select`, `sort_by`
- String processing: `@base64`, `ascii_upcase`, `length`
- Complex data extraction from nested JSON structures
- Backward compatible with existing `{field}` syntax

**Examples:**
- `{jq: .transcript_path}` - Extract transcript path
- `{jq: .data | length}` - Count array elements
- `{jq: [.data[] | select(.type == \"assistant\") | .content]}` - Get all assistant messages
- `{jq: .message | @base64}` - Base64 encode message
- `{tool_input.nested.field}` - nested fields

## Event Types

- PreToolUse - Before tool execution
- PostToolUse - After tool execution
- Notification - System notifications
- Stop - Session stop
- SubagentStop - Subagent termination
- PreCompact - Before conversation compaction

## Conditions

- `file_extension` - Match file extension in tool_input.file_path
- `command_contains` - Match substring in tool_input.command

## Actions

- `command` - Execute shell command
- `output` - Print message to stdout

## Examples

Auto-format Go files:
```yaml
PostToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".go"
    actions:
      - type: command
        command: "gofmt -w {tool_input.file_path}"
```

Warn about git add:
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "Warning: direct git add detected"
```

## Development

```bash
go test ./...
go build -o cchook
```

## License

MIT