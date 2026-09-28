#!/bin/bash
# Gate every agent's git commit on gofmt cleanliness.
# A commit is allowed only when gofmt -l finds no .go files to rewrite.

set -u

if ! command -v gofmt >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1; then
  export PATH="/opt/homebrew/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
fi

if ! command -v python3 >/dev/null 2>&1; then
  printf '%s\n' '{"permission":"deny","user_message":"Commit blocked: python3 is required to run the gofmt gate.","agent_message":"Commit blocked. python3 is not on PATH, so the gofmt gate could not verify that every .go file is gofmt-clean."}'
  exit 0
fi

input_file=$(mktemp)
trap 'rm -f "$input_file"' EXIT
cat > "$input_file"

python3 - "$input_file" <<'PY'
import json
import os
import shutil
import subprocess
import sys

MAX_OFFENDERS = 30
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
            tokens = shlex_split(part)
        except ValueError:
            if unparsed_looks_like_commit(part):
                return True
            continue
        if command_commits(tokens):
            return True
    return False


def shlex_split(part):
    import shlex
    return shlex.split(part, posix=True)


def unparsed_looks_like_commit(part):
    import re
    return re.search(
        r"(?:^|[;&|\n]|&&|\|\|)\s*(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s+)*git\s+commit(?:\s|$)",
        part,
    ) is not None


def scan_roots(payload):
    roots = []
    for root in payload.get("workspace_roots") or []:
        if isinstance(root, str) and os.path.isdir(root):
            roots.append(root)
    if roots:
        return roots
    return [os.getcwd()]


def clip(text):
    text = text.strip()
    if len(text) <= MAX_ERROR_CHARS:
        return text
    return text[:MAX_ERROR_CHARS] + "\n…"


def list_dirty(gofmt, root):
    completed = subprocess.run(
        [gofmt, "-l", "."],
        cwd=root,
        capture_output=True,
        text=True,
        check=False,
    )
    if completed.returncode != 0:
        detail = clip(completed.stderr or completed.stdout or f"gofmt exited {completed.returncode}")
        raise RuntimeError(detail)
    return [line for line in completed.stdout.splitlines() if line.strip()]


def offenders_message(paths):
    shown = paths[:MAX_OFFENDERS]
    lines = [f"- {path}" for path in shown]
    extra = len(paths) - len(shown)
    if extra > 0:
        lines.append(f"- … and {extra} more")
    noun = "file" if len(paths) == 1 else "files"
    return (
        "Commit blocked. Every .go file must be gofmt-clean. "
        f"{len(paths)} {noun} would be rewritten by gofmt:\n"
        + "\n".join(lines)
        + "\nRun gofmt -w on those files (or let the afterFileEdit hook format them), "
        "then create the commit again."
    )


def main():
    raw = open(sys.argv[1], encoding="utf-8").read()
    try:
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        deny(
            "Commit blocked: the gofmt hook could not read its input.",
            "Commit blocked. The gofmt hook received invalid JSON, so it could not verify that every .go file is gofmt-clean.",
        )

    command = payload.get("command", "")
    if not is_git_commit(command):
        allow()

    gofmt = shutil.which("gofmt")
    if not gofmt:
        deny(
            "Commit blocked: gofmt is not available.",
            "Commit blocked. gofmt must be on PATH before a commit can be created. "
            "Install Go (gofmt ships with it), then rerun the commit.",
        )

    dirty = []
    try:
        for root in scan_roots(payload):
            dirty.extend(list_dirty(gofmt, root))
    except (OSError, RuntimeError) as exc:
        deny(
            "Commit blocked: gofmt check failed.",
            "Commit blocked. gofmt check failed, so it could not prove every .go file is gofmt-clean.\n"
            + clip(str(exc)),
        )

    # De-dupe while preserving order.
    seen = set()
    unique = []
    for path in dirty:
        if path not in seen:
            seen.add(path)
            unique.append(path)

    if unique:
        deny(
            "Commit blocked: Go files are not gofmt-clean.",
            offenders_message(unique),
        )

    allow("gofmt check passed. Every .go file is gofmt-clean.")


if __name__ == "__main__":
    main()
PY
