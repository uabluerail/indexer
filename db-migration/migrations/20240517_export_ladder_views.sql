drop materialized view export_dids_ladder;
drop materialized view export_replies_ladder;
drop materialized view export_likes_ladder;

create materialized view export_likes_ladder
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3) as ":END_ID",
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '30' DAY) * 10 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '60' DAY) * 5 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '90' DAY) * 3 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '360' DAY) * 1 as "count:long"
  from records join repos on records.repo = repos.id
  where records.collection = 'app.bsky.feed.like'
    and repos.did <> split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
  group by repos.did, split_part(jsonb_extract_path_text(content, 'subject',  'uri'), '/', 3)
with no data;
create index export_like_subject_ladder on export_likes_ladder (":END_ID");

create materialized view export_replies_ladder
as select repos.did as ":START_ID",
    split_part(jsonb_extract_path_text(content, 'reply', 'parent',  'uri'), '/', 3) as ":END_ID",
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '30' DAY) * 10 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '60' DAY) * 5 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '90' DAY) * 3 +
    count(*) FILTER (WHERE records.created_at > CURRENT_DATE - INTERVAL '360' DAY) * 1 as "count:long"
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

create index idx_records_created_at on records (created_at);