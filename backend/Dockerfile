# Stage 1: Build
FROM golang:1.25-alpine AS builder

ARG GOOSE_VERSION=v3.27.1

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/worker ./cmd/worker/

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/bin/worker .
COPY --from=builder /go/bin/goose .
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:${PORT:-8080}/healthz || exit 1

ENTRYPOINT ["./server"]
