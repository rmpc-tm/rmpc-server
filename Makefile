DB_CONTAINER = rmpc-postgres
DB_PORT      = 5432
DB_USER      = rmpc
DB_PASS      = rmpc
DB_NAME      = rmpc
LOCAL_DSN    = postgresql://$(DB_USER):$(DB_PASS)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

DB_DSN  ?= $(DATABASE_URL)
JET_DSN ?= $(DB_DSN)
JET_BIN  = $(shell go env GOPATH)/bin/jet
LINT_BIN = $(shell go env GOPATH)/bin/golangci-lint

.PHONY: build vet test lint generate migrate migrate-down dev clean db-start db-stop db-reset

build: ## Build all packages
	go build ./api/...

vet: ## Run go vet
	go vet ./api/...

TEST_PKGS = rmpc-server/api/_pkg/auth rmpc-server/api/_pkg/config rmpc-server/api/_pkg/ratelimit

test: ## Run tests
	go test $(TEST_PKGS) -v

lint: $(LINT_BIN) ## Run golangci-lint
	$(LINT_BIN) run ./api/...

$(LINT_BIN):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

generate: $(JET_BIN) ## Generate go-jet types from database (requires JET_DSN or DATABASE_URL)
	@test -n "$(JET_DSN)" || { echo "Set DATABASE_URL or JET_DSN"; exit 1; }
	$(JET_BIN) -dsn="$(JET_DSN)" -schema=public -path=./db/.gen

$(JET_BIN):
	go install github.com/go-jet/jet/v2/cmd/jet@latest

migrate: ## Run migrations (uses local container or DATABASE_URL)
	@if [ -n "$(DB_DSN)" ]; then \
		psql "$(DB_DSN)" -f db/migrations/001_initial.up.sql; \
	else \
		docker cp db/migrations/001_initial.up.sql $(DB_CONTAINER):/tmp/migration.sql && \
		docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -f /tmp/migration.sql; \
	fi

migrate-down: ## Roll back migrations (uses local container or DATABASE_URL)
	@if [ -n "$(DB_DSN)" ]; then \
		psql "$(DB_DSN)" -f db/migrations/001_initial.down.sql; \
	else \
		docker cp db/migrations/001_initial.down.sql $(DB_CONTAINER):/tmp/migration.sql && \
		docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -f /tmp/migration.sql; \
	fi

db-start: ## Start local Postgres in Docker
	@docker inspect -f '{{.State.Running}}' $(DB_CONTAINER) 2>/dev/null | grep -q true \
		&& echo "$(DB_CONTAINER) is already running" \
		|| docker run -d --name $(DB_CONTAINER) \
			-e POSTGRES_USER=$(DB_USER) \
			-e POSTGRES_PASSWORD=$(DB_PASS) \
			-e POSTGRES_DB=$(DB_NAME) \
			-p $(DB_PORT):5432 \
			postgres:16-alpine
	@echo "Waiting for Postgres..."
	@for i in $$(seq 1 30); do docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -c 'SELECT 1' >/dev/null 2>&1 && break; sleep 0.5; done
	@echo "Ready: $(LOCAL_DSN)"

db-stop: ## Stop and remove local Postgres container
	@docker rm -f $(DB_CONTAINER) 2>/dev/null || true

db-reset: db-stop db-start ## Recreate local DB and run migrations
	@docker cp db/migrations/001_initial.up.sql $(DB_CONTAINER):/tmp/migration.sql
	@docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -f /tmp/migration.sql

dev: ## Run locally with Vercel CLI
	vercel dev

clean: ## Remove build artifacts
	go clean ./...
