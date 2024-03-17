# Data consistency model

## Indicators received from upstream

### `rev`

String value, sequencing each commit within a given repo. Each next commit must have a `rev` value strictly greater than the previous commit.

### Cursor

Integer number, sent with each message in firehose. Must be strictly increasing. Messages also contain `rev` value for the corresponding repo event, and we assume that within each repo all commits with smaller `rev` values also were sent with smaller cursor values. That is, cursor sequences all events recorded by the PDS and we assume that events of each given repo are sent in proper order.

#### Cursor reset

"Cursor reset" is a situation where upon reconnecting to a PDS we find out that the PDS is unable to send us all events that happened since the cursor value we have recorded. It is **Very Bad**™, because we have no idea what events did we miss between our recorded cursor and the new cursor that PDS has sent us.

This gap in data from a PDS must be addressed somehow, and most of this document revolves around detecting when a given repo is affected by a cursor reset and how to recover missing data with minimal effort.

## Available operations

### Repo fetch

We can fetch a full copy of a repo. Each commit contains a `rev` - string value
that is strictly increasing with each new commit.

### Consuming firehose

We can stream new events from each PDS. Every event comes with a cursor value -
integer number that is strictly increasing, scoped to a PDS. Events also contain
repo-specific `rev` which is the same with a full repo fetch.

## Metadata fields

### PDS

* `Cursor` - last cursor value received from this PDS.
* `FirstCursorSinceReset` - earliest cursor we have uninterrupted sequence of
  records up to now.

### Repo

* `LastIndexedRev` - last `rev` recorded during most recent full repo re-index
  * Up to this `rev` we do have all records
* `FirstRevSinceReset` - first `rev` seen on firehose since the most recent
  cursor reset.
  * Changes only when an event for this repo is received, so it alone doesn't
    guarantee that we have all subsequent records
* `FirstCursorSinceReset` - copy of the PDS field with the same name.
  * If `FirstCursorSinceReset` >= `PDS`.`FirstCursorSinceReset` and PDS's
    firehose is live - then we indeed have all records since
    `FirstRevSinceReset`
* `LastFirehoseRev` - last `rev` seen on the firehose
  * Currently recorded, but not used for anything

## Guarantees

* Up to and including `LastIndexedRev` - all records have been indexed.

* If `FirstCursorSinceReset` >= `PDS`.`FirstCursorSinceReset`:
  * Starting from and including `FirstRevSinceReset` - we have indexed all newer
    records
    * Consequently, if `LastIndexedRev` >= `FirstRevSinceReset` - we have a
      complete copy of the repo

* If `FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`:
  * There was a cursor reset, we might be missing some records after
    `FirstRevSinceReset`

## Operations

### Indexing a repo

* Resolve the current PDS hosting the repo and store its `FirstCursorSinceReset` in a variable
  * If the PDS is different from the one we have on record (i.e., the repo migrated) - update accordingly
* Fetch the repo
* Upsert all fetched records
* Set `LastIndexedRev` to `rev` of the fetched repo
* In a transaction check if `Repo`.`FirstCursorSinceReset` >= the value stored in the first step, and set it to that value if it isn't.
  * Assumption here is that a PDS returns strongly consistent responses for a single repo, and fetching the repo will include all records corresponding to a cursor value generated before that.

### Connecting to firehose

* If the first message is `#info` - this means that our cursor is too old
  * Update PDS's `FirstCursorSinceReset` to the value supplied in the `#info`
    message

Workaround for a buggy relay that doesn't send `#info`:

* If the first message has cursor value that is different from `Cursor`+1:
  * Assume there was a cursor reset and update PDS's `FirstCursorSinceReset` to
    the value provided in the message

### Receiving event on firehose

* Check that the event is coming from the correct PDS for a given repo
  * TODO: maybe drop this and just check the signature
* Process the event normally
* If `Repo`.`FirstCursorSinceReset` >= `PDS`.`FirstCursorSinceReset`:
  * No metadata updates needed for the repo
* If `Repo`.`FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`:
  * Set repo's `FirstRevSinceReset` to the event's `rev` and
    `FirstCursorSinceReset` to `PDS`.`FirstCursorSinceReset`

* If `tooBig` flag is set on the message (MST diff was larger than PDS's size
  limit, so some records were dropped):
  * Set repo's `FirstRevSinceReset` to the event's `rev` and
    `FirstCursorSinceReset` to `PDS`.`FirstCursorSinceReset`
    * Note: `FirstCursorSinceReset` might be the same, but moving forward
      `FirstRevSinceReset` likely will trigger repo reindexing

* Update `LastFirehoseRev` to event's `rev`
* Update PDS's `Cursor` to the value provided in the message

### Listing repos

* Fetch a list of repos from a PDS. Response also includes the last `rev` for
  every repo.
* For each repo:
  * If `FirstRevSinceReset` is not set:
    * Set `FirstRevSinceReset` to received `rev`
    * Set `FirstCursorSinceReset` to the PDS's `FirstCursorSinceReset`

### Repo migrating to a different PDS

TODO

Currently we're simply resetting `FirstRevSinceReset`.

### Finding repos that need indexing

* Repo index is incomplete and needs to be indexed if one of these is true:
  * `LastIndexedRev` is not set
  * `LastIndexedRev` < `FirstCursorSinceReset`
  * `Repo`.`FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`
