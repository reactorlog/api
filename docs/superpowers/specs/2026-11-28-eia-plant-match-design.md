# EIA plant enrichment

## Goal

Fetch EIA-860 nuclear plant metadata, match every stored NRC site, and write plant code, address, and coordinates.

## Decisions

- Match order: exact `MatchKey`, then deterministic soft tokens, then checked-in site name → plant code aliases.
- Soft tokens strip station suffixes and operator or person prefixes, map `saint` to `st`, drop parentheticals, and allow token-subset matches. No fuzzy scores. No LLM.
- Alias today: `Shearon Harris` → `6015` (`Harris (NC)`).
- Unmatched sites fail the import. Extra EIA plants (for example Crane Clean Energy Center) are logged and skipped.
- Source: Form EIA-860 annual zip. Plant workbook plus operable generators with energy source `NUC`.
- CLI: `reactorlog ingest-eia-plants` writes enrichment. Observability is `slog` plus a non-zero exit on failure.

## Data shape

```text
EIAPlant {
  Code         string
  Name         string
  MatchKey     string
  AddressLine1 string
  City         string
  State        string
  Zip          string
  Latitude     float64
  Longitude    float64
}

MatchReport {
  Matched           []{ SiteName, EIAPlant }
  UnmatchedSites    []SiteName
  UnmatchedPlants   []EIAPlant
}
```

`NormalizeName` trims, lowercases, collapses whitespace, and strips punctuation.

## Package and CLI

- `Fetch(ctx)` downloads the current EIA-860 zip.
- `Parse` reads plant and operable-generator sheets. Keeps plants with at least one operable `NUC` generator. Missing required columns fail the parse.
- `Match(sites, plants)` is pure. Duplicate `MatchKey` on either side is an error.
- CLI: `reactorlog ingest-eia-plants --match-report`
  - Loads `select name from sites`
  - Fetches, parses, matches
  - Prints matched count, unmatched sites, unmatched nuclear plants
  - Exits non-zero on fetch/parse failure or any unmatched site or plant

Zip and Excel stay inside the `eia` package. Callers see `[]EIAPlant` and `MatchReport` only.

## Tests

- `NormalizeName` known pairs
- `Parse` fixture with nuclear and non-nuclear rows
- `Match` all-match, unmatched site, unmatched plant, duplicate key

Live check: run the CLI against a DB after `ingest-nrc` and capture the miss list. Exact-name gaps feed a later alias table. Do not add fuzzy matching in this phase.

## Live result (2026-11-28)

Against EIA-860 2025 and the current NRC-imported sites after soft match and aliases:

- matched 55 (exact 25, soft 29, alias 1)
- unused plants 1 (`Crane Clean Energy Center` / 8011)
- first run updated 55 rows; second run updated 0 (idempotent)

## Out of scope

- Writing `eia_plant_code`, address, or coordinates
- Schema or PostGIS changes
- EIA-930 hourly data
- API key config
