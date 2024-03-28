package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/gorilla/websocket"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/util/resolver"
	"gorm.io/gorm"
)

type RelayConsumer struct {
	url string
	db  *gorm.DB
}

func NewRelayConsumer(ctx context.Context, host string, db *gorm.DB) (*RelayConsumer, error) {
	addr, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %s", host, err)
	}
	addr.Scheme = "wss"
	addr.Path = path.Join(addr.Path, "xrpc/com.atproto.sync.subscribeRepos")
	return &RelayConsumer{db: db, url: addr.String()}, nil
}

func (c *RelayConsumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *RelayConsumer) run(ctx context.Context) {
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

func (c *RelayConsumer) runOnce(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, http.Header{})
	if err != nil {
		return fmt.Errorf("establishing websocker connection: %w", err)
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, b, err := conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("websocket.ReadMessage: %w", err)
			}

			r := bytes.NewReader(b)
			proto := basicnode.Prototype.Any
			headerNode := proto.NewBuilder()
			if err := (&dagcbor.DecodeOptions{DontParseBeyondEnd: true}).Decode(headerNode, r); err != nil {
				return fmt.Errorf("unmarshaling message header: %w", err)
			}
			header, err := parseHeader(headerNode.Build())
			if err != nil {
				return fmt.Errorf("parsing message header: %w", err)
			}
			switch header.Op {
			case 1:
				if err := c.processMessage(ctx, header.Type, r); err != nil {
					log.Info().Err(err).Msgf("Relay consumer failed to process a message: %s", err)
				}
			case -1:
				bodyNode := proto.NewBuilder()
				if err := (&dagcbor.DecodeOptions{DontParseBeyondEnd: true, AllowLinks: true}).Decode(bodyNode, r); err != nil {
					return fmt.Errorf("unmarshaling message body: %w", err)
				}
				body, err := parseError(bodyNode.Build())
				if err != nil {
					return fmt.Errorf("parsing error payload: %w", err)
				}
				return &body
			default:
				log.Warn().Msgf("Unknown 'op' value received: %d", header.Op)
			}
		}
	}

}

func (c *RelayConsumer) processMessage(ctx context.Context, typ string, r io.Reader) error {
	log := zerolog.Ctx(ctx)

	did := ""

	switch typ {
	case "#commit":
		payload := &comatproto.SyncSubscribeRepos_Commit{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		did = payload.Repo

	case "#handle":
		payload := &comatproto.SyncSubscribeRepos_Handle{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		did = payload.Did

	case "#migrate":
		payload := &comatproto.SyncSubscribeRepos_Migrate{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		did = payload.Did

	case "#tombstone":
		payload := &comatproto.SyncSubscribeRepos_Tombstone{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		did = payload.Did

	case "#info":
		// Ignore
	case "#identity":
		payload := &comatproto.SyncSubscribeRepos_Identity{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		did = payload.Did

	default:
		b, err := io.ReadAll(r)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to read message payload: %s", err)
		}
		log.Warn().Msgf("Unknown message type received: %s payload=%q", typ, string(b))
	}

	if did == "" {
		return nil
	}

	u, err := resolver.GetPDSEndpoint(ctx, did)
	if err != nil {
		return err
	}
	_, err = pds.EnsureExists(ctx, c.db, u.String())

	return err
}
