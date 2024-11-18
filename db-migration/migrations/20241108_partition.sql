alter table records rename to records_old;

create table records
(like records_old including defaults)
partition by hash (repo);

alter sequence records_id_seq owned by records.id;

do $$
begin
for i in 0..15 loop
   execute 'create table records_' || i || ' partition of records for values with (modulus 16, remainder ' || i || ')';
end loop;
end $$;

with moved_rows as (
        delete from records_old r
        returning r.*
)
insert into records select * from moved_rows;
drop table records_old;
