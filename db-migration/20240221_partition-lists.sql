alter table records detach partition records_default;

create table records_list
partition of records for values in ('app.bsky.graph.list');
create table records_listblock
partition of records for values in ('app.bsky.graph.listblock');
create table records_listitem
partition of records for values in ('app.bsky.graph.listitem');

with moved_rows as (
        delete from records_default r
        where collection in ('app.bsky.graph.list', 'app.bsky.graph.listblock', 'app.bsky.graph.listitem')
        returning r.*
)
insert into records select * from moved_rows;

alter table records attach partition records_default default;
