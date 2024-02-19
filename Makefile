.PHONY: all build up update down start-db status logs psql init-db

all:
	go test -v ./...

.env:
	@cp example.env .env
	@echo "Please edit .env to suit your environment before proceeding"
	@exit 1

build: .env
	@docker compose build

up: .env
	@docker compose up -d --build

update: up

down:
	@docker compose down

start-db:
	@docker compose up -d postgres

status:
	@docker compose stats

logs:
	@docker compose logs -f -n 50 lister consumer record-indexer

psql:
	@docker compose exec -it postgres psql -U postgres -d bluesky

init-db: init.sql
	@docker compose up -d --build lister
	@sleep 10
	@docker compose stop lister
	@cat init.sql | docker exec -i "$$(docker compose ps --format '{{.Names}}' postgres)" psql -U postgres -d bluesky
