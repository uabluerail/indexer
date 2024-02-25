-- Only affects future writes
alter table records_like alter content set compression lz4;
alter table records alter content set compression lz4;
alter table records_default alter content set compression lz4;
alter table records_post alter content set compression lz4;
alter table records_follow alter content set compression lz4;
alter table records_block alter content set compression lz4;
alter table records_repost alter content set compression lz4;
alter table records_profile alter content set compression lz4;
alter table records_list alter content set compression lz4;
alter table records_listblock alter content set compression lz4;
alter table records_listitem alter content set compression lz4;
