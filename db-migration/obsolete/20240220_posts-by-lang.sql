create index post_langs on records_post using gin (jsonb_extract_path(content, 'langs') jsonb_ops);

-- There are invalid/non-conforming values that need to be handled somehow.
create function parse_timestamp(text)
  returns timestamp
  returns null on null input
  immutable
  as
  $$
  begin
    begin
      return $1::timestamp;
    exception
      when others then
        return null;
    end;
  end;
  $$
  language plpgsql;

create index post_created_at on records_post (parse_timestamp(jsonb_extract_path_text(content, 'createdAt')));

create view posts as
  select *, jsonb_extract_path(content, 'langs') as langs,
    parse_timestamp(jsonb_extract_path_text(content, 'createdAt')) as content_created_at
    from records_post;

explain select count(*) from posts where langs ? 'uk' and content_created_at > now() - interval '1 day';
