run:
	@set -a; \
	. ./.env; \
	set +a; \
	go run ./cmd/server

migrate-up:
	@set -a; \
	. ./.env; \
	set +a; \
	migrate -path ./migrations -database $$DATABASE_URL up

migrate-down:
	@set -a; \
	. ./.env; \
	set +a; \
	migrate -path ./migrations -database $$DATABASE_URL down 1

db:
	docker compose exec postgres psql -U reactorlog -d reactorlog