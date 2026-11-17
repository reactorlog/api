#!/usr/bin/env python3
"""Fail closed when a git commit touches formally verified sources.

Discovery reads .cursor/skills/verify-*/references/coverage.json from the
worktree, the index, and HEAD. A matching commit re-runs

    python3 .cursor/skills/<skill>/scripts/check_go_mutation.py

Dry run:

    python3 .cursor/scripts/verify-commit-gate.py --dry-run --path cmd/reactorlog/main.go

Proof:

    python3 .cursor/scripts/verify-commit-gate.py --self-test
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import subprocess
import sys
import tempfile
import traceback
from dataclasses import dataclass
from pathlib import Path

CHECK_TIMEOUT_SECONDS = 540
MAX_ERROR_CHARS = 4000
COVERAGE_PATH = re.compile(
    r"^\.cursor/skills/(verify-[A-Za-z0-9][A-Za-z0-9-]*)/references/coverage\.json\Z"
)
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
COMMIT_VALUE_OPTIONS = {
    "-m",
    "--message",
    "-F",
    "--file",
    "-c",
    "--reedit-message",
    "-C",
    "--reuse-message",
    "--fixup",
    "--squash",
    "-t",
    "--template",
    "--author",
    "--date",
    "--cleanup",
    "-u",
    "--untracked-files",
    "--trailer",
}
SHORT_VALUE_LETTERS = set("mFcCut")


@dataclass(frozen=True)
class Skill:
    name: str
    tree: str
    files: frozenset[str]
    active_files: frozenset[str]
    check_rel: str
    check_exists: bool


@dataclass(frozen=True)
class Catalog:
    skills: tuple[Skill, ...]
    errors: tuple[str, ...]


@dataclass(frozen=True)
class RawManifest:
    name: str
    origin: str
    text: str | None
    error: str | None


@dataclass(frozen=True)
class CommitIntent:
    workdir: Path | None
    mode: str
    pathspecs: tuple[str, ...]


@dataclass(frozen=True)
class Run:
    skill: Skill
    paths: tuple[str, ...]


@dataclass(frozen=True)
class Block:
    skill: Skill
    reason: str
    paths: tuple[str, ...]


@dataclass(frozen=True)
class Plan:
    runs: tuple[Run, ...]
    blocked: tuple[Block, ...]


@dataclass(frozen=True)
class Decision:
    permission: str
    user_message: str
    agent_message: str


class GitError(RuntimeError):
    pass


def default_root() -> Path:
    return Path(__file__).resolve().parents[2]


def command_line(skill: Skill) -> str:
    return f"python3 {skill.check_rel}"


def clip(text: str) -> str:
    text = text.strip()
    if len(text) <= MAX_ERROR_CHARS:
        return text
    return text[:MAX_ERROR_CHARS] + "\n…"


def normalize_repo_path(path: str) -> str:
    path = path.replace("\\", "/").strip()
    if path.startswith("./"):
        path = path[2:]
    parts = path.split("/")
    if (
        not path
        or path.startswith("/")
        or any(part in ("", "..") for part in parts)
    ):
        raise ValueError(f"invalid coverage path {path}")
    return path


def git(root: Path, args: list[str]) -> subprocess.CompletedProcess[bytes]:
    env = os.environ.copy()
    env["GIT_TERMINAL_PROMPT"] = "0"
    env["GIT_PAGER"] = "cat"
    try:
        return subprocess.run(
            ["git", "-c", "diff.renames=true", "-C", str(root), *args],
            capture_output=True,
            check=False,
            env=env,
        )
    except OSError as exc:
        raise GitError(str(exc)) from exc


def git_ok(root: Path, args: list[str]) -> bytes:
    result = git(root, args)
    if result.returncode != 0:
        detail = result.stderr.decode("utf-8", "replace").strip()
        raise GitError(detail or f"git {' '.join(args)} exited {result.returncode}")
    return result.stdout


def git_toplevel(start: Path) -> Path | None:
    if not start.is_dir():
        return None
    result = git(start, ["rev-parse", "--show-toplevel"])
    if result.returncode != 0:
        return None
    text = result.stdout.decode("utf-8", "replace").strip()
    if not text:
        return None
    return Path(text)


def has_head(root: Path) -> bool:
    return git(root, ["rev-parse", "--verify", "-q", "HEAD"]).returncode == 0


def split_z(blob: bytes) -> list[str]:
    if not blob:
        return []
    parts = blob.split(b"\0")
    if parts and parts[-1] == b"":
        parts.pop()
    try:
        return [part.decode("utf-8") for part in parts]
    except UnicodeDecodeError as exc:
        raise GitError(f"git path is not utf-8: {exc}") from exc


def parse_name_status(blob: bytes) -> list[str]:
    parts = split_z(blob)
    paths: list[str] = []
    index = 0
    while index < len(parts):
        status = parts[index]
        index += 1
        if not status:
            raise GitError("empty git status")
        if status[0] in "RC":
            if index + 1 >= len(parts):
                raise GitError("truncated git rename record")
            paths.append(parts[index])
            paths.append(parts[index + 1])
            index += 2
            continue
        if index >= len(parts):
            raise GitError("truncated git status record")
        paths.append(parts[index])
        index += 1
    return paths


def parse_files(text: str) -> frozenset[str]:
    try:
        data = json.loads(text)
    except json.JSONDecodeError as exc:
        raise ValueError(f"is not valid JSON: {exc.msg}") from exc
    if not isinstance(data, dict):
        raise ValueError("is not a JSON object")
    files = data.get("files")
    if not isinstance(files, list):
        raise ValueError("files is not a list")
    parsed: set[str] = set()
    for item in files:
        if not isinstance(item, str):
            raise ValueError("files contains a non-string")
        parsed.add(normalize_repo_path(item))
    return frozenset(parsed)


def worktree_manifests(root: Path) -> list[RawManifest]:
    skills = root / ".cursor" / "skills"
    if not skills.is_dir():
        return []
    found: list[RawManifest] = []
    for path in sorted(skills.glob("verify-*/references/coverage.json")):
        relative = path.relative_to(root).as_posix()
        match = COVERAGE_PATH.match(relative)
        if not match:
            found.append(
                RawManifest(
                    relative,
                    "worktree",
                    None,
                    f"{relative} path is not a verify skill coverage file",
                )
            )
            continue
        name = match.group(1)
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:
            found.append(
                RawManifest(name, "worktree", None, f"{name} coverage.json (worktree) {exc}")
            )
            continue
        found.append(RawManifest(name, "worktree", text, None))
    return found


def git_manifests(root: Path, origin: str) -> list[RawManifest]:
    if origin == "head":
        if not has_head(root):
            return []
        blob = git_ok(root, ["ls-tree", "-r", "--name-only", "-z", "HEAD", "--", ".cursor/skills"])
    else:
        blob = git_ok(root, ["ls-files", "-z", "--", ".cursor/skills"])
    found: list[RawManifest] = []
    for relative in split_z(blob):
        match = COVERAGE_PATH.match(relative)
        if not match:
            continue
        name = match.group(1)
        spec = f"HEAD:{relative}" if origin == "head" else f":{relative}"
        result = git(root, ["show", spec])
        if result.returncode != 0:
            detail = result.stderr.decode("utf-8", "replace").strip()
            found.append(
                RawManifest(name, origin, None, f"{name} coverage.json ({origin}) {detail}")
            )
            continue
        try:
            text = result.stdout.decode("utf-8")
        except UnicodeDecodeError as exc:
            found.append(RawManifest(name, origin, None, f"{name} coverage.json ({origin}) {exc}"))
            continue
        found.append(RawManifest(name, origin, text, None))
    return found


def discover(root: Path) -> Catalog:
    grouped: dict[str, list[RawManifest]] = {}
    for raw in (
        worktree_manifests(root) + git_manifests(root, "index") + git_manifests(root, "head")
    ):
        grouped.setdefault(raw.name, []).append(raw)
    skills: list[Skill] = []
    errors: list[str] = []
    for name in sorted(grouped):
        files: set[str] = set()
        active: frozenset[str] | None = None
        usable = True
        for raw in grouped[name]:
            if raw.error:
                errors.append(raw.error)
                usable = False
                continue
            try:
                parsed = parse_files(raw.text or "")
            except ValueError as exc:
                errors.append(f"{name} coverage.json ({raw.origin}) {exc}")
                usable = False
                continue
            files |= parsed
            if raw.origin == "worktree":
                active = parsed
        if not usable:
            continue
        check_rel = f".cursor/skills/{name}/scripts/check_go_mutation.py"
        skills.append(
            Skill(
                name=name,
                tree=f".cursor/skills/{name}",
                files=frozenset(files),
                active_files=frozenset() if active is None else active,
                check_rel=check_rel,
                check_exists=(root / check_rel).is_file(),
            )
        )
    return Catalog(tuple(skills), tuple(errors))


def covers(skill: Skill, path: str) -> bool:
    # Fingerprint and other JSON metadata are not state-machine source.
    # Editing coverage.json must not require re-running the mutation check.
    if path.endswith(".json"):
        return False
    if path in skill.files:
        return True
    return path == skill.tree or path.startswith(skill.tree + "/")


def plan_paths(catalog: Catalog, paths: list[str]) -> Plan:
    path_set = set(paths)
    runs: list[Run] = []
    blocked: list[Block] = []
    for skill in catalog.skills:
        matched = tuple(sorted(path for path in path_set if covers(skill, path)))
        if not matched:
            continue
        dropped = tuple(sorted((skill.files - skill.active_files) & path_set))
        if dropped:
            blocked.append(Block(skill, "stops covering", dropped))
            continue
        if not skill.check_exists:
            blocked.append(Block(skill, "missing check", matched))
            continue
        runs.append(Run(skill, matched))
    return Plan(tuple(runs), tuple(blocked))


def untracked(root: Path, pathspecs: list[str] | None) -> list[str]:
    args = ["ls-files", "--others", "--exclude-standard", "-z"]
    if pathspecs is not None:
        args.extend(["--", *pathspecs])
    return split_z(git_ok(root, args))


def changes_without_head(
    root: Path, pathspecs: list[str] | None, include_untracked: bool
) -> list[str]:
    cached = ["diff", "--cached", "--name-status", "-z"]
    worktree = ["diff", "--name-status", "-z"]
    if pathspecs is not None:
        cached.extend(["--", *pathspecs])
        worktree.extend(["--", *pathspecs])
    paths = parse_name_status(git_ok(root, cached)) + parse_name_status(git_ok(root, worktree))
    if include_untracked:
        paths.extend(untracked(root, pathspecs))
    return paths


def diff_head(root: Path, pathspecs: list[str] | None, include_untracked: bool) -> list[str]:
    if not has_head(root):
        return changes_without_head(root, pathspecs, include_untracked)
    args = ["diff", "HEAD", "--name-status", "-z"]
    if pathspecs is not None:
        args.extend(["--", *pathspecs])
    paths = parse_name_status(git_ok(root, args))
    if include_untracked:
        paths.extend(untracked(root, pathspecs))
    return paths


def collect_paths(root: Path, intent: CommitIntent) -> list[str]:
    if intent.mode == "all":
        return diff_head(root, None, True)
    if intent.mode == "tracked":
        return diff_head(root, None, False)
    if intent.mode == "paths":
        return diff_head(root, list(intent.pathspecs), True)
    return parse_name_status(git_ok(root, ["diff", "--cached", "--name-status", "-z"]))


def clean_paths(paths: list[str]) -> list[str]:
    seen: set[str] = set()
    cleaned: list[str] = []
    for path in paths:
        try:
            normal = normalize_repo_path(path)
        except ValueError as exc:
            raise GitError(str(exc)) from exc
        if normal not in seen:
            seen.add(normal)
            cleaned.append(normal)
    return cleaned


def split_commands(source: str) -> list[str]:
    parts: list[str] = []
    buf: list[str] = []
    index = 0
    quote: str | None = None
    while index < len(source):
        char = source[index]
        if quote:
            buf.append(char)
            if char == "\\" and quote == '"' and index + 1 < len(source):
                buf.append(source[index + 1])
                index += 2
                continue
            if char == quote:
                quote = None
            index += 1
            continue
        if char in ("'", '"'):
            quote = char
            buf.append(char)
            index += 1
            continue
        if source.startswith("&&", index) or source.startswith("||", index):
            parts.append("".join(buf))
            buf = []
            index += 2
            continue
        if char in ";\n|":
            parts.append("".join(buf))
            buf = []
            index += 1
            continue
        buf.append(char)
        index += 1
    parts.append("".join(buf))
    return parts


def unparsed_looks_like_commit(part: str) -> bool:
    return (
        re.search(
            r"(?:^|[;&|\n]|&&|\|\|)\s*(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s+)*git\s+commit(?:\s|$)",
            part,
        )
        is not None
    )


def skip_wrappers(tokens: list[str]) -> int:
    index = 0
    while index < len(tokens) and "=" in tokens[index] and not tokens[index].startswith("-"):
        index += 1
    if index < len(tokens) and tokens[index] in ("command", "env", "nice", "nohup"):
        index += 1
        while index < len(tokens) and (
            tokens[index].startswith("-") or "=" in tokens[index]
        ):
            index += 1
    return index


def strip_git_globals(
    tokens: list[str],
) -> tuple[Path | None, bool, list[str]] | None:
    index = 1
    workdir: Path | None = None
    opaque = False
    while index < len(tokens):
        token = tokens[index]
        if token == "commit":
            return workdir, opaque, tokens[index + 1 :]
        if token == "--":
            return None
        if token in ("-C", "--work-tree") and index + 1 < len(tokens):
            workdir = Path(tokens[index + 1])
            index += 2
            continue
        if token.startswith("--work-tree="):
            workdir = Path(token.split("=", 1)[1])
            index += 1
            continue
        if token.startswith("-C") and not token.startswith("--") and token != "-C":
            workdir = Path(token[2:])
            index += 1
            continue
        if token == "--git-dir" and index + 1 < len(tokens):
            opaque = True
            index += 2
            continue
        if token.startswith("--git-dir="):
            opaque = True
            index += 1
            continue
        if token in GIT_OPTIONS_WITH_VALUE:
            index += 2
            continue
        if token.startswith("--") and "=" in token:
            index += 1
            continue
        if token.startswith("-"):
            index += 1
            continue
        return None
    return None


def parse_commit_args(tokens: list[str]) -> tuple[str, tuple[str, ...]]:
    mode = "staged"
    pathspecs: list[str] = []
    index = 0
    while index < len(tokens):
        token = tokens[index]
        if token == "--":
            pathspecs.extend(tokens[index + 1 :])
            break
        if token in ("-a", "--all") or token == "--all=true":
            if mode == "staged":
                mode = "tracked"
            index += 1
            continue
        if token == "--pathspec-from-file" or token.startswith("--pathspec-from-file="):
            return "all", ()
        if token.startswith("--") and "=" in token:
            index += 1
            continue
        if token in COMMIT_VALUE_OPTIONS:
            index += 2
            continue
        if token.startswith("--"):
            index += 1
            continue
        if token.startswith("-") and len(token) > 1:
            cluster = token[1:]
            if "a" in cluster and mode == "staged":
                mode = "tracked"
            if any(char in SHORT_VALUE_LETTERS for char in cluster):
                index += 2 if cluster[-1] in SHORT_VALUE_LETTERS else 1
            else:
                index += 1
            continue
        pathspecs.append(token)
        index += 1
    if pathspecs:
        return "paths", tuple(pathspecs)
    return mode, ()


def intent_from_tokens(tokens: list[str]) -> CommitIntent | None:
    index = skip_wrappers(tokens)
    if index >= len(tokens) or os.path.basename(tokens[index]) != "git":
        return None
    parsed = strip_git_globals(tokens[index:])
    if parsed is None:
        return None
    workdir, opaque, commit_args = parsed
    if opaque:
        return CommitIntent(workdir, "all", ())
    mode, pathspecs = parse_commit_args(commit_args)
    return CommitIntent(workdir, mode, pathspecs)


def parse_commits(command: str) -> list[CommitIntent]:
    intents: list[CommitIntent] = []
    for part in split_commands(command):
        part = part.strip()
        if not part:
            continue
        try:
            tokens = shlex.split(part, posix=True)
        except ValueError:
            if unparsed_looks_like_commit(part):
                intents.append(CommitIntent(None, "all", ()))
            continue
        intent = intent_from_tokens(tokens)
        if intent is not None:
            intents.append(intent)
    return intents


def hook_cwd_from(payload: dict) -> Path:
    cwd = payload.get("cwd")
    if isinstance(cwd, str) and cwd:
        return Path(cwd)
    return Path.cwd()


def payload_roots(payload: dict, cwd: Path) -> list[Path]:
    roots: list[Path] = []
    values = payload.get("workspace_roots")
    if isinstance(values, list):
        for value in values:
            if isinstance(value, str) and value:
                roots.append(Path(value))
    if roots:
        return roots
    return [cwd]


def roots_for(
    payload: dict, intent: CommitIntent, cwd: Path
) -> tuple[list[Path], str | None]:
    if intent.workdir is not None:
        path = intent.workdir if intent.workdir.is_absolute() else cwd / intent.workdir
        top = git_toplevel(path)
        if top is None:
            return [], str(path)
        return [top], None
    found: list[Path] = []
    for root in payload_roots(payload, cwd):
        top = git_toplevel(root)
        if top is not None and top not in found:
            found.append(top)
    return found, None


def show_paths(paths: tuple[str, ...] | list[str]) -> str:
    shown = list(paths[:30])
    text = ", ".join(shown)
    extra = len(paths) - len(shown)
    if extra:
        text += f", and {extra} more"
    return text


def deny(user_message: str, agent_message: str) -> Decision:
    return Decision("deny", user_message, agent_message)


def allow_clear() -> Decision:
    return Decision("allow", "", "No formally verified paths are in this commit.")


def deny_catalog(catalog: Catalog) -> Decision:
    lines = "\n".join(f"- {error}" for error in catalog.errors)
    return deny(
        "Commit blocked: the formal verification catalog cannot be read.",
        "Commit blocked. The formal verification catalog cannot be read, "
        "so the commit cannot prove which sources are covered.\n"
        + lines,
    )


def deny_blocked(blocked: tuple[Block, ...]) -> Decision:
    lines: list[str] = []
    stopped = any(block.reason == "stops covering" for block in blocked)
    missing = any(block.reason == "missing check" for block in blocked)
    for block in blocked:
        command = command_line(block.skill)
        if block.reason == "stops covering":
            lines.append(
                f"This commit changes {show_paths(block.paths)} and stops covering "
                f"it with {block.skill.name}."
            )
            lines.append(command)
            lines.append(
                "Land the coverage edit in a commit that does not change that path. "
                "A passing check uses the coverage on disk and does not clear a path "
                "removed from coverage.json in the same commit."
            )
        else:
            lines.append(
                f"{block.skill.name} matches {show_paths(block.paths)} and its check script is missing."
            )
            lines.append(command)
    if stopped:
        user = "Commit blocked: this commit changes a path it stops covering."
    elif missing:
        user = "Commit blocked: the verification check script is missing."
    else:
        user = "Commit blocked: formal verification could not run."
    return deny(user, "Commit blocked. " + "\n".join(lines))


def deny_check(
    run: Run,
    detail: str,
    remaining: tuple[Run, ...],
) -> Decision:
    lines = [
        f"{run.skill.name} covers {show_paths(run.paths)}.",
        command_line(run.skill),
        detail.strip(),
    ]
    if remaining:
        lines.append("Not run yet:")
        lines.extend(command_line(item.skill) for item in remaining)
    return deny(
        "Commit blocked: a formal verification check failed.",
        "Commit blocked. " + "\n".join(line for line in lines if line),
    )


def execute(root: Path, runs: tuple[Run, ...]) -> Decision:
    for index, run in enumerate(runs):
        script = root / run.skill.check_rel
        try:
            completed = subprocess.run(
                [sys.executable, str(script)],
                cwd=root,
                capture_output=True,
                text=True,
                timeout=CHECK_TIMEOUT_SECONDS,
                check=False,
            )
        except subprocess.TimeoutExpired:
            return deny_check(
                run,
                f"The check exceeded {CHECK_TIMEOUT_SECONDS} seconds.",
                runs[index + 1 :],
            )
        except OSError as exc:
            return deny_check(run, str(exc), runs[index + 1 :])
        if completed.returncode != 0:
            output = clip(f"{completed.stdout or ''}\n{completed.stderr or ''}")
            return deny_check(
                run,
                output or f"The check exited {completed.returncode}.",
                runs[index + 1 :],
            )
    commands = "\n".join(command_line(run.skill) for run in runs)
    return Decision("allow", "", "Formal verification checks passed.\n" + commands)


def evaluate_root(root: Path, intent: CommitIntent) -> Decision:
    catalog = discover(root)
    if catalog.errors:
        return deny_catalog(catalog)
    paths = clean_paths(collect_paths(root, intent))
    chosen = plan_paths(catalog, paths)
    if chosen.blocked:
        return deny_blocked(chosen.blocked)
    if not chosen.runs:
        return allow_clear()
    return execute(root, chosen.runs)


def merge_decisions(decisions: list[Decision]) -> Decision:
    denied = [decision for decision in decisions if decision.permission == "deny"]
    if denied:
        return deny(
            denied[0].user_message,
            "\n\n".join(decision.agent_message for decision in denied if decision.agent_message),
        )
    messages = [decision.agent_message for decision in decisions if decision.agent_message]
    return Decision("allow", "", "\n".join(messages))


def evaluate(payload: dict, cwd: Path | None = None) -> Decision:
    command = payload.get("command", "")
    if not isinstance(command, str):
        return deny(
            "Commit blocked: the formal verification gate could not read its input.",
            "Commit blocked. The command field was not a string.",
        )
    try:
        intents = parse_commits(command)
    except ValueError as exc:
        return deny(
            "Commit blocked: the formal verification gate could not read the git command.",
            "Commit blocked. " + str(exc),
        )
    if not intents:
        return Decision("allow", "", "")
    hook_cwd = cwd or hook_cwd_from(payload)
    decisions: list[Decision] = []
    try:
        for intent in intents:
            roots, explicit_error = roots_for(payload, intent, hook_cwd)
            if explicit_error:
                decisions.append(
                    deny(
                        "Commit blocked: the formal verification gate could not read the git repository.",
                        "Commit blocked. git -C does not name a repository: " + explicit_error,
                    )
                )
                continue
            if not roots:
                decisions.append(
                    deny(
                        "Commit blocked: the formal verification gate could not read the git repository.",
                        "Commit blocked. No workspace root is a git repository.",
                    )
                )
                continue
            for root in roots:
                decisions.append(evaluate_root(root, intent))
    except GitError as exc:
        return deny(
            "Commit blocked: the formal verification gate could not read git.",
            "Commit blocked. " + clip(str(exc)),
        )
    return merge_decisions(decisions)


def emit(decision: Decision) -> None:
    payload: dict[str, str] = {"permission": decision.permission}
    if decision.user_message:
        payload["user_message"] = decision.user_message
    if decision.agent_message:
        payload["agent_message"] = decision.agent_message
    json.dump(payload, sys.stdout)
    sys.stdout.write("\n")


def hook_main(raw: str) -> int:
    if not raw.strip():
        payload: object = {}
    else:
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            emit(
                deny(
                    "Commit blocked: the formal verification gate could not read its input.",
                    "Commit blocked. The formal verification gate received invalid JSON.",
                )
            )
            return 0
    if not isinstance(payload, dict):
        emit(
            deny(
                "Commit blocked: the formal verification gate could not read its input.",
                "Commit blocked. The formal verification gate received a JSON value that is not an object.",
            )
        )
        return 0
    try:
        decision = evaluate(payload)
    except Exception as exc:
        traceback.print_exc(file=sys.stderr)
        emit(
            deny(
                "Commit blocked: the formal verification gate failed.",
                "Commit blocked. The formal verification gate failed.\n" + clip(str(exc)),
            )
        )
        return 0
    emit(decision)
    return 0


def skill_json(skill: Skill) -> dict[str, object]:
    return {
        "name": skill.name,
        "command": command_line(skill),
        "files": sorted(skill.files),
        "check_exists": skill.check_exists,
    }


def dry_run_main(root: Path, paths: list[str]) -> int:
    try:
        catalog = discover(root)
        normal = [normalize_repo_path(path) for path in paths]
        chosen = plan_paths(catalog, normal)
    except (GitError, ValueError) as exc:
        json.dump(
            {"errors": [str(exc)], "skills": [], "runs": [], "blocked": []},
            sys.stdout,
            indent=2,
        )
        sys.stdout.write("\n")
        return 1
    report = {
        "errors": list(catalog.errors),
        "skills": [skill_json(skill) for skill in catalog.skills],
        "runs": [
            {
                "name": run.skill.name,
                "command": command_line(run.skill),
                "paths": list(run.paths),
            }
            for run in chosen.runs
        ],
        "blocked": [
            {
                "name": block.skill.name,
                "command": command_line(block.skill),
                "reason": block.reason,
                "paths": list(block.paths),
            }
            for block in chosen.blocked
        ],
    }
    json.dump(report, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 1 if catalog.errors else 0


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hook", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--self-test", action="store_true")
    parser.add_argument("--root", type=Path)
    parser.add_argument("--path", action="append", default=[])
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    if args.self_test:
        return self_test()
    if args.dry_run:
        root = (args.root or default_root()).resolve()
        return dry_run_main(root, args.path)
    if args.hook:
        return hook_main(sys.stdin.read())
    parse_args(["--help"])
    return 2


PASS_CHECK = """#!/usr/bin/env python3
from pathlib import Path
Path(__file__).with_name("ran").write_text("yes\\n")
"""

FAIL_ALPHA = """#!/usr/bin/env python3
import sys
from pathlib import Path
Path(__file__).with_name("ran").write_text("yes\\n")
sys.stderr.write("MUTATION SURVIVED token-alpha\\n")
raise SystemExit(1)
"""

FAIL_BETA = """#!/usr/bin/env python3
import sys
from pathlib import Path
Path(__file__).with_name("ran").write_text("yes\\n")
sys.stderr.write("MUTATION SURVIVED token-beta\\n")
raise SystemExit(1)
"""


def self_test() -> int:
    hook = default_root() / ".cursor" / "hooks" / "verify-before-commit.sh"
    script = Path(__file__).resolve()
    checks = 0

    def expect(condition: bool, message: str) -> None:
        nonlocal checks
        checks += 1
        if not condition:
            raise SystemExit(f"FAIL: {message}")

    def git_cmd(root: Path, args: list[str]) -> None:
        env = os.environ.copy()
        env.update(
            {
                "GIT_AUTHOR_NAME": "gate-test",
                "GIT_AUTHOR_EMAIL": "gate-test@example.com",
                "GIT_COMMITTER_NAME": "gate-test",
                "GIT_COMMITTER_EMAIL": "gate-test@example.com",
            }
        )
        completed = subprocess.run(
            ["git", "-C", str(root), "-c", "commit.gpgsign=false", *args],
            capture_output=True,
            text=True,
            env=env,
            check=False,
        )
        if completed.returncode != 0:
            raise SystemExit(f"git {args} failed\n{completed.stderr}")

    def write_skill(root: Path, name: str, files: list[str], body: str) -> None:
        skill = root / ".cursor" / "skills" / name
        (skill / "references").mkdir(parents=True, exist_ok=True)
        (skill / "scripts").mkdir(parents=True, exist_ok=True)
        (skill / "references" / "coverage.json").write_text(
            json.dumps({"files": files}) + "\n", encoding="utf-8"
        )
        script_path = skill / "scripts" / "check_go_mutation.py"
        script_path.write_text(body, encoding="utf-8")

    def build(parent: Path, name: str, skills: dict[str, tuple[list[str], str]]) -> Path:
        root = parent / name
        root.mkdir()
        git_cmd(root, ["init", "-q", "-b", "main"])
        (root / "pkg").mkdir()
        (root / "README.md").write_text("readme\n", encoding="utf-8")
        (root / "pkg" / "a.go").write_text("package a\n", encoding="utf-8")
        (root / "pkg" / "b.go").write_text("package b\n", encoding="utf-8")
        for skill, (files, body) in skills.items():
            write_skill(root, skill, files, body)
        git_cmd(root, ["add", "."])
        git_cmd(root, ["commit", "-q", "-m", "base"])
        return root

    def marker(root: Path, name: str) -> Path:
        return root / ".cursor" / "skills" / name / "scripts" / "ran"

    def clear_markers(root: Path) -> None:
        for path in root.glob(".cursor/skills/*/scripts/ran"):
            path.unlink()

    def invoke(command: str, workspace: list[Path], cwd: Path) -> dict[str, str]:
        payload = {
            "command": command,
            "workspace_roots": [str(path) for path in workspace],
        }
        completed = subprocess.run(
            [str(hook)],
            input=json.dumps(payload),
            text=True,
            capture_output=True,
            cwd=cwd,
            check=False,
        )
        if completed.returncode != 0:
            raise SystemExit(
                f"hook exit {completed.returncode}\n{completed.stdout}\n{completed.stderr}"
            )
        try:
            decision = json.loads(completed.stdout)
        except json.JSONDecodeError as exc:
            raise SystemExit(f"hook JSON {exc}\n{completed.stdout}\n{completed.stderr}") from exc
        if not isinstance(decision, dict):
            raise SystemExit(f"hook JSON was not an object\n{completed.stdout}")
        return decision

    def dry(root: Path, paths: list[str]) -> tuple[int, dict[str, object]]:
        args = [sys.executable, str(script), "--dry-run", "--root", str(root)]
        for path in paths:
            args.extend(["--path", path])
        completed = subprocess.run(args, capture_output=True, text=True, check=False)
        try:
            report = json.loads(completed.stdout)
        except json.JSONDecodeError as exc:
            raise SystemExit(f"dry-run JSON {exc}\n{completed.stdout}\n{completed.stderr}") from exc
        return completed.returncode, report

    with tempfile.TemporaryDirectory(prefix="verify-commit-gate-") as temporary:
        parent = Path(temporary)
        neutral = parent / "cwd"
        neutral.mkdir()
        alpha = {"verify-alpha": (["pkg/a.go"], FAIL_ALPHA)}
        passing = {"verify-alpha": (["pkg/a.go"], PASS_CHECK)}

        repo = build(parent, "unrelated", alpha)
        git_cmd(repo, ["add", "README.md"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)
        expect("No formally verified paths" in decision.get("agent_message", ""), decision)
        expect(not marker(repo, "verify-alpha").exists(), "unrelated commit ran the check")

        repo = build(parent, "covered", alpha)
        git_cmd(repo, ["add", "-u"])
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        git_cmd(repo, ["add", "pkg/a.go"])
        clear_markers(repo)
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(
            "python3 .cursor/skills/verify-alpha/scripts/check_go_mutation.py"
            in decision.get("agent_message", ""),
            decision,
        )
        expect("MUTATION SURVIVED token-alpha" in decision.get("agent_message", ""), decision)
        expect(marker(repo, "verify-alpha").exists(), "covered commit did not run the check")

        repo = build(parent, "dry", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        code, report = dry(repo, ["pkg/a.go"])
        expect(code == 0, report)
        expect(not marker(repo, "verify-alpha").exists(), "dry-run executed the check")
        runs = report["runs"]
        expect(isinstance(runs, list) and runs[0]["name"] == "verify-alpha", report)
        expect(
            runs[0]["command"]
            == "python3 .cursor/skills/verify-alpha/scripts/check_go_mutation.py",
            report,
        )

        repo = build(parent, "tree", alpha)
        skill_md = repo / ".cursor" / "skills" / "verify-alpha" / "SKILL.md"
        skill_md.write_text("edited\n", encoding="utf-8")
        git_cmd(repo, ["add", ".cursor/skills/verify-alpha/SKILL.md"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "skill tree edit did not run the check")

        repo = build(parent, "unstaged", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)
        expect(not marker(repo, "verify-alpha").exists(), "unstaged path was treated as staged")

        repo = build(parent, "all", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        decision = invoke("git commit -am x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "-a did not include the worktree path")

        repo = build(parent, "pathspec", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        decision = invoke("git commit -m x -- pkg/a.go", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "pathspec commit skipped the covered file")

        repo = build(
            parent,
            "untracked-all",
            {"verify-alpha": (["pkg/a.go", "pkg/new.go"], FAIL_ALPHA)},
        )
        (repo / "pkg" / "new.go").write_text("package new\n", encoding="utf-8")
        decision = invoke("git commit -am x", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)
        expect(not marker(repo, "verify-alpha").exists(), "-a included an untracked file")
        decision = invoke("git commit -m x -- pkg/new.go", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "pathspec skipped a new covered file")

        repo = build(parent, "not-commit", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        git_cmd(repo, ["add", "pkg/a.go"])
        decision = invoke("git commit-tree HEAD", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)
        expect(decision.get("agent_message", "") == "", decision)
        expect(not marker(repo, "verify-alpha").exists(), "commit-tree ran the check")
        decision = invoke("echo git commit", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)

        repo = build(parent, "shrink", passing)
        coverage = repo / ".cursor" / "skills" / "verify-alpha" / "references" / "coverage.json"
        coverage.write_text(json.dumps({"files": []}) + "\n", encoding="utf-8")
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        git_cmd(repo, ["add", "pkg/a.go", ".cursor/skills/verify-alpha/references/coverage.json"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect("stops covering" in decision.get("agent_message", ""), decision)
        expect(
            "python3 .cursor/skills/verify-alpha/scripts/check_go_mutation.py"
            in decision.get("agent_message", ""),
            decision,
        )
        expect(not marker(repo, "verify-alpha").exists(), "shrink deny ran the check")

        repo = build(parent, "skill-only", passing)
        coverage = repo / ".cursor" / "skills" / "verify-alpha" / "references" / "coverage.json"
        coverage.write_text(json.dumps({"files": []}) + "\n", encoding="utf-8")
        git_cmd(repo, ["add", ".cursor/skills/verify-alpha/references/coverage.json"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "allow", decision)
        expect(
            not marker(repo, "verify-alpha").exists(),
            "coverage json edit ran the check",
        )

        repo = build(parent, "broken", alpha)
        coverage = repo / ".cursor" / "skills" / "verify-alpha" / "references" / "coverage.json"
        coverage.write_text("{", encoding="utf-8")
        git_cmd(repo, ["add", "README.md"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect("not valid JSON" in decision.get("agent_message", ""), decision)
        expect(not marker(repo, "verify-alpha").exists(), "broken catalog ran the check")

        repo = build(
            parent,
            "two",
            {
                "verify-alpha": (["pkg/a.go"], FAIL_ALPHA),
                "verify-beta": (["pkg/b.go"], FAIL_BETA),
            },
        )
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        (repo / "pkg" / "b.go").write_text("package b\nchanged\n", encoding="utf-8")
        git_cmd(repo, ["add", "pkg/a.go", "pkg/b.go"])
        decision = invoke("git commit -m x", [repo], neutral)
        message = decision.get("agent_message", "")
        expect(decision.get("permission") == "deny", decision)
        expect("python3 .cursor/skills/verify-alpha/scripts/check_go_mutation.py" in message, decision)
        expect("python3 .cursor/skills/verify-beta/scripts/check_go_mutation.py" in message, decision)
        expect("MUTATION SURVIVED token-alpha" in message, decision)
        expect("token-beta" not in message, decision)
        expect(marker(repo, "verify-alpha").exists(), "first check did not run")
        expect(not marker(repo, "verify-beta").exists(), "later check ran after a failure")

        repo = build(parent, "rename", alpha)
        git_cmd(repo, ["mv", "pkg/a.go", "pkg/moved.go"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "rename dropped the old covered path")

        repo = build(parent, "delete", alpha)
        git_cmd(repo, ["rm", "-q", "pkg/a.go"])
        decision = invoke("git commit -m x", [repo], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "deletion skipped the covered path")

        repo = build(parent, "discover", passing)
        write_skill(repo, "verify-http-read-status", ["pkg/http.go"], PASS_CHECK)
        code, report = dry(repo, ["pkg/http.go"])
        expect(code == 0, report)
        names = [item["name"] for item in report["runs"]]
        expect(names == ["verify-http-read-status"], report)
        expect(not marker(repo, "verify-http-read-status").exists(), "discovery dry-run executed")

        other = build(parent, "other", {})
        repo = build(parent, "dash-c", alpha)
        (repo / "pkg" / "a.go").write_text("package a\nchanged\n", encoding="utf-8")
        git_cmd(repo, ["add", "pkg/a.go"])
        decision = invoke(f"git -C {repo} commit -m x", [other], neutral)
        expect(decision.get("permission") == "deny", decision)
        expect(marker(repo, "verify-alpha").exists(), "git -C did not select the commit repository")
        expect(not marker(other, "verify-alpha").exists(), "git -C scanned the workspace instead")

        raw = subprocess.run(
            [str(hook)],
            input="{not json",
            text=True,
            capture_output=True,
            cwd=neutral,
            check=False,
        )
        expect(raw.returncode == 0, raw.stderr)
        invalid = json.loads(raw.stdout)
        expect(invalid.get("permission") == "deny", invalid)

    code, report = dry(default_root(), ["this-path-is-not-covered.txt"])
    expect(code == 0 and report["runs"] == [] and report["blocked"] == [], report)
    code, catalog = dry(default_root(), [])
    expect(code == 0, catalog.get("errors"))
    discovered = {item["name"] for item in catalog["skills"]}
    for coverage in sorted(default_root().glob(".cursor/skills/verify-*/references/coverage.json")):
        name = coverage.parent.parent.name
        data = json.loads(coverage.read_text(encoding="utf-8"))
        expect(name in discovered, f"missing {name}")
        command = f"python3 .cursor/skills/{name}/scripts/check_go_mutation.py"
        skill = next(item for item in catalog["skills"] if item["name"] == name)
        expect(skill["command"] == command, skill)
        expect((default_root() / skill["command"].split(" ", 1)[1]).is_file(), skill["command"])
        covered = data.get("files") or []
        if not covered:
            continue
        code, report = dry(default_root(), [covered[0]])
        expect(code == 0, report.get("errors"))
        expect(name in {item["name"] for item in report["runs"]}, (name, report["runs"]))

    hooks = json.loads((default_root() / ".cursor" / "hooks.json").read_text(encoding="utf-8"))
    entries = hooks["hooks"]["beforeShellExecution"]
    commands = [entry["command"] for entry in entries]
    expect(".cursor/hooks/go-crap-before-commit.sh" in commands, commands)
    expect(".cursor/hooks/gofmt-before-commit.sh" in commands, commands)
    verify = [
        entry
        for entry in entries
        if entry["command"] == ".cursor/hooks/verify-before-commit.sh"
    ]
    expect(len(verify) == 1, verify)
    expect(verify[0]["failClosed"] is True, verify)
    expect(verify[0]["matcher"] == "git commit", verify)
    expect(hooks["hooks"]["afterFileEdit"][0]["command"] == ".cursor/hooks/gofmt-after-edit.sh", hooks)

    print(f"PASS {checks} checks")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
