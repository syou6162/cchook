# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cchook is a CLI tool that simplifies Claude Code hook configuration by providing YAML-based configuration and template syntax instead of complex JSON one-liners. It processes JSON input from Claude Code hooks via stdin and executes configured actions based on matching conditions.

## Development Commands

```bash
# Build the project
go build -o cchook

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test file
go test -v ./hooks_test.go

# Run specific test function
go test -v -run TestExecutePreToolUseHooks ./hooks_test.go

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
- Defines all event types: PreToolUse, PostToolUse, Stop, SubagentStop, Notification, PreCompact
- Event-specific input structures with embedded BaseInput
- Hook and Action interfaces for polymorphic behavior
- Separate condition and action types for each event

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
- Condition checking functions per event type
- Command execution wrapper
- File existence, extension, and URL pattern matching

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

**Condition System**: Event-specific condition types with common patterns (file_extension, command_contains, etc.)

**Error Handling**: Custom ExitError type for precise control over exit codes and stderr/stdout routing

**Caching**: JQ query compilation caching for performance optimization

## Testing Strategy

Tests are organized by component with comprehensive coverage:
- Unit tests for each module (*_test.go files)
- Integration tests for hook execution flows
- Template processing tests with real-world examples
- Dry-run functionality testing
- Error condition coverage

## Configuration Format

The tool uses YAML configuration with event-specific hook definitions. Each hook contains:
- `matcher`: Tool name pattern matching (partial, pipe-separated)
- `conditions`: Event-specific condition checks
- `actions`: Command execution or output with optional exit_status

Template variables are available based on the event type and include fields from BaseInput, tool-specific data, and full jq query support.

## Common Workflows

### Adding a New Hook Type
1. Define the input structure in `types.go` with embedded BaseInput
2. Add condition types if needed in `types.go`
3. Implement parsing logic in `parser.go`
4. Add hook execution function in `hooks.go`
5. Implement condition checking in `utils.go`
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
