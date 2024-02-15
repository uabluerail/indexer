package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/pds"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

const defaultRateLimit = 10

type Limiter struct {
	mu      sync.RWMutex
	db      *gorm.DB
	limiter map[string]*rate.Limiter
}

func NewLimiter(db *gorm.DB) (*Limiter, error) {
	remotes := []pds.PDS{}

	if err := db.Find(&remotes).Error; err != nil {
		return nil, fmt.Errorf("failed to get the list of known PDSs: %w", err)
	}

	l := &Limiter{
		db:      db,
		limiter: map[string]*rate.Limiter{},
	}

	for _, remote := range remotes {
		limit := remote.CrawlLimit
		if limit == 0 {
			limit = defaultRateLimit
		}
		l.limiter[remote.Host] = rate.NewLimiter(rate.Limit(limit), limit*2)
	}
	return l, nil
}

func (l *Limiter) getLimiter(name string) *rate.Limiter {
	l.mu.RLock()
	limiter := l.limiter[name]
	l.mu.RUnlock()

	if limiter != nil {
		return limiter
	}

	limiter = rate.NewLimiter(defaultRateLimit, defaultRateLimit*2)
	l.mu.Lock()
	l.limiter[name] = limiter
	l.mu.Unlock()
	return limiter
}

func (l *Limiter) Wait(ctx context.Context, name string) error {
	return l.getLimiter(name).Wait(ctx)
}

func (l *Limiter) SetLimit(ctx context.Context, name string, limit rate.Limit) {
	l.getLimiter(name).SetLimit(limit)
	err := l.db.Model(&pds.PDS{}).Where(&pds.PDS{Host: name}).Updates(&pds.PDS{CrawlLimit: int(limit)}).Error
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msgf("Failed to persist rate limit change for %q: %s", name, err)
	}
}

func (l *Limiter) SetAllLimits(ctx context.Context, limit rate.Limit) {
	l.mu.RLock()
	for name, limiter := range l.limiter {
		limiter.SetLimit(limit)
		err := l.db.Model(&pds.PDS{}).Where(&pds.PDS{Host: name}).Updates(&pds.PDS{CrawlLimit: int(limit)}).Error
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("Failed to persist rate limit change for %q: %s", name, err)
		}
	}
	l.mu.RUnlock()
}
