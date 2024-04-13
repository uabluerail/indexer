.PHONY: all build up update down start-db status logs psql init-db start-plc wait-for-plc

# ---------------------------- Docker ----------------------------

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

start-db: .env
	@docker compose up -d postgres

status:
	@docker compose stats

logs:
	@docker compose logs -f -n 50

start-plc: .env
	@docker compose up -d --build postgres plc

wait-for-plc:
	@. ./.env && while ! curl -s --fail-with-body http://$${METRICS_ADDR:-localhost}:11004/ready; do sleep 10; done

# ---------------------------- Docker ----------------------------



# ---------------------------- Database ----------------------------

psql:
	@docker compose exec -it postgres psql -U postgres -d bluesky

init-db: .env
	@docker compose up -d --build lister
	@sleep 10
	@docker compose stop lister
	@cat ./db-migration/init.sql | docker exec -i "$$(docker compose ps --format '{{.Names}}' postgres)" psql -U postgres -d bluesky

sqltop:
	watch -n 1 'cat top.sql|docker compose exec -i postgres psql -U postgres -d bluesky'

sqldu:
	cat du.sql | docker compose exec -iT postgres psql -U postgres -d bluesky

# ---------------------------- Database ----------------------------



# ---------------------------- CSV Export ----------------------------

# NOT RECOMMENDED TO RUN for the firts time on hot live db, will chomp all available IO. stop services first
csv-export:
	@docker compose up -d postgres
	@sleep 10
	@nohup ./csv_export.sh > csv_export.out &

csv-iexport:
	@docker compose up -d postgres
	@sleep 10
	@nohup ./csv_iexport.sh > csv_iexport.out &

kill-csv-export:
	@kill -9 `pgrep csv_export.sh`

kill-csv-iexport:
	@kill -9 `pgrep csv_iexport.sh`

# ---------------------------- CSV Export ----------------------------
