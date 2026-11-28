#!/bin/bash
# Re-run formal verification checks when a commit touches covered sources.
# Discovery reads verify-*/references/coverage.json. There is no receipt file.

set -u

if ! command -v python3 >/dev/null 2>&1 || ! command -v git >/dev/null 2>&1; then
  export PATH="${HOME}/.local/bin:/opt/homebrew/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
fi

if ! command -v python3 >/dev/null 2>&1; then
  printf '%s\n' '{"permission":"deny","user_message":"Commit blocked: python3 is required to run the formal verification gate.","agent_message":"Commit blocked. python3 is not on PATH, so the formal verification gate could not run."}'
  exit 0
fi

script=$(CDPATH= cd -- "$(dirname "$0")/../scripts" && pwd)/verify-commit-gate.py
exec python3 "$script" --hook
