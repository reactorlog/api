#!/usr/bin/env python3
"""Run TLC 1.7.4 on this skill's spec. The broken config must violate the invariant."""
import hashlib
import os
import re
import subprocess
import sys
from pathlib import Path

SKILL = Path(__file__).resolve().parents[1]
ROOT = SKILL.parents[2]
SPEC = "NrcSiteLink.tla"
INVARIANT = "NewDocketJoins"
CONFIG = __import__("json").loads((SKILL / "references" / "coverage.json").read_text())


def tlc(cfg):
    jar = os.environ.get("TLA2TOOLS")
    if not jar:
        raise SystemExit("Set TLA2TOOLS to tla2tools.jar from TLC 1.7.4, and JAVA to Java 11 or later.")
    java = os.environ.get("JAVA", "java")
    return subprocess.run(
        [java, "-jar", jar, "-workers", "1", "-config", str(SKILL / cfg), str(SKILL / SPEC)],
        cwd=SKILL,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )


def main():
    for filename, expected in CONFIG["sha256"].items():
        actual = hashlib.sha256((ROOT / filename).read_bytes()).hexdigest()
        if actual != expected:
            raise SystemExit(f"UNREVIEWED SOURCE: {filename}; review correspondence before updating coverage.json")
    passed = tlc(SPEC.replace(".tla", ".cfg"))
    if passed.returncode != 0:
        print(passed.stdout)
        return 1
    count = re.search(r"(\d+) distinct states found", passed.stdout)
    broken = tlc(SPEC.replace(".tla", "Broken.cfg"))
    text = broken.stdout
    if broken.returncode == 0 or f"Invariant {INVARIANT} is violated" not in text:
        print(text)
        raise SystemExit(f"broken model did not violate {INVARIANT}")
    states = count.group(1) if count else "unknown"
    print(f"PASS: {states} distinct states; {CONFIG['property']}")
    print(f"BROKEN MODEL CAUGHT: {INVARIANT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
