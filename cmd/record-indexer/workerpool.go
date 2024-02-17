package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/imax9000/errors"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/uabluerail/bsky-tools/xrpcauth"
	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/repo"
	"github.com/uabluerail/indexer/util/resolver"
)

type WorkItem struct {
	Repo   *repo.Repo
	signal chan struct{}
}

type WorkerPool struct {
	db      *gorm.DB
	input   <-chan WorkItem
	limiter *Limiter

	workerSignals []chan struct{}
	resize        chan int
}

func NewWorkerPool(input <-chan WorkItem, db *gorm.DB, size int, limiter *Limiter) *WorkerPool {
	r := &WorkerPool{
		db:      db,
		input:   input,
		limiter: limiter,
		resize:  make(chan int),
	}
	r.workerSignals = make([]chan struct{}, size)
	for i := range r.workerSignals {
		r.workerSignals[i] = make(chan struct{})
	}
	return r
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

	doc, err := resolver.GetDocument(ctx, work.Repo.DID)
	if err != nil {
		return fmt.Errorf("resolving did %q: %w", work.Repo.DID, err)
	}

	pdsHost := ""
	for _, srv := range doc.Service {
		if srv.Type != "AtprotoPersonalDataServer" {
			continue
		}
		pdsHost = srv.ServiceEndpoint
	}
	if pdsHost == "" {
		return fmt.Errorf("did not find any PDS in DID Document")
	}
	u, err := url.Parse(pdsHost)
	if err != nil {
		return fmt.Errorf("PDS endpoint (%q) is an invalid URL: %w", pdsHost, err)
	}
	if u.Host == "" {
		return fmt.Errorf("PDS endpoint (%q) doesn't have a host part", pdsHost)
	}

	client := xrpcauth.NewAnonymousClient(ctx)
	client.Host = u.String()
	client.Client = util.RobustHTTPClient()
	client.Client.Timeout = 30 * time.Minute

retry:
	if p.limiter != nil {
		if err := p.limiter.Wait(ctx, u.String()); err != nil {
			return fmt.Errorf("failed to wait on rate limiter: %w", err)
		}
	}

	b, err := comatproto.SyncGetRepo(ctx, client, work.Repo.DID, "")
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
	reposFetched.WithLabelValues(u.String(), "true").Inc()

	newRev, err := repo.GetRev(ctx, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to read 'rev' from the fetched repo: %w", err)
	}

	newRecs, err := repo.ExtractRecords(ctx, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to extract records: %w", err)
	}
	recs := []repo.Record{}
	for k, v := range newRecs {
		parts := strings.SplitN(k, "/", 2)
		if len(parts) != 2 {
			log.Warn().Msgf("Unexpected key format: %q", k)
			continue
		}
		recs = append(recs, repo.Record{
			Repo:       models.ID(work.Repo.ID),
			Collection: parts[0],
			Rkey:       parts[1],
			Content:    v,
		})
	}
	recordsFetched.Add(float64(len(recs)))
	if len(recs) > 0 {
		for _, batch := range splitInBatshes(recs, 500) {
			result := p.db.Model(&repo.Record{}).
				Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"content"}),
					Columns: []clause.Column{{Name: "repo"}, {Name: "collection"}, {Name: "rkey"}}}).
				Create(batch)
			if err := result.Error; err != nil {
				return fmt.Errorf("inserting records into the database: %w", err)
			}
			recordsInserted.Add(float64(result.RowsAffected))

		}
	}

	err = p.db.Model(&repo.Repo{}).Where(&repo.Repo{ID: work.Repo.ID}).
		Updates(&repo.Repo{LastIndexedRev: newRev}).Error
	if err != nil {
		return fmt.Errorf("updating repo rev: %w", err)
	}

	return nil
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
