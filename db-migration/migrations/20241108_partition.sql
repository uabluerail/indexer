alter table records rename to records_old;

create table records
(like records_old including defaults)
partition by hash (repo);

do $$
begin
for i in 0..1023 loop
   execute 'create table records_' || i || ' partition of records for values with (modulus 1024, remainder ' || i || ')';
end loop;
end $$;

with moved_rows as (
        delete from records_old r
        returning r.*
)
insert into records select * from moved_rows;
