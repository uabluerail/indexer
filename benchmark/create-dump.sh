#!/bin/sh

cd ..

docker compose exec -i postgres pg_dump -U postgres -d bluesky -t records -t records_id_seq --schema-only | sed -E -e 's/PARTITION BY.*/;/' > records.sql
docker compose exec -i postgres pg_dump -U postgres -d bluesky --table-and-children records --load-via-partition-root --data-only | lz4 > records.sql.lz4
