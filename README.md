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

Use `{jq_query}` for JSON processing with jq-compatible queries:

```yaml
Stop:
  - actions:
      - type: command
        command: |
          MESSAGE=$(cat '{.transcript_path}' | 
            jq -sr 'reverse 
                   | map(select(.type == "assistant" and .message.content[0].type == "text")) 
                   | .[0].message.content[0].text')
          ntfy publish \
            --markdown \
            --title 'Claude Code Complete' \
            --message "${MESSAGE}"

Notification:  
  - actions:
      - type: command
        command: ntfy publish --markdown --title "{.hook_event_name}" "{.message}"
```

**JQ Features:**
- Full jq query language support via [gojq](https://github.com/itchyny/gojq)
- Array manipulation: `reverse`, `map`, `select`, `sort_by`
- String processing: `@base64`, `ascii_upcase`, `length`
- Complex data extraction from nested JSON structures

**YAML Multi-line Support:**
- `>` - Folded style (spaces preserved, newlines become spaces)
- `|` - Literal style (preserves all formatting)

**Access Patterns:**
- `{.transcript_path}` - Access root fields directly
- `{.data | length}` - Count array elements
- `{[.data[] | select(.type == "assistant") | .content]}` - Filter and extract from arrays
- `{.message | @base64}` - String transformations
- `{.}` - Access entire input JSON object
- `{.tool_input.file_path}` - Access nested fields

**Complex Example:**
```yaml
Stop:
  - actions:
      - type: command
        command: |
          MESSAGE=$(cat '{.transcript_path}' | 
            jq -sr 'reverse 
                   | map(select(.type == "assistant" and .message.content[0].type == "text")) 
                   | .[0].message.content[0].text')
          echo "Session completed!" &&
          ntfy publish \
            --markdown \
            --title "Claude Code Complete" \
            --tags "checkmark" \
            --message "${MESSAGE}"
```

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