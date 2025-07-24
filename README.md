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
        command: "gofmt -w {ToolInput.file_path}"
      - type: output
        message: "Formatted {ToolInput.file_path}"

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

Use `{path.to.field}` to access input data:

- `{SessionID}` - session_id field
- `{ToolName}` - tool_name field  
- `{ToolInput.file_path}` - file_path in tool_input
- `{ToolInput.content}` - content in tool_input
- `{ToolInput.nested.field}` - nested fields

## Event Types

- PreToolUse - Before tool execution
- PostToolUse - After tool execution
- Notification - System notifications
- Stop - Session stop
- SubagentStop - Subagent termination
- PreCompact - Before conversation compaction

## Conditions

- `file_extension` - Match file extension in ToolInput.file_path
- `command_contains` - Match substring in ToolInput.command

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
        command: "gofmt -w {ToolInput.file_path}"
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