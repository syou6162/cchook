#!/bin/bash
# Output deny behavior with message
cat <<'EOF'
{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Operation blocked"}}}
EOF
