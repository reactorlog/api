---
name: verify-database-pool
description: Verify reactorlog database-pool state transitions when modifying openPool, verifyPool, Connect, runServer. Run the bounded model and review its correspondence to the Go implementation.
---

# verify database-pool

The property is that a pool allocated before a failed ping is closed. A violation means failed connection attempts can retain resources.

## run

From the api repository root, run:

```sh
python3 .cursor/skills/verify-database-pool/scripts/check_go_mutation.py
```

Verified with CPython 3.9.6 and Go 1.27.1. Models use the Python standard library. Go checks use the dependencies pinned in go.mod and go.sum. The command runs the model, then baseline, mutated, and restored Go checks in a temporary copy. Go dependencies must be available locally or downloadable.
A nonzero exit indicates a violated property, a surviving mutation, or source that needs correspondence review. Read the printed trace in order.

## coverage and model

Read `references/coverage.json` and `scripts/verify.py` before interpreting a result. Covered functions are openPool, verifyPool, Connect, runServer. Covered source paths and exact SHA-256 fingerprints are in the coverage record.

Bounds are allocation success/failure crossed with ping success/failure. The checker exhaustively enumerates that finite abstraction. External operations are represented by outcomes. Model transitions are explained at the corresponding statements in the script. Database atomicity and transaction-scoped advisory locking are assumptions supplied by PostgreSQL. A commit error may represent either durable outcome.

Baseline commit is `231c1cb5c2b7338cdbc2e854d72daea2e647c493`. The inspected checkout contains uncommitted changes, so the fingerprints identify the actual baseline. The finite abstraction passed and the generated command caught a real Go source mutation, then passed again after restoration on 2026-11-16. The mutation and test selection are recorded in coverage.json. SQL mock checks establish the expected calls and ordering, while PostgreSQL semantics remain assumptions. Use `scripts/verify.py --mutation-test` for the faster model-only mutation check. The source-to-model correspondence remains manually reviewed.

## maintain

When source fingerprints differ, inspect the actual diff, remap every affected transition, guard, and atomic boundary, then update the model and fingerprint. Never refresh fingerprints merely to obtain a pass. Advance the sync commit only after confirming correspondence at the new commit. Preserve failures in intended properties and report their shortest available trace. Distinguish implementation counterexamples from incorrect abstraction assumptions. Use `/maintain-formal-verification-skill` after changes that skipped this review.

The checker does not verify arbitrary Go execution, SQL semantics, liveness, crashes, networking, or unbounded input sizes. The source fingerprint gate detects drift but does not prove correspondence. Review these assumptions before relying on a pass.
