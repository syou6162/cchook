#!/bin/bash
# Output deny behavior with message and interrupt
cat <<'EOF'
{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Initial block","interrupt":true}}}
EOF
