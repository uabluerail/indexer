#!/bin/sh

set -e

follows_query="$(cat <<- EOF
  select repos.did as ":START_ID", records.content ->> 'subject' as ":END_ID" from repos join records on repos.id = records.repo where records.collection = 'app.bsky.graph.follow' and records.content ->> 'subject' <> repos.did
EOF
)"

echo "Dumping follows..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${follows_query}) to stdout with csv header;" > follows.csv
echo "Done: $(ls -lh follows.csv)"

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
  -c "copy (${likes_query}) to stdout with csv header;" > like_counts2.csv
echo "Done: $(ls -lh like_counts2.csv)"

posts_query="$(cat <<- EOF
      select repos.did as ":START_ID", subject_did as ":END_ID", "count:long" from
        repos join lateral (
          select repo, split_part(content['reply']['parent'] ->> 'uri', '/', 3) as subject_did, count(*) as "count:long" from records where repos.id = records.repo AND records.collection = 'app.bsky.feed.post' group by repo, split_part(content['reply']['parent'] ->> 'uri', '/', 3)
        ) as r on repos.id = r.repo
      where repos.did <> subject_did
EOF
)"

echo "Dumping posts..."
docker compose exec -it postgres psql -U postgres -d bluesky \
  -c "copy (${posts_query}) to stdout with csv header;" > post_counts.csv
echo "Done: $(ls -lh post_counts.csv)"
