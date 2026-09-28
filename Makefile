run:
	@set -a; \
	. ./.env; \
	set +a; \
	go run ./cmd/reactorlog server

migrate:
	@set -a; \
	. ./.env; \
	set +a; \
	go run ./cmd/reactorlog migrate

db:
	docker compose exec postgres psql -U reactorlog -d reactorlog

test:
	@set -a; \
	. ./.env; \
	set +a; \
	go test -count=1 ./...

test-db:
	@docker compose exec -T postgres \
		psql -U reactorlog -d postgres -tAc \
		"SELECT 1 FROM pg_database WHERE datname = 'reactorlog_test'" \
		| grep -q 1 || \
	docker compose exec -T postgres \
		createdb -U reactorlog reactorlog_test