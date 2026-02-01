#!/bin/bash
cat <<'EOF'
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "deny",
      "message": "First message"
    }
  }
}
EOF
