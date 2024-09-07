\timing

CREATE EXTENSION pg_partman SCHEMA public;

alter table records rename to records_like;

create table records
(like records_like including defaults)
partition by list (collection);

drop index idx_repo_record_key;
drop index idx_repo_rev;
alter sequence records_id_seq owned by records.id;

drop table records_like;

create index on records (collection, repo, rkey);

CREATE OR REPLACE FUNCTION setup_partition(in collection text, in suffix text) RETURNS boolean AS $$
   BEGIN
      EXECUTE 'CREATE TABLE records_' || suffix ||
      ' PARTITION OF records FOR VALUES IN (' || quote_literal(collection) || ')
      PARTITION BY RANGE (created_at)';
      EXECUTE 'CREATE INDEX ON records_' || suffix || ' (created_at)';
      EXECUTE 'alter table records_' || suffix || ' add check (collection = ' || quote_literal(collection) || ')';

      PERFORM public.create_parent('public.records_' || suffix, 'created_at', '1 month',
        p_start_partition := '2024-02-01');
      RETURN true;
   END;
$$ LANGUAGE plpgsql;


select setup_partition('app.bsky.feed.like', 'like');
select setup_partition('app.bsky.feed.post', 'post');
select setup_partition('app.bsky.graph.follow', 'follow');
select setup_partition('app.bsky.graph.block', 'block');
select setup_partition('app.bsky.feed.repost', 'repost');
select setup_partition('app.bsky.actor.profile', 'profile');
select setup_partition('app.bsky.graph.list', 'list');
select setup_partition('app.bsky.graph.listblock', 'listblock');
select setup_partition('app.bsky.graph.listitem', 'listitem');


CREATE TABLE records_default
PARTITION OF records DEFAULT
PARTITION BY RANGE (created_at);
CREATE INDEX ON records_default (created_at);

SELECT public.create_parent('public.records_default', 'created_at', '1 month',
  p_start_partition := '2024-02-01');



create index idx_like_subject
on records_like
(split_part(jsonb_extract_path_text(content, 'subject', 'uri'), '/', 3));

create index idx_follow_subject
on records_follow
(jsonb_extract_path_text(content, 'subject'));

create index idx_reply_subject
on records_post
(split_part(jsonb_extract_path_text(content, 'reply', 'parent', 'uri'), '/', 3));

create index listitem_uri_subject
on records_listitem
(
  jsonb_extract_path_text(content, 'list'),
  jsonb_extract_path_text(content, 'subject'))
include (deleted);

create index listitem_subject_uri
on records_listitem
(
  jsonb_extract_path_text(content, 'subject'),
  jsonb_extract_path_text(content, 'list'))
include (deleted);

create view listitems as
  select *, jsonb_extract_path_text(content, 'list') as list,
    jsonb_extract_path_text(content, 'subject') as subject
    from records_listitem;


create view lists as
  select records_list.*,
    jsonb_extract_path_text(content, 'name') as name,
    jsonb_extract_path_text(content, 'description') as description,
    jsonb_extract_path_text(content, 'purpose') as purpose,
    'at://' || repos.did || '/app.bsky.graph.list/' || rkey as uri
  from records_list join repos on records_list.repo = repos.id;
