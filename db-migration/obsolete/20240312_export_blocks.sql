-- Create a block materialized view, don't populate right away.

create materialized view export_blocks
as select repos.did as ":START_ID",
  records.content ->> 'subject' as ":END_ID"
  from repos join records on repos.id = records.repo
  where records.collection = 'app.bsky.graph.block'
  and records.content ->> 'subject' <> repos.did
with no data;
create index export_block_subject on export_blocks (":END_ID");