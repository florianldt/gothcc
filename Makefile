.DEFAULT_GOAL := dev

.PHONY: dev down clean sh templ sqlc generate test check

# Everything in Docker with live rebuilds: Postgres plus a container carrying
# Go, templ, sqlc, Tailwind and air. Edit on the host, air rebuilds inside.
# http://localhost:8000
dev:
	docker compose up --build

down:
	docker compose down

# Also drops the database, the module caches and the locally built dev image.
clean:
	docker compose down -v --rmi local

# A shell in the dev container, for one-off commands.
sh:
	docker compose exec app sh

# These helpers use the running app container, so the tool versions are the
# pinned ones from dev/Dockerfile; start it with `make dev` first.
templ:
	docker compose exec -T app go tool templ generate

sqlc:
	docker compose exec -T app sqlc generate

generate:
	docker compose exec -T app sh -ec 'go tool templ generate && sqlc generate'

test:
	docker compose exec -T app go test ./...

# Regenerate code, require a clean tracked working tree and gofmt-ed code,
# then run the Go checks. CI runs the same steps.
check: generate
	docker compose exec -T app sh -ec 'git diff --exit-code -- . && test -z "$$(gofmt -l .)" && go vet ./... && go test ./...'
