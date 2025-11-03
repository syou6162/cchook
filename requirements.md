# cchook Requirements

## SessionStart JSON Output

### Requirement 1: Output Format

#### 1.1 Type: Output Action

Type: `output` actions produce `hookSpecificOutput` with these fields:
- `hookEventName`: Always set to "SessionStart"
- `additionalContext`: Set from the `message` field of the action

Example:
```yaml
SessionStart:
  - actions:
      - type: output
        message: "Welcome message"
```

Output:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Welcome message"
  }
}
```

#### 1.2 Type: Command Action - Valid JSON Output

Type: `command` actions must output valid JSON with the following structure:
```json
{
  "continue": boolean,
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "string (optional)"
  },
  "systemMessage": "string (optional)"
}
```

#### 1.3 Type: Command Action - Non-JSON Output

When the command outputs non-JSON text:
- Parse as `additionalContext` (raw text)
- Set `hookEventName` to "SessionStart"
- Set `continue` to true

Example:
```bash
# Command outputs: "Project initialized"
# Result:
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Project initialized"
  },
  "continue": true
}
```

#### 1.4 Type: Command Action - Invalid JSON Output

When the command outputs invalid JSON:
- Log error to stderr as a warning
- Return fallback JSON with `continue: true` and `hookEventName: "SessionStart"`
- No `additionalContext` or `systemMessage` provided

#### 1.5 Type: Command Action - Non-Zero Exit Code

When the command exits with non-zero code:
- Log error to stderr as a warning
- Return fallback JSON with `continue: true` and `hookEventName: "SessionStart"`
- No `additionalContext` or `systemMessage` provided

#### 1.6 Type: Command Action - Empty Output

When the command outputs empty string (requirement 3.7):
- Treat as success
- Return `continue: true` and `hookEventName: "SessionStart"`
- No `additionalContext` provided
- **Rationale**: Validation-type CLI tools (fmt, linter, pre-commit) exit 0 with no output when everything is OK. In this case, we should allow the session to proceed.

### Requirement 2: Field Merging

When multiple actions execute in sequence:

- `continue`: Last value wins (early return on false)
- `hookEventName`: Set once by first action (always "SessionStart")
- `additionalContext` and `systemMessage`: Concatenated with newline separator

### Requirement 3: Exit Code Behavior

#### 3.1 SessionStart Always Exits with Code 0

SessionStart hooks **always exit with code 0**, even when:
- Command actions fail or return non-zero exit codes
- JSON parsing errors occur
- Invalid/unsupported fields are detected in command output

#### 3.2 Exit Code Implications

This ensures Claude Code always receives a response and can handle the result appropriately based on the `continue` field in the returned JSON.

#### 3.3 Error Logging

Errors are logged to stderr as warnings, but cchook continues to output JSON and exits successfully. This ensures Claude Code always receives a response.

#### 3.4 Empty Output as Success

When a command outputs empty string with exit code 0, this is treated as successful validation (requirement 1.6, not an error condition). No warning is logged.

#### 3.5 Empty Output with Non-Zero Exit Code

When a command outputs empty string with non-zero exit code:
- Treat as command failure
- Return fallback JSON with `continue: true`
- Log warning to stderr

#### 3.6 Combined Behavior

The `continue` field in the response indicates whether the session should proceed:
- `continue: true`: Session continues normally
- `continue: false`: Session is blocked/halted (if supported by Claude Code)

#### 3.7 Validation Tool Exit Code 0 as Success

Commands that exit with code 0 and empty output are treated as successful validation:
- No error message is generated
- `continue: true` is returned
- This supports the common pattern of validation tools that output nothing when validation passes

### Requirement 4: JSON Output Structure

All SessionStart hook responses follow this JSON structure:
```json
{
  "continue": boolean,
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "string or omitted"
  },
  "systemMessage": "string or omitted"
}
```

### Requirement 5: Command Output Processing Priority

1. Try to parse as JSON
   - If valid JSON with expected structure, use as-is
   - If valid JSON with unexpected structure, merge fields
   - If invalid JSON, parse as additionalContext
2. Process non-JSON output as additionalContext
3. Handle empty output according to requirement 1.6
