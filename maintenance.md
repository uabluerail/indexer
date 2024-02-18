# Control number of workers

Full throttle
`curl 'localhost:11003/pool/resize?size=50'`

Half throttle
`curl 'localhost:11003/pool/resize?size=25'`

Stop eating all of my Internet
`curl 'localhost:11003/pool/resize?size=10'`

# Peak into db

`docker compose exec -it postgres psql -U postgres -d bluesky`

Seen repos
`select count(*) from repos;`

Fully indexed repos
`select count(*) from repos where last_indexed_rev <> '' and (last_indexed_rev >= first_rev_since_reset or first_rev_since_reset is null or first_rev_since_reset = '');`

```
SELECT pid, age(clock_timestamp(), query_start), state, query
FROM pg_stat_activity
WHERE query != '<IDLE>' AND query NOT ILIKE '%pg_stat_activity%'
ORDER BY query_start asc;
```