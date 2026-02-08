# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cchook is a CLI tool that simplifies Claude Code hook configuration by providing YAML-based configuration and template syntax instead of complex JSON one-liners. It processes JSON input from Claude Code hooks via stdin and executes configured actions based on matching conditions.

## Example Configuration

```yaml
# Prevent building when build directory already exists
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: dir_exists
        value: "build"
      - type: command_starts_with
        value: "make"
    actions:
      - type: output
        message: "Build directory already exists. Run 'make clean' first."
        exit_status: 1

# Warn when package-lock.json doesn't exist
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: file_not_exists
        value: "package-lock.json"
      - type: command_starts_with
        value: "npm install"
    actions:
      - type: output
        message: "Warning: package-lock.json not found. This may cause dependency issues."

# Create backup directory if it doesn't exist
PreToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: dir_not_exists
        value: "backups"
    actions:
      - type: command
        command: "mkdir -p backups && echo 'Created backup directory'"

# Check for missing test files
PostToolUse:
  - matcher: "Write"
    conditions:
      - type: file_extension
        value: ".go"
      - type: file_not_exists_recursive
        value: "main_test.go"
    actions:
      - type: output
        message: "Consider adding tests for {.tool_input.file_path}"

# Pass complex JSON data to external commands via stdin
PreToolUse:
  - matcher: "Write|Edit"
    conditions:
      - type: file_extension
        value: ".sql"
    actions:
      - type: command
        # use_stdin: true safely handles special characters (newlines, quotes, etc.)
        # without shell escaping issues
        command: "python validate_sql.py"
        use_stdin: true
```

## Development Commands

```bash
# Build the project
go build -o cchook

# Install dependencies
go mod download
go mod tidy

# Run unit tests only (fast, no external dependencies)
go test ./...

# Run unit tests with verbose output
go test -v ./...

# Run integration tests (requires external commands like cat, jq)
go test -tags=integration ./...

# Run integration tests with verbose output
go test -v -tags=integration ./...

# Run specific test function (more practical than test file)
go test -v -run TestCheckGitTrackedFileOperation ./...
go test -v -run TestExecutePreToolUseHooks ./...
go test -v -run TestCheckUserPromptSubmitCondition ./...

# Run specific integration test
go test -v -tags=integration -run TestExecutePreToolUseAction_WithUseStdin ./...

# Run with coverage
go test -cover ./...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Install locally for testing
go install

# Lint code (via pre-commit) - requires pre-commit to be installed
pre-commit run --all-files

# Lint Go code directly with golangci-lint
golangci-lint run

# Test the tool manually (requires JSON input via stdin)
echo '{"session_id":"test","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"test.go"}}' | ./cchook -event PreToolUse

# Dry-run mode for testing configurations
echo '{"session_id":"test","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"test.go"}}' | ./cchook -event PreToolUse -command "echo 'would execute: {.tool_name}'"
```

## Architecture

### Core Components

The application follows a modular architecture with clear separation of concerns:

**Main Entry Point** (`main.go`)
- CLI argument parsing (`-config`, `-command`, `-event`)
- Delegates to appropriate hook execution functions
- Handles ExitError for proper exit codes and output routing

**Type System** (`types.go`)
- Defines all event types: PreToolUse, PostToolUse, Stop, SubagentStop, Notification, PreCompact, SessionStart, SessionEnd, UserPromptSubmit
- Event-specific input structures with embedded BaseInput
- Hook and Action interfaces for polymorphic behavior
- Separate condition and action types for each event
- Opaque struct pattern for ConditionType with predefined singletons

**Configuration** (`config.go`)
- YAML configuration loading with XDG_CONFIG_HOME support
- Default path: `~/.config/cchook/config.yaml`
- Custom path via `-config` flag

**Input Processing** (`parser.go`)
- Generic parsing function with type constraints
- Tool-specific parsing for PreToolUse/PostToolUse (complex tool_input handling)
- Returns both structured data and raw JSON for template processing

**Hook Execution** (`hooks.go`)
- Event-specific hook execution functions
- Matcher checking (partial string matching with pipe separation)
- Condition evaluation per event type
- Dry-run mode for debugging configurations

**Action Execution** (`actions.go`)
- Command execution via shell
- Output handling with exit status control
- ExitError creation for blocking execution (PreToolUse only)

