#!/bin/bash
source .env

set -e

# ------------------------------ Write data timestamp ----------------------------------

date=$(date -Idate --utc)

mkdir -p ${CSV_DIR}/monthly
mkdir -p ${CSV_DIR}/monthly/${date}

echo "Output directory: ${CSV_DIR}/monthly/${date}"

to_timestamp=$(date -Iseconds --utc)
echo "export_start" > ${CSV_DIR}/monthly/${date}/timestamp.csv
echo "${to_timestamp}" >> ${CSV_DIR}/monthly/${date}/timestamp.csv

# ------------------------------ Refresh views ----------------------------------

docker compose exec -iT postgres psql -U postgres -d bluesky <<- EOF
\timing
\echo Refreshing follows...
refresh materialized view export_follows_month;
\echo Refreshing like counts...
refresh materialized view export_likes_month;
\echo Refreshing reply counts...
refresh materialized view export_replies_month;
\echo Refreshing block list...
refresh materialized view export_blocks_month;
\echo Refreshing DID list...
refresh materialized view export_dids_month;
\echo Refreshing optout list...
refresh materialized view export_optouts;
EOF

# ------------------------------ Dump views into .csv ----------------------------------

echo "Writing .csv files..."

echo "Starting follows export..."
folows_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$folows_started', '$to_timestamp', 'app.bsky.graph.follow_month')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_follows_month) to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/follows.csv
echo "Finishing follows export..."
folows_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$folows_finished' where started='$folows_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.graph.follow_month'"

echo "Starting blocks export..."
block_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$block_started', '$to_timestamp', 'app.bsky.graph.block_month')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_blocks_month) to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/blocks.csv
echo "Finishing blocks export..."
block_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$block_finished' where started='$block_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.graph.block_month'"


echo "Starting likes export..."
likes_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$likes_started', '$to_timestamp', 'app.bsky.feed.like_month')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_likes_month) to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/like_counts.csv
echo "Finishing likes export..."
likes_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$likes_finished' where started='$likes_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.feed.like_month'"

echo "Starting posts export..."
posts_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$posts_started', '$to_timestamp', 'app.bsky.feed.post_month')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_replies_month) to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/post_counts.csv
echo "Finishing posts export..."
posts_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$posts_finished' where started='$posts_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.feed.post_month'"

echo "Starting dids export..."
dids_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$dids_started', '$to_timestamp', 'did_month')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_dids_month) to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/dids.csv
echo "Finishing dids export..."
dids_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$dids_finished' where started='$dids_started' and to_tsmp='$to_timestamp' and collection = 'did_month'"

echo "Starting optouts export..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select did from repos as r inner join records_block as rb on r.id=rb.repo where rb.content['subject']::text like '%did:plc:qevje4db3tazfbbialrlrkza%') to stdout with csv header;" > ${CSV_DIR}/monthly/${date}/optout.csv
echo "Finishing optouts export..."


# ------------------------------ DO NOT Free up space used by materialized views for incremental refresh ----------------------------------

# ------------------------------ Dump handles from plc-mirror ----------------------------------

echo "Starting handles export..."
handles_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$handles_started', '$to_timestamp', 'handle_month')"
docker exec -t plc-postgres-1 psql -U postgres -d plc \
  -c 'copy (select handle, did as "did:ID" from actors) to stdout with (format csv , header, force_quote ("handle"));' | sed -E -e 's/([^\\])\\",/\1\\\\",/g' > ${CSV_DIR}/monthly/${date}/handles.csv
echo "Finishing handles export..."
handles_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$handles_finished' where started='$handles_started' and to_tsmp='$to_timestamp' and collection = 'handle_month'"

