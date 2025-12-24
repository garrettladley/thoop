FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /proxy ./cmd/proxy

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /proxy /proxy

EXPOSE 8080

CMD ["/proxy"]
