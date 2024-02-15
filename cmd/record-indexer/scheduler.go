package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/repo"
	"gorm.io/gorm"
)

type Scheduler struct {
	db     *gorm.DB
	output chan<- WorkItem

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
		if len(s.queue) > 0 {
			next := WorkItem{signal: make(chan struct{})}
			for _, r := range s.queue {
				next.Repo = r
				break
			}

			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := s.fillQueue(ctx); err != nil {
					log.Error().Err(err).Msgf("Failed to get more tasks for the queue: %s", err)
				}
			case s.output <- next:
				delete(s.queue, next.Repo.DID)
				s.inProgress[next.Repo.DID] = next.Repo
				go func(did string, ch chan struct{}) {
					select {
					case <-ch:
					case <-ctx.Done():
					}
					done <- did
				}(next.Repo.DID, next.signal)
				s.updateQueueLenMetrics()
			case did := <-done:
				delete(s.inProgress, did)
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
				delete(s.inProgress, did)
				s.updateQueueLenMetrics()
			}
		}
	}
}

func (s *Scheduler) fillQueue(ctx context.Context) error {
	const maxQueueLen = 10000

	if len(s.queue)+len(s.inProgress) >= maxQueueLen {
		return nil
	}

	remotes := []pds.PDS{}
	if err := s.db.Find(&remotes).Error; err != nil {
		return fmt.Errorf("failed to get the list of PDSs: %w", err)
	}

	perPDSLimit := maxQueueLen * 2 / len(remotes)

	for _, remote := range remotes {
		repos := []repo.Repo{}

		err := s.db.Raw(`SELECT * FROM "repos" WHERE pds = ? AND (last_indexed_rev is null OR last_indexed_rev = '')
UNION
SELECT * FROM "repos" WHERE pds = ? AND (first_rev_since_reset is not null AND first_rev_since_reset <> '' AND last_indexed_rev < first_rev_since_reset) LIMIT ?`,
			remote.ID, remote.ID, perPDSLimit).
			Scan(&repos).Error

		if err != nil {
			return fmt.Errorf("querying DB: %w", err)
		}
		for _, r := range repos {
			if s.queue[r.DID] != nil || s.inProgress[r.DID] != nil {
				continue
			}
			copied := r
			s.queue[r.DID] = &copied
			reposQueued.Inc()
		}
		s.updateQueueLenMetrics()
	}

	return nil
}

func (s *Scheduler) updateQueueLenMetrics() {
	queueLenght.WithLabelValues("queued").Set(float64(len(s.queue)))
	queueLenght.WithLabelValues("inProgress").Set(float64(len(s.inProgress)))
}