**Template Engine** (`template_jq.go`)
- Unified `{.field}` syntax for JSON field access
- Full jq query support within template braces
- Query caching for performance
- Error handling with `[JQ_ERROR: ...]` format

**Utilities** (`utils.go`)
- Condition checking functions per event type with `(bool, error)` return
- Sentinel error pattern (`ErrConditionNotHandled`) for unknown condition types
- Command execution wrapper
- File existence, extension, and URL pattern matching
- Prompt regex matching for UserPromptSubmit events
- Transcript file parsing for `every_n_prompts` condition using json.Decoder

### Data Flow

1. **Input**: JSON from Claude Code hooks via stdin
2. **Parsing**: Event-specific parsing with tool input handling
3. **Hook Matching**: Matcher patterns (partial string matching) and conditions
4. **Template Processing**: jq-based field substitution in commands/messages
5. **Action Execution**: Shell commands or output with exit status control
6. **Exit Handling**: ExitError for blocking vs allowing execution

### Key Design Patterns

**Event-Driven Architecture**: Each Claude Code event type has dedicated input/hook/action structures

**Template System**: Consistent `{.field}` syntax across all actions, powered by gojq for complex queries

**Condition System**: Event-specific condition types with common patterns (file_extension, command_contains, etc.). UserPromptSubmit uses `prompt_regex` for flexible pattern matching and `every_n_prompts` for periodic triggers based on transcript history. SessionEnd uses `reason_is` to match session end reasons ("clear", "logout", "prompt_input_exit", "other").

**Error Handling**:
- Custom ExitError type for precise control over exit codes and stderr/stdout routing
- Sentinel error pattern for condition type handling to distinguish between "condition not matched" and "condition type unknown"

**Caching**: JQ query compilation caching for performance optimization

## Testing Strategy

Tests are organized into two tiers:

**Unit Tests** (*_test.go files)
- Fast execution with no external dependencies
- Command execution is mocked using `stubRunner` implementation for dependency injection testing
- Tests cover ActionExecutor methods, logic, error handling, and template processing
- Run by default with `go test ./...`
- Examples:
  - `TestExecuteNotificationAction_CommandWithStubRunner`: Tests command execution mocking
  - `TestExecutePreToolUseAction_CommandWithStubRunner`: Tests exit code 2 blocking behavior
  - `TestGetExitStatus`: Tests exit status logic

**Integration Tests** (*_integration_test.go files with `//go:build integration` tag)
- Load real YAML configuration files (e.g., `testdata/integration_test_config.yaml`)
- Execute real shell commands (requires `cat`, `jq`, etc.)
- Test end-to-end hook execution with real config parsing
- Separated to avoid dependency issues and slow test runs
- Run explicitly with `go test -tags=integration ./...`
- Examples:
  - `TestPreToolUseIntegration`: Tests complete PreToolUse flow with YAML config
  - `TestComplexJSONTemplateProcessing`: Tests jq template processing with complex JSON
  - `TestExecutePreToolUseAction_WithUseStdin`: Tests stdin handling with real commands

**Coverage includes:**
- Dependency injection pattern with ActionExecutor
- Hook execution flows with real YAML configuration
- Condition matching and evaluation
- Template processing with jq queries (nested objects, arrays, transformations)
- Exit status control and error handling
- Dry-run functionality
- Transcript parsing for `every_n_prompts` condition
- Git-tracked file operation detection

## Configuration Format

The tool uses YAML configuration with event-specific hook definitions. Each hook contains:
- `matcher`: Tool name pattern matching (partial, pipe-separated)
- `conditions`: Event-specific condition checks
- `actions`: Command execution or output with optional exit_status

Available condition types:
- **Common**:
  - File operations: `file_exists`, `file_exists_recursive`, `file_not_exists`, `file_not_exists_recursive`
  - Directory operations: `dir_exists`, `dir_exists_recursive`, `dir_not_exists`, `dir_not_exists_recursive`
  - Working directory: `cwd_is`, `cwd_is_not`, `cwd_contains`, `cwd_not_contains`
  - Permission mode: `permission_mode_is`
