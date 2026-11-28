---
name: verify-http-read-status
description: Verify reactorlog http-read-status state transitions when modifying NewMux, health, listSites, getSite, listReactors, SiteByID. Run the bounded model and review its correspondence to the Go implementation.
---

# verify http-read-status

The property is that response status follows the route and the store outcome. A miss is not 200. A store failure is not 200. A hit is 200. A violation means a client can treat a missing or failed read as a successful payload.

## run

From the api repository root, run:

```sh
python3 .cursor/skills/verify-http-read-status/scripts/check_go_mutation.py
```

Verified with CPython 3.9.6 and Go 1.27.1. Models use the Python standard library. Go checks use the dependencies pinned in go.mod and go.sum. The command runs the model, then baseline, mutated, and restored Go checks in a temporary copy. Go dependencies must be available locally or downloadable.
A nonzero exit indicates a violated property, a surviving mutation, or source that needs correspondence review. Read the printed trace in order.

## coverage and model

Read `references/coverage.json` and `scripts/verify.py` before interpreting a result. Covered functions are NewMux, health, listSites, getSite, listReactors, SiteByID. Covered source paths and exact SHA-256 fingerprints are in the coverage record.

Bounds are the four GET routes crossed with store outcomes. Lists and health use ok and error. `GET /sites/{id}` uses hit, miss, and error. The checker also includes one unmatched path and one wrong method. Empty ok is still 200. The checker exhaustively enumerates that finite abstraction. External operations are represented by outcomes. Model transitions are explained at the corresponding statements in the script. JSON body shape, CORS, listen and bind, Postgres, and concurrency are outside this model. SQL scan correctness stays with the database integration test.

Baseline commit is `472f85d4755f53a6eea822e9d56a6011e9fe42e7`. The inspected checkout contains uncommitted changes, so the fingerprints identify the actual baseline. The finite abstraction passed and the generated command caught a real Go source mutation, then passed again after restoration on 2026-11-17. The mutation and test selection are recorded in coverage.json. `TestHandlerStatusMapping` is the Go oracle for the status table. Use `scripts/verify.py --mutation-test` for the faster model-only mutation check. The source-to-model correspondence remains manually reviewed.

## maintain

When source fingerprints differ, inspect the actual diff, remap every affected transition, guard, and atomic boundary, then update the model and fingerprint. Never refresh fingerprints merely to obtain a pass. Advance the sync commit only after confirming correspondence at the new commit. Preserve failures in intended properties and report their shortest available trace. Distinguish implementation counterexamples from incorrect abstraction assumptions. Use `/maintain-formal-verification-skill` after changes that skipped this review.

The checker does not verify arbitrary Go execution, SQL semantics, liveness, crashes, networking, or unbounded input sizes. The source fingerprint gate detects drift but does not prove correspondence. Review these assumptions before relying on a pass.
