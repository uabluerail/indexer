package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/imax9000/errors"
	"github.com/rs/zerolog"
	"github.com/scylladb/gocqlx/qb"
	"github.com/scylladb/gocqlx/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/uabluerail/bsky-tools/xrpcauth"
	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/repo"
	"github.com/uabluerail/indexer/util/fix"
	"github.com/uabluerail/indexer/util/resolver"
)

type WorkItem struct {
	Repo   *repo.Repo
	signal chan struct{}
}

type WorkerPool struct {
	db                  *gorm.DB
	recordsDB           *gocqlx.Session
	input               <-chan WorkItem
	limiter             *Limiter
	collectionBlacklist map[string]bool

	workerSignals []chan struct{}
	resize        chan int
}

func NewWorkerPool(input <-chan WorkItem, db *gorm.DB, session *gocqlx.Session, size int, limiter *Limiter) *WorkerPool {
	r := &WorkerPool{
		db:                  db,
		recordsDB:           session,
		input:               input,
		limiter:             limiter,
		resize:              make(chan int),
		collectionBlacklist: map[string]bool{},
	}
	r.workerSignals = make([]chan struct{}, size)
	for i := range r.workerSignals {
		r.workerSignals[i] = make(chan struct{})
	}
	return r
}

func (p *WorkerPool) BlacklistCollections(colls []string) {
	for _, c := range colls {
		p.collectionBlacklist[c] = true
	}
}
func (p *WorkerPool) Start(ctx context.Context) error {
	go p.run(ctx)
	return nil
}

func (p *WorkerPool) Resize(ctx context.Context, size int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.resize <- size:
		return nil
	}
}

func (p *WorkerPool) run(ctx context.Context) {
	for _, ch := range p.workerSignals {
		go p.worker(ctx, ch)
	}
	workerPoolSize.Set(float64(len(p.workerSignals)))

	for {
		select {
		case <-ctx.Done():
			for _, ch := range p.workerSignals {
				close(ch)
			}
			// also wait for all workers to stop?
			return
		case newSize := <-p.resize:
			switch {
			case newSize > len(p.workerSignals):
				ch := make([]chan struct{}, newSize-len(p.workerSignals))
				for i := range ch {
					ch[i] = make(chan struct{})
					go p.worker(ctx, ch[i])
				}
				p.workerSignals = append(p.workerSignals, ch...)
				workerPoolSize.Set(float64(len(p.workerSignals)))
			case newSize < len(p.workerSignals) && newSize > 0:
				for _, ch := range p.workerSignals[newSize:] {
					close(ch)
				}
				p.workerSignals = p.workerSignals[:newSize]
				workerPoolSize.Set(float64(len(p.workerSignals)))
			}
		}
	}
}

func (p *WorkerPool) worker(ctx context.Context, signal chan struct{}) {
	log := zerolog.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-signal:
			return
		case work := <-p.input:
			updates := &repo.Repo{}
			if err := p.doWork(ctx, work); err != nil {
				log.Error().Err(err).Msgf("Work task %q failed: %s", work.Repo.DID, err)
				updates.LastError = err.Error()
				updates.FailedAttempts = work.Repo.FailedAttempts + 1
				reposIndexed.WithLabelValues("false").Inc()
			} else {
				updates.FailedAttempts = 0
				reposIndexed.WithLabelValues("true").Inc()
			}
			updates.LastIndexAttempt = time.Now()
			err := p.db.Model(&repo.Repo{}).
				Where(&repo.Repo{ID: work.Repo.ID}).
				Select("last_error", "last_index_attempt", "failed_attempts").
				Updates(updates).Error
			if err != nil {
				log.Error().Err(err).Msgf("Failed to update repo info for %q: %s", work.Repo.DID, err)
			}
		}
	}
}

