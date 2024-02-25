#!/bin/sh

set -e

# ------------------------------ Write data timestamp ----------------------------------

echo "export_start" > timestamp.csv
date -Iseconds --utc >> timestamp.csv

# ------------------------------ Refresh views ----------------------------------

docker compose exec -iT postgres psql -U postgres -d bluesky <<- EOF
\timing
\echo Refreshing follows...
refresh materialized view export_follows;
\echo Refreshing like counts...
refresh materialized view export_likes;
\echo Refreshing reply counts...
refresh materialized view export_replies;
\echo Refreshing DID list...
refresh materialized view export_dids;
EOF

# ------------------------------ Dump views into .csv ----------------------------------

echo "Writing .csv files..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_follows) to stdout with csv header;" > follows.csv
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_likes) to stdout with csv header;" > like_counts.csv
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_replies) to stdout with csv header;" > post_counts.csv
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (select * from export_dids) to stdout with csv header;" > dids.csv

# ------------------------------ Free up space used by materialized views ----------------------------------

docker compose exec -iT postgres psql -U postgres -d bluesky <<- EOF
\timing
refresh materialized view export_follows with no data;
refresh materialized view export_likes with no data;
refresh materialized view export_replies with no data;
refresh materialized view export_dids with no data;
EOF

# ------------------------------ Dump handles from plc-mirror ----------------------------------

docker exec -it plc-postgres-1 psql -U postgres -d plc \
  -c 'copy (select handle, did as "did:ID" from actors) to stdout with (format csv , header, force_quote ("handle"));' | sed -E -e 's/([^\\])\\",/\1\\\\",/g' > handles.csv
