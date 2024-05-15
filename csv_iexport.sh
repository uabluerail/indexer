#!/bin/bash
source .env

set -e

# ------------------------------ Write data timestamp ----------------------------------

date=$(date -Idate --utc)

mkdir -p ${CSV_DIR}/full
mkdir -p ${CSV_DIR}/full/${date}

echo "Output directory: ${CSV_DIR}/full/${date}"

to_timestamp=$(date -Iseconds --utc)

echo "export_start" > ${CSV_DIR}/full/${date}/timestamp.csv
echo "${to_timestamp}" >> ${CSV_DIR}/full/${date}/timestamp.csv

# ------------------------------ Refresh views ----------------------------------

docker compose exec -iT postgres psql -U postgres -d bluesky <<- EOF
\timing
\echo Refreshing follows...
refresh materialized view export_follows;
\echo Refreshing like counts...
refresh materialized view export_likes_ladder;
\echo Refreshing reply counts...
refresh materialized view export_replies_ladder;
\echo Refreshing block list...
refresh materialized view export_blocks;
\echo Refreshing DID list...
refresh materialized view export_dids_ladder;
\echo Refreshing optout list...
refresh materialized view export_optouts;
EOF

# ------------------------------ Dump views into .csv ----------------------------------

echo "Writing .csv files..."

echo "Starting follows export..."
folows_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$folows_started', '$to_timestamp', 'app.bsky.graph.follow')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_follows) to stdout with csv header;" > ${CSV_DIR}/full/${date}/follows.csv
echo "Finishing follows export..."
folows_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$folows_finished' where started='$folows_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.graph.follow'"

echo "Starting blocks export..."
block_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$block_started', '$to_timestamp', 'app.bsky.graph.block')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_blocks) to stdout with csv header;" > ${CSV_DIR}/full/${date}/blocks.csv
echo "Finishing blocks export..."
block_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$block_finished' where started='$block_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.graph.block'"


echo "Starting likes export..."
likes_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$likes_started', '$to_timestamp', 'app.bsky.feed.like')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_likes_ladder) to stdout with csv header;" > ${CSV_DIR}/full/${date}/like_counts.csv
echo "Finishing likes export..."
likes_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$likes_finished' where started='$likes_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.feed.like'"

echo "Starting posts export..."
posts_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$posts_started', '$to_timestamp', 'app.bsky.feed.post')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_replies_ladder) to stdout with csv header;" > ${CSV_DIR}/full/${date}/post_counts.csv
echo "Finishing posts export..."
posts_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$posts_finished' where started='$posts_started' and to_tsmp='$to_timestamp' and collection = 'app.bsky.feed.post'"

echo "Starting dids export..."
dids_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$dids_started', '$to_timestamp', 'did')"
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_dids_ladder) to stdout with csv header;" > ${CSV_DIR}/full/${date}/dids.csv
echo "Finishing dids export..."
dids_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$dids_finished' where started='$dids_started' and to_tsmp='$to_timestamp' and collection = 'did'"

echo "Starting optouts export..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select did as "did:ID" from repos as r inner join records_block as rb on r.id=rb.repo where rb.content['subject']::text like '%did:plc:qevje4db3tazfbbialrlrkza%') to stdout with csv header;" > ${CSV_DIR}/full/${date}/optout.csv
echo "Finishing optouts export..."


# ------------------------------ DO NOT Free up space used by materialized views for incremental refresh ----------------------------------

# ------------------------------ Dump handles from plc-mirror ----------------------------------

echo "Starting handles export..."
handles_started=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "insert into incremental_export_log (started, to_tsmp, collection) values ('$handles_started', '$to_timestamp', 'handle')"
docker exec -t plc-postgres-1 psql -U postgres -d plc \
  -c 'copy (select handle, did as "did:ID" from actors) to stdout with (format csv , header, force_quote ("handle"));' | sed -E -e 's/([^\\])\\",/\1\\\\",/g' > ${CSV_DIR}/full/${date}/handles.csv
echo "Finishing handles export..."
handles_finished=$(date -Iseconds --utc)
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "update incremental_export_log set finished='$handles_finished' where started='$handles_started' and to_tsmp='$to_timestamp' and collection = 'handle'"

