#!/bin/sh
set -eu

MIN_PORT="${TURN_MIN_PORT:-49152}"
MAX_PORT="${TURN_MAX_PORT:-65535}"

set -- turnserver -n --log-file=stdout \
  --listening-port=3478 \
  --listening-ip=0.0.0.0 \
  --fingerprint \
  --use-auth-secret \
  --static-auth-secret="${TURN_SECRET}" \
  --realm="${TURN_REALM}" \
  --min-port="${MIN_PORT}" \
  --max-port="${MAX_PORT}"

if [ -n "${EXTERNAL_IP:-}" ]; then
  set -- "$@" \
    --external-ip="${EXTERNAL_IP}" \
    --relay-ip="${EXTERNAL_IP}"
fi

if [ "${TURN_TLS:-true}" = "true" ] && [ -f "/etc/letsencrypt/live/${SERVER_NAME}/fullchain.pem" ]; then
  set -- "$@" \
    --tls-listening-port=5349 \
    --cert="/etc/letsencrypt/live/${SERVER_NAME}/fullchain.pem" \
    --pkey="/etc/letsencrypt/live/${SERVER_NAME}/privkey.pem"
fi

exec "$@"
