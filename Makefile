.PHONY: dev dev-down build docker-prod-up docker-prod-down nginx nginx-apply

dev:
	@if [ -f .env.dev ]; then \
		docker-compose -f docker-compose.dev.yml --env-file .env.dev up --build; \
	else \
		docker-compose -f docker-compose.dev.yml up --build; \
	fi

dev-down:
	docker compose -f docker-compose.dev.yml down

build:
	go build -trimpath -o ./tmp/server ./cmd/server

docker-prod-up:
	@test -f .env.prod || (echo "Copy .env.prod.example to .env.prod"; exit 1)
	docker compose --env-file .env.prod up -d --build

docker-prod-down:
	docker compose --env-file .env.prod down

nginx:
	@test -f .env.prod || (echo "Copy .env.prod.example to .env.prod"; exit 1)
	sudo bash -c 'set -a && source "$1/.env.prod" && set +a && exec "$1/scripts/nginx/setup.sh"' _ "$(pwd)"

nginx-apply:
	@test -f .env.prod || (echo "Copy .env.prod.example to .env.prod"; exit 1)
	sudo SKIP_CERTBOT=1 bash -c 'set -a && source "$1/.env.prod" && set +a && exec "$1/scripts/nginx/setup.sh"' _ "$(pwd)"
