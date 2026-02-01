#!/bin/bash
cat <<'EOF'
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "deny",
      "message": "Second message"
    }
  },
  "systemMessage": "System message"
}
EOF
