# Data consistency model

## Indicators received from upstream

We have two interconnected strictly ordered values: `rev` and cursor. `rev` is
local to each repo, cursor provides additional ordering across all repos hosted
on a PDS.

### `rev`

String value, sequencing each commit within a given repo. Each next commit must
have a `rev` value strictly greater than the previous commit.

### Cursor

Integer number, sent with each message in firehose. Must be strictly increasing.
Messages also contain `rev` value for the corresponding repo event, and we
assume that within each repo all commits with smaller `rev` values also were
sent with smaller cursor values. That is, cursor sequences all events recorded
by the PDS and we assume that events of each given repo are sent in proper
order.

#### Cursor reset

"Cursor reset" is a situation where upon reconnecting to a PDS we find out that
the PDS is unable to send us all events that happened since the cursor value we
have recorded. It is **Very Bad**™, because we have no idea what events did we
miss between our recorded cursor and the new cursor that PDS has sent us.

This gap in data from a PDS must be addressed somehow, and most of this document
revolves around detecting when a given repo is affected by a cursor reset and
how to recover missing data with minimal effort.

## Available operations

### Repo fetch

We can fetch a full copy of a repo. Each commit contains a `rev` - string value
that is strictly increasing with each new commit.

We also have the option to only fetch records created after a particular `rev` -
this is useful for reducing the amount of data received when we already have
some of the records.

### Consuming firehose

We can stream new events from each PDS. Every event comes with a cursor value -
integer number that is strictly increasing, scoped to a PDS. Events also contain
repo-specific `rev` which is the same with a full repo fetch.

## High-level overview

With `rev` imposing strict ordering on repo operations, we maintain the
following two indicators for each repo:

1. `LastCompleteRev` - largest `rev` value that we are sure we have the complete
   set of records at. For example, we can set this after processing the output
   of `getRepo` call.
2. `FirstUninterruptedFirehoseRev` - smallest `rev` value from which we are sure
   to have a complete set of records up until ~now.

These indicators define two intervals of `rev` values (`(-Infinity,
LastCompleteRev]`, `[FirstUninterruptedFirehoseRev, now)`) that we assume to
have already processed. If these intervals overlap - we assume that we've
covered `(-Infinity, now)`, i.e., have a complete set of records of a given
repo. If they don't overlap - we might have missed some records, and can
remediate that by fetching the whole repo, indexing records we don't have and
updating `LastCompleteRev`.

Both of these indicators should never decrease. When a PDS tells us that our
cursor value is invalid, we move `FirstUninterruptedFirehoseRev` forward, which
in turn can make the above intervals non-overlapping.

These indicators also can be uninitialized, which means that we have no data
about the corresponding interval.

Note that for performance and feasibility reasons we don't store these two
indicators in the database directly. Instead, to minimize the number of writes,
we derive them from a few other values.

### Updating `LastCompleteRev`

We can move `LastCompleteRev` forward when either:

* We just indexed a full repo checkout
* We got a new record from firehose AND the repo currently has no gaps
  (`LastCompleteRev` >= `FirstUninterruptedFirehoseRev`)

### Updating `FirstUninterruptedFirehoseRev`

Once initialized, stays constant during normal operation. Can move forward if a
PDS informs us that we missed some records and it can't replay all of them (and
resets our cursor).

## Handling cursor resets

### Naive approach

We could store `FirstUninterruptedFirehoseRev` in a column for each repo, and
when we detect a cursor reset - unset it for every repo from a particular PDS.

There are a couple of issues with this:

1. Cursor reset will trigger a lot of writes: row for each repo from the
   affected PDS will have to be updated.
2. We have no information about `[FirstUninterruptedFirehoseRev, now)` interval
   until we see a new commit for a repo, which might take a long time, or never
   happen at all.

### Reducing the number of writes

We can rely on the firehose cursor value imposing additional ordering on
commits.

1. Start tracking firehose stream continuity by storing
   `FirstUninterruptedCursor` for each PDS