func (p *WorkerPool) doWork(ctx context.Context, work WorkItem) error {
	log := zerolog.Ctx(ctx)
	defer close(work.signal)

	u, pubKey, err := resolver.GetPDSEndpointAndPublicKey(ctx, work.Repo.DID)
	if err != nil {
		return err
	}

	remote, err := pds.EnsureExists(ctx, p.db, u.String())
	if err != nil {
		return fmt.Errorf("failed to get PDS records for %q: %w", u, err)
	}
	if work.Repo.PDS != remote.ID {
		if err := p.db.Model(&work.Repo).Where(&repo.Repo{ID: work.Repo.ID}).Updates(&repo.Repo{PDS: remote.ID}).Error; err != nil {
			return fmt.Errorf("failed to update repo's PDS to %q: %w", u, err)
		}
		work.Repo.PDS = remote.ID
	}

	client := xrpcauth.NewAnonymousClient(ctx)
	client.Host = u.String()
	client.Client = util.RobustHTTPClient()
	client.Client.Timeout = 30 * time.Minute

	knownCursorBeforeFetch := remote.FirstCursorSinceReset

retry:
	if p.limiter != nil {
		if err := p.limiter.Wait(ctx, u.String()); err != nil {
			return fmt.Errorf("failed to wait on rate limiter: %w", err)
		}
	}

	// TODO: add a configuration knob for switching between full and partial fetch.
	sinceRev := work.Repo.LastIndexedRev
	b, err := comatproto.SyncGetRepo(ctx, client, work.Repo.DID, sinceRev)
	if err != nil {
		if err, ok := errors.As[*xrpc.Error](err); ok {
			if err.IsThrottled() && err.Ratelimit != nil {
				log.Debug().Str("pds", u.String()).Msgf("Hit a rate limit (%s), sleeping until %s", err.Ratelimit.Policy, err.Ratelimit.Reset)
				time.Sleep(time.Until(err.Ratelimit.Reset))
				goto retry
			}
		}

		reposFetched.WithLabelValues(u.String(), "false").Inc()
		return fmt.Errorf("failed to fetch repo: %w", err)
	}
	if len(b) == 0 {
		reposFetched.WithLabelValues(u.String(), "false").Inc()
		return fmt.Errorf("PDS returned zero bytes")
	}
	reposFetched.WithLabelValues(u.String(), "true").Inc()

	if work.Repo.PDS == pds.Unknown {
		remote, err := pds.EnsureExists(ctx, p.db, u.String())
		if err != nil {
			return err
		}
		work.Repo.PDS = remote.ID
		if err := p.db.Model(&work.Repo).Where(&repo.Repo{ID: work.Repo.ID}).Updates(&repo.Repo{PDS: work.Repo.PDS}).Error; err != nil {
			return fmt.Errorf("failed to set repo's PDS: %w", err)
		}
	}

	newRev, err := repo.GetRev(ctx, bytes.NewReader(b))
	if sinceRev != "" && errors.Is(err, repo.ErrZeroBlocks) {
		// No new records since the rev we requested above.
		if work.Repo.FirstCursorSinceReset < knownCursorBeforeFetch {
			if err := p.bumpFirstCursorSinceReset(work.Repo.ID, knownCursorBeforeFetch); err != nil {
				return fmt.Errorf("updating first_cursor_since_reset: %w", err)
			}
		}
		return nil
	} else if err != nil {
		l := 25
		if len(b) < l {
			l = len(b)
		}
		log.Debug().Err(err).Msgf("Total bytes fetched: %d. First few bytes: %q", len(b), string(b[:l]))
		return fmt.Errorf("failed to read 'rev' from the fetched repo: %w", err)
	}

	newRecs, err := repo.ExtractRecords(ctx, bytes.NewReader(b), pubKey)
	if err != nil {
		return fmt.Errorf("failed to extract records: %w", err)
	}
	recordsFetched.Add(float64(len(newRecs)))

	return p.insertRecords(ctx, newRecs, work, newRev, knownCursorBeforeFetch)
}

