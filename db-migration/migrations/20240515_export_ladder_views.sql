CREATE OR REPLACE FUNCTION ladderq(date TIMESTAMP) RETURNS integer AS $$
SELECT
CASE
  WHEN date > CURRENT_DATE - INTERVAL '30' DAY THEN 10
  WHEN date > CURRENT_DATE - INTERVAL '90' DAY THEN 5
  ELSE 1
END;
$$ LANGUAGE sql STRICT IMMUTABLE;

create materialized view export_likes_ladder
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3) as ":END_ID",
    sum(ladderq(records.created_at::TIMESTAMP)) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.like'
    and repos.did <> split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
with no data;
create index export_like_subject_ladder on export_likes_ladder (":END_ID");

create materialized view export_replies_ladder
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3) as ":END_ID",
    sum(ladderq(records.created_at::TIMESTAMP)) as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.post'
    and repos.did <> split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3)
with no data;
create index export_reply_subject_ladder on export_replies_ladder (":END_ID");

create materialized view export_dids_ladder
as select distinct did as "did:ID" from (
    select did from repos
      union
    select distinct ":END_ID" as did from export_follows
      union
    select distinct ":END_ID" as did from export_likes_ladder
      union
    select distinct ":END_ID" as did from export_replies_ladder
      union
    select distinct ":END_ID" as did from export_blocks
  )
with no data;