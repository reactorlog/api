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

python3 - "$input_file" <<'PY'
import json
import os
import shutil
import subprocess
import sys

LIMIT = 6.0
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


def entry_score(entry):
    effective = entry.get("effective_crap")
    crap = entry.get("crap")
    if isinstance(effective, (int, float)) and effective != 0:
        return float(effective)
    if isinstance(crap, (int, float)):
        return float(crap)
    if isinstance(effective, (int, float)):
        return float(effective)
    return None


def parse_report(text):
    start = text.find("{")
    if start < 0:
        raise ValueError("go-crap did not return a JSON report")
    report, _ = json.JSONDecoder().raw_decode(text[start:])
    if not isinstance(report, dict) or not isinstance(report.get("entries"), list):
        raise ValueError("go-crap JSON report has no entries list")
    return report


def format_offender(entry, score):
    location = entry.get("file") or "?"
    line = entry.get("line")
    if isinstance(line, int) and line > 0:
        location = f"{location}:{line}"
    name = entry.get("function") or "?"
    receiver = entry.get("receiver")
    if receiver:
        name = f"{receiver}.{name}"
    details = [f"CRAP {score:.2f}"]
    complexity = entry.get("cyclomatic")
    if isinstance(complexity, int):
        details.append(f"complexity {complexity}")
    coverage = entry.get("coverage")
    if isinstance(coverage, (int, float)):
        details.append(f"coverage {coverage:.1f}%")
    return f"- {location} {name} ({', '.join(details)})"


def clip(text):
    text = text.strip()
    if len(text) <= MAX_ERROR_CHARS:
        return text
    return text[:MAX_ERROR_CHARS] + "\n…"


def offenders_message(offenders):
    shown = offenders[:MAX_OFFENDERS]
    lines = [format_offender(entry, score) for entry, score in shown]
    extra = len(offenders) - len(shown)
    if extra > 0:
        lines.append(f"- … and {extra} more")
    noun = "function" if len(offenders) == 1 else "functions"
    return (
        "Commit blocked. Every function must have a go-crap CRAP score under 6 "
        f"(effective score < {LIMIT:.0f}). "
        f"{len(offenders)} {noun} scored 6 or higher:\n"
        + "\n".join(lines)
        + "\nLower the complexity of those functions or cover them with tests, "
        "then create the commit again."
    )


def scan_root(binary, root):
    completed = subprocess.run(
        [binary, "scan", "--format", "json", "--no-progress", "--timeout", "9m"],
        cwd=root,
        capture_output=True,
        text=True,
        timeout=540,
        check=False,
    )
    stdout = completed.stdout or ""
    stderr = completed.stderr or ""
    if completed.returncode != 0:
        detail = clip(stderr or stdout or f"go-crap exited {completed.returncode}")
        raise RuntimeError(detail)
    return parse_report(stdout)


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

    binary = shutil.which("go-crap")
    go_binary = shutil.which("go")
    if not binary or not go_binary:
        deny(
            "Commit blocked: go-crap is not available.",
            "Commit blocked. go-crap and go must both be on PATH before a commit can be created. "
            "Install go-crap, then rerun the commit. Every function must score under 6.",
        )

    found = []
    try:
        for root in scan_roots(payload):
            report = scan_root(binary, root)
            for entry in report["entries"]:
                score = entry_score(entry)
                if score is None or score >= LIMIT:
                    found.append((entry, LIMIT if score is None else score))
    except subprocess.TimeoutExpired:
        deny(
            "Commit blocked: go-crap scan timed out.",
            "Commit blocked. go-crap scan exceeded 9 minutes, so it could not prove every function scores under 6. Fix the test run and commit again.",
        )
    except (OSError, RuntimeError, ValueError) as exc:
        deny(
            "Commit blocked: go-crap scan failed.",
            "Commit blocked. go-crap scan failed, so it could not prove every function scores under 6.\n"
            + clip(str(exc)),
        )

    if found:
        found.sort(key=lambda item: item[1], reverse=True)
        deny(
            "Commit blocked: every function must have a go-crap score under 6.",
            offenders_message(found),
        )

    allow("go-crap scan passed. Every function scores under 6.")


if __name__ == "__main__":
    main()
PY
