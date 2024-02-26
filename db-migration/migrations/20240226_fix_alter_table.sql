DROP VIEW posts;
DROP VIEW lists;
DROP VIEW listitems;

ALTER TABLE "records" ALTER COLUMN "deleted" TYPE boolean USING "deleted"::boolean;

create view posts as
  select *, jsonb_extract_path(content, 'langs') as langs,
    parse_timestamp(jsonb_extract_path_text(content, 'createdAt')) as content_created_at
    from records_post;

create view lists as
  select records_list.*,
    jsonb_extract_path_text(content, 'name') as name,
    jsonb_extract_path_text(content, 'description') as description,
    jsonb_extract_path_text(content, 'purpose') as purpose,
    'at://' || repos.did || '/app.bsky.graph.list/' || rkey as uri
  from records_list join repos on records_list.repo = repos.id;

create view listitems as
  select *, jsonb_extract_path_text(content, 'list') as list,
    jsonb_extract_path_text(content, 'subject') as subject
    from records_listitem;