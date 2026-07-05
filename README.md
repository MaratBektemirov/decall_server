<p align="center">
  <img src="docs/logo.svg" alt="Decall" width="160">
</p>

# decall_server

Backend for [Decall](https://github.com/MaratBektemirov/decall_server) — decentralized calls with public-key authentication.

Go API with no database. In production: Docker API behind **nginx** (proxies `/api` only; no static client hosting).

Production env: `.env.prod` · local dev: `.env.dev`.

## Layout

```text
cmd/server/main.go
internal/auth/          # SecretAuth challenge
internal/signal/        # WebRTC signaling hub
scripts/nginx/          # prod edge (HTTPS + /api → localhost)
docker-compose.dev.yml  # dev: Air hot reload
docker-compose.yml      # prod: API container
```

## Local development

Create `.env.dev` (or use the one in the repo):

```bash
API_HOST_PORT=8080
CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
AUTH_DOMAIN=localhost
CHALLENGE_TTL_SEC=300
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

## Production

### Firewall (UFW)

On the VPS, allow only what the stack needs. WebRTC **signaling** is served by the Go API and proxied by nginx over **HTTPS/WSS** (port 443). **Media** (RTP, `RTCDataChannel`) goes **peer-to-peer** between clients — this server does not relay audio/video, so no extra UDP/TCP ports are required for WebRTC today.

```bash
# SSH first — do not lock yourself out
sudo ufw allow OpenSSH

# nginx: ACME + HTTPS API (signaling WebSocket is wss://…/api/signal)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw enable
sudo ufw status verbose
```

Keep the Go API off the public internet: bind Docker to localhost only (`127.0.0.1:${API_HOST_PORT:-8080}:8080` in `docker-compose.yml`) and **do not** `ufw allow` the API port. nginx listens on **80/443** and proxies `/api` → `127.0.0.1:${API_PORT}` (default **8080**, Go container).

| Port | Protocol | Purpose |
|------|----------|---------|
| 22 | tcp | SSH (or your custom SSH port) |
| 80 | tcp | nginx: ACME + redirect to HTTPS |
| 443 | tcp | nginx: HTTPS + WSS (`/api/signal` → container) |
| 8080 | tcp | Go API on localhost only (Docker → host) |

**Later (TURN in Go):** if you add a TURN relay on this host for NAT traversal, also open the TURN listener (commonly `3478/udp` and `3478/tcp`) and a UDP relay range (e.g. `49152:65535/udp`). Tune the range to match your TURN config.

### 1. API in Docker

Create `.env.prod` on the VPS:

```bash
SERVER_NAME=api.decall.example
CERTBOT_EMAIL=you@example.com
API_PORT=8080
API_HOST_PORT=8080
AUTH_DOMAIN=api.decall.example
CORS_ORIGINS=https://decall.example
CHALLENGE_TTL_SEC=300
```

```bash
make docker-prod-up
make docker-prod-down
```

API container on `127.0.0.1:8080`; nginx on **80/443** proxies `https://$SERVER_NAME/api/*` to it.

### 2. nginx + Let's Encrypt

On the VPS:

- DNS `A`/`AAAA` for `SERVER_NAME` points to the server
- UFW allows 80 and 443 (see [Firewall](#firewall-ufw))
- API is running (`make docker-prod-up`)

```bash
sudo make nginx
```

The script installs nginx and certbot, serves HTTP (ACME + `/api`), obtains a certificate, then switches to HTTPS (redirect 80 → 443).

```bash
curl https://api.decall.example/api/health
```

Re-apply nginx config without certbot (e.g. after port changes):

```bash
sudo make nginx-apply
```

| `.env.prod` | Description |
|-------------|-------------|
| `SERVER_NAME` | API hostname, e.g. `api.decall.example` |
| `CERTBOT_EMAIL` | Let's Encrypt contact email |
| `API_PORT` | Go API on localhost — nginx upstream (default `8080`) |
| `API_HOST_PORT` | Host port mapped by Docker (`8080` → container `8080`) |
| `CORS_ORIGINS` | Allowed browser origins (comma-separated) |

Client static assets (`decall_client`) are **not** served here — host them separately.

## API

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /auth/challenge` | SecretAuth challenge (`domain`, `nonce`, `exp`) |
| `WS /signal` | WebRTC signaling (`join`, `offer`, `answer`, `ice`) |

Chat messages travel P2P over `RTCDataChannel`; the server only relays signaling.

**Client:** [decall_client](https://github.com/MaratBektemirov/decall_client) · auth via [cruzo-web3](https://www.npmjs.com/package/cruzo-web3).

**Planned:** proof verification, voice calls.
