#!/usr/bin/env python3
"""Fail when SQL keywords are written in ALL CAPS.

Repo SQL uses lowercase keywords (select, create, where). Agents that copy
generic UPPERCASE SQL break that house style. This check fails closed on
.SQL files and on SQL-looking string literals in .go files.

Run:

    python3 .cursor/scripts/sql-keyword-case.py
    python3 .cursor/scripts/sql-keyword-case.py --self-test
"""

from __future__ import annotations

import argparse
import re
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

# Keywords that must not appear as ALL-CAPS tokens in SQL.
KEYWORDS = frozenset(
    {
        "select",
        "insert",
        "update",
        "delete",
        "create",
        "alter",
        "drop",
        "table",
        "index",
        "view",
        "constraint",
        "primary",
        "key",
        "foreign",
        "references",
        "unique",
        "check",
        "default",
        "values",
        "set",
        "into",
        "returning",
        "with",
        "from",
        "where",
        "join",
        "left",
        "right",
        "inner",
        "outer",
        "cross",
        "full",
        "on",
        "as",
        "and",
        "or",
        "not",
        "null",
        "is",
        "in",
        "between",
        "exists",
        "case",
        "when",
        "then",
        "else",
        "end",
        "order",
        "by",
        "group",
        "having",
        "limit",
        "offset",
        "union",
        "distinct",
        "cascade",
        "schema",
        "if",
        "include",
        "unnest",
        "current_timestamp",
        "count",
        "coalesce",
        "nulls",
        "last",
        "first",
        "do",
        "conflict",
        "excluded",
        "true",
        "false",
    }
)

TOKEN = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
SQL_HINT = re.compile(
    r"\b(select|insert|update|delete|create|alter|drop|with|returning)\b",
    re.IGNORECASE,
)
SKIP_DIRS = frozenset(
    {
        ".git",
        ".worktrees",
        "node_modules",
        "vendor",
        ".benchmarks",
    }
)


def strip_sql_comments(text: str) -> str:
    out: list[str] = []
    i = 0
    n = len(text)
    while i < n:
        if text.startswith("--", i):
            while i < n and text[i] != "\n":
                i += 1
            continue
        if text.startswith("/*", i):
            end = text.find("*/", i + 2)
            if end < 0:
                break
            i = end + 2
            continue
        if text[i] == "'":
            out.append(text[i])
            i += 1
            while i < n:
                if text[i] == "'" and i + 1 < n and text[i + 1] == "'":
                    out.append("''")
                    i += 2
                    continue
                out.append(text[i])
                if text[i] == "'":
                    i += 1
                    break
                i += 1
            continue
        out.append(text[i])
        i += 1
    return "".join(out)


def find_upper_keywords(text: str) -> list[tuple[int, str]]:
    hits: list[tuple[int, str]] = []
    for match in TOKEN.finditer(text):
        token = match.group(0)
        if token.isupper() and token.lower() in KEYWORDS:
            line = text.count("\n", 0, match.start()) + 1
            hits.append((line, token))
    return hits


def iter_sources(root: Path) -> list[Path]:
    files: list[Path] = []
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if any(part in SKIP_DIRS for part in path.parts):
            continue
        if path.suffix == ".sql" or path.suffix == ".go":
            files.append(path)
    return sorted(files)


def go_string_literals(source: str) -> list[tuple[int, str]]:
    """Return (start_line, literal_text) for "..." and `...` strings."""
    literals: list[tuple[int, str]] = []
    i = 0
    n = len(source)
    line = 1
    while i < n:
        char = source[i]
        if char == "\n":
            line += 1
            i += 1
            continue
        if source.startswith("//", i):
            while i < n and source[i] != "\n":
                i += 1
            continue
        if source.startswith("/*", i):
            i += 2
            while i + 1 < n and not source.startswith("*/", i):
                if source[i] == "\n":
                    line += 1
                i += 1
            i = min(i + 2, n)
            continue
        if char == '"':
            start_line = line
            i += 1
            buf: list[str] = []
            while i < n:
                if source[i] == "\\":
                    if i + 1 < n:
                        buf.append(source[i : i + 2])
                        if source[i + 1] == "\n":
                            line += 1
                        i += 2
                        continue
                if source[i] == '"':
                    i += 1
                    break
                if source[i] == "\n":
                    line += 1
                buf.append(source[i])
                i += 1
            literals.append((start_line, "".join(buf)))
            continue
        if char == "`":
            start_line = line
            i += 1
            buf = []
            while i < n and source[i] != "`":
                if source[i] == "\n":
                    line += 1
                buf.append(source[i])
                i += 1
            if i < n:
                i += 1
            literals.append((start_line, "".join(buf)))
            continue
        i += 1
    return literals


