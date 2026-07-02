#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=scripts/nginx/common.sh
source "$(cd "$(dirname "$0")" && pwd)/common.sh"
require_root

echo "[*] decall edge bootstrap (prod)"
echo "    SERVER_NAME=$SERVER_NAME  API_PORT=$API_PORT  ACME_ROOT=$ACME_ROOT"

install_nginx
mkdir -p "$ACME_ROOT"

if ! has_cert && [[ "${SKIP_CERTBOT:-0}" != "1" ]]; then
  deploy_nginx_conf http
  reload_nginx
  run_certbot || true
fi

deploy_nginx_conf auto
reload_nginx

if has_cert; then
  echo "[+] edge online: https://${SERVER_NAME}/api"
else
  echo "[!] no TLS cert — HTTP only at http://${SERVER_NAME}/api"
  echo "    fix DNS + CERTBOT_EMAIL, then: sudo make nginx"
fi
