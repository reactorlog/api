#!/bin/bash
# Gate every agent's git commit on a go-crap scan.
# A commit is allowed only when every function scores strictly under 6.

set -u

if ! command -v go-crap >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1; then
  export PATH="${HOME}/.local/bin:/opt/homebrew/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
fi

if ! command -v python3 >/dev/null 2>&1; then
  printf '%s\n' '{"permission":"deny","user_message":"Commit blocked: python3 is required to run the go-crap score gate.","agent_message":"Commit blocked. python3 is not on PATH, so the go-crap gate could not verify that every function scores under 6."}'
  exit 0
fi

input_file=$(mktemp)
trap 'rm -f "$input_file"' EXIT
cat > "$input_file"

script=$(CDPATH= cd -- "$(dirname "$0")/../scripts" && pwd)/go-crap-report.py
python3 - "$input_file" "$script" <<'PY'
import json
import os
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
    # Fail closed only when an unquoted git commit invocation is visible.
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


def load_report(completed):
    text = completed.stdout or ""
    start = text.find("{")
    if start < 0:
        detail = clip(completed.stderr or text or f"go-crap report exited {completed.returncode}")
        raise ValueError(detail or f"go-crap report exited {completed.returncode}")
    report, _ = json.JSONDecoder().raw_decode(text[start:])
    if not isinstance(report, dict):
        raise ValueError("go-crap report was not a JSON object")
    return report


def deny_from_report(report):
    error = report.get("error")
    if error == "missing_binary":
        deny(
            "Commit blocked: go-crap is not available.",
            "Commit blocked. go-crap and go must both be on PATH before a commit can be created. "
            "Install go-crap, then rerun the commit. Every function must score under 6.",
        )
    if error == "timeout":
        deny(
            "Commit blocked: go-crap scan timed out.",
            "Commit blocked. go-crap scan exceeded 9 minutes, so it could not prove every function scores under 6. Fix the test run and commit again.",
        )
    if error:
        deny(
            "Commit blocked: go-crap scan failed.",
            "Commit blocked. go-crap scan failed, so it could not prove every function scores under 6.\n"
            + clip(str(report.get("detail") or "")),
        )
    summary = report.get("summary")
    if not isinstance(summary, str) or not summary.strip():
        deny(
            "Commit blocked: go-crap scan failed.",
            "Commit blocked. go-crap scan failed, so it could not prove every function scores under 6.\n"
            "go-crap report did not include an offender summary.",
        )
    deny(
        "Commit blocked: every function must have a go-crap score under 6.",
        "Commit blocked. "
        + summary
        + "\nLower the complexity of those functions or cover them with tests, "
        "then create the commit again.",
    )


def main():
    raw = open(sys.argv[1], encoding="utf-8").read()
    try:
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        deny(
            "Commit blocked: the go-crap hook could not read its input.",
            "Commit blocked. The go-crap hook received invalid JSON, so it could not verify that every function scores under 6.",
        )

    command = payload.get("command", "")
    if not is_git_commit(command):
        allow()

    script = sys.argv[2]
    command_args = [sys.executable, script]
    for root in scan_roots(payload):
        command_args.extend(["--root", root])
    try:
        completed = subprocess.run(
            command_args,
            capture_output=True,
            text=True,
            timeout=560,
            check=False,
        )
        report = load_report(completed)
    except subprocess.TimeoutExpired:
        deny(
            "Commit blocked: go-crap scan timed out.",
            "Commit blocked. go-crap scan exceeded 9 minutes, so it could not prove every function scores under 6. Fix the test run and commit again.",
        )
    except (OSError, ValueError) as exc:
        deny(
            "Commit blocked: go-crap scan failed.",
            "Commit blocked. go-crap scan failed, so it could not prove every function scores under 6.\n"
            + clip(str(exc)),
        )

    if completed.returncode == 0 and report.get("ok") is True:
        allow("go-crap scan passed. Every function scores under 6.")

    deny_from_report(report)


if __name__ == "__main__":
    main()
PY
