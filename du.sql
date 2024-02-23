SELECT
   relname  as table_name,
   pg_size_pretty(pg_total_relation_size(relid)) As "Total Size",
   pg_size_pretty(pg_indexes_size(relid)) as "Index Size",
   pg_size_pretty(pg_table_size(relid)) as "Actual Size"
   FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;

SELECT
   relname  as table_name,
   indexrelname as index_name,
   pg_size_pretty(pg_table_size(indexrelid)) as "Index Size"
   FROM pg_catalog.pg_statio_user_indexes
ORDER BY pg_table_size(indexrelid) DESC;

