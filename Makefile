.PHONY: init up down

COMPOSE ?= docker-compose

init:
	echo "== Инициализация окружения =="
	test -f .env.example || (echo "Error: .env.example not found" && exit 1)
	test -f .env || cp .env.example .env
	mkdir -p .docker/postgres-data
	echo "Done."

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

