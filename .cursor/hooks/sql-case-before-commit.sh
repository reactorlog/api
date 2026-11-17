#!/bin/bash
# Gate every agent's git commit on lowercase SQL keywords.
# A commit is allowed only when .sql files and SQL strings in .go files
# use lowercase keywords (select, not SELECT).

set -u

if ! command -v python3 >/dev/null 2>&1; then
  export PATH="${HOME}/.local/bin:/opt/homebrew/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
fi

if ! command -v python3 >/dev/null 2>&1; then
  printf '%s\n' '{"permission":"deny","user_message":"Commit blocked: python3 is required to run the SQL keyword-case gate.","agent_message":"Commit blocked. python3 is not on PATH, so the SQL keyword-case gate could not run."}'
  exit 0
fi

input_file=$(mktemp)
trap 'rm -f "$input_file"' EXIT
cat > "$input_file"

script=$(CDPATH= cd -- "$(dirname "$0")/../scripts" && pwd)/sql-keyword-case.py
python3 - "$input_file" "$script" <<'PY'
import json
import os
import re
import shlex
import subprocess
import sys

MAX_ERROR_CHARS = 4000

GIT_OPTIONS_WITH_VALUE = {
    "-C",
    "-c",
    "--git-dir",
    "--work-tree",
    "--namespace",
    "--config-env",
    "--super-prefix",
    "--list-cmds",
    "--exec-path",
}


def emit(permission, user_message="", agent_message=""):
    payload = {"permission": permission}
    if user_message:
        payload["user_message"] = user_message
    if agent_message:
        payload["agent_message"] = agent_message
    json.dump(payload, sys.stdout)
    sys.stdout.write("\n")


def deny(user_message, agent_message):
    emit("deny", user_message, agent_message)
    raise SystemExit(0)


def allow(agent_message=""):
    emit("allow", agent_message=agent_message)
    raise SystemExit(0)


def clip(text):
    text = text.strip()
    if len(text) <= MAX_ERROR_CHARS:
        return text
    return text[:MAX_ERROR_CHARS] + "\n…"


def split_commands(source):
    parts = []
    buf = []
    i = 0
    quote = None
    while i < len(source):
        char = source[i]
        if quote:
            buf.append(char)
            if char == "\\" and quote == '"' and i + 1 < len(source):
                buf.append(source[i + 1])
                i += 2
                continue
            if char == quote:
                quote = None
            i += 1
            continue
        if char in ("'", '"'):
            quote = char
            buf.append(char)
            i += 1
            continue
        if source.startswith("&&", i) or source.startswith("||", i):
            parts.append("".join(buf))
            buf = []
            i += 2
            continue
        if char in ";\n|":
            parts.append("".join(buf))
            buf = []
            i += 1
            continue
        buf.append(char)
        i += 1
    parts.append("".join(buf))
    return parts


def is_git_token(token):
    return os.path.basename(token) == "git"


def command_commits(tokens):
    index = 0
    while index < len(tokens) and "=" in tokens[index] and not tokens[index].startswith("-"):
        index += 1
    if index < len(tokens) and tokens[index] in ("command", "env", "nice", "nohup"):
        index += 1
        while index < len(tokens) and (
            tokens[index].startswith("-") or "=" in tokens[index]
        ):
            index += 1
    if index >= len(tokens) or not is_git_token(tokens[index]):
        return False
    index += 1
    while index < len(tokens):
        token = tokens[index]
        if token == "commit":
            return True
        if token == "--":
            return False
        if token.startswith("--") and "=" in token:
            index += 1
            continue
        if token in GIT_OPTIONS_WITH_VALUE:
            index += 2
            continue
        if token.startswith("-"):
            index += 1
            continue
        return False
    return False


def is_git_commit(command):
    if not isinstance(command, str) or not command.strip():
        return False
    for part in split_commands(command):
        part = part.strip()
        if not part:
            continue
        try:
            tokens = shlex.split(part, posix=True)
        except ValueError:
            if re.search(
                r"(?:^|[;&|\n]|&&|\|\|)\s*(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s+)*git\s+commit(?:\s|$)",
                part,
            ):
                return True
            continue
        if command_commits(tokens):
            return True
    return False


def scan_roots(payload):
    roots = []
    for root in payload.get("workspace_roots") or []:
        if isinstance(root, str) and os.path.isdir(root):
            roots.append(root)
    if roots:
        return roots
    return [os.getcwd()]


def main():
    raw = open(sys.argv[1], encoding="utf-8").read()
    try:
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        deny(
            "Commit blocked: the SQL keyword-case hook could not read its input.",
            "Commit blocked. The SQL keyword-case hook received invalid JSON.",
        )

    command = payload.get("command", "")
    if not is_git_commit(command):
        allow()

    script = sys.argv[2]
    errors = []
    for root in scan_roots(payload):
        completed = subprocess.run(
            [sys.executable, script],
            cwd=root,
            capture_output=True,
            text=True,
            check=False,
        )
        if completed.returncode == 0:
            continue
        detail = completed.stderr.strip() or completed.stdout.strip() or f"exit {completed.returncode}"
        errors.append(clip(detail))

    if errors:
        body = "\n\n".join(errors)
        deny(
            "Commit blocked: SQL keywords must be lowercase.",
            "Commit blocked. SQL keywords must be lowercase (select, not SELECT).\n"
            + body
            + "\nRun: python3 .cursor/scripts/sql-keyword-case.py",
        )

    allow("SQL keyword-case check passed.")


if __name__ == "__main__":
    main()
PY
