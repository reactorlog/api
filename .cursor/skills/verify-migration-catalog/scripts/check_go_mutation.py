#!/usr/bin/env python3
"""Run the model, then the Go check on the original, a broken copy, and the restored copy."""
import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path

skill = Path(__file__).resolve().parents[1]
root = skill.parents[2]
config = json.loads((skill / "references" / "coverage.json").read_text())
subprocess.run(["python3", str(skill / "scripts" / "verify.py")], check=True)
with tempfile.TemporaryDirectory(prefix="reactorlog-go-mutation-") as directory:
    copy = Path(directory)
    for source in root.rglob("*"):
        relative = source.relative_to(root)
        if any(part.startswith(".") for part in relative.parts):
            continue
        if source.is_file() and (source.suffix in (".go", ".html", ".sql") or source.name in ("go.mod", "go.sum")):
            target = copy / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, target)
    extra = skill / "references" / "mutation_test.go"
    if extra.exists():
        shutil.copyfile(extra, copy / config["test_package"] / "verification_mutation_test.go")
    environment = dict(os.environ, GOCACHE="/tmp/reactorlog-go-cache", TEST_DATABASE_URL="")
    command = ["go", "test", "-count=1", config["test_package"], "-run", config["test_name"]]

    def run():
        result = subprocess.run(command, cwd=copy, env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        print(result.stdout, end="")
        return result

    if run().returncode:
        raise SystemExit("baseline Go check failed")
    target = copy / config["files"][0]
    original = target.read_text()
    if config["old"] not in original:
        raise SystemExit("mutation anchor missing")
    target.write_text(original.replace(config["old"], config["new"], 1))
    mutated = run()
    if mutated.returncode == 0:
        raise SystemExit("mutation survived")
    if "--- FAIL:" not in mutated.stdout:
        raise SystemExit("mutation failed without a test assertion; investigate")
    target.write_text(original)
    if run().returncode:
        raise SystemExit("restored Go check failed")
    print("PASS: actual Go mutation caught and restored baseline passed")
