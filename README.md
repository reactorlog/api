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
