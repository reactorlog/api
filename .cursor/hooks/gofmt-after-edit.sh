#!/bin/bash
# Format a single edited Go file with gofmt after an agent write.
# afterFileEdit has no useful stdout schema; formatting is a side effect.

set -u

if ! command -v gofmt >/dev/null 2>&1; then
  export PATH="/opt/homebrew/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
fi

if ! command -v gofmt >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then
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

def main():
    raw = open(sys.argv[1], encoding="utf-8").read()
    try:
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        return

    path = payload.get("file_path")
    if not isinstance(path, str) or not path:
        return
    if not path.endswith(".go"):
        return
    if not os.path.isfile(path):
        return

    gofmt = shutil.which("gofmt")
    if not gofmt:
        return

    # Only rewrite when gofmt would change the file (avoids rewrite loops).
    listed = subprocess.run(
        [gofmt, "-l", path],
        capture_output=True,
        text=True,
        check=False,
    )
    if listed.returncode != 0:
        return
    if not listed.stdout.strip():
        return

    subprocess.run([gofmt, "-w", path], check=False)


if __name__ == "__main__":
    main()
PY

exit 0
