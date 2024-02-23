alter table records detach partition records_default;

create table records_list
partition of records for values in ('app.bsky.graph.list');
create table records_listblock
partition of records for values in ('app.bsky.graph.listblock');
create table records_listitem
partition of records for values in ('app.bsky.graph.listitem');

ALTER TABLE records_list
   ADD CHECK (collection in ('app.bsky.graph.list'));

ALTER TABLE records_listblock
   ADD CHECK (collection in ('app.bsky.graph.listblock'));

ALTER TABLE records_listitem
   ADD CHECK (collection in ('app.bsky.graph.listitem'));

with moved_rows as (
        delete from records_default r
        where collection in ('app.bsky.graph.list', 'app.bsky.graph.listblock', 'app.bsky.graph.listitem')
        returning r.*
)
insert into records select * from moved_rows;

alter table records attach partition records_default default;

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
