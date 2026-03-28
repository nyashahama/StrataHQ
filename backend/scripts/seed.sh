#!/usr/bin/env bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

echo "Seeding database..."
# Add seed SQL commands here as the schema is built out.
# Example:
# psql "$DATABASE_URL" -f db/seed.sql
echo "No seed data configured yet. Add seed SQL to this script."
