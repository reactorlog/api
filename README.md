# reactorlog

a go/postgres project for exploring public nuclear-reactor data: where reactors are, how they are designed, and what happens during operation.

**status:** early development. the repository implements the database, migrations, the NRC operating-reactor import, and a read-only HTTP API for sites and reactors. scram events are planned.

## implemented

- separate site and reactor models, with a foreign key from each reactor to its site
- site coordinates and timezones; unique reactor docket numbers
- postgres uuidv7 identifiers and timestamptz defaults
- schema and import statements in `.sql` files, executed by the migration runner and the NRC importer
- versioned migrations that apply each up script and record the version in one transaction
- NRC operating-reactor import in one serialized transaction
- tests for database connection handling, migrations, NRC parsing and import, and cli behavior

## run locally

requires go matching `go.mod` (currently 1.27.1), docker compose, and make. the compose file runs postgres 18, which provides the `uuidv7()` default used by the schema.

```sh
git clone https://github.com/reactorlog/api.git
cd api
docker compose up -d
```

create a `.env` file in the repository root:

```dotenv
DATABASE_URL=postgresql://reactorlog:reactorlog@localhost:5432/reactorlog?sslmode=disable
```

omit the file when `DATABASE_URL` is already set.

### Neon (hosted Postgres + object storage)

The repo links to Neon project `nameless-term-76560583` via `.neon` and `neon.ts` (Postgres + private `excel` bucket). After `neon login`:

```sh
neon deploy          # reconcile neon.ts and refresh .env.local
npm run build
./reactorlog migrate
./reactorlog storage-check
./reactorlog server
```

`neon deploy` writes `DATABASE_URL` (pooled), `DATABASE_URL_UNPOOLED` (direct), and AWS S3 env vars into `.env.local`. The CLI loads `.env` then overloads `.env.local`. `migrate` prefers `DATABASE_URL_UNPOOLED`. Object Storage uses the `excel` bucket (override with `STORAGE_BUCKET`).

### Fly.io (API + website)

Same-origin deploy: one Machine serves the Go API under `/api/*` and the Vite `dist` from `STATIC_DIR`. Free plan needs a card for the recurring $5 allowance. Deploy with one Machine:

```sh
# build web into ./static, then:
fly secrets set DATABASE_URL=...
fly deploy --ha=false
```

`fly.toml` checks `GET /live` (no DB ping). Do not probe `/health` on a short interval. GitHub Actions workflows: `migrate.yml`, `ingest.yml`, `deploy.yml` (secrets: `DATABASE_URL`, optional `DATABASE_URL_UNPOOLED`, `FLY_API_TOKEN`).

When the SPA is on another origin (for example `https://www.reactorlog.com`), set `CORS_ORIGINS` to a comma-separated allowlist. `fly.toml` defaults it to `https://www.reactorlog.com`. Leave it empty for same-origin-only local runs. Browser calls stay under `/api/*` on the Fly host.

When the SPA is on another origin (for example `https://www.reactorlog.com`), set `CORS_ORIGINS` to a comma-separated allowlist. `fly.toml` defaults it to `https://www.reactorlog.com`. Leave it empty for same-origin-only local runs. Browser calls stay under `/api/*` on the Fly host.

once postgres is ready:

```sh
make migrate
make ingest-nrc
make run
```

open a database shell with `make db`. `make` and the cli load `.env` when the file is present, and overload `.env.local` when Neon has written one. the cli also accepts `go run ./cmd/reactorlog migrate`, `go run ./cmd/reactorlog ingest-nrc`, `go run ./cmd/reactorlog ingest-eia-plants`, `go run ./cmd/reactorlog storage-check`, and `go run ./cmd/reactorlog server`. `server`, `migrate`, `ingest-nrc`, and `ingest-eia-plants` still require `DATABASE_URL`.

`server` listens on `HTTP_ADDR` (default `:8080`) and serves:

- `GET /health`
- `GET /sites`
- `GET /sites/{id}`
- `GET /reactors`

## NRC reference data

