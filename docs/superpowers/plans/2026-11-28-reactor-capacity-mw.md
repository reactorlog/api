# Reactor capacity_mw Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist and expose EIA nameplate MW per reactor as `capacity_mw` via the existing `ingest-eia-plants` path.

**Architecture:** Parse operable NUC generators with nameplate into a plant-code index. Keep plant↔site match unchanged. After match, require generator ID ↔ `unit_number` bijection per site, then update `reactors.capacity_mw` in the same EIA transaction as site enrich. Edit migration 1 in place.

**Tech Stack:** Go, Postgres, embedded SQL, existing `internal/ingestion/eia` package.

## Global Constraints

- Field JSON/SQL name is `capacity_mw` only.
- No additive migration. Edit `migrations/sql/1_create_sites_and_reactors.up.sql`.
- Fail the EIA import on incomplete generator↔reactor bijection.
- NRC import must not write `capacity_mw`.
- SQL keywords lowercase.
- Pre-release: update callers and tests together. No compatibility aliases.

## File map

| File | Role |
| --- | --- |
| `migrations/sql/1_create_sites_and_reactors.up.sql` | add `capacity_mw` |
| `internal/ingestion/eia/plant.go` (or `generator.go`) | `Generator` type |
| `internal/ingestion/eia/parse.go` | parse Generator ID + Nameplate Capacity (MW) |
| `internal/ingestion/eia/store.go` + new SQL | bijection check + capacity update |
| `internal/api/types.go` + reactor SQL + store scan | read API |
| NRC import enrichment verify | ensure capacity preserved / not written by NRC |
| README + EIA design doc | document source and match rule |

---

### Task 1: Schema and read API

- [x] Add `capacity_mw double precision` to reactors in migration 1
- [x] Add `CapacityMW *float64 \`json:"capacity_mw"\`` to API Reactor
- [x] Update list/get reactor SQL and scan
- [x] Update API tests
- [x] `go test ./internal/api/... ./migrations/...`

### Task 2: Parse generators with nameplate

- [x] Add `Generator` type and parse operable NUC rows for Plant Code, Generator ID, Nameplate Capacity (MW)
- [x] Return generators indexed by plant code alongside plants
- [x] Extend parse fixtures/tests
- [x] `go test ./internal/ingestion/eia/... -count=1`

### Task 3: Match bijection and write capacity

- [x] After plant match, load reactors for matched sites
- [x] Fail if unit_number set ≠ Generator ID set for that plant
- [x] SQL update `capacity_mw` in the EIA transaction (idempotent distinct-from)
- [x] Store tests for pass, fail, and idempotent re-run
- [x] Confirm NRC import SQL does not mention `capacity_mw`; extend enrichment tests if needed
- [x] `go test ./internal/ingestion/... -count=1`

### Task 4: Docs

- [x] README: capacity from EIA nameplate via generator ID = unit number
- [x] Update or note in `docs/superpowers/specs/2026-11-28-eia-plant-match-design.md`
- [x] Commit when tasks are green
