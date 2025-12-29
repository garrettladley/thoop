# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=devel

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -tags release \
    -ldflags="-s -w -X github.com/garrettladley/thoop/internal/version.version=${VERSION}" \
    -o /server ./cmd/server

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
