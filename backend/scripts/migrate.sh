#!/usr/bin/env bash
set -euo pipefail

MIGRATIONS_DIR="db/migrations"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

case "${1:-up}" in
    up)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" up
        ;;
    down)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" down
        ;;
    status)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" status
        ;;
    *)
        echo "Usage: $0 {up|down|status}"
        exit 1
        ;;
esac
