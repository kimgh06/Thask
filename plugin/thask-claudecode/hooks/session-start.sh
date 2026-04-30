#!/usr/bin/env bash
# Thask SessionStart hook — inject project context into Claude Code sessions.
# Output on stdout becomes additional context for the agent.

set -uo pipefail

if ! command -v thask >/dev/null 2>&1; then
  echo "[thask plugin] CLI not found in PATH. Install: npm i -g @thask-org/cli" >&2
  exit 0
fi

# Skip silently if user has not configured token yet.
if ! thask config show >/dev/null 2>&1; then
  exit 0
fi

# Pull project guide (current state, recent mistakes, in-progress tasks).
# Falls back gracefully if MCP-style guide is unavailable.
output=$(thask guide 2>/dev/null || true)

if [ -n "$output" ]; then
  printf '## Thask project context\n\n%s\n' "$output"
fi

exit 0
