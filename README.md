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
      - type: structured_output
        continue: true

PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: structured_output
        continue: true
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

**WebFetch tool input example:**
```json
{
  "session_id": "abc123",
  "transcript_path": "/tmp/transcript",
  "hook_event_name": "PreToolUse",
  "tool_name": "WebFetch",
  "tool_input": {
    "url": "https://api.example.com/data",
    "prompt": "Summarize the API response"
  }
}
```

## Templates

Use `{jq_query}` for JSON processing with jq-compatible queries:

```yaml
Stop:
  - actions:
      - type: command
        command: >
          cat '{.transcript_path}' | 
          jq -s 'reverse | map(select(.type == "assistant" and .message.content[0].type == "text")) | .[0].message.content[0].text' |
          xargs -I {} ntfy publish --markdown --title 'Claude Code Complete' "{}"

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
- `{.tool_input.file_path}` - Access nested fields (Write/Edit tools)
- `{.tool_input.url}` - Access URL field (WebFetch tool)
- `{.tool_input.prompt}` - Access prompt field (WebFetch tool)

**Complex Example:**
```yaml
Stop:
  - actions:
      - type: command
        command: >
          cat '{.transcript_path}' | 
          jq -s 'reverse | map(select(.type == "assistant" and .message.content[0].type == "text")) | .[0].message.content[0].text' |
          xargs -I {} ntfy publish --markdown --title 'Claude Code Complete' --tags 'checkmark' "{}"
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
- `command_starts_with` - Match if command starts with specified string
- `file_exists` - Match if specified file exists
- `url_starts_with` - Match if URL starts with specified string (WebFetch tool)

## Actions

- `command` - Execute shell command
- `structured_output` - Generate Claude Code-compatible structured JSON output

### Structured JSON Output

The `structured_output` action generates Claude Code-compatible JSON using YAML fields directly:

**Structured Output Configuration**:
```yaml
actions:
  - type: "structured_output"
    permission_decision: "allow"
    permission_reason: "Auto-approved"
    continue: true
    suppress_output: false
```

#### Supported Hook Types and YAML Fields

**PreToolUse**:
- `permission_decision`: `"allow"`, `"deny"`, or `"ask"`
- `permission_reason`: Optional explanation
- Common fields: `continue`, `stop_reason`, `suppress_output`

**PostToolUse**:
- `decision`: `"block"` to automatically prompt Claude
- `reason`: Optional explanation for blocking
- Common fields: `continue`, `stop_reason`, `suppress_output`

**Stop/SubagentStop**:
- `decision`: `"block"` to prevent stopping
- `reason`: Required explanation when blocking
- Common fields: `continue`, `stop_reason`, `suppress_output`

**Notification/PreCompact**:
- Common fields only: `continue`, `stop_reason`, `suppress_output`

Fields are mapped directly from YAML to JSON output automatically.

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
      - type: structured_output
        continue: true
```

Check for Docker commands:
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_starts_with
        value: "docker"
      - type: file_exists
        value: "Dockerfile"
    actions:
      - type: structured_output
        continue: true
```

Monitor WebFetch access to specific sites:
```yaml
PreToolUse:
  - matcher: "WebFetch"
    conditions:
      - type: url_starts_with
        value: "https://api."
    actions:
      - type: output
        message: "🌐 API access detected: {.tool_input.url}"
      - type: command
        command: 'echo "API access: {.tool_input.url}" >> /tmp/api_access.log'
        
PostToolUse:
  - matcher: "WebFetch"
    conditions:
      - type: url_starts_with
        value: "https://news."
    actions:
      - type: command
        command: 'echo "News content fetched from {.tool_input.url}" | ntfy publish "WebFetch News"'
```

## Development

```bash
go test ./...
go build -o cchook
```

## License

MIT