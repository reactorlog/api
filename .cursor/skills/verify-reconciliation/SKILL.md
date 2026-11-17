---
name: verify-reconciliation
description: Verify reactorlog reconciliation state transitions when modifying store, import.sql, or the sites_name_location index. Run the bounded model and review its correspondence to the Go implementation.
---

# verify reconciliation

The property is that resolving an existing docket returns its stored site identity. A violation means metadata can be written to a different site from the reactor’s stored site.

## run

From the api repository root, run:

```sh
python3 .cursor/skills/verify-reconciliation/scripts/check_go_mutation.py
```

Verified with CPython 3.9.6 and Go 1.27.1. Models use the Python standard library. Go checks use the dependencies pinned in go.mod and go.sum. The command runs the model, then baseline, mutated, and restored Go checks in a temporary copy. Go dependencies must be available locally or downloadable.
A nonzero exit indicates a violated property, a surviving mutation, or source that needs correspondence review. Read the printed trace in order.

## coverage and model

Read `references/coverage.json` and `scripts/verify.py` before interpreting a result. Covered functions are store and import.sql; sites_name_location index. Covered source paths and exact SHA-256 fingerprints are in the coverage record.

Bounds are two sites, one known docket, matching and nonmatching source identities. The checker exhaustively enumerates that finite abstraction. External operations are represented by outcomes. Model transitions are explained at the corresponding statements in the script. Database atomicity and transaction-scoped advisory locking are assumptions supplied by PostgreSQL. A commit error may represent either durable outcome.

Baseline commit is `231c1cb5c2b7338cdbc2e854d72daea2e647c493`. The inspected checkout contains uncommitted changes, so the fingerprints identify the actual baseline. The finite abstraction passed and the generated command caught a real Go source mutation, then passed again after restoration on 2026-11-16. The mutation and test selection are recorded in coverage.json. SQL mock checks establish the expected calls and ordering, while PostgreSQL semantics remain assumptions. Use `scripts/verify.py --mutation-test` for the faster model-only mutation check. The source-to-model correspondence remains manually reviewed.

## maintain

When source fingerprints differ, inspect the actual diff, remap every affected transition, guard, and atomic boundary, then update the model and fingerprint. Never refresh fingerprints merely to obtain a pass. Advance the sync commit only after confirming correspondence at the new commit. Preserve failures in intended properties and report their shortest available trace. Distinguish implementation counterexamples from incorrect abstraction assumptions. Use `/maintain-formal-verification-skill` after changes that skipped this review.

The checker does not verify arbitrary Go execution, SQL semantics, liveness, crashes, networking, or unbounded input sizes. The source fingerprint gate detects drift but does not prove correspondence. Review these assumptions before relying on a pass.

Known audit concern outside this property: renaming a site while adding a new docket can make success depend on row order. The bounded identity check does not establish order independence.