- **Tool-specific** (PreToolUse/PostToolUse): `file_extension`, `command_contains`, `command_starts_with`, `url_starts_with`, `git_tracked_file_operation`
- **Prompt-specific** (UserPromptSubmit):
  - `prompt_regex`: Supports regex patterns including OR conditions with `|`
  - `every_n_prompts`: Triggers every N prompts based on transcript file parsing (counts `type: "user"` entries)
- **Session-specific** (SessionEnd):
  - `reason_is`: Matches session end reason ("clear", "logout", "prompt_input_exit", "other")

Template variables are available based on the event type and include fields from BaseInput, tool-specific data, and full jq query support.

### SessionStart JSON Output

SessionStart hooks support JSON output format for Claude Code integration. Actions can return structured output:

**Output Action** (type: `output`):
```yaml
SessionStart:
  - actions:
      - type: output
        message: "Welcome message"
        continue: true  # optional, defaults to true
```

**Command Action** (type: `command`):
Commands must output JSON with the following structure:
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Message to display"
  },
  "systemMessage": "Optional system message"
}
```

**Field Merging**:
When multiple actions execute:
- `continue`: Last value wins (early return on `false`)
- `hookEventName`: Set once by first action
- `additionalContext` and `systemMessage`: Concatenated with newline separator

**Exit Code Behavior**:
SessionStart hooks **always exit with code 0**, even when:
- Command actions fail or return non-zero exit codes
- JSON parsing errors occur
- Invalid/unsupported fields are detected in command output

Errors are logged to stderr as warnings, but cchook continues to output JSON and exits successfully. This ensures Claude Code always receives a response.

**Example**:
```yaml
SessionStart:
  - matcher: "startup"
    actions:
      - type: output
        message: "Session started"
      - type: command
        command: "get-project-info.sh"  # Returns JSON
```

### UserPromptSubmit JSON Output

UserPromptSubmit hooks support JSON output format for Claude Code integration. Actions can return structured output with decision control:

**Output Action** (type: `output`):
```yaml
UserPromptSubmit:
  - actions:
      - type: output
        message: "Prompt validation message"
        decision: "block"  # optional: "block" only; omit to allow prompt
```

**Command Action** (type: `command`):
Commands must output JSON with the following structure:
```json
{
  "continue": true,
  "decision": "block",
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "Message to display"
  },
  "systemMessage": "Optional system message"
}
```

Note: To allow the prompt, omit the `decision` field entirely.

**Field Merging**:
When multiple actions execute:
- `continue`: Always `true` (cannot be changed for UserPromptSubmit)
- `decision`: Last value wins (early return on `"block"`)
- `hookEventName`: Set once by first action
- `additionalContext` and `systemMessage`: Concatenated with newline separator

**Exit Code Behavior**:
UserPromptSubmit hooks **always exit with code 0**. The `decision` field controls whether the prompt is blocked:
- `decision` field omitted: Prompt processing continues normally
- `"block"`: Prompt processing is blocked (early return)

Errors are logged to stderr as warnings, but cchook continues to output JSON and exits successfully.

**Empty stdout behavior**:
When a command action returns empty stdout (e.g., validation tools that only output on failure):
- The decision field is omitted (allowing the prompt)
- This supports validation-type CLI tools following the Unix philosophy ("silence is golden")

Example:
```yaml
UserPromptSubmit:
  - actions:
      - type: command
        command: "lint-prompt.sh"  # No output on success → decision omitted (allow)
```

Best practice: For clarity, always return explicit JSON output from commands.

**Example**:
```yaml
UserPromptSubmit:
  - conditions:
      - type: prompt_regex
        value: "delete|rm -rf"
    actions:
      - type: output
        message: "Dangerous command detected"
        decision: "block"
```

### PreToolUse JSON Output

PreToolUse hooks support JSON output format for Claude Code integration. Actions can return structured output with 3-stage permission control:

**Output Action** (type: `output`):
```yaml
PreToolUse:
  - matcher: "Write"
    actions:
      - type: output
        message: "Operation validated"
        permission_decision: "allow"  # "allow", "deny", or "ask"
```

**Command Action** (type: `command`):
Commands must output JSON with the following structure:
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "Message to display",
    "updatedInput": {
      "file_path": "modified_path.txt"
    }
  },
  "systemMessage": "Optional system message"
}
```

**Permission Control** (3-stage):
- `"allow"`: Tool execution proceeds normally
- `"deny"`: Tool execution is blocked (early return)
- `"ask"`: Claude Code prompts user for confirmation

