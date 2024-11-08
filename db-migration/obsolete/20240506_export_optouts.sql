drop materialized view export_optouts;

create materialized view export_optouts
as select did as "did:ID" from repos as r inner join records_block as rb on r.id=rb.repo where rb.content['subject']::text like '%did:plc:qevje4db3tazfbbialrlrkza%'
with no data;
