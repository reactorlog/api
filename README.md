# reactorlog

a go/postgres project for exploring public nuclear-reactor data: where reactors are, how they are designed, and what happens during operation.

**status:** early development. the repository implements the database, migrations, and the NRC operating-reactor import. scram events and http endpoints are planned. the `server` command currently verifies a database connection and exits.

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

once postgres is ready:

```sh
make migrate
make ingest-nrc
make run
```

open a database shell with `make db`. the cli also accepts `go run ./cmd/reactorlog migrate`, `go run ./cmd/reactorlog ingest-nrc`, and `go run ./cmd/reactorlog server`; it loads `.env` on startup.

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
coordinates, EIA codes, or contact details. The initial schema leaves timezone and
design nullable so those fields stay NULL on new records and are preserved on
existing records. No timezone is inferred from state.

The parser fixture is the NRC unit table retrieved on November 5, 2026. It is
used to detect markup changes without network-dependent tests.

## tests

```sh
make test
```

`make test` loads the root `.env` file. `make test-db` creates an optional `reactorlog_test` database for local development. Database integration tests require `TEST_DATABASE_URL` pointing to a disposable PostgreSQL 18 database.

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
| `internal/database` | postgres connection pool |
| `internal/ingestion/nrc` | NRC list parsing and the import statement |
| `migrations` | sql files, version tracking, and the transactional migration runner |

## next

add reactor-linked scram events and expose the data through an http api. the goal is a searchable dataset and map with traceable source records.
