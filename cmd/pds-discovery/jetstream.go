package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog"
	"gorm.io/gorm"

	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"

	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/util/resolver"
)

type JetstreamConsumer struct {
	url   string
	db    *gorm.DB
	cache *ristretto.Cache[string, struct{}]
}

func NewJetstreamConsumer(ctx context.Context, host string, db *gorm.DB) (*JetstreamConsumer, error) {
	addr, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %s", host, err)
	}
	// Fixup protocol name, just in case.
	switch addr.Scheme {
	case "http":
		addr.Scheme = "ws"
	case "https":
		addr.Scheme = "wss"
	}
	addr.Path = path.Join(addr.Path, "subscribe")
	cache, err := ristretto.NewCache(&ristretto.Config[string, struct{}]{
		MaxCost:     100_000,
		NumCounters: 1_000_000,
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}
	return &JetstreamConsumer{
		db:    db,
		url:   addr.String(),
		cache: cache,
	}, nil
}

func (c *JetstreamConsumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *JetstreamConsumer) run(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("relay", c.url).Logger()
	ctx = log.WithContext(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("Relay consumer stopped")
			return
		default:
			if err := c.runOnce(ctx); err != nil {
				log.Error().Err(err).Msgf("Consumer of relay %q failed (will be restarted): %s", c.url, err)
			}
			time.Sleep(time.Second)
		}
	}
}

func (c *JetstreamConsumer) runOnce(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	slog := slog.New(slogzerolog.Option{Level: slog.LevelDebug, Logger: log}.NewZerologHandler())

	cfg := client.DefaultClientConfig()
	cfg.Compress = true
	cfg.WebsocketURL = c.url
	jetstream, err := client.NewClient(cfg, slog, sequential.NewScheduler("uabluerail/indexer/pds-discovery", slog, c.handleEvent))

	if err != nil {
		return fmt.Errorf("creating jetstream client: %w", err)
	}

	return jetstream.ConnectAndRead(ctx, nil)
}

func (c *JetstreamConsumer) handleEvent(ctx context.Context, event *models.Event) error {
	if event.Did == "" {
		return nil
	}

	_, found := c.cache.Get(event.Did)
	if found {
		return nil
	}

	u, _, err := resolver.GetPDSEndpointAndPublicKey(ctx, event.Did)
	if err != nil {
		return err
	}
	_, err = pds.EnsureExists(ctx, c.db, u.String())
	if err != nil {
		return err
	}

	c.cache.Set(event.Did, struct{}{}, 1)
	return nil
}
