#!/bin/bash
# Output allow behavior with updatedInput containing template variable
cat <<'EOF'
{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","updatedInput":{"file_path":"test.go"}}}}
EOF
