CREATE TABLE incremental_export_log (
  id SERIAL PRIMARY KEY,
  collection text NOT NULL,
  to_tsmp TIMESTAMP NOT NULL,
  started TIMESTAMP,
  finished TIMESTAMP,
);

CREATE UNIQUE INDEX incremental_export_log_idx on incremental_export_log ("collection", "to_tsmp");

-- manually insert your latest snapshot here
-- insert into incremental_export_log (started, finished, to_tsmp, collection) values ('2024-02-27T05:53:30+00:00', '2024-02-27T07:23:30+00:00', '2024-02-27T05:53:30+00:00', 'app.bsky.graph.follow');
-- insert into incremental_export_log (started, finished, to_tsmp, collection) values ('2024-02-27T05:53:30+00:00', '2024-02-27T07:23:30+00:00', '2024-02-27T05:53:30+00:00', 'app.bsky.feed.like');
-- insert into incremental_export_log (started, finished, to_tsmp, collection) values ('2024-02-27T05:53:30+00:00', '2024-02-27T07:23:30+00:00', '2024-02-27T05:53:30+00:00', 'app.bsky.feed.post');
-- insert into incremental_export_log (started, finished, to_tsmp, collection) values ('2024-02-27T05:53:30+00:00', '2024-02-27T07:23:30+00:00', '2024-02-27T05:53:30+00:00', 'did');
-- insert into incremental_export_log (started, finished, to_tsmp, collection) values ('2024-02-27T05:53:30+00:00', '2024-02-27T07:23:30+00:00', '2024-02-27T05:53:30+00:00', 'handle');