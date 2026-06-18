#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deploy/docker-compose.yml}"

if [ "${1:-}" = "" ]; then
  echo "Usage: $0 'temporary-password-at-least-8-chars'" >&2
  exit 2
fi

docker compose -f "$COMPOSE_FILE" exec -T \
  -e OPSCORE_RESET_ADMIN_PASSWORD="$1" \
  backend /app/opscore-api reset-admin-password
