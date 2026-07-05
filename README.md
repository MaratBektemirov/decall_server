<p align="center">
  <img src="docs/logo.svg" alt="Decall" width="160">
</p>

# decall_server

Backend for [Decall](https://github.com/MaratBektemirov/decall_server) — decentralized calls with public-key authentication.

Go API with no database. In production: Docker API + **coturn** behind **nginx** (proxies `/api` only; no static client hosting).

Production env: `.env.prod` · local dev: `.env.dev`.

## Layout

```text
cmd/server/main.go
internal/auth/          # SecretAuth challenge + proof verify
internal/turn/          # TURN credentials (iceServers)
internal/signal/        # WebRTC signaling hub
coturn/                 # coturn entrypoint
scripts/nginx/          # prod edge (HTTPS + /api → localhost)
docker-compose.dev.yml  # dev: API + coturn
docker-compose.yml      # prod: API + coturn (host network)
```

## Local development

Create `.env.dev`:

```bash
API_HOST_PORT=8080
CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
AUTH_DOMAIN=localhost
CHALLENGE_TTL_SEC=300
TURN_SECRET=dev-turn-secret
TURN_HOST=localhost
TURN_REALM=localhost
TURN_TLS=false
```

```bash
make dev
make dev-down
```

API: `http://localhost:8080` · client: `http://localhost:5173` (proxies `/api`).

```bash
curl http://localhost:8080/health
curl http://localhost:8080/auth/challenge
```

WebSocket signaling: `ws://localhost:8080/signal` (via client proxy: `/api/signal`).

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP listen address (inside container) |
| `API_HOST_PORT` | `8080` | Host port mapped by Docker Compose (dev) |
| `CORS_ORIGINS` | — | Allowed browser origins (comma-separated) |
| `AUTH_DOMAIN` | — | Domain embedded in auth challenges |
| `CHALLENGE_TTL_SEC` | `300` | Challenge lifetime in seconds |
| `TURN_SECRET` | — | Shared secret for coturn + credential HMAC |
| `TURN_HOST` | — | Hostname clients use in `iceServers` |
| `TURN_REALM` | `TURN_HOST` | coturn realm |
| `TURN_CREDENTIAL_TTL_SEC` | `86400` | TURN username expiry |
| `TURN_TLS` | `true` (prod) | Enable TURNS on port 5349 when certs exist |

## Production

### 1. DNS

Point `server-01.decall.app` (or your API hostname) to the VPS public IP:

```text
A    server-01.decall.app  →  YOUR_VPS_IP
```

### 2. Firewall (UFW)

```bash
# SSH first — do not lock yourself out
sudo ufw allow OpenSSH

# nginx: ACME + HTTPS API (signaling WebSocket is wss://…/api/signal)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# coturn: STUN/TURN + relay media when P2P fails
sudo ufw allow 3478/tcp
sudo ufw allow 3478/udp
sudo ufw allow 5349/tcp
sudo ufw allow 49152:65535/udp

sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw enable
sudo ufw status verbose
```

Keep the Go API off the public internet: bind Docker to localhost only (`127.0.0.1:8080`) and **do not** `ufw allow` port 8080.

| Port | Protocol | Purpose |
|------|----------|---------|
| 22 | tcp | SSH |
| 80 | tcp | nginx: ACME + redirect to HTTPS |
| 443 | tcp | nginx: HTTPS + WSS (`/api/signal` → Go API) |
| 3478 | tcp, udp | coturn: STUN/TURN |
| 5349 | tcp | coturn: TURNS (TLS) |
| 49152–65535 | udp | coturn: relay ports |
| 8080 | tcp | Go API on localhost only (Docker → host) |

### 3. `.env.prod` on the VPS

Copy the template and fill in secrets:

```bash
cp .env.prod.example .env.prod
```

```bash
SERVER_NAME=server-01.decall.app
CERTBOT_EMAIL=you@example.com
API_PORT=8080
API_HOST_PORT=8080
AUTH_DOMAIN=server-01.decall.app
CORS_ORIGINS=https://decall.app
CHALLENGE_TTL_SEC=300

# TURN — generate secret: openssl rand -hex 32
TURN_SECRET=your-64-char-hex-secret
TURN_HOST=server-01.decall.app
TURN_REALM=server-01.decall.app
EXTERNAL_IP=YOUR_VPS_PUBLIC_IP
TURN_TLS=true
TURN_CREDENTIAL_TTL_SEC=86400
```

`AUTH_DOMAIN` and `TURN_HOST` are hostnames only (no `https://`). `TURN_SECRET` must match between the Go API and coturn (same value in `.env.prod`).

### 4. Deploy API + coturn

On the VPS, from the repo directory:

```bash
git pull
make docker-prod-up
```

Check containers:

```bash
docker compose --env-file .env.prod ps
make docker-prod-logs
```

### 5. nginx + Let's Encrypt

Requires DNS, UFW (80/443), and running API:

```bash
sudo make nginx
```

Re-apply nginx without certbot (e.g. after port changes):

```bash
sudo make nginx-apply
```

Verify:

```bash
curl https://server-01.decall.app/api/health
curl https://server-01.decall.app/api/auth/challenge
```

### 6. Full deploy (API + nginx config)

```bash
git pull
make deploy
```

This runs `docker-prod-up` then `nginx-apply`.

### 7. After certificate renewal

coturn reads TLS certs from `/etc/letsencrypt`. Restart coturn after certbot renews:

```bash
docker compose --env-file .env.prod restart coturn
```

Or add to `scripts/nginx/certbot-deploy-hook.sh` if you automate renewals.

### Troubleshooting TURN

```bash
# coturn listening?
sudo ss -ulnp | grep 3478
sudo ss -tlnp | grep 5349

# relay UDP range
sudo ss -ulnp | grep turnserver

# test from client: chrome://webrtc-internals → look for typ relay candidates
```

## API

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /auth/challenge` | SecretAuth challenge (`domain`, `nonce`, `exp`) |
| `POST /turn-credentials` | TURN `iceServers` (body: `{ "proof": SecretAuthProof }`) |
| `WS /signal` | WebRTC signaling (`join`, `offer`, `answer`, `ice`) |

Chat messages travel P2P over `RTCDataChannel`; the server relays signaling. Media uses P2P when possible, otherwise **coturn**.

**Client:** [decall_client](https://github.com/MaratBektemirov/decall_client) · auth via [cruzo-web3](https://www.npmjs.com/package/cruzo-web3).

**Planned:** proof verification on signaling, voice calls polish.
