# reactorlog

a go/postgres project for exploring public nuclear-reactor data: where reactors are, how they are designed, and what happens during operation.

**status:** early development. the repository currently implements the database foundation and cli. data ingestion, scram events, and http endpoints are planned. the `server` command currently verifies a database connection and exits.

## implemented

- separate site and reactor models, with a foreign key from each reactor to its site
- site coordinates and timezones; unique reactor docket numbers
- postgres uuidv7 identifiers and timestamp defaults
- schema builders for columns, foreign keys, and check constraints
- sql query builders with postgres placeholders and separate value arguments
- versioned migrations that apply schema changes and record the version in one transaction
- tests for database connection handling, expressions, schema/query builders, migrations, and cli behavior

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
make run
```

open a database shell with `make db`. the cli also accepts `go run ./cmd/reactorlog migrate` and `go run ./cmd/reactorlog server`; it loads `.env` on startup.

## tests

```sh
make test
```

`make test` loads the root `.env` file. `make test-db` creates an optional `reactorlog_test` database for local development.

## benchmarks

```sh
make bench
make bench BENCH_COUNT=1 BENCH_TIME=100ms # quick smoke check
```

No postgres or `.env` is needed. The initial benchmarks measure `Build()` for a
docket lookup, combined reactor filters, and a reactor insert. Query setup and
output validation are outside the timed `b.Loop()`; SQL rendering and argument
allocation are inside. These measure query construction, not database execution
or API latency.

Capture measurements before and after an implementation change:

```sh
make bench-save BENCH_OUTPUT=.benchmarks/before.txt
# change the implementation, keeping the benchmark inputs unchanged
make bench-save BENCH_OUTPUT=.benchmarks/after.txt
make bench-compare BASE=.benchmarks/before.txt NEW=.benchmarks/after.txt
```

Each benchmark runs ten times for one second per measurement by default. Override
`BENCH_PACKAGES`, `BENCH_PATTERN`, `BENCH_COUNT`, or `BENCH_TIME` to narrow or extend
a run. For example:

```sh
make bench BENCH_PACKAGES=./internal/database/query BENCH_PATTERN=BenchmarkSelectBuild
```

`bench-compare` downloads a pinned version of `benchstat` on first use without
adding application dependencies. Results report time, bytes allocated, and
allocations per operation. Local results are ignored by git. A failed capture
leaves a `.tmp` file and preserves the previous result.

Use the same machine, Go version, input data, and runtime settings for both sides.
Record the commit and `go version` with results you share. Keep the machine idle;
alternate baseline and candidate runs when checking small differences. Treat an
inconclusive comparison as inconclusive. Check correctness and end-to-end impact
before claiming a speedup. Collect profiles separately from reported timings:

```sh
mkdir -p .benchmarks
go test ./internal/database/query -run='^$' -bench=BenchmarkSelectBuild/docket \
  -benchtime=5s -cpuprofile=.benchmarks/cpu.out -o .benchmarks/query.test
go tool pprof .benchmarks/query.test .benchmarks/cpu.out
```

## code map

| path | responsibility |
| --- | --- |
| `cmd/reactorlog` | cli commands and startup |
| `internal/database` | postgres connection pool |
| `internal/database/expr` | sql condition expressions |
| `internal/database/query` | select, insert, update, and delete builders |
| `internal/database/schema` | table definitions and ddl generation |
| `migrations` | site/reactor definitions, version tracking, and transactional migration runner |

## next

ingest public nrc reactor data and eia plant coordinates, add reactor-linked scram events, and expose the data through an http api. the goal is a searchable dataset and map with traceable source records.
