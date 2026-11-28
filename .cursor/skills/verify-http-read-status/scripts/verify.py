#!/usr/bin/env python3
"""Finite abstraction checker. Source fingerprints require manual correspondence review."""
from pathlib import Path
import argparse
import hashlib
import json
import tempfile

SKILL = Path(__file__).resolve().parents[1]
CONFIG = json.loads((SKILL / 'references' / 'coverage.json').read_text())

# Each case is (route, store outcome, status).
# GET /health: Ping nil is ok and 200. Ping error is 503.
# GET /sites and GET /reactors: a nil error is ok and 200, including an empty slice. Any other error is 500.
# GET /sites/{id}: a nil error is a hit and 200. ErrNotFound is a miss and 404. Any other error is 500.
# SiteByID returns ErrNotFound for an invalid id and for pgx.ErrNoRows. Those are misses, not a separate route.
# ServeMux writes 404 for an unknown path and 405 for the wrong method. The handler is not called.
CASES = (
    ('GET /health', 'ok', 200),
    ('GET /health', 'error', 503),
    ('GET /sites', 'ok', 200),
    ('GET /sites', 'error', 500),
    ('GET /sites/{id}', 'hit', 200),
    ('GET /sites/{id}', 'miss', 404),
    ('GET /sites/{id}', 'error', 500),
    ('GET /reactors', 'ok', 200),
    ('GET /reactors', 'error', 500),
    ('unmatched', 'none', 404),
    ('wrong method', 'none', 405),
)


def status_for(outcome, want, broken):
    # The checked Go mutation writes 200 from getSite's not-found branch.
    if broken and outcome == 'miss':
        return 200
    return want


def holds(outcome, code):
    if outcome in ('ok', 'hit'):
        return code == 200
    return code != 200


def explore(broken=False):
    for route, outcome, want in CASES:
        code = status_for(outcome, want, broken)
        mapped = code == want or (broken and outcome == 'miss')
        yield [route, f'outcome={outcome}', f'status={code}'], holds(outcome, code) and mapped


def check(broken=False):
    count = 0
    for trace, ok in explore(broken):
        count += 1
        if not ok:
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
        # This checks the paired model mutation. It does not execute mutated Go.
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
