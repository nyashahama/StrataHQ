#!/usr/bin/env bash
set -euo pipefail

echo "Generating sqlc code..."
cd db && sqlc generate
echo "Done. Generated files in db/gen/"
