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
API_PORT="${API_PORT:-8080}"
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

ensure_nginx_dirs() {
  mkdir -p "$SITES_AVAILABLE" "$SITES_ENABLED" "$ACME_ROOT"
}

remove_default_site() {
  rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf
}

ensure_nginx_includes_sites() {
  local conf=/etc/nginx/nginx.conf

  if grep -q 'sites-enabled' "$conf"; then
    return 0
  fi

  local tmp
  tmp="$(mktemp)"
  awk '
    /^http \{/ { in_http=1 }
    in_http && /^}/ {
      print "    include /etc/nginx/sites-enabled/*;"
      in_http=0
    }
    { print }
  ' "$conf" >"$tmp"
  mv "$tmp" "$conf"
  echo "[+] nginx.conf: enabled sites-enabled include"
}

install_nginx() {
  ensure_nginx_dirs
  ensure_nginx_includes_sites
  remove_default_site

  if command -v nginx >/dev/null && command -v certbot >/dev/null; then
    echo "[*] nginx + certbot already installed"
    return 0
  fi

  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y nginx certbot
  ensure_nginx_includes_sites
  remove_default_site
}

ensure_ssl_params() {
  local ssl_dir=/etc/nginx/snippets/decall
  mkdir -p "$ssl_dir"
  install -m644 "${NGINX_DIR}/conf/options-ssl-nginx.conf" "${ssl_dir}/options-ssl-nginx.conf"
  install -m644 "${NGINX_DIR}/conf/upgrade-map.conf" /etc/nginx/conf.d/decall-upgrade-map.conf
  if [[ ! -f "${ssl_dir}/ssl-dhparams.pem" ]]; then
    echo "[*] generating dhparam (once, ~1 min)..."
    openssl dhparam -out "${ssl_dir}/ssl-dhparams.pem" 2048
  fi
}

ensure_proxy_snippets() {
  install -m644 "${NGINX_DIR}/conf/upgrade-map.conf" /etc/nginx/conf.d/decall-upgrade-map.conf
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
  if systemctl is-active --quiet nginx; then
    systemctl reload nginx
  else
    systemctl enable --now nginx
  fi
}

cert_path() {
  echo "/etc/letsencrypt/live/${SERVER_NAME}/fullchain.pem"
}

has_cert() {
  [[ -f "$(cert_path)" ]]
}

deploy_nginx_conf() {
  local mode="${1:-auto}"

  ensure_nginx_dirs
  ensure_nginx_includes_sites
  remove_default_site

  if [[ "$mode" == "http" ]] || { [[ "$mode" == "auto" ]] && ! has_cert; }; then
    ensure_proxy_snippets
    render_conf "${NGINX_DIR}/conf/site.http.conf" "${SITES_AVAILABLE}/${SITE_FILE}"
    echo "[+] HTTP config deployed (API + ACME webroot)"
  else
    ensure_ssl_params
    render_conf "${NGINX_DIR}/conf/site.conf" "${SITES_AVAILABLE}/${SITE_FILE}"
    echo "[+] TLS config deployed"
  fi

  enable_site
}

install_certbot_hook() {
  install -d /etc/letsencrypt/renewal-hooks/deploy
  install -m755 "${NGINX_DIR}/certbot-deploy-hook.sh" /etc/letsencrypt/renewal-hooks/deploy/decall-nginx.sh
}

run_certbot() {
  if [[ "${SKIP_CERTBOT:-0}" == "1" ]]; then
    return 0
  fi
  if [[ -z "$CERTBOT_EMAIL" ]]; then
    echo "[!] CERTBOT_EMAIL required for Let's Encrypt" >&2
    return 1
  fi

  install_certbot_hook

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
