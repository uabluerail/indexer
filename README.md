# Bluesky indexer

This is a bunch of code that can download all of Bluesky into a giant table in
PostgreSQL.

The structure of that table is roughly `(repo, collection, rkey) -> JSON`, and
it is a good idea to partition it by collection.

## System requirements

NOTE: all of this is valid as of December 2024, when Bluesky has ~24M accounts,
~4.7B records total, and average daily peak of ~1000 commits/s.

* Local PLC mirror. Without it you'll get throttled hard all the time. So go
  to https://github.com/bsky-watch/plc-mirror and set it up now. It'll need
  a few hours to replicate everything and become useable.
* 32GB of RAM, but the more the better, obviously.
* One decent SATA SSD is plenty fast to keep up. Preferably a dedicated one
  (definitely not the same that your system is installed on). There will be a
  lot of writes happening, so the total durability of the disk will be used up
  at non-negligible rate.
* XFS is mandatory to get a decent performance out of ScyllaDB.

With a SATA SSD dedicated to ScyllaDB it can handle about 6000 commits/s from firehose. The actual number you'll get might be lower, if your CPU is not fast enough.

## Overview of components

### Lister

Once a day get a list of all repos from all known PDSs and adds any that are
missing to the database.

### Consumer

Connects to firehose of each PDS and stores all received records in the
database.

If `CONSUMER_RELAYS` is specified, it will also add any new PDSs to the database
that have records sent through a relay.

### Record indexer

Goes over all repos that might have missing data, gets a full checkout from the
PDS and adds all missing records to the database.

## Setup

* Set up a [PLC mirror](https://github.com/bsky-watch/plc-mirror). It'll need
  a few hours to fetch all the data. You can use any other implementation too,
  the only requirement is that `/${did}` request returns a DID document.
* Decide where do you want to store the data. It needs to be on XFS,
  otherwise ScyllaDB's performance will be very poor.
* Copy `example.env` to `.env` and edit it to your liking.
    * `POSTGRES_PASSWORD` can be anything, it will be used on the first start of
      `postgres` container to initialize the database.
* Optional: copy `docker-compose.override.yml.example` to
  `docker-compose.override.yml` to change some parts of `docker-compose.yml`
  without actually editing it (and introducing possibility of merge conflicts
  later on).
* `make init-db`
    * This will add the initial set of PDS hosts into the database.
    * You can skip this if you're specifying `CONSUMER_RELAYS` in
      `docker-compose.override.yml`
* `make up`

## Additional commands

* `make status` - will show container status and resource usage
* `make psql` - starts up SQL shell inside the `postgres` container
* `make logs` - streams container logs into your terminal
* `make sqltop` - will show you currently running queries
* `make sqldu` - will show disk space usage for each table and index

## Tweaking the number of indexer threads at runtime

Record indexer exposes a simple HTTP handler that allows to do this:

`curl -s 'http://localhost:11003/pool/resize?size=10'`

## Advanced topics

### Table partitioning

With partitioning by collection you can have separate indexes for each record
type. Also, doing any kind of heavy processing on a particular record type will
be also faster, because all of these records will be in a separate table and
PostgreSQL will just read them sequentially, instead of checking `collection`
column for each row.

You can do the partitioning at any point, but the more data you already have in
the database, the longer will it take.

Before doing this you need to run `lister` at least once in order to create the
tables (`make init-db` does this for you as well).

* Stop all containers except for `postgres`.
* Run the [SQL script](db-migration/migrations/20240217_partition.sql) in
  `psql`.
* Check [`migrations`](db-migration/migrations/) dir for any additional
  migrations you might be interested in.
* Once all is done, start the other containers again.
