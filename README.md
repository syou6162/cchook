# cchook

A CLI tool for executing hooks at various stages of Claude Code operations.

## Background & Motivation

Claude Code has a powerful [hook system](https://docs.anthropic.com/ja/docs/claude-code/hooks) that allows executing custom commands at various stages of operation. However, writing hooks can become unwieldy for several reasons:

- **Complex JSON configuration**: Hooks are configured in JSON format within settings, making them hard to read and maintain
- **Repetitive jq processing**: When using multiple elements from input JSON, you need temporary files and repeated jq filters
- **Single-line limitations**: JSON strings don't support multi-line formatting like YAML, leading to very long, hard-to-read command lines

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
- **YAML configuration**: Clean, readable multi-line configuration
- **Template syntax**: Simple `{.field}` syntax for accessing JSON data with full jq query support
- **Conditional logic**: Built-in conditions for common scenarios (file extensions, command patterns, etc.)
- **Better maintainability**: Structured configuration that's easy to understand and modify

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


## Configuration

Create a YAML configuration file at `~/.config/cchook/config.yaml`:

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
      - type: output
        message: "Formatted {.tool_input.file_path}"

PreToolUse:
  - conditions:
      - type: tool_name
        value: "Bash"
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "Consider using semantic commit workflow"
```

## Input Format

cchook receives JSON input from Claude Code hooks via stdin. For details on the JSON structure and available fields, see the [Claude Code hook documentation](https://docs.anthropic.com/ja/docs/claude-code/hooks).

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
- `{.session_id}` - Access session identifier
- `{.hook_event_name}` - Access hook event name
- `{.tool_name}` - Access tool name
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
- `output` - Print message to stdout/stderr

### ExitStatus Control

Actions support an optional `exit_status` field to control execution behavior:

- **Default for `output` actions**: `2` (blocks tool execution, outputs to stderr)
- **ExitStatus `0`**: Normal execution (outputs to stdout)
- **ExitStatus `2`**: Blocks tool execution (outputs to stderr) - **Useful for PreToolUse**
- **Other values**: Exits with specified code (outputs to stdout)

**Examples:**

Block dangerous commands (recommended for PreToolUse):
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_starts_with
        value: "rm -rf"
    actions:
      - type: output
        message: "🚫 Dangerous command blocked!"
        # exit_status: 2 (default for output actions)
```

Allow with warning (outputs to stdout):
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git push"
    actions:
      - type: output
        message: "⚠️ Pushing to remote repository"
        exit_status: 0  # Allows tool execution
```

Custom exit behavior:
```yaml
PreToolUse:
  - matcher: "Bash"
    actions:
      - type: output
        message: "Custom exit status for specific workflows"
        exit_status: 1  # Custom exit code
```

**Important Notes:**
- **ExitStatus `2` in PreToolUse**: Blocks tool execution and sends message to Claude via stderr
- **ExitStatus `0`**: Allows tool execution and outputs informational message to stdout

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
        command: "gofmt -w {.tool_input.file_path}"
```

Block git add (recommended approach):
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "🚫 Direct git add blocked. Use semantic commit workflow instead."
        # exit_status: 2 (default - blocks execution)
```

Or warn about git add (allows execution):
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "git add"
    actions:
      - type: output
        message: "⚠️ Warning: direct git add detected"
        exit_status: 0  # Allows execution with warning
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
      - type: output
        message: "Docker operation detected in project with Dockerfile"
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