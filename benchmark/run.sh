#!/bin/sh

output="psql_$(date '+%y%m%d_%H%M%S').log"

set -x

docker compose stop postgres

. ./.env
sudo rm -rf ${DATA_DIR:?DATA_DIR not set}/benchmark

echo "$(date): Starting data import..."

docker compose up -d postgres

while ! docker compose exec postgres psql -U postgres -d bluesky -c 'select 1;'; do sleep 1; done

cat ../records.sql | docker compose exec -iT postgres psql -U postgres -d bluesky
lz4cat ../records.sql.lz4 | docker compose exec -iT postgres psql -U postgres -d bluesky

echo "$(date): Data import done"

cat ../db-migration/20240217_partition.sql \
  | docker compose exec -iT postgres psql -U postgres -d bluesky --echo-queries -c '\timing' \
  | tee -a "${output}"