func (p *WorkerPool) insertRecords(ctx context.Context, newRecs map[string]json.RawMessage, work WorkItem, newRev string, knownCursorBeforeFetch int64) error {
	log := zerolog.Ctx(ctx)

	recs := []repo.Record{}
	for k, v := range newRecs {
		parts := strings.SplitN(k, "/", 2)
		if len(parts) != 2 {
			log.Warn().Msgf("Unexpected key format: %q", k)
			continue
		}
		if p.collectionBlacklist[parts[0]] {
			continue
		}
		rec := repo.Record{
			Repo:       models.ID(work.Repo.ID),
			Collection: parts[0],
			Rkey:       parts[1],
			Content:    v,
			AtRev:      newRev,
		}
		if p.recordsDB == nil {
			// XXX: proper replacement of \u0000 would require full parsing of JSON
			// and recursive iteration over all string values, but this
			// should work well enough for now.
			rec.Content = fix.EscapeNullCharForPostgres(rec.Content)
		}
		recs = append(recs, rec)
	}
	if len(recs) > 0 {
		if p.recordsDB != nil {
			// We're querying for a matching record before insert to avoid
			// unnecessary record duplication. If we didn't, virtually all
			// records from a repo download would create new rows, because `at_rev`
			// would not match any existing entries.
			// So we fetch the last version of a record and skip insert if
			// it's identical to the one we just fetched.
			for _, rec := range recs {
				iter := p.recordsDB.Query(
					qb.Select("bluesky.records").Columns("at_rev", "deleted", "record").
						Where(qb.Eq("repo"), qb.Eq("collection"), qb.Eq("rkey")).
						OrderBy("at_rev", qb.DESC).Limit(1).ToCql()).
					WithContext(ctx).Bind(work.Repo.DID, rec.Collection, rec.Rkey).Iter()

				var (
					skip    = false
					atRev   string
					deleted bool
					record  json.RawMessage
				)
				for iter.Scan(&atRev, &deleted, &record) {
					if bytes.Equal(record, rec.Content) {
						skip = true
					}
				}
				if err := iter.Close(); err != nil {
					return fmt.Errorf("failed to check for existing record: %w", err)
				}
				if skip {
					continue
				}
				err := p.recordsDB.Query(qb.Insert("bluesky.records").
					Columns("repo", "collection", "rkey", "at_rev", "record", "created_at").
					ToCql()).WithContext(ctx).
					Bind(work.Repo.DID, rec.Collection, rec.Rkey, rec.AtRev, rec.Content, time.Now()).ExecRelease()
				if err != nil {
					return fmt.Errorf("inserting record %s/%s/%s into the database: %w", work.Repo.DID, rec.Collection, rec.Rkey, err)
				}
				recordsInserted.Add(1)
			}
		} else {
			for _, batch := range splitInBatshes(recs, 500) {
				result := p.db.Model(&repo.Record{}).
					Clauses(clause.OnConflict{
						Where: clause.Where{Exprs: []clause.Expression{
							clause.Neq{
								Column: clause.Column{Name: "content", Table: "records"},
								Value:  clause.Column{Name: "content", Table: "excluded"}},
							clause.Or(
								clause.Eq{Column: clause.Column{Name: "at_rev", Table: "records"}, Value: nil},
								clause.Eq{Column: clause.Column{Name: "at_rev", Table: "records"}, Value: ""},
								clause.Lt{
									Column: clause.Column{Name: "at_rev", Table: "records"},
									Value:  clause.Column{Name: "at_rev", Table: "excluded"}},
							)}},
						DoUpdates: clause.AssignmentColumns([]string{"content", "at_rev"}),
						Columns:   []clause.Column{{Name: "repo"}, {Name: "collection"}, {Name: "rkey"}}}).
					Create(batch)
				if err := result.Error; err != nil {
					return fmt.Errorf("inserting records into the database: %w", err)
				}
				recordsInserted.Add(float64(result.RowsAffected))
			}
		}
	}

	err := p.db.Model(&repo.Repo{}).Where(&repo.Repo{ID: work.Repo.ID}).
		Updates(&repo.Repo{LastIndexedRev: newRev}).Error
	if err != nil {
		return fmt.Errorf("updating repo rev: %w", err)
	}

	if work.Repo.FirstCursorSinceReset < knownCursorBeforeFetch {
		if err := p.bumpFirstCursorSinceReset(work.Repo.ID, knownCursorBeforeFetch); err != nil {
			return fmt.Errorf("updating first_cursor_since_reset: %w", err)
		}
	}
	// TODO: check for records that are missing in the repo download
	// and mark them as deleted.

	return nil
}

// bumpFirstCursorSinceReset increases repo's FirstCursorSinceReset iff it is currently lower than the supplied value.
func (p *WorkerPool) bumpFirstCursorSinceReset(repoId models.ID, cursorValue int64) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var currentCursor int64
		err := tx.Model(&repo.Repo{}).Where(&repo.Repo{ID: repoId}).
			Select("first_cursor_since_reset").First(&currentCursor).Error
		if err != nil {
			return fmt.Errorf("failed to get current cursor value: %w", err)
		}
		if currentCursor < cursorValue {
			return tx.Model(&repo.Repo{}).Where(&repo.Repo{ID: repoId}).
				Updates(&repo.Repo{FirstCursorSinceReset: cursorValue}).Error
		}
		return nil
	})
}

func splitInBatshes[T any](s []T, batchSize int) [][]T {
	var r [][]T
	for i := 0; i < len(s); i += batchSize {
		if i+batchSize < len(s) {
			r = append(r, s[i:i+batchSize])
		} else {
			r = append(r, s[i:])
		}
	}
	return r
}
