# Graceful shutdown/restart

`docker compose stop lister`
`docker compose stop consumer`
`docker compose stop record-indexer`

Take a look at grafana, once all quiet

`docker compose stop postgres`

Start everything up

`docker compose up -d --build`

# Control number of workers

Full throttle
`curl 'localhost:11003/pool/resize?size=50'`

Half throttle (recommended)
`curl 'localhost:11003/pool/resize?size=25'`

Stop eating all of my Internet
`curl 'localhost:11003/pool/resize?size=10'`

# Peak into db

`docker compose exec -it postgres psql -U postgres -d bluesky`

Seen repos
`select count(*) from repos;`

Fully indexed repos
`select count(*) from repos where last_indexed_rev <> '' and (last_indexed_rev >= first_rev_since_reset or first_rev_since_reset is null or first_rev_since_reset = '');`

Get list blocks

non-partitioned (very slow)

```
select count(*) from (select distinct repo from records where collection in ('app.bsky.graph.listblock') and deleted=false and content['subject']::text like '"at://did:plc:bmjomljebcsuxolnygfgqtap/%');
```

partitioned (slow)
`select count(*) from (select distinct repo from records_listblock where deleted=false and content['subject']::text like '"at:///%');`

`select count(*) from (select distinct repo from records_listblock where deleted=false and (split_part(jsonb_extract_path_text(content, 'subject'), '/', 3))='did:plc:bmjomljebcsuxolnygfgqtap');`

Count all records

`analyze records; select relname, reltuples::int from pg_class where relname like 'records';`

View errors

`select last_error, count(*) from repos where failed_attempts > 0 group by last_error;`

Restart errors

`update repos set failed_attempts=0, last_error='' where failed_attempts >0;`

# MONITORING

More verbose logging for queries DEBUG1-DEBUG5
`set client_min_messages = 'DEBUG5';`

Take a look at slow queries
```
SELECT pid, age(clock_timestamp(), query_start), state, query
FROM pg_stat_activity
WHERE query != '<IDLE>' AND query NOT ILIKE '%pg_stat_activity%'
ORDER BY query_start asc;
```

Monitor index progress
`select * from pg_stat_progress_create_index;`

Explore new collection types

```
select * from records where collection not in (
    'app.bsky.actor.profile',
    'app.bsky.feed.generator',
    'app.bsky.feed.like',
    'app.bsky.feed.post',
    'app.bsky.feed.repost',
    'app.bsky.feed.threadgate',
    'app.bsky.graph.block',
    'app.bsky.graph.follow',
    'app.bsky.graph.listitem',
    'app.bsky.graph.list',
    'app.bsky.graph.listblock'
    ) limit 20;

```

count listitems
`select count(*) from listitems where list='at://did:plc:2yqylcqgxier4l5uplp6w6jh/app.bsky.graph.list/3kkud7l6s4v2m';`