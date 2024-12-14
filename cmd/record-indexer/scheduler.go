package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/repo"
	"gorm.io/gorm"
)

type Scheduler struct {
	db     *gorm.DB
	output chan<- WorkItem

	mu         sync.Mutex
	queue      map[string]*repo.Repo
	inProgress map[string]*repo.Repo
}

func NewScheduler(output chan<- WorkItem, db *gorm.DB) *Scheduler {
	return &Scheduler{
		db:         db,
		output:     output,
		queue:      map[string]*repo.Repo{},
		inProgress: map[string]*repo.Repo{},
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	go s.run(ctx)
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	log := zerolog.Ctx(ctx)
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	if err := s.fillQueue(ctx); err != nil {
		log.Error().Err(err).Msgf("Failed to get more tasks for the queue: %s", err)
	}

	done := make(chan string)
	for {
		s.mu.Lock()
		q := len(s.queue)
		s.mu.Unlock()
		if q > 0 {
			next := WorkItem{signal: make(chan struct{})}
			s.mu.Lock()
			for _, r := range s.queue {
				next.Repo = r
				break
			}
			s.mu.Unlock()

			select {
			case <-ctx.Done():
				return
			case <-t.C:
				go func() {
					if err := s.fillQueue(ctx); err != nil {
						log.Error().Err(err).Msgf("Failed to get more tasks for the queue: %s", err)
					}
				}()
			case s.output <- next:
				s.mu.Lock()
				delete(s.queue, next.Repo.DID)
				s.inProgress[next.Repo.DID] = next.Repo
				s.mu.Unlock()

				go func(did string, ch chan struct{}) {
					select {
					case <-ch:
					case <-ctx.Done():
					}
					done <- did
				}(next.Repo.DID, next.signal)
				s.updateQueueLenMetrics()
			case did := <-done:
				s.mu.Lock()
				delete(s.inProgress, did)
				s.mu.Unlock()
				s.updateQueueLenMetrics()
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := s.fillQueue(ctx); err != nil {
					log.Error().Err(err).Msgf("Failed to get more tasks for the queue: %s", err)
				}
			case did := <-done:
				s.mu.Lock()
				delete(s.inProgress, did)
				s.mu.Unlock()
				s.updateQueueLenMetrics()
			}
		}
	}
}

func (s *Scheduler) fillQueue(ctx context.Context) error {
	const maxQueueLen = 300000
	const lowWatermark = 30000
	const maxAttempts = 3
	log := zerolog.Ctx(ctx)

	s.mu.Lock()
	queueLen := len(s.queue) + len(s.inProgress)
	s.mu.Unlock()
	if queueLen >= lowWatermark {
		return nil
	}

	counts := []pdsCounts{}
	err := s.db.Raw(`select * from (
	  SELECT pds, count(*) FROM "repos" left join "pds" on repos.pds = pds.id WHERE
	    (
	      (last_indexed_rev is null OR last_indexed_rev = '') OR
	      (first_rev_since_reset is not null AND first_rev_since_reset <> ''
	        AND last_indexed_rev < first_rev_since_reset)
	      OR
	      ("repos".first_cursor_since_reset is not null AND "repos".first_cursor_since_reset <> 0
	        AND "repos".first_cursor_since_reset < "pds".first_cursor_since_reset)
	    )
	  AND failed_attempts < 3
	  AND (not pds.disabled OR pds.disabled is null)
	  GROUP BY pds
	) order by count desc`).Scan(&counts).Error
	if err != nil {
		return fmt.Errorf("querying DB: %w", err)
	}
	log.Debug().Msgf("Found %d PDSs with repos that need fetching", len(counts))

	batches := batchBySize(counts, maxQueueLen)
	perBatchLimit := maxQueueLen
	// Avoid division by zero if there is no work.
	if len(batches) > 0 {
		perBatchLimit = maxQueueLen / len(batches)
	}

	for _, batch := range batches {
		repos := []repo.Repo{}

		ids := []models.ID{}
		for _, c := range batch {
			ids = append(ids, c.PDS)
		}

		err := s.db.Raw(`SELECT repos.* FROM repos left join pds on repos.pds = pds.id WHERE pds IN ?
			AND
				(
					(last_indexed_rev is null OR last_indexed_rev = '')
					OR
					(first_rev_since_reset is not null AND first_rev_since_reset <> ''
						AND last_indexed_rev < first_rev_since_reset)
					OR
					(repos.first_cursor_since_reset is not null AND repos.first_cursor_since_reset <> 0
						AND repos.first_cursor_since_reset < pds.first_cursor_since_reset)
				)
			AND failed_attempts < ? LIMIT ?`,
			ids, maxAttempts, perBatchLimit).
			Scan(&repos).Error

		if err != nil {
			return fmt.Errorf("querying DB: %w", err)
		}
		s.mu.Lock()
		for _, r := range repos {
			if s.queue[r.DID] != nil || s.inProgress[r.DID] != nil {
				continue
			}
			copied := r
			s.queue[r.DID] = &copied
			reposQueued.Inc()
		}
		s.mu.Unlock()
		s.updateQueueLenMetrics()
	}

	return nil
}

func (s *Scheduler) updateQueueLenMetrics() {
	s.mu.Lock()
	queueLenght.WithLabelValues("queued").Set(float64(len(s.queue)))
	queueLenght.WithLabelValues("inProgress").Set(float64(len(s.inProgress)))
	s.mu.Unlock()
}

type pdsCounts struct {
	PDS   models.ID
	Count int
}

func batchBySize(counts []pdsCounts, size int) [][]pdsCounts {
	r := [][]pdsCounts{}
	sum := 0
	start := 0

	for i := 0; i < len(counts); i++ {
		sum += counts[i].Count

		if sum >= size {
			r = append(r, counts[start:i+1])
			start = i + 1
			sum = 0
		}
	}
	if start < len(counts) {
		r = append(r, counts[start:])
	}

	return r
}