2. When receiving a commit from firehose, compare `FirstUninterruptedCursor`
   between repo and PDS entries:
    * If `Repo`.`FirstUninterruptedCursor` < `PDS`.`FirstUninterruptedCursor`,
      set `FirstUninterruptedFirehoseRev` to the commit's `rev` and copy
      `FirstUninterruptedCursor` from PDS entry.

Now during a cursor reset we need to only change `FirstUninterruptedCursor` in
the PDS entry. And if `Repo`.`FirstUninterruptedCursor` <
`PDS`.`FirstUninterruptedCursor` - we know that repo's hosting PDS reset our
cursor at some point and `FirstUninterruptedFirehoseRev` value is no longer
valid.

### Avoiding long wait for the first firehose event

We can fetch the full repo to index any missing records and advance
`LastCompleteRev` accordingly. But if we don't update
`Repo`.`FirstUninterruptedCursor` - it will stay smaller than
`PDS`.`FirstUninterruptedCursor` and `FirstUninterruptedFirehoseRev` will remain
invalid.

We can fix that with an additional assumption: PDS provides strong consistency
between the firehose and `getRepo` - if we have already seen cursor value `X`,
then `getRepo` response will be up to date with all commits corresponding to
cursor values smaller or equal to `X`.

1. Before fetching the repo, note the current `FirstUninterruptedCursor` value
   of the repo's hosting PDS. (Or even the latest `Cursor` value)
2. Fetch and process the full repo checkout, setting `LastCompleteRev`
3. If `Repo`.`FirstUninterruptedCursor` < `PDS`.`FirstUninterruptedCursor` still
   holds (i.e., no new records on firehose while we were re-indexing), then set
   `Repo`.`FirstUninterruptedCursor` to the cursor value recorded in step 1.
   With the above assumption, all records that happened between
   `FirstUninterruptedFirehoseRev` and this cursor value were already processed
   in step 2, so `FirstUninterruptedFirehoseRev` is again valid, until
   `PDS`.`FirstUninterruptedCursor` moves forward again.

## Repo discovery

We have the ability to get a complete list of hosted repos from a PDS. The
response includes last known `rev` for each repo, but does not come attached
with a firehose cursor value. We're assuming here the same level of consistency
as with `getRepo`, and can initialize `Repo`.`FirstUninterruptedCursor` with the
value from the PDS entry recorded before making the call to list repos, and
`FirstUninterruptedFirehoseRev` to the returned `rev`.

TODO: consider if it's worth to not touch cursor/`rev` values here and offload
initializing them to indexing step described above.

## Updating `LastCompleteRev` based on firehose events

We have the option to only advance `LastCompleteRev` when processing the full
repo checkout. While completely valid, it's rather pessimistic in that, in
absence of cursor resets, this value will remain arbitrarily old despite us
actually having a complete set of records for the repo. Consequently, when a
cursor reset eventually does happen - we'll be assuming that we're missing much
more records than we actually do.

Naively, we can simply update `LastCompleteRev` on every event (iff the
completeness intervals are currently overlapping). The drawback is that each
event, in addition to new record creation, will update the corresponding repo
entry. If we could avoid this, it would considerably reduce the number of
writes.

### Alternative 1: delay updates

We can delay updating `LastCompleteRev` from firehose events for some time and
elide multiple updates to the same repo into a single write. Delay duration
would have to be at least on the order of minutes for this to be effective,
since writes to any single repo are usually initiated by human actions and have
a very low rate.

This way we can trade some RAM for reduction in writes.

### Alternative 2: skip frequent updates

Similar to the above, but instead of delaying updates, simply skip them if last
update was recent enough. This will often result in `LastCompleteRev` not
reflecting *actual* last complete `rev` for a repo, but it will keep it recent
enough.

## Detailed design

### Bad naming

In the implementation not enough attention was paid to naming things, and their
usage and meaning slightly changed over time, so in the sections below and in
the code some of the things mentioned above are named differently:

