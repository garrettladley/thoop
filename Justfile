set quiet := true

redis_port := env("REDIS_PORT", "6420")
postgres_port := env("POSTGRES_PORT", "6767")
server_port := env("SERVER_PORT", "8989")
migrations_path := env("MIGRATIONS_PATH", "internal/migrations/sql")
db_path := env("DB_PATH", "thoop.db")
templ_version := env("TEMPL_VERSION", "v0.3.865")

[private]
default:
    just --list

[private]
confirm:
    echo 'Are you sure? [y/N]' && read ans && [ ${ans:-N} = y ]

# install dependencies
[group('dev')]
install: install-go

# install go dependencies
[group('dev')]
install-go:
    go mod download

# tidy go modules
[group('dev')]
tidy:
    go mod tidy

# run tests
[group('dev')]
test:
    go test -v -race ./...

# run tests with coverage output for CI
[group('ci')]
test-ci:
    go test -v -json -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt ./... -timeout 5m

# format and modernize code
[group('dev')]
fmt:
    go fix ./...
    golangci-lint fmt

# modernize Go code
[group('dev')]
go-fix:
    go fix ./...

# verify go fix does not produce changes
[group('ci')]
go-fix-verify:
    #!/usr/bin/env bash
    set -euo pipefail
    go fix ./...
    git diff --exit-code || (echo "go fix produced changes - run 'go fix ./...' locally and commit" && exit 1)

# run linter
[group('dev')]
lint:
    go fix ./...
    golangci-lint run --path-mode=abs --config=".golangci.yml" --timeout=5m

# run linter with auto-fix
[group('dev')]
lint-fix:
    go fix ./...
    golangci-lint run --path-mode=abs --config=".golangci.yml" --timeout=5m --fix

# build all binaries (dev mode, includes dev commands)
[group('build')]
build:
    go build -tags dev -ldflags="-s -w" -o bin/thoop ./cmd/thoop
    go build -tags dev -ldflags="-s -w" -o bin/thoop-server ./cmd/server
    go build -tags dev -ldflags="-s -w" -o bin/thoop-db ./cmd/db

# build all binaries for release (no dev commands)
[group('build')]
build-release:
    go build -ldflags="-s -w" -o bin/thoop ./cmd/thoop
    go build -ldflags="-s -w" -o bin/thoop-server ./cmd/server
    go build -ldflags="-s -w" -o bin/thoop-db ./cmd/db

# build TUI client with version (dev mode)
[group('build')]
build-thoop:
    go build -tags dev -ldflags="-s -w" -o bin/thoop ./cmd/thoop

# build server with version
[group('build')]
build-server:
    go build -ldflags="-s -w" -o bin/thoop-server ./cmd/server

# generate version_gen.go from manifest
[group('version')]
version-gen:
    go run -tags dev ./cmd/gen_version

# generate snapshot version for local dev
[group('version')]
version-snapshot:
    go run -tags dev ./cmd/gen_version -snapshot

# print current version
[group('version')]
version:
    grep 'Version.*string' version_gen.go | sed 's/.*"\(.*\)".*/\1/'

# run main CLI (dev mode)
[group('run')]
thoop:
    go run -tags dev -ldflags="-s -w" ./cmd/thoop

# run database CLI
[group('run')]
db:
    go run ./cmd/db

# apply all up migrations
[group('migrate')]
migrate-up:
    echo 'Running migrations...'
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path={{ migrations_path }} -database=sqlite3://{{ db_path }} up

# rollback all migrations
[group('migrate')]
migrate-down: confirm
    echo 'Rolling back migrations...'
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path={{ migrations_path }} -database=sqlite3://{{ db_path }} down

# rollback the last migration
[group('migrate')]
migrate-down-1:
    echo 'Rolling back last migration...'
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path={{ migrations_path }} -database=sqlite3://{{ db_path }} down 1

# force migration version
[group('migrate')]
migrate-force version:
    echo 'Forcing migration version {{ version }}...'
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path={{ migrations_path }} -database=sqlite3://{{ db_path }} force {{ version }}

# show current migration version
[group('migrate')]
migrate-version:
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate -path={{ migrations_path }} -database=sqlite3://{{ db_path }} version

# create a new migration file
[group('migrate')]
migrate-create name:
    echo 'Creating migration files for {{ name }}...'
    go run -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate create -ext sql -dir {{ migrations_path }} -seq {{ name }}

# install templ CLI
[group('codegen')]
templ-install:
    go install github.com/a-h/templ/cmd/templ@{{ templ_version }}

# generate Go code from templ files
[group('codegen')]
templ-generate:
    templ generate

# verify templ generated code is up to date
[group('ci')]
templ-verify: templ-generate
    git diff --exit-code

# install sqlc
[group('codegen')]
sqlc-install:
    echo 'Installing sqlc...'
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# generate sqlc code from SQL
[group('codegen')]
sqlc-generate:
    echo 'Generating sqlc code...'
    sqlc generate

# verify sqlc schema and queries
[group('ci')]
sqlc-vet:
    sqlc vet

# verify sqlc generated code is up to date
[group('ci')]
sqlc-diff:
    sqlc diff

# start all services (redis, postgres)
[group('db')]
up:
    docker compose up -d
    echo 'Services running:'
    echo '  Redis:    localhost:{{ redis_port }}'
    echo '  Postgres: localhost:{{ postgres_port }} (user: thoop, pass: thoop, db: thoop)'

# stop all services
[group('db')]
down:
    docker compose down

# connect to local postgres
[group('db')]
psql:
    docker compose exec postgres psql -U thoop -d thoop

# flush all redis data
[group('db')]
redis-flush:
    redis-cli -p {{ redis_port }} FLUSHALL

# run server (requires .env or env vars)
[group('run')]
server:
    PORT={{ server_port }} go run -ldflags="-s -w" ./cmd/server

alias b := build
alias t := test
alias r := server
