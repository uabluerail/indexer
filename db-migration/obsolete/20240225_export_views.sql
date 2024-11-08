-- Create a bunch of materialized views, but don't populate them right away.

create materialized view export_follows
as select repos.did as ":START_ID",
  records.content ->> 'subject' as ":END_ID"
  from repos join records on repos.id = records.repo
  where records.collection = 'app.bsky.graph.follow'
  and records.content ->> 'subject' <> repos.did
with no data;
create index export_follow_subject on export_follows (":END_ID");

-- Thanks to `join`, eats up 30GB+ of space while refreshing, but
-- finishes in under an hour.
create materialized view export_likes
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3) as ":END_ID",
    count(*) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.like'
    and repos.did <> split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
with no data;
create index export_like_subject on export_likes (":END_ID");

create materialized view export_replies
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3) as ":END_ID",
    count(*) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.post'
    and repos.did <> split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
with no data;
create index export_reply_subject on export_replies (":END_ID");

create materialized view export_dids
as select distinct did as "did:ID" from (
    select did from repos
      union
    select distinct ":END_ID" as did from export_follows
      union
    select distinct ":END_ID" as did from export_likes
      union
    select distinct ":END_ID" as did from export_replies
  )
with no data;
