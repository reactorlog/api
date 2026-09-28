#!/usr/bin/env python3
"""Report functions whose go-crap score is not strictly under 6.

Stdout is one JSON object.
Exit 0 when every function scores under 6.
Exit 1 when at least one function scores 6 or higher, or has no score.
Exit 2 when the scan times out.
Exit 3 when go-crap or go is missing, or the scan fails.
"""

import json
import os
import shutil
import subprocess
import sys

LIMIT = 6.0
MAX_OFFENDERS = 30
MAX_ERROR_CHARS = 4000
SCAN_TIMEOUT_SECONDS = 540


def augment_path():
    extra = [
        os.path.expanduser("~/.local/bin"),
        "/opt/homebrew/bin",
        "/usr/local/bin",
    ]
    current = os.environ.get("PATH", "/usr/bin:/bin")
    os.environ["PATH"] = ":".join(extra + [current])


def clip(text):
    text = (text or "").strip()
    if len(text) <= MAX_ERROR_CHARS:
        return text
    return text[:MAX_ERROR_CHARS] + "\n…"


def emit(payload, code):
    json.dump(payload, sys.stdout)
    sys.stdout.write("\n")
    raise SystemExit(code)


def fail(kind, detail, code):
    emit({"ok": False, "error": kind, "detail": clip(detail)}, code)


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


def offender_record(entry, score, missing):
    record = {
        "file": entry.get("file") or "",
        "line": entry.get("line") if isinstance(entry.get("line"), int) else 0,
        "function": entry.get("function") or "",
        "receiver": entry.get("receiver") or "",
        "score": None if missing else score,
        "missing_score": missing,
    }
    complexity = entry.get("cyclomatic")
    if isinstance(complexity, int):
        record["cyclomatic"] = complexity
    coverage = entry.get("coverage")
    if isinstance(coverage, (int, float)):
        record["coverage"] = float(coverage)
    package = entry.get("package")
    if isinstance(package, str) and package:
        record["package"] = package
    return record


def summary(found):
    shown = found[:MAX_OFFENDERS]
    lines = [format_offender(entry, score) for entry, score, _missing in shown]
    extra = len(found) - len(shown)
    if extra > 0:
        lines.append(f"- … and {extra} more")
    noun = "function" if len(found) == 1 else "functions"
    return (
        "Every function must have a go-crap CRAP score under 6 "
        f"(effective score < {LIMIT:.0f}). "
        f"{len(found)} {noun} scored 6 or higher:\n" + "\n".join(lines)
    )


# Numbered files under migrations/ are migration steps. They are not unit tested.
EXCLUDED_FILE = r"migrations/(?:.+/)?[0-9][^/]*\.go"


def scan_root(binary, root):
    completed = subprocess.run(
        [
            binary,
            "scan",
            "--format",
            "json",
            "--no-progress",
            "--timeout",
            "9m",
            "--exclude",
            EXCLUDED_FILE,
        ],
        cwd=root,
        capture_output=True,
        text=True,
        timeout=SCAN_TIMEOUT_SECONDS,
        check=False,
    )
    stdout = completed.stdout or ""
    stderr = completed.stderr or ""
    if completed.returncode != 0:
        detail = clip(stderr or stdout or f"go-crap exited {completed.returncode}")
        raise RuntimeError(detail)
    return parse_report(stdout)


def roots_from_args(argv):
    roots = []
    index = 0
    while index < len(argv):
        arg = argv[index]
        if arg == "--root":
            index += 1
            if index >= len(argv):
                fail("scan_failed", "--root requires a directory", 3)
            roots.append(argv[index])
        elif arg.startswith("--root="):
            roots.append(arg.split("=", 1)[1])
        else:
            fail("scan_failed", f"unknown argument {arg}", 3)
        index += 1
    if roots:
        return roots
    return [os.getcwd()]


def main():
    augment_path()
    binary = shutil.which("go-crap")
    go_binary = shutil.which("go")
    if not binary or not go_binary:
        fail(
            "missing_binary",
            "go-crap and go must both be on PATH. Install go-crap, then run the scan again.",
            3,
        )

    found = []
    try:
        for root in roots_from_args(sys.argv[1:]):
            if not os.path.isdir(root):
                fail("scan_failed", f"scan root is not a directory: {root}", 3)
            report = scan_root(binary, root)
            for entry in report["entries"]:
                score = entry_score(entry)
                if score is None or score >= LIMIT:
                    display = LIMIT if score is None else score
                    found.append((entry, display, score is None))
    except subprocess.TimeoutExpired:
        fail("timeout", "go-crap scan exceeded 9 minutes.", 2)
    except (OSError, RuntimeError, ValueError) as exc:
        fail("scan_failed", str(exc), 3)

    if found:
        found.sort(key=lambda item: item[1], reverse=True)
        emit(
            {
                "ok": False,
                "limit": LIMIT,
                "summary": summary(found),
                "offenders": [
                    offender_record(entry, score, missing)
                    for entry, score, missing in found
                ],
            },
            1,
        )

    emit({"ok": True, "limit": LIMIT, "offenders": []}, 0)


if __name__ == "__main__":
    main()
