---
name: verify-migration-catalog
description: Verify the migration catalog walk when modifying Run, applyPending, or runMigration. Run the TLA+ model and the Go mutation check.
---

# verify migration catalog

The property is that Run applies each missing catalog version by calling Up, never Down, and Result.Applied is true only for versions this call executed. A violation means Down runs, or a version this call skipped is reported as applied.

## run

From the api repository root, with `TLA2TOOLS` set to `tla2tools.jar` from TLC 1.7.4 and `JAVA` set to Java 11 or later (`JAVA` defaults to `java` on `PATH`):

```sh
python3 .cursor/skills/verify-migration-catalog/scripts/check_go_mutation.py
```

A nonzero exit means the invariant failed, the broken model passed, a Go mutation survived, or a covered file no longer matches its fingerprint. Read a TLC result with `/statechart`. The spec is `MigrationCatalog.tla`, and `view.mjs` beside it is the chart's wording. Do not treat a fingerprint mismatch as a passing model.

## coverage

Read `references/coverage.json` and `MigrationCatalog.tla` before interpreting a result. Covered code is `Run`, `applyPending`, and `runMigration` in `migrations/runner.go`.

Bounds are catalog versions 1 and 2 and every subset already present in the migrations table. One action is one version in `applyPending`: `Skip` when the version is present, `Execute` when it is missing. `Execute` is `runMigration` calling `Up` and then `recordMigration`. `executed` is the set of `Result.Applied == true`. `Finish` is the return from `applyPending` after the catalog. `Broken = TRUE` calls `Down` instead of `Up`.

`RunOnlyMovesForward` requires that Down was not called, and that a finished run has applied every catalog version, with `executed` equal to the versions that were missing at the start. A later version may already be present while an earlier one is applied underneath it. That is the `{2}` seed.

Baseline commit is `e1622e04eaf33bbc4002c6a87c5b02e09f6ececb`. `migrations/runner.go` may differ from that commit, so the fingerprint in `coverage.json` identifies the baseline. Move the sync commit forward only after you confirm that this spec still describes the code at the new commit.

## when a check fails

Replay the TLC path against `applyPending` and `runMigration`. If a step cannot happen there, the model is wrong. If every step can happen, the code breaks the property. Say which version was skipped, applied, or sent to Down. Do not weaken `RunOnlyMovesForward` to obtain a pass.

`TestVerificationMigrationDirection` runs catalog `1, 2` with version 2 already recorded. It requires one `Up`, zero `Down` calls, `Applied` true only on version 1, and no second transaction. The mutation replaces the `Up` call with `Down`.

## leaves out

Whether `Up` and the migrations-table insert become visible together is `verify-migrations`. This model starts from a successful pass: it does not include a failed `Up`, a failed insert, or a failed commit. `Down` exists on `Migration` and `validateMigrationEntry` requires it. `Run` does not call it. Making `Down` succeed is out of this check.
