---
name: verify-nrc-site-link
description: Verify NRC site linking when modifying store, import.sql, or the sites_name_location index. Run the TLA+ model and the Go mutation check.
---

# verify nrc site link

The property is that a new docket joins the site with its name and location, and a stored docket keeps its site. A violation means a second site is created for a plant that already exists, or a rename that collides commits a partial import.

## run

From the api repository root, with `TLA2TOOLS` set to `tla2tools.jar` from TLC 1.7.4 and `JAVA` set to Java 11 or later (`JAVA` defaults to `java` on `PATH`):

```sh
python3 .cursor/skills/verify-nrc-site-link/scripts/check_go_mutation.py
```

A nonzero exit means the invariant failed, the broken model passed, a Go mutation survived, or a covered file no longer matches its fingerprint. Read a TLC result with `/statechart`. The spec is `NrcSiteLink.tla`, and `view.mjs` beside it is the chart's wording. Do not treat a fingerprint mismatch as a passing model.

## coverage

Read `references/coverage.json` and `NrcSiteLink.tla` before interpreting a result. Covered code is `store` in `internal/ingestion/nrc/store.go` and `internal/ingestion/nrc/sql/import.sql`, and the unique index `sites_name_location` in `migrations/sql/1_create_sites.up.sql`.

Bounds are two sites, two dockets, two name-location keys, and six batches of at most two rows inside one statement. `Broken = FALSE` is that statement. `NrcSiteLinkBroken.cfg` skips the name-and-location join.

`Import` renames every stored site, then links new dockets. `Reject` is the unique-index error followed by `defer tx.Rollback`. The reactor upsert does not change `site_id` or `design`.

The six batches are `two-new-same`, `join-existing`, `rename-keep`, `rename-collision`, `rename-then-join`, and `create-then-collide`. The last two are the same stored reactor and the same two rows in opposite order. Renames happen before inserts, so both commit with the new reactor on the renamed site.

Baseline commit is `e1622e04eaf33bbc4002c6a87c5b02e09f6ececb`. The covered files are not all in that commit, so the fingerprints in `coverage.json` identify the baseline. Move the sync commit forward only after you confirm that this spec still describes the code at the new commit.

## when a check fails

Replay the TLC path against `import.sql`: stored sites are renamed, then a new docket joins by name and location or a site is inserted, then reactors are upserted. If a step cannot happen in that statement, the model is wrong. If every step can happen, the code breaks the property. Say the path as lookups and writes, not as TLA+ names. Do not weaken `NewDocketJoins` to obtain a pass.

A Go failure of `TestVerificationNewDocketJoinsSite` is the same property on the SQL file the function executes. The mutation adds `and false` to the name-and-location match. The test must fail, and the restored file must pass.

## leaves out

Advisory-lock mutual exclusion is `verify-import-run`. Which enrichment columns are written is `verify-import-enrichment`. Fetch, parse, deletes, and more than two rows are out of this model. PostgreSQL treats nulls in the unique index as distinct; this model has no null name or location, because NRC import always writes both.
