# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/db/migrations ./db/migrations
COPY scripts/entrypoint.sh /app/entrypoint

RUN chmod +x /app/entrypoint

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:${PORT:-8080}/healthz || exit 1

ENTRYPOINT ["/app/entrypoint"]
CMD ["serve"]
