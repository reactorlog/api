---
name: verify-import-enrichment
description: Verify which enrichment fields an NRC import writes when modifying store, import.sql, or migration 3. Run the TLA+ model and the Go mutation check.
---

# verify import enrichment

The property is that a committed import leaves design, timezone, coordinates, and address unchanged on rows that already exist. A violation means reimport erases enrichment that another writer stored.

## run

From the api repository root, with `TLA2TOOLS` set to `tla2tools.jar` from TLC 1.7.4 and `JAVA` set to Java 11 or later (`JAVA` defaults to `java` on `PATH`):

```sh
python3 .cursor/skills/verify-import-enrichment/scripts/check_go_mutation.py
```

A nonzero exit means the invariant failed, the broken model passed, a Go mutation survived, or a covered file no longer matches its fingerprint. Read a TLC result with `/statechart`. The spec is `ImportEnrichment.tla`, and `view.mjs` beside it is the chart's wording. Do not treat a fingerprint mismatch as a passing model.

## coverage

Read `references/coverage.json` and `ImportEnrichment.tla` before interpreting a result. Covered code is `store` in `internal/ingestion/nrc/store.go`, `internal/ingestion/nrc/sql/import.sql`, and migration 3 in `migrations/sql/3_nrc_reference_data.up.sql`, which drops `NOT NULL` on `sites.timezone` and `reactors.design`.

Bounds are three imports: a stored reactor whose design, timezone, and address are set and whose coordinates are half-set; a new reactor joining that site; a new site and reactor. `Reimport` is the docket hit: the site update writes name, location, and state, and the reactor upsert writes name, unit, and type. `Join` returns the existing site id and does not update the site. `Create` inserts a site with name, location, and state, and a reactor without design. `Broken` sets design to empty on reimport.

`EnrichmentSurvives` requires the committed site and reactor enrichment to equal the snapshot. Half-set coordinates stay half-set. This check does not forbid that state. The schema allows it, and the latitude and longitude checks pass on null.

Baseline commit is `e1622e04eaf33bbc4002c6a87c5b02e09f6ececb`. The covered files are not all in that commit, so the fingerprints in `coverage.json` identify the baseline. Move the sync commit forward only after you confirm that this spec still describes the code at the new commit.

## when a check fails

Replay the TLC path against `import.sql`. The site update writes name, location, and state. The site insert writes name, location, and state. The reactor upsert writes name, unit, and type. If a step writes a column that statement does not mention, the model is wrong. If the statement writes design, timezone, coordinates, or address on a row that already had them, the code breaks the property. Do not weaken `EnrichmentSurvives` to obtain a pass.

`TestVerificationImportKeepsEnrichment` executes that file and rejects an import statement that names an enrichment column. The mutation adds `design = null` to the upsert. The test must fail, and the restored file must pass.

## leaves out

Which site a docket is linked to is `verify-nrc-site-link`. A writer other than `store` may set these columns. This model does not include that writer. It checks that `store` does not clear the value. Null `sites.location` and the unique index are out of this model.
