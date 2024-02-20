#!/bin/sh

set -e

# ------------------------------ FOLLOWS ----------------------------------

follows_query="$(cat <<- EOF
  select repos.did as ":START_ID", records.content ->> 'subject' as ":END_ID" from repos join records on repos.id = records.repo where records.collection = 'app.bsky.graph.follow' and records.content ->> 'subject' <> repos.did
EOF
)"

echo "Dumping follows..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${follows_query}) to stdout with csv header;" > follows.csv
echo "Done: $(ls -lh follows.csv)"

# ------------------------------ FOLLOWS ----------------------------------

# ------------------------------ LIKES ----------------------------------

likes_query="$(cat <<- EOF
      select repos.did as ":START_ID", subject_did as ":END_ID", "count:long" from
        repos join lateral (
          select repo, split_part(content['subject'] ->> 'uri', '/', 3) as subject_did, count(*) as "count:long" from records where repos.id = records.repo AND records.collection = 'app.bsky.feed.like' group by repo, split_part(content['subject'] ->> 'uri', '/', 3)
        ) as r on repos.id = r.repo
      where repos.did <> subject_did
EOF
)"

echo "Dumping likes..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${likes_query}) to stdout with csv header;" > like_counts.csv
echo "Done: $(ls -lh like_counts.csv)"

posts_query="$(cat <<- EOF
      select repos.did as ":START_ID", subject_did as ":END_ID", "count:long" from
        repos join lateral (
          select repo, split_part(content['reply']['parent'] ->> 'uri', '/', 3) as subject_did, count(*) as "count:long" from records where repos.id = records.repo AND records.collection = 'app.bsky.feed.post' group by repo, split_part(content['reply']['parent'] ->> 'uri', '/', 3)
        ) as r on repos.id = r.repo
      where repos.did <> subject_did
EOF
)"

# ------------------------------ LIKES ----------------------------------

# ------------------------------ REPLIES ----------------------------------

echo "Dumping posts..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${posts_query}) to stdout with csv header;" > post_counts.csv
echo "Done: $(ls -lh post_counts.csv)"

# ------------------------------ REPLIES ----------------------------------

# ------------------------------ HANDLES ----------------------------------

dids_query="$(cat <<- EOF
insert into repos (did)
select distinct did from (
  select distinct (split_part(jsonb_extract_path_text(content, 'reply', 'parent', 'uri'), '/', 3)) as did from records_post where collection='app.bsky.feed.post'
  union
  select distinct (split_part(jsonb_extract_path_text(content, 'subject', 'uri'), '/', 3)) from records where collection='app.bsky.feed.like'
  union
  select distinct (jsonb_extract_path_text(content, 'subject')) from records where collection='app.bsky.graph.follow'
)
on conflict (did) do nothing;
copy (select did as "did:ID" from repos) to stdout with csv header;
EOF
)"

echo "Dumping DIDs..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${posts_query}) to stdout with csv header;" > dids.csv
echo "Done: $(ls -lh dids.csv)"

docker exec -it plc-postgres-1 psql -U postgres -d plc \
  -c 'copy (select handle, did as "did:ID" from actors) to stdout with (format csv , header, force_quote ("handle"));' > handles.csv

# ------------------------------ HANDLES ----------------------------------