DB_CONTAINER = rmpc-postgres
DB_PORT      = 5432
DB_USER      = rmpc
DB_PASS      = rmpc
DB_NAME      = rmpc
LOCAL_DSN    = postgresql://$(DB_USER):$(DB_PASS)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

DB_DSN  ?= $(DATABASE_URL)
JET_DSN ?= $(or $(DB_DSN),$(LOCAL_DSN))
JET_BIN  = $(shell go env GOPATH)/bin/jet
LINT_BIN = $(shell go env GOPATH)/bin/golangci-lint

# The "..." wildcard skips any directory whose name starts with "_" or ".", so
# ./... does not match anything under api/_pkg. Those packages have to be named
# explicitly or they are never built, vetted, tested or linted. Discover them
# rather than listing them, so a new package is picked up automatically.
MODULE        = $(shell go list -m)
INTERNAL_PKGS = $(shell find api/_pkg -name '*.go' -exec dirname {} \; | sort -u | sed 's|^|$(MODULE)/|')
PKGS          = ./... $(INTERNAL_PKGS)

.PHONY: build vet test lint fmt check pkgs generate migrate migrate-down dev clean db-start db-stop db-reset

pkgs: ## List every package the checks cover
	@for p in $(PKGS); do echo $$p; done

build: ## Build all packages
	go build $(PKGS)

vet: ## Run go vet
	go vet $(PKGS)

fmt: ## Check formatting (prints offending files)
	@out="$$(gofmt -l api cmd)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi; \
	echo "gofmt: clean"

test: ## Run tests
	go test $(PKGS)

check: fmt vet test ## Everything CI runs

lint: $(LINT_BIN) ## Run golangci-lint
	$(LINT_BIN) run $(PKGS)

$(LINT_BIN):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

generate: $(JET_BIN) ## Generate go-jet types from database (defaults to local Docker DB)
	$(JET_BIN) -dsn="$(JET_DSN)" -schema=public -path=./db/.gen

$(JET_BIN):
	go install github.com/go-jet/jet/v2/cmd/jet@latest

# There is no migration tracking table, so these apply every file every time.
# That is fine on a fresh database; against one that is already migrated, pass
# MIGRATION=<path> to apply a single file.
MIGRATIONS_UP   = $(sort $(wildcard db/migrations/*.up.sql))
MIGRATIONS_DOWN = $(shell ls -r db/migrations/*.down.sql 2>/dev/null)

# $(1) is the list of .sql files to apply, in order.
define run_sql
	@for f in $(1); do \
		echo "applying $$f"; \
		if [ -n "$(DB_DSN)" ]; then \
			psql "$(DB_DSN)" -v ON_ERROR_STOP=1 -f "$$f" || exit 1; \
		else \
			docker cp "$$f" $(DB_CONTAINER):/tmp/migration.sql >/dev/null && \
			docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 -f /tmp/migration.sql || exit 1; \
		fi; \
	done
endef

migrate: ## Apply migrations in order (MIGRATION=<path> for one file)
	$(call run_sql,$(if $(MIGRATION),$(MIGRATION),$(MIGRATIONS_UP)))

migrate-down: ## Roll back migrations, newest first (MIGRATION=<path> for one)
	$(call run_sql,$(if $(MIGRATION),$(MIGRATION),$(MIGRATIONS_DOWN)))

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
	$(call run_sql,$(MIGRATIONS_UP))

dev: ## Run local dev server with mock Openplanet auth
	DATABASE_URL=$(LOCAL_DSN) \
	OPENPLANET_PLUGIN_SECRET=dev-secret \
	OPENPLANET_AUTH_URL=http://localhost:8081/api/auth/validate \
	PLAYER_LINK_SECRET=dev-link-secret \
	SCORE_COOLDOWN=5s \
	go run ./cmd/dev

clean: ## Remove build artifacts
	go clean ./...
