package main

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/pds"
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
	const maxQueueLen = 30000
	const maxAttempts = 3

	s.mu.Lock()
	queueLen := len(s.queue) + len(s.inProgress)
	s.mu.Unlock()
	if queueLen >= maxQueueLen {
		return nil
	}

	remotes := []pds.PDS{}
	if err := s.db.Find(&remotes).Error; err != nil {
		return fmt.Errorf("failed to get the list of PDSs: %w", err)
	}

	remotes = slices.DeleteFunc(remotes, func(pds pds.PDS) bool {
		return pds.Disabled
	})
	perPDSLimit := maxQueueLen
	if len(remotes) > 0 {
		perPDSLimit = maxQueueLen * 2 / len(remotes)
	}
	if perPDSLimit < maxQueueLen/10 {
		perPDSLimit = maxQueueLen / 10
	}

	// Fake remote to account for repos we didn't have a PDS for yet.
	remotes = append(remotes, pds.PDS{ID: pds.Unknown})

	for _, remote := range remotes {
		repos := []repo.Repo{}

		err := s.db.Raw(`SELECT * FROM "repos" WHERE pds = ? AND (last_indexed_rev is null OR last_indexed_rev = '') AND failed_attempts < ?
UNION
SELECT "repos".* FROM "repos" left join "pds" on repos.pds = pds.id WHERE pds = ?
	AND
		(
			(first_rev_since_reset is not null AND first_rev_since_reset <> ''
				AND last_indexed_rev < first_rev_since_reset)
			OR
			("repos".first_cursor_since_reset is not null AND "repos".first_cursor_since_reset <> 0
				AND "repos".first_cursor_since_reset < "pds".first_cursor_since_reset)
		)
	AND failed_attempts < ? LIMIT ?`,
			remote.ID, maxAttempts, remote.ID, maxAttempts, perPDSLimit).
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
