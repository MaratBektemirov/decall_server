#!/usr/bin/env bash
set -euo pipefail

nginx -t && systemctl reload nginx

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${REPO_ROOT}/.env.prod"

if [[ -f "$ENV_FILE" ]] && command -v docker >/dev/null; then
  docker compose -f "${REPO_ROOT}/docker-compose.yml" --env-file "$ENV_FILE" restart coturn || true
fi
