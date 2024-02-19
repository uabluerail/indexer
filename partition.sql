alter table records rename to records_like;

create table records
(like records_like including defaults)
partition by list (collection);

drop index idx_repo_record_key;
drop index idx_repo_rev;
alter table records_like drop constraint records_pkey;
create unique index records_pkey on records (id, collection);

create table records_default
partition of records default;

create table records_post
partition of records for values in ('app.bsky.feed.post');
create table records_follow
partition of records for values in ('app.bsky.graph.follow');
create table records_block
partition of records for values in ('app.bsky.graph.block');
create table records_repost
partition of records for values in ('app.bsky.feed.repost');
create table records_profile
partition of records for values in ('app.bsky.actor.profile');

with moved_rows as (
        delete from records_like r
        where collection <> 'app.bsky.feed.like'
        returning r.*
)
insert into records select * from moved_rows;

alter table records attach partition records_like for values in ('app.bsky.feed.like');
