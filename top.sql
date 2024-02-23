SELECT pid, age(clock_timestamp(), query_start), state, query
FROM pg_stat_activity
WHERE query != '<IDLE>' AND query NOT ILIKE '%pg_stat_activity%' AND state <> 'idle'
ORDER BY query_start asc;
