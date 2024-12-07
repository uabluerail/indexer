package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"time"

	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/util/resolver"
	"gorm.io/gorm"
)

type JetstreamConsumer struct {
	url string
	db  *gorm.DB
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
	return &JetstreamConsumer{db: db, url: addr.String()}, nil
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

	jetstream, err := client.NewClient(&client.ClientConfig{
		Compress:     true,
		WebsocketURL: c.url,
	}, slog, sequential.NewScheduler("uabluerail/indexer/pds-discovery", slog, c.handleEvent))

	if err != nil {
		return fmt.Errorf("creating jetstream client: %w", err)
	}

	return jetstream.ConnectAndRead(ctx, nil)
}

func (c *JetstreamConsumer) handleEvent(ctx context.Context, event *models.Event) error {
	if event.Did == "" {
		return nil
	}

	// TODO: add some in-process caching to avoid needlessly repeating the same
	// query.
	u, _, err := resolver.GetPDSEndpointAndPublicKey(ctx, event.Did)
	if err != nil {
		return err
	}
	_, err = pds.EnsureExists(ctx, c.db, u.String())

	return err
}