* `LastCompleteRev` - max(`LastIndexedRev`, `LastFirehoseRev`)
* `FirstUninterruptedCursor` - `FirstCursorSinceReset`
* `FirstUninterruptedFirehoseRev` - `FirstRevSinceReset`

### Metadata fields

#### PDS

* `Cursor` - last cursor value received from this PDS.
* `FirstCursorSinceReset` - earliest cursor we have uninterrupted sequence of
  records up to now.

#### Repo

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
* `LastFirehoseRev` - last `rev` seen on the firehose while we didn't have any
  interruptions

### Guarantees

* Up to and including `LastIndexedRev` - all records have been indexed.
* If `LastFirehoseRev` is set - all records up to and including it have been
  indexed.

* If `FirstCursorSinceReset` >= `PDS`.`FirstCursorSinceReset`:
  * Starting from and including `FirstRevSinceReset` - we have indexed all newer
    records
    * Consequently, if max(`LastIndexedRev`, `LastFirehoseRev`) >=
      `FirstRevSinceReset` - we have a complete copy of the repo

* If `FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`:
  * There was a cursor reset, we might be missing some records after
    `FirstRevSinceReset`

* `FirstCursorSinceReset` on both repos and PDSs never gets rolled back
* `LastIndexedRev` never gets rolled back

### Operations

#### Indexing a repo

* Resolve the current PDS hosting the repo and store its `FirstCursorSinceReset`
  in a variable
  * If the PDS is different from the one we have on record (i.e., the repo
    migrated) - update accordingly
* Fetch the repo
* Upsert all fetched records
* Set `LastIndexedRev` to `rev` of the fetched repo
* In a transaction check if `Repo`.`FirstCursorSinceReset` >= the value stored
  in the first step, and set it to that value if it isn't.
  * Assumption here is that a PDS returns strongly consistent responses for a
    single repo, and fetching the repo will include all records corresponding to
    a cursor value generated before that.

#### Connecting to firehose

* If the first message is `#info` - this means that our cursor is too old
  * Update PDS's `FirstCursorSinceReset` to the value supplied in the `#info`
    message

Workaround for a buggy relay that doesn't send `#info`:

* If the first message has cursor value that is different from `Cursor`+1:
  * Assume there was a cursor reset and update PDS's `FirstCursorSinceReset` to
    the value provided in the message

#### Receiving event on firehose

* Check that the event is coming from the correct PDS for a given repo
  * TODO: maybe drop this and just check the signature
* Process the event normally
* If `Repo`.`FirstCursorSinceReset` >= `PDS`.`FirstCursorSinceReset`:
  * Update `LastFirehoseRev` to event's `rev`
* If `Repo`.`FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`:
  * Set repo's `FirstRevSinceReset` to the event's `rev` and
    `FirstCursorSinceReset` to `PDS`.`FirstCursorSinceReset`

* If `tooBig` flag is set on the message (MST diff was larger than PDS's size
  limit, so some records were dropped):
  * Set repo's `FirstRevSinceReset` to the event's `rev` and
    `FirstCursorSinceReset` to `PDS`.`FirstCursorSinceReset`
    * Note: `FirstCursorSinceReset` might be the same, but moving forward
      `FirstRevSinceReset` likely will trigger repo reindexing

* Update PDS's `Cursor` to the value provided in the message

#### Listing repos

* Fetch a list of repos from a PDS. Response also includes the last `rev` for
  every repo.
* For each repo:
  * If `FirstRevSinceReset` is not set:
    * Set `FirstRevSinceReset` to received `rev`
    * Set `FirstCursorSinceReset` to the PDS's `FirstCursorSinceReset`

#### Repo migrating to a different PDS

TODO

Currently we're simply resetting `FirstRevSinceReset`.

#### Finding repos that need indexing

* Repo index is incomplete and needs to be indexed if one of these is true:
  * `LastIndexedRev` is not set
  * max(`LastIndexedRev`, `LastFirehoseRev`) < `FirstRevSinceReset`
  * `Repo`.`FirstCursorSinceReset` < `PDS`.`FirstCursorSinceReset`
