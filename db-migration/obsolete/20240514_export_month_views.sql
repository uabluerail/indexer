-- Create a bunch of materialized views, but don't populate them right away.

create materialized view export_follows_month
as select repos.did as ":START_ID",
  records.content ->> 'subject' as ":END_ID"
  from repos join records on repos.id = records.repo
  where records.collection = 'app.bsky.graph.follow'
  and records.content ->> 'subject' <> repos.did
  and records.created_at > CURRENT_DATE - INTERVAL '30' DAY
with no data;
create index export_follow_subject_month on export_follows_month (":END_ID");

-- Thanks to `join`, eats up 30GB+ of space while refreshing, but
-- finishes in under an hour.
create materialized view export_likes_month
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3) as ":END_ID",
    count(*) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.like'
    and records.created_at > CURRENT_DATE - INTERVAL '30' DAY
    and repos.did <> split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
with no data;
create index export_like_subject_month on export_likes_month (":END_ID");

create materialized view export_replies_month
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3) as ":END_ID",
    count(*) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.post'
    and records.created_at > CURRENT_DATE - INTERVAL '30' DAY
    and repos.did <> split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
with no data;
create index export_reply_subject_month on export_replies_month (":END_ID");

create materialized view export_blocks_month
as select repos.did as ":START_ID",
  records.content ->> 'subject' as ":END_ID"
  from repos join records on repos.id = records.repo
  where records.collection = 'app.bsky.graph.block'
  and records.created_at > CURRENT_DATE - INTERVAL '30' DAY
  and records.content ->> 'subject' <> repos.did
with no data;
create index export_block_subject_month on export_blocks_month (":END_ID");


create materialized view export_dids_month
as select distinct did as "did:ID" from (
    select did from repos
      union
    select distinct ":END_ID" as did from export_follows_month
      union
    select distinct ":END_ID" as did from export_likes_month
      union
    select distinct ":END_ID" as did from export_replies_month
      union
    select distinct ":END_ID" as did from export_blocks_month
  )
with no data;