The importer reads the NRC [List of Power Reactor Units](https://www.nrc.gov/facilities-safety/operating-reactors/list-power-reactor-units).
This covers the units in that list, rather than retired, proposed, or research reactors.
It imports reactor names, unit numbers, eight-digit docket numbers (including leading
zeros), reactor types, site names, locations, and state abbreviations. An unnumbered
unit in this list is represented as unit 1. Site names are the unit names with the
trailing unit number removed. A site is identified by its name. Two units that
share a site name share that site. Location is descriptive text from the list.
Named facilities stay separate when they share a location string, such as two
plants near Oswego.

Reactors match by docket number. A stored reactor keeps its site, and that site's
name, location, and state are updated from the list. A new reactor joins the site
with the same name, or a site is created. Giving one site another site's name
fails the transaction rather than merging the two sites.

All rows are validated before database writes. Each run is transactional, concurrent
imports are serialized, and repeated imports update existing records. Units absent
from a later list are retained. The list does not provide timezone, design,
capacity_mw, coordinates, EIA codes, or contact details. The initial schema leaves
timezone, design, and capacity_mw nullable so those fields stay NULL on new
records and are preserved on existing records. No timezone is inferred from state.

The parser fixture is the NRC unit table retrieved on November 5, 2026. It is
used to detect markup changes without network-dependent tests.

## EIA plant enrichment

`ingest-eia-plants` downloads Form EIA-860, keeps operable nuclear plants, and
writes `eia_plant_code`, address fields, and coordinates onto existing sites.
Matching uses exact names, then deterministic soft tokens (strip station
suffixes and operator prefixes), then a small checked-in alias map. Every site
must match or the command fails. Extra EIA plants are logged and skipped.
Run it after `ingest-nrc`.

The same command writes each reactor's `capacity_mw` from Nameplate Capacity (MW)
on the operable sheet, for rows whose Energy Source 1 is NUC. After a site matches
a plant, that site's reactor `unit_number` values must be the same set as the
plant's Generator IDs. The import fails when a unit has no generator, a generator
has no unit, or either id is repeated. The update runs in the site enrichment transaction and changes
`capacity_mw` only when the stored value differs. `ingest-nrc` does not write
`capacity_mw`.

## tests

```sh
make test
```

`make test` loads the root `.env` file when it is present. `make test-db` creates an optional `reactorlog_test` database for local development. Database integration tests require `TEST_DATABASE_URL` pointing to a disposable PostgreSQL 18 database.

## benchmarks

```sh
make bench
make bench BENCH_COUNT=1 BENCH_TIME=100ms # quick smoke check
```

No postgres or `.env` is needed. Capture measurements before and after an implementation change:

```sh
make bench-save BENCH_OUTPUT=.benchmarks/before.txt
# change the implementation, keeping the benchmark inputs unchanged
make bench-save BENCH_OUTPUT=.benchmarks/after.txt
make bench-compare BASE=.benchmarks/before.txt NEW=.benchmarks/after.txt
```

Each benchmark runs ten times for one second per measurement by default. Override
`BENCH_PACKAGES`, `BENCH_PATTERN`, `BENCH_COUNT`, or `BENCH_TIME` to narrow or extend
a run. `bench-compare` downloads a pinned version of `benchstat` on first use without
adding application dependencies. Results report time, bytes allocated, and
allocations per operation. Local results are ignored by git. A failed capture
leaves a `.tmp` file and preserves the previous result.

Use the same machine, Go version, input data, and runtime settings for both sides.
Record the commit and `go version` with results you share. Keep the machine idle;
alternate baseline and candidate runs when checking small differences. Treat an
inconclusive comparison as inconclusive.

## code map

| path | responsibility |
| --- | --- |
| `cmd/reactorlog` | cli commands and startup |
| `internal/api` | read-only HTTP handlers and embedded SQL |
| `internal/database` | postgres connection pool |
| `internal/ingestion/nrc` | NRC list parsing and the import statement |
| `internal/ingestion/eia` | EIA-860 plant download, match, site enrichment, and reactor capacity |
| `migrations` | sql files, version tracking, and the transactional migration runner |

## next

add reactor-linked scram events and a hosted frontend over the plant-list API.
