#!/usr/bin/env sh
set -eu

MODE="${1:-serve}"

case "$MODE" in
    serve)
        exec /app/server
        ;;
    migrate)
        : "${DATABASE_URL:?DATABASE_URL is required}"
        exec goose -dir /app/db/migrations postgres "$DATABASE_URL" up
        ;;
    *)
        echo "Usage: $0 [serve|migrate]" >&2
        exit 1
        ;;
esac
