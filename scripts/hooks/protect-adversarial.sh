#!/usr/bin/env bash
# protect-adversarial.sh — PreToolUse hook for Crush.
#
# Blocks any bash command that destructively targets .adversarial/ or its
# subdirectories. Report files and state files are the permanent audit trail
# and must never be deleted by an agent.
#
# Configure in crush.json:
#   {
#     "hooks": {
#       "PreToolUse": [
#         {
#           "matcher": "^bash$",
#           "command": "./scripts/hooks/protect-adversarial.sh"
#         }
#       ]
#     }
#   }
#
# Exit codes (Crush PreToolUse):
#   0  — no opinion; tool proceeds normally
#   2  — block the tool call; stderr is shown to the model as the deny reason

cmd="${CRUSH_TOOL_INPUT_COMMAND:-}"

# Match destructive patterns targeting .adversarial/
# Covers: rm, rmdir, Remove-Item, find -delete, git clean
if echo "$cmd" | grep -qE '(rm|rmdir|Remove-Item|find.*-delete|git\s+clean).*\.adversarial'; then
    echo "BLOCKED: Modifying or deleting files under .adversarial/ is not allowed." >&2
    echo "Report files and state files are the permanent audit trail." >&2
    echo "Only esquisse-mcp or the adversarial-review skill may write to .adversarial/." >&2
    exit 2
fi

exit 0
