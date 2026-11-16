#!/usr/bin/env python3
"""Finite abstraction checker. Source fingerprints require manual correspondence review."""
from collections import deque
from itertools import product
from pathlib import Path
import argparse
import hashlib
import json
import tempfile

SKILL = Path(__file__).resolve().parents[1]
CONFIG = json.loads((SKILL / 'references' / 'coverage.json').read_text())

def explore(broken=False):

    for dotenv_ok, command_ok in product((False, True), repeat=2):
        if not dotenv_ok:
            yield ['dotenv failed', 'exit 1'], True
            continue
        code = 0 if command_ok or broken else 1
        yield ['dotenv loaded', f'command success={command_ok}', f'exit {code}'], command_ok or code != 0

def check(broken=False):
    count = 0
    for trace, holds in explore(broken):
        count += 1
        if not holds:
            return count, trace
    return count, None

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mutation-test', action='store_true')
    parser.add_argument('--source-root', type=Path, default=SKILL.parents[2])
    args = parser.parse_args()
    for filename, expected in CONFIG['sha256'].items():
        actual = hashlib.sha256((args.source_root / filename).read_bytes()).hexdigest()
        if actual != expected:
            raise SystemExit(f'UNREVIEWED SOURCE: {filename}; review correspondence before updating coverage.json')
    count, failure = check()
    if failure:
        print('FAIL:', ' -> '.join(failure))
        return 1
    print(f'PASS: {count} bounded cases/states; ' + CONFIG['property'])
    if args.mutation_test:
        # Preserve actual source bytes with one real-code mutation in scratch.
        # This checks the paired model mutation; it does not execute mutated Go.
        source = (args.source_root / CONFIG['files'][0]).read_text()
        if CONFIG['old'] not in source:
            raise SystemExit('mutation anchor missing')
        with tempfile.TemporaryDirectory(prefix='reactorlog-verification-') as temporary:
            changed = source.replace(CONFIG['old'], CONFIG['new'], 1)
            path = Path(temporary) / Path(CONFIG['files'][0]).name
            path.write_text(changed)
            assert path.read_text() != source
            _, counterexample = check(broken=True)
            if counterexample is None:
                raise SystemExit('FAIL: model mutation survived')
            print('MODEL MUTATION CAUGHT:', ' -> '.join(counterexample))
        assert check()[1] is None
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
