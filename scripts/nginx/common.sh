#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env.prod}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

SERVER_NAME="${SERVER_NAME:-api.decall.example}"
API_PORT="${API_PORT:-80}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
ACME_ROOT="${ACME_ROOT:-/var/www/certbot}"

NGINX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SITES_AVAILABLE="/etc/nginx/sites-available"
SITES_ENABLED="/etc/nginx/sites-enabled"
SITE_FILE="${SERVER_NAME}"

require_root() {
  if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
    echo "[!] root required — run with sudo" >&2
    exit 1
  fi
}

install_nginx() {
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y nginx certbot
}

ensure_ssl_params() {
  local ssl_dir=/etc/nginx/snippets/decall
  mkdir -p "$ssl_dir"
  install -m644 "${NGINX_DIR}/conf/options-ssl-nginx.conf" "${ssl_dir}/options-ssl-nginx.conf"
  if [[ ! -f "${ssl_dir}/ssl-dhparams.pem" ]]; then
    echo "[*] generating dhparam (once, ~1 min)..."
    openssl dhparam -out "${ssl_dir}/ssl-dhparams.pem" 2048
  fi
}

render_conf() {
  local src="$1" dest="$2"
  sed \
    -e "s|__SERVER_NAME__|${SERVER_NAME}|g" \
    -e "s|__API_PORT__|${API_PORT}|g" \
    -e "s|__ACME_ROOT__|${ACME_ROOT}|g" \
    "$src" >"$dest"
}

enable_site() {
  ln -sf "${SITES_AVAILABLE}/${SITE_FILE}" "${SITES_ENABLED}/${SITE_FILE}"
}

reload_nginx() {
  nginx -t
  systemctl reload nginx
}

cert_path() {
  echo "/etc/letsencrypt/live/${SERVER_NAME}/fullchain.pem"
}

has_cert() {
  [[ -f "$(cert_path)" ]]
}

deploy_nginx_conf() {
  local mode="${1:-auto}"

  rm -f /etc/nginx/sites-enabled/default
  rm -f "${SITES_ENABLED}/${SITE_FILE}" "${SITES_AVAILABLE}/${SITE_FILE}"

  if [[ "$mode" == "http" ]] || { [[ "$mode" == "auto" ]] && ! has_cert; }; then
    render_conf "${NGINX_DIR}/conf/site.http.conf" "${SITES_AVAILABLE}/${SITE_FILE}"
    echo "[+] HTTP config deployed (API + ACME webroot)"
  else
    ensure_ssl_params
    render_conf "${NGINX_DIR}/conf/site.conf" "${SITES_AVAILABLE}/${SITE_FILE}"
    echo "[+] TLS config deployed"
  fi

  enable_site
}

run_certbot() {
  if [[ "${SKIP_CERTBOT:-0}" == "1" ]]; then
    return 0
  fi
  if [[ -z "$CERTBOT_EMAIL" ]]; then
    echo "[!] CERTBOT_EMAIL required for Let's Encrypt" >&2
    return 1
  fi

  mkdir -p "$ACME_ROOT"

  install -d /etc/letsencrypt/renewal-hooks/deploy
  install -m755 "${NGINX_DIR}/certbot-deploy-hook.sh" /etc/letsencrypt/renewal-hooks/deploy/decall-nginx.sh

  if has_cert; then
    echo "[*] cert present, renew if due"
    certbot renew --quiet || true
    return 0
  fi

  echo "[*] issuing cert via HTTP-01 (webroot): ${SERVER_NAME}"
  certbot certonly --webroot \
    -w "$ACME_ROOT" \
    -d "$SERVER_NAME" \
    --email "$CERTBOT_EMAIL" \
    --agree-tos \
    --non-interactive \
    --keep-until-expiring
}
