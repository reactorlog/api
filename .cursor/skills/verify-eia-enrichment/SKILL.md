---
name: verify-eia-enrichment
description: Verify which fields an EIA enrich writes when modifying writeEnrichment, enrich.sql, or the initial schema nullability for timezone and location. Run the TLA+ model and the Go mutation check.
---

# verify eia enrichment

The property is that a committed EIA enrichment writes plant fields onto the site matched by name when they differ, and leaves timezone, location, and reactors unchanged. A violation means plant import clears NRC-owned fields or skips the named site.

## run

From the api repository root, with `TLA2TOOLS` set to `tla2tools.jar` from TLC 1.7.4 and `JAVA` set to Java 11 or later (`JAVA` defaults to `java` on `PATH`):

```sh
python3 .cursor/skills/verify-eia-enrichment/scripts/check_go_mutation.py
```

A nonzero exit means the invariant failed, the broken model passed, a Go mutation survived, or a covered file no longer matches its fingerprint. Read a TLC result with `/statechart`. The spec is `EiaEnrichment.tla`, and `view.mjs` beside it is the chart's wording. Do not treat a fingerprint mismatch as a passing model.

## coverage

Read `references/coverage.json` and `EiaEnrichment.tla` before interpreting a result. Covered code is `writeEnrichment` in `internal/ingestion/eia/store.go`, `internal/ingestion/eia/sql/enrich.sql`, and `migrations/sql/1_create_sites_and_reactors.up.sql`, which leaves `sites.timezone` and `sites.location` nullable.

Bounds are four enrichments of one site with one reactor whose design is set: fill empty enrichment, replace prior enrichment, skip when the plant tuple already matches, and miss when `site_name` does not equal `sites.name`. `Apply` writes the plant enrichment columns on a name match when values differ. `Skip` and `Miss` leave the row unchanged. `Broken` clears timezone on apply.

`EnrichmentAppliesAndProtects` requires timezone, location, and the reactor to equal the pre-enrich snapshot after commit. Fill and replace require enrichment equal to the incoming plant. Skip and miss require the whole site row unchanged.

Baseline commit is `ec164ab223b0debb511fc9c3cbdc5d599060ee43`. The covered files are not all introduced in that commit, so the fingerprints in `coverage.json` identify the baseline. Move the sync commit forward only after you confirm that this spec still describes the code at the new commit.

## when a check fails

Replay the TLC path against `enrich.sql`. The update sets `eia_plant_code`, `address_line_1`, `city`, `state`, `zip`, `latitude`, `longitude`, and `updated_at`. It matches `site.name = incoming.site_name` and skips when the enrichment tuple `is distinct from` is false. If a step writes timezone, location, design, or another column that statement does not mention, the model is wrong. If the statement clears timezone or location, or drops the name match, the code breaks the property. Do not weaken `EnrichmentAppliesAndProtects` to obtain a pass.

`TestVerificationEiaEnrichmentColumns` rejects an enrich statement that names a protected column, omits a plant column, drops the name match, or drops `is distinct from`. The mutation adds `timezone = null` to the update. The test must fail, and the restored file must pass.

## leaves out

Which plant matches a site name is the Go match tables, not this model. Advisory-lock mutual exclusion is a separate concern. Concurrent NRC rename after `listSites` can miss the update. Fetch, zip parse, and more than one site are out. `state` is written by this statement. NRC import also writes `state`. Null `address_line_2`, phone, and email stay out because enrich never sets them. Which enrichment columns NRC import preserves is `verify-import-enrichment`.