def check_sql_file(path: Path, text: str) -> list[str]:
    hits = find_upper_keywords(strip_sql_comments(text))
    return [f"{path}:{line}: uppercase SQL keyword {token!r}; use {token.lower()!r}" for line, token in hits]


def check_go_file(path: Path, text: str) -> list[str]:
    errors: list[str] = []
    for start_line, literal in go_string_literals(text):
        if not SQL_HINT.search(literal):
            continue
        for line_offset, token in find_upper_keywords(strip_sql_comments(literal)):
            line = start_line + line_offset - 1
            errors.append(
                f"{path}:{line}: uppercase SQL keyword {token!r}; use {token.lower()!r}"
            )
    return errors


def check_tree(root: Path) -> list[str]:
    errors: list[str] = []
    for path in iter_sources(root):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(root)
        if path.suffix == ".sql":
            errors.extend(check_sql_file(rel, text))
        else:
            errors.extend(check_go_file(rel, text))
    return errors


def self_test() -> int:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        bad_sql = root / "bad.sql"
        bad_sql.write_text("SELECT id FROM sites;\n", encoding="utf-8")
        good_sql = root / "good.sql"
        good_sql.write_text("select id from sites;\n", encoding="utf-8")
        bad_go = root / "bad.go"
        bad_go.write_text(
            'package p\nconst q = "SELECT count(*) FROM sites"\n',
            encoding="utf-8",
        )
        good_go = root / "good.go"
        good_go.write_text(
            'package p\nconst q = "select count(*) from sites"\n',
            encoding="utf-8",
        )

        errors = check_tree(root)
        joined = "\n".join(errors)
        if "bad.sql:1: uppercase SQL keyword 'SELECT'" not in joined:
            print("self-test: expected SELECT failure in bad.sql", file=sys.stderr)
            print(joined, file=sys.stderr)
            return 1
        if "bad.go:2: uppercase SQL keyword 'SELECT'" not in joined:
            print("self-test: expected SELECT failure in bad.go", file=sys.stderr)
            print(joined, file=sys.stderr)
            return 1
        if "good.sql" in joined or "good.go" in joined:
            print("self-test: good files must pass", file=sys.stderr)
            print(joined, file=sys.stderr)
            return 1

        # Replay a real past mistake shape from migrations_test.go.
        past = root / "past.go"
        past.write_text(
            "package migrations\n"
            "const reset = `\n"
            "\t\tDROP TABLE IF EXISTS reactors CASCADE;\n"
            "`\n",
            encoding="utf-8",
        )
        past_errors = check_go_file(Path("past.go"), past.read_text(encoding="utf-8"))
        if not any("DROP" in err for err in past_errors):
            print("self-test: expected DROP failure for past migrations_test shape", file=sys.stderr)
            print("\n".join(past_errors), file=sys.stderr)
            return 1

    print("sql-keyword-case self-test passed")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--self-test",
        action="store_true",
        help="prove the checker fails on uppercase SQL and passes on lowercase",
    )
    args = parser.parse_args()
    if args.self_test:
        return self_test()

    errors = check_tree(ROOT)
    if errors:
        for err in errors:
            print(err, file=sys.stderr)
        print(
            "SQL keywords must be lowercase (select, not SELECT). "
            "Fix the lines above.",
            file=sys.stderr,
        )
        return 1
    print("sql-keyword-case: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
