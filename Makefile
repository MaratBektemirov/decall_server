.PHONY: dev dev-down build docker-prod-up docker-prod-down nginx nginx-apply

REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

dev:
	@test -f .env.dev || (echo "Create .env.dev (see README)"; exit 1)
	docker compose -f docker-compose.dev.yml --env-file .env.dev up --build

dev-down:
	docker compose -f docker-compose.dev.yml down

build:
	go build -trimpath -o ./tmp/server ./cmd/server

docker-prod-up:
	@test -f .env.prod || (echo "Create .env.prod (see README)"; exit 1)
	docker compose -f docker-compose.yml --env-file .env.prod up -d --build

docker-prod-down:
	docker compose -f docker-compose.yml --env-file .env.prod down

nginx:
	@test -f .env.prod || (echo "Create .env.prod (see README)"; exit 1)
	sudo bash -c 'set -a && source "$$1/.env.prod" && set +a && exec "$$1/scripts/nginx/setup.sh"' _ "$(REPO_ROOT)"

nginx-apply:
	@test -f .env.prod || (echo "Create .env.prod (see README)"; exit 1)
	sudo SKIP_CERTBOT=1 bash -c 'set -a && source "$$1/.env.prod" && set +a && exec "$$1/scripts/nginx/setup.sh"' _ "$(REPO_ROOT)"