**Updated Input**:
The `updatedInput` field allows hooks to modify tool input parameters before execution. This enables:
- Path normalization
- Parameter validation and correction
- Adding default values

**Field Merging**:
When multiple actions execute:
- `continue`: Always `true` (cannot be changed for PreToolUse)
- `permissionDecision`: Last value wins (early return on `"deny"`)
- `permissionDecisionReason` and `systemMessage`: Concatenated with newline separator
- `updatedInput`: Last non-null value wins (merged at top level, not deep merge)
- `hookEventName`: Set once by first action

**Exit Code Behavior**:
PreToolUse hooks **always exit with code 0**. The `permissionDecision` field controls whether the tool execution is allowed, denied, or requires user confirmation.

Errors are logged to stderr as warnings, but cchook continues to output JSON and exits successfully. On errors, `permissionDecision` defaults to `"deny"` for safety.

**Example**:
```yaml
PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "rm -rf"
    actions:
      - type: output
        message: "Dangerous command blocked"
        permission_decision: "deny"
  - matcher: "Write"
    conditions:
      - type: file_extension
        value: ".env"
    actions:
      - type: output
        message: "Modifying sensitive file"
        permission_decision: "ask"
```

### Stop JSON Output

Stop hooks support JSON output format for Claude Code integration. Actions can return structured output with decision control to block or allow Claude's stopping:

**Output Action** (type: `output`):
```yaml
Stop:
  - actions:
      - type: output
        message: "Stop reason message"
        decision: "block"  # optional: "block" only; omit to allow stop
        reason: "Detailed reason for blocking"  # required when decision is "block"
```

**Command Action** (type: `command`):
Commands must output JSON with the following structure:
```json
{
  "continue": true,
  "decision": "block",
  "reason": "Detailed reason for blocking",
  "stopReason": "Optional stop reason",
  "suppressOutput": false,
  "systemMessage": "Optional system message"
}
```

Note: To allow the stop, omit the `decision` field entirely.

**Important**: Unlike other hook types, Stop does NOT use `hookSpecificOutput`. All fields are at the top level.

**Field Merging**:
When multiple actions execute:
- `continue`: Always `true` (cannot be changed for Stop)
- `decision`: Last value wins (early return on `"block"`)
- `reason`: Reset when decision changes; concatenated with newline within same decision
- `systemMessage`: Concatenated with newline separator
- `stopReason` and `suppressOutput`: Last value wins

**Exit Code Behavior**:
Stop hooks **always exit with code 0**. The `decision` field controls whether Claude's stopping is blocked:
- `decision` field omitted: Stop proceeds normally
- `"block"`: Stop is blocked (early return)

Errors are logged to stderr as warnings, but cchook continues to output JSON and exits successfully. On errors, `decision` defaults to `"block"` for safety (fail-safe).

**Backward Compatibility**:
Prior to JSON output support, Stop hooks used exit codes:
- `exit_status: 0` allowed the stop
- `exit_status: 2` (default) blocked the stop

After JSON migration:
- `exit_status` field is **ignored** in output actions
- Use `decision` field instead: omit for allow, `"block"` for deny
- A stderr warning is emitted if `exit_status` is set (migration reminder)

**Example**:
```yaml
Stop:
  - conditions:
      - type: cwd_contains
        value: "/important-project"
    actions:
      - type: output
        message: "Cannot stop in important project directory"
        decision: "block"
        reason: "Stopping Claude in this directory may lose work context"
```

## Common Workflows

### Adding a New Hook Type
1. Define the input structure in `types.go` with embedded BaseInput
2. Add condition types if needed in `types.go` using the opaque struct pattern
3. Implement parsing logic in `parser.go`
4. Add hook execution function in `hooks.go`
5. Implement condition checking in `utils.go` with `(bool, error)` return
6. Add tests in corresponding `*_test.go` files

### Testing Template Processing
Template processing can be tested independently:
```go
// See template_jq_test.go for examples
result := processTemplate("{.tool_name | ascii_upcase}", jsonData)
```

### Debugging Hook Execution
1. Use dry-run mode with `-command` flag to test without side effects
2. Check template expansion with simple echo commands
3. Use verbose test output (`go test -v`) to see detailed execution flow
