#!/bin/bash
# Read from stdin and output allow behavior with updatedInput from input
FILE_PATH=$(jq -r '.tool_input.file_path')
cat <<EOF
{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","updatedInput":{"file_path":"$FILE_PATH"}}}}
EOF
