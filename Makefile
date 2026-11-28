run:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go run ./cmd/reactorlog server

migrate:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go run ./cmd/reactorlog migrate

db:
	docker compose exec postgres psql -U reactorlog -d reactorlog

ingest-nrc:
	go run ./cmd/reactorlog ingest-nrc

ingest-eia-plants:
	go run ./cmd/reactorlog ingest-eia-plants

test:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go test -count=1 ./...

lint-sql:
	@python3 .cursor/scripts/sql-keyword-case.py


test-db:
	@docker compose exec -T postgres \
		psql -U reactorlog -d postgres -tAc \
		"select 1 from pg_database where datname = 'reactorlog_test'" \
		| grep -q 1 || \
	docker compose exec -T postgres \
		createdb -U reactorlog reactorlog_test

BENCH_PACKAGES ?= ./...
BENCH_PATTERN ?= .
BENCH_COUNT ?= 10
BENCH_TIME ?= 1s
BENCH_OUTPUT ?= .benchmarks/latest.txt
BENCHSTAT_VERSION := v0.0.0-20261009192801-be2c69fb417e

.PHONY: bench bench-save bench-compare

bench:
	@go test $(BENCH_PACKAGES) -run='^$$' -bench='$(BENCH_PATTERN)' -benchmem -count=$(BENCH_COUNT) -benchtime=$(BENCH_TIME)

bench-save:
	@mkdir -p "$$(dirname "$(BENCH_OUTPUT)")"
	@$(MAKE) --no-print-directory bench > "$(BENCH_OUTPUT).tmp" && mv "$(BENCH_OUTPUT).tmp" "$(BENCH_OUTPUT)"
	@echo "saved $(BENCH_OUTPUT)"

bench-compare:
	@test -n "$(BASE)" -a -n "$(NEW)" || { echo 'usage: make bench-compare BASE=before.txt NEW=after.txt' >&2; exit 1; }
	@go run golang.org/x/perf/cmd/benchstat@$(BENCHSTAT_VERSION) "$(BASE)" "$(NEW)"
