.PHONY: all build up update down start-db status logs

all:
	go test -v ./...

.env: example.env
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

