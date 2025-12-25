VERSION ?= devel
LDFLAGS := -s -w -X github.com/garrettladley/thoop/internal/version.version=$(VERSION)

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

## build: build all binaries with version
.PHONY: build
build:
	@echo "Building with version: $(VERSION)"
	@go build -ldflags="$(LDFLAGS)" -o bin/thoop ./cmd/thoop
	@go build -ldflags="$(LDFLAGS)" -o bin/thoop-proxy ./cmd/proxy
	@go build -ldflags="$(LDFLAGS)" -o bin/thoop-db ./cmd/db

## build/thoop: build TUI client with version
.PHONY: build/thoop
build/thoop:
	@go build -ldflags="$(LDFLAGS)" -o bin/thoop ./cmd/thoop

## build/proxy: build proxy server with version
.PHONY: build/proxy
build/proxy:
	@go build -ldflags="$(LDFLAGS)" -o bin/thoop-proxy ./cmd/proxy

## version: print current version
.PHONY: version
version:
	@echo $(VERSION)

## thoop: run main CLI
.PHONY: thoop
thoop:
	@go run -ldflags="$(LDFLAGS)" ./cmd/thoop

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

# SQL and Code Generation
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

# Redis
## redis: start Redis container for local dev
.PHONY: redis
redis:
	@docker compose up -d redis
	@echo 'Redis running on localhost:6379'

## redis/stop: stop Redis container
.PHONY: redis/stop
redis/stop:
	@docker compose down

# Proxy
## proxy: run proxy server (requires .env.proxy or env vars)
.PHONY: proxy
proxy:
	@go run -ldflags="$(LDFLAGS)" ./cmd/proxy
