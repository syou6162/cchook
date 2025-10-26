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
```

## Development Commands

```bash
# Build the project
go build -o cchook

# Install dependencies
go mod download
go mod tidy

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test function (more practical than test file)
go test -v -run TestCheckGitTrackedFileOperation ./...
go test -v -run TestExecutePreToolUseHooks ./...
go test -v -run TestCheckUserPromptSubmitCondition ./...

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

Tests are organized by component with comprehensive coverage:
- Unit tests for each module (*_test.go files)
- Integration tests for hook execution flows
- Template processing tests with real-world examples
- Dry-run functionality testing
- Error condition coverage including unknown condition types
- Transcript parsing tests for `every_n_prompts` condition

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
- **Tool-specific** (PreToolUse/PostToolUse): `file_extension`, `command_contains`, `command_starts_with`, `url_starts_with`, `git_tracked_file_operation`
- **Prompt-specific** (UserPromptSubmit):
  - `prompt_regex`: Supports regex patterns including OR conditions with `|`
  - `every_n_prompts`: Triggers every N prompts based on transcript file parsing (counts `type: "user"` entries)
- **Session-specific** (SessionEnd):
  - `reason_is`: Matches session end reason ("clear", "logout", "prompt_input_exit", "other")

Template variables are available based on the event type and include fields from BaseInput, tool-specific data, and full jq query support.

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
