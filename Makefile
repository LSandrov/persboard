.PHONY: init up down prepare-logs proto

COMPOSE ?= docker-compose

init:
	echo "== Инициализация окружения =="
	test -f .env.example || (echo "Error: .env.example not found" && exit 1)
	test -f .env || cp .env.example .env
	mkdir -p .docker/postgres-data
	mkdir -p .docker/logs
	$(MAKE) prepare-logs
	echo "Done."

prepare-logs:
	@mkdir -p .docker/logs
	@docker run --rm -v "$(PWD)/.docker/logs:/logs" alpine:3.20 \
		sh -c 'chown -R 1000:1000 /logs && chmod 0775 /logs'

up:
	# Workaround for docker-compose v1.29 + newer Docker API ("ContainerConfig" recreate error).
	-ids=$$(docker ps -aq --filter "name=persboard-backend" --filter "name=persboard-frontend"); \
	if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	$(MAKE) prepare-logs
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

proto:
	cd backend && buf dep update && buf generate

