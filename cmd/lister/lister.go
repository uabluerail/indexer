package main

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/did"

	"github.com/uabluerail/bsky-tools/pagination"
	"github.com/uabluerail/bsky-tools/xrpcauth"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/repo"
	"github.com/uabluerail/indexer/util/resolver"
)

type Lister struct {
	db       *gorm.DB
	resolver did.Resolver

	pollInterval        time.Duration
	listRefreshInterval time.Duration
}

func NewLister(ctx context.Context, db *gorm.DB) (*Lister, error) {
	return &Lister{
		db:                  db,
		resolver:            resolver.Resolver,
		pollInterval:        5 * time.Minute,
		listRefreshInterval: 24 * time.Hour,
	}, nil
}

func (l *Lister) Start(ctx context.Context) error {
	go l.run(ctx)
	return nil
}

func (l *Lister) run(ctx context.Context) {
	log := zerolog.Ctx(ctx)
	ticker := time.NewTicker(l.pollInterval)

	log.Info().Msgf("Lister starting...")
	t := make(chan time.Time, 1)
	t <- time.Now()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("Lister stopped (context expired)")
			return
		case <-t:
			db := l.db.WithContext(ctx)

			remote := pds.PDS{}
			if err := db.Model(&remote).
				Where("last_list is null or last_list < ?", time.Now().Add(-l.listRefreshInterval)).
				Take(&remote).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Error().Err(err).Msgf("Failed to query DB for a PDS to list repos from: %s", err)
				}
				break
			}
			client := xrpcauth.NewAnonymousClient(ctx)
			client.Host = remote.Host

			log.Info().Msgf("Listing repos from %q...", remote.Host)
			dids, err := pagination.Reduce(
				func(cursor string) (resp *comatproto.SyncListRepos_Output, nextCursor string, err error) {
					resp, err = comatproto.SyncListRepos(ctx, client, cursor, 200)
					if err == nil && resp.Cursor != nil {
						nextCursor = *resp.Cursor
					}
					return
				},
				func(resp *comatproto.SyncListRepos_Output, acc []string) ([]string, error) {
					for _, repo := range resp.Repos {
						if repo == nil {
							continue
						}
						acc = append(acc, repo.Did)
					}
					return acc, nil
				})

			if err != nil {
				log.Error().Err(err).Msgf("Failed to list repos from %q: %s", remote.Host, err)
				break
			}
			log.Info().Msgf("Received %d DIDs from %q", len(dids), remote.Host)

			for _, did := range dids {
				if _, err := repo.EnsureExists(ctx, l.db, did); err != nil {
					log.Error().Err(err).Msgf("Failed to ensure that we have a record for the repo %q: %s", did, err)
				}
			}

			if err := db.Model(&remote).Updates(&pds.PDS{LastList: time.Now()}).Error; err != nil {
				log.Error().Err(err).Msgf("Failed to update the timestamp of last list for %q: %s", remote.Host, err)
			}
		case v := <-ticker.C:
			t <- v
		}
	}
}
