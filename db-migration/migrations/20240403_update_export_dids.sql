
drop materialized view export_dids;

create materialized view export_dids
as select distinct did as "did:ID" from (
    select did from repos
      union
    select distinct ":END_ID" as did from export_follows
      union
    select distinct ":END_ID" as did from export_likes
      union
    select distinct ":END_ID" as did from export_replies
      union
    select distinct ":END_ID" as did from export_blocks
  )
with no data;
