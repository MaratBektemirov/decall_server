<p align="center">
  <img src="docs/logo.svg" alt="Decall" width="160">
</p>

# decall_server

Backend for [Decall](https://github.com/MaratBektemirov/decall_server) — decentralized calls with public-key authentication.

Go API with no database. In production: Docker API behind **nginx** (proxies `/api` only; no static client hosting).

Production env: `.env.prod` (see `.env.prod.example`).

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

```bash
# Docker + Air (reloads on .go changes)
make dev
make dev-down
```

API: `http://localhost:8080` · client: `http://localhost:5173` (proxies `/api`).

Optional: copy `.env.dev.example` → `.env.dev` (`CORS_ORIGINS`, `AUTH_DOMAIN`).

```bash
curl http://localhost:8080/health
curl http://localhost:8080/auth/challenge
```

WebSocket signaling: `ws://localhost:8080/signal` (via client proxy: `/api/signal`).

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `CORS_ORIGINS` | — | Allowed browser origins (comma-separated) |
| `AUTH_DOMAIN` | — | Domain embedded in auth challenges |
| `CHALLENGE_TTL_SEC` | `300` | Challenge lifetime in seconds |

## Production

### 1. API in Docker

```bash
cp .env.prod.example .env.prod
# edit SERVER_NAME, CERTBOT_EMAIL, API_PORT

make docker-prod-up
make docker-prod-down
```

API binds on the host at `127.0.0.1:${API_PORT}` (default `8080`).

### 2. nginx + Let's Encrypt

On the VPS:

- DNS `A`/`AAAA` for `SERVER_NAME` points to the server
- ports 80 and 443 are open
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
| `API_PORT` | API port on localhost (nginx upstream) |
| `API_HOST_PORT` | Host port mapped by Docker Compose |

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
