# Ports
REDIS_PORT := 6420
POSTGRES_PORT := 6767
SERVER_PORT := 8989

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^//'

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]


## install: install dependencies
.PHONY: install
install:
	@make install/go


## install/go: install go dependencies
.PHONY: install/go
install/go:
	@go mod tidy

## test: run tests
.PHONY: test
test:
	@go test -v -race ./...

## fmt: format code
.PHONY: fmt
fmt:
	@golangci-lint fmt

## lint: run linter
.PHONY: lint
lint:
	@golangci-lint run --path-mode=abs --config=".golangci.yml" --timeout=5m

## lint/fix: run linter with auto-fix
.PHONY: lint/fix
lint/fix:
	@golangci-lint run --path-mode=abs --config=".golangci.yml" --timeout=5m --fix

## build: build all binaries (dev mode, includes dev commands)
.PHONY: build
build:
	@go build -tags dev -ldflags="-s -w" -o bin/thoop ./cmd/thoop
	@go build -tags dev -ldflags="-s -w" -o bin/thoop-server ./cmd/server
	@go build -tags dev -ldflags="-s -w" -o bin/thoop-db ./cmd/db

## build/release: build all binaries for release (no dev commands)
.PHONY: build/release
build/release:
	@go build -ldflags="-s -w" -o bin/thoop ./cmd/thoop
	@go build -ldflags="-s -w" -o bin/thoop-server ./cmd/server
	@go build -ldflags="-s -w" -o bin/thoop-db ./cmd/db

## build/thoop: build TUI client with version (dev mode)
.PHONY: build/thoop
build/thoop:
	@go build -tags dev -ldflags="-s -w" -o bin/thoop ./cmd/thoop

## build/server: build server with version
.PHONY: build/server
build/server:
	@go build -ldflags="-s -w" -o bin/thoop-server ./cmd/server

## version/gen: generate version_gen.go from manifest
.PHONY: version/gen
version/gen:
	@go run -tags dev ./cmd/gen_version

## version/snapshot: generate snapshot version for local dev
.PHONY: version/snapshot
version/snapshot:
	@go run -tags dev ./cmd/gen_version -snapshot

## version: print current version
.PHONY: version
version:
	@grep 'Version.*string' version_gen.go | sed 's/.*"\(.*\)".*/\1/'

## thoop: run main CLI (dev mode)
.PHONY: thoop
thoop:
	@go run -tags dev -ldflags="-s -w" ./cmd/thoop

## db: run database CLI
.PHONY: db
db:
	@go run ./cmd/db

# Database migrations (golang-migrate CLI - alternative to db commands)
MIGRATIONS_PATH = internal/migrations/sql
DB_PATH ?= thoop.db

## migrate/up: apply all up migrations
.PHONY: migrate/up
migrate/up:
	@echo 'Running migrations...'
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path=$(MIGRATIONS_PATH) -database=sqlite3://$(DB_PATH) up

## migrate/down: rollback all migrations
.PHONY: migrate/down
migrate/down: confirm
	@echo 'Rolling back migrations...'
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path=$(MIGRATIONS_PATH) -database=sqlite3://$(DB_PATH) down

## migrate/down/1: rollback the last migration
.PHONY: migrate/down/1
migrate/down/1:
	@echo 'Rolling back last migration...'
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path=$(MIGRATIONS_PATH) -database=sqlite3://$(DB_PATH) down 1

## migrate/force: force migration version (use: make migrate/force VERSION=1)
.PHONY: migrate/force
migrate/force:
	@echo 'Forcing migration version $(VERSION)...'
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path=$(MIGRATIONS_PATH) -database=sqlite3://$(DB_PATH) force $(VERSION)

## migrate/version: show current migration version
.PHONY: migrate/version
migrate/version:
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path=$(MIGRATIONS_PATH) -database=sqlite3://$(DB_PATH) version

## migrate/create: create a new migration file (use: make migrate/create NAME=your_migration_name)
.PHONY: migrate/create
migrate/create:
	@echo 'Creating migration files for $(NAME)...'
	@go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

# Code Generation
TEMPL_VERSION := v0.3.865

## templ/install: install templ CLI
.PHONY: templ/install
templ/install:
	@go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)

## templ/generate: generate Go code from templ files
.PHONY: templ/generate
templ/generate:
	@templ generate

## sqlc/install: install sqlc
.PHONY: sqlc/install
sqlc/install:
	@echo 'Installing sqlc...'
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

## sqlc/generate: generate sqlc code from SQL
.PHONY: sqlc/generate
sqlc/generate:
	@echo 'Generating sqlc code...'
	@sqlc generate

## sqlc/verify: verify sqlc generated code is up to date
.PHONY: sqlc/verify
sqlc/verify: sqlc/generate
	@if ! git diff --exit-code internal/sqlc/ > /dev/null; then \
		echo "Error: sqlc generated code is out of date"; \
		echo "Please run 'make sqlc/generate' and commit the changes"; \
		exit 1; \
	fi
	@echo 'sqlc code is up to date'

# Docker Compose
## up: start all services (redis, postgres)
.PHONY: up
up:
	@docker compose up -d
	@echo 'Services running:'
	@echo '  Redis:    localhost:$(REDIS_PORT)'
	@echo '  Postgres: localhost:$(POSTGRES_PORT) (user: thoop, pass: thoop, db: thoop)'

## down: stop all services
.PHONY: down
down:
	@docker compose down

## psql: connect to local postgres
.PHONY: psql
psql:
	@docker compose exec postgres psql -U thoop -d thoop

## redis/flush: flush all redis data
.PHONY: redis/flush
redis/flush:
	@redis-cli -p $(REDIS_PORT) FLUSHALL

# Server
## server: run server (requires .env or env vars)
.PHONY: server
server:
	@go run -ldflags="-s -w" ./cmd/server
