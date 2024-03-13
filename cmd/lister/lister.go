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
				Where("(disabled=false or disabled is null) and (last_list is null or last_list < ?)", time.Now().Add(-l.listRefreshInterval)).
				Take(&remote).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Error().Err(err).Msgf("Failed to query DB for a PDS to list repos from: %s", err)
				}
				break
			}

			if !pds.IsWhitelisted(remote.Host) {
				log.Info().Msgf("PDS %q is not whitelisted, disabling it", remote.Host)
				if err := db.Model(&remote).Where(&pds.PDS{ID: remote.ID}).Updates(&pds.PDS{Disabled: true}).Error; err != nil {
					log.Error().Err(err).Msgf("Failed to disable PDS %q: %s", remote.Host, err)
				}
				break
			}

			client := xrpcauth.NewAnonymousClient(ctx)
			client.Host = remote.Host

			log.Info().Msgf("Listing repos from %q...", remote.Host)
			repos, err := pagination.Reduce(
				func(cursor string) (resp *comatproto.SyncListRepos_Output, nextCursor string, err error) {
					resp, err = comatproto.SyncListRepos(ctx, client, cursor, 200)
					if err == nil && resp.Cursor != nil {
						nextCursor = *resp.Cursor
					}
					return
				},
				func(resp *comatproto.SyncListRepos_Output, acc []*comatproto.SyncListRepos_Repo) ([]*comatproto.SyncListRepos_Repo, error) {
					for _, repo := range resp.Repos {
						if repo == nil {
							continue
						}
						acc = append(acc, repo)
					}
					return acc, nil
				})

			if err != nil {
				log.Error().Err(err).Msgf("Failed to list repos from %q: %s", remote.Host, err)
				break
			}
			log.Info().Msgf("Received %d DIDs from %q", len(repos), remote.Host)
			reposListed.WithLabelValues(remote.Host).Add(float64(len(repos)))

			for _, repoInfo := range repos {
				record, created, err := repo.EnsureExists(ctx, l.db, repoInfo.Did)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to ensure that we have a record for the repo %q: %s", repoInfo.Did, err)
				} else if created {
					reposDiscovered.WithLabelValues(remote.Host).Inc()
				}

				if err == nil && record.FirstRevSinceReset == "" {
					// Populate this field in case it's empty, so we don't have to wait for the first firehose event
					// to trigger a resync.
					err := l.db.Transaction(func(tx *gorm.DB) error {
						var currentRecord repo.Repo
						if err := tx.Model(&record).Where(&repo.Repo{ID: record.ID}).Take(&currentRecord).Error; err != nil {
							return err
						}
						if currentRecord.FirstRevSinceReset != "" {
							// Someone else already updated it, nothing to do.
							return nil
						}
						var remote pds.PDS
						if err := tx.Model(&remote).Where(&pds.PDS{ID: record.PDS}).Take(&remote).Error; err != nil {
							return err
						}
						return tx.Model(&record).Where(&repo.Repo{ID: record.ID}).Updates(&repo.Repo{
							FirstRevSinceReset:    repoInfo.Rev,
							FirstCursorSinceReset: remote.FirstCursorSinceReset,
						}).Error
					})
					if err != nil {
						log.Error().Err(err).Msgf("Failed to set the initial FirstRevSinceReset value for %q: %s", repoInfo.Did, err)
					}
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
