package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"

	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/repo"
	"github.com/uabluerail/indexer/util/fix"
	"github.com/uabluerail/indexer/util/resolver"
)

type BadRecord struct {
	ID        models.ID `gorm:"primarykey"`
	CreatedAt time.Time
	PDS       models.ID `gorm:"index"`
	Cursor    int64
	Error     string
	Content   []byte
}

type Consumer struct {
	db     *gorm.DB
	remote pds.PDS

	lastCursorPersist time.Time
}

func NewConsumer(ctx context.Context, remote *pds.PDS, db *gorm.DB) (*Consumer, error) {
	if err := db.AutoMigrate(&BadRecord{}); err != nil {
		return nil, fmt.Errorf("db.AutoMigrate: %s", err)
	}

	return &Consumer{
		db:     db,
		remote: *remote,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	go c.run(ctx)
	return nil
}

func (c *Consumer) run(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("pds", c.remote.Host).Logger()
	ctx = log.WithContext(ctx)

	for {
		if err := c.runOnce(ctx); err != nil {
			log.Error().Err(err).Msgf("Consumer of %q failed (will be restarted): %s", c.remote.Host, err)
		}
		time.Sleep(time.Second)
	}
}

func (c *Consumer) runOnce(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	addr, err := url.Parse(c.remote.Host)
	if err != nil {
		return fmt.Errorf("parsing URL %q: %s", c.remote.Host, err)
	}
	addr.Scheme = "wss"
	addr.Path = path.Join(addr.Path, "xrpc/com.atproto.sync.subscribeRepos")

	if c.remote.Cursor > 0 {
		params := url.Values{"cursor": []string{fmt.Sprint(c.remote.Cursor)}}
		addr.RawQuery = params.Encode()
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, addr.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("establishing websocker connection: %w", err)
	}
	defer conn.Close()

	ch := make(chan bool)
	defer close(ch)
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ch:
				return
			case <-t.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Minute)); err != nil {
					log.Error().Err(err).Msgf("Failed to send ping: %s", err)
				}
			}
		}
	}()

	first := true
	for {
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
			if err := c.processMessage(ctx, header.Type, r, first); err != nil {
				const maxBadRecords = 500
				var count int64
				if err2 := c.db.Model(&BadRecord{}).Where(&BadRecord{PDS: c.remote.ID}).Count(&count).Error; err2 != nil {
					return err
				}

				if count >= maxBadRecords {
					return err
				}

				log.Error().Err(err).Str("pds", c.remote.Host).Msgf("Failed to process message at cursor %d: %s", c.remote.Cursor, err)
				err := c.db.Create(&BadRecord{
					PDS:     c.remote.ID,
					Cursor:  c.remote.Cursor,
					Error:   err.Error(),
					Content: b,
				}).Error
				if err != nil {
					return fmt.Errorf("failed to store bad message: %s", err)
				}
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
		first = false
	}
}

func (c *Consumer) checkForCursorReset(ctx context.Context, seq int64) error {
	// hack to detect cursor resets upon connection for implementations
	// that don't emit an explicit #info when connecting with an outdated cursor.

	if seq == c.remote.Cursor+1 {
		// No reset.
		return nil
	}

	return c.resetCursor(ctx, seq)
}

func (c *Consumer) resetCursor(ctx context.Context, seq int64) error {
	zerolog.Ctx(ctx).Warn().Str("pds", c.remote.Host).Msgf("Cursor reset: %d -> %d", c.remote.Cursor, seq)
	err := c.db.Model(&c.remote).
		Where(&pds.PDS{ID: c.remote.ID}).
		Updates(&pds.PDS{FirstCursorSinceReset: seq}).Error
	if err != nil {
		return fmt.Errorf("updating FirstCursorSinceReset: %w", err)
	}
	c.remote.FirstCursorSinceReset = seq
	return nil
}

func (c *Consumer) updateCursor(ctx context.Context, seq int64) error {
	if math.Abs(float64(seq-c.remote.Cursor)) < 100 && time.Since(c.lastCursorPersist) < 5*time.Second {
		c.remote.Cursor = seq
		return nil
	}

	err := c.db.Model(&c.remote).
		Where(&pds.PDS{ID: c.remote.ID}).
		Updates(&pds.PDS{Cursor: seq}).Error
	if err != nil {
		return fmt.Errorf("updating Cursor: %w", err)
	}
	c.remote.Cursor = seq
	return nil

}

func (c *Consumer) processMessage(ctx context.Context, typ string, r io.Reader, first bool) error {
	log := zerolog.Ctx(ctx)

	eventCounter.WithLabelValues(c.remote.Host, typ).Inc()

	switch typ {
	case "#commit":
		payload := &comatproto.SyncSubscribeRepos_Commit{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		exportEventTimestamp(ctx, c.remote.Host, payload.Time)

		if c.remote.FirstCursorSinceReset == 0 {
			if err := c.resetCursor(ctx, payload.Seq); err != nil {
				return fmt.Errorf("handling cursor reset: %w", err)
			}
		}
		if first {
			if err := c.checkForCursorReset(ctx, payload.Seq); err != nil {
				return err
			}
		}

		repoInfo, created, err := repo.EnsureExists(ctx, c.db, payload.Repo)
		if err != nil {
			return fmt.Errorf("repo.EnsureExists(%q): %w", payload.Repo, err)
		}
		if repoInfo.PDS != c.remote.ID {
			u, err := resolver.GetPDSEndpoint(ctx, payload.Repo)
			if err == nil {
				cur, err := pds.EnsureExists(ctx, c.db, u.String())
				if err == nil {
					if repoInfo.PDS != cur.ID {
						// Repo was migrated, lets update our record.
						err := c.db.Model(repoInfo).Where(&repo.Repo{ID: repoInfo.ID}).Updates(&repo.Repo{PDS: cur.ID}).Error
						if err != nil {
							log.Error().Err(err).Msgf("Repo %q was migrated to %q, but updating the repo has failed: %s", payload.Repo, cur.Host, err)
						}
					}
					repoInfo.PDS = cur.ID
				} else {
					log.Error().Err(err).Msgf("Failed to get PDS record for %q: %s", u, err)
				}
			} else {
				log.Error().Err(err).Msgf("Failed to get PDS endpoint for repo %q: %s", payload.Repo, err)
			}

			if repoInfo.PDS != c.remote.ID {
				// We checked a recent version of DID doc and this is still not a correct PDS.
				log.Error().Str("did", payload.Repo).Str("rev", payload.Rev).
					Msgf("Commit from an incorrect PDS, skipping")
				return nil
			}
		}
		if created {
			reposDiscovered.WithLabelValues(c.remote.Host).Inc()
		}

		// TODO: verify signature

		expectRecords := false
		deletions := []string{}
		for _, op := range payload.Ops {
			switch op.Action {
			case "create":
				expectRecords = true
			case "update":
				expectRecords = true
			case "delete":
				deletions = append(deletions, op.Path)
			}
		}
		for _, d := range deletions {
			parts := strings.SplitN(d, "/", 2)
			if len(parts) != 2 {
				continue
			}
			err := c.db.Model(&repo.Record{}).
				Where(&repo.Record{
					Repo:       models.ID(repoInfo.ID),
					Collection: parts[0],
					Rkey:       parts[1]}).
				Updates(&repo.Record{Deleted: true}).Error
			if err != nil {
				return fmt.Errorf("failed to mark %s/%s as deleted: %w", payload.Repo, d, err)
			}
		}

		newRecs, err := repo.ExtractRecords(ctx, bytes.NewReader(payload.Blocks))
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
			langs, _, err := repo.GetLang(ctx, v)
			if err == nil {
				for _, lang := range langs {
					postsByLanguageIndexed.WithLabelValues(c.remote.Host, lang).Inc()
				}
			}
			recs = append(recs, repo.Record{
				Repo:       models.ID(repoInfo.ID),
				Collection: parts[0],
				Rkey:       parts[1],
				// XXX: proper replacement of \u0000 would require full parsing of JSON
				// and recursive iteration over all string values, but this
				// should work well enough for now.
				Content: fix.EscapeNullCharForPostgres(v),
				AtRev:   payload.Rev,
			})
		}
		if len(recs) == 0 && expectRecords {
			log.Debug().Int64("seq", payload.Seq).Str("pds", c.remote.Host).Msgf("len(recs) == 0")
		}
		if len(recs) > 0 {
			err = c.db.Model(&repo.Record{}).
				Clauses(clause.OnConflict{
					Where: clause.Where{Exprs: []clause.Expression{clause.Or(
						clause.Eq{Column: clause.Column{Name: "at_rev", Table: "records"}, Value: nil},
						clause.Eq{Column: clause.Column{Name: "at_rev", Table: "records"}, Value: ""},
						clause.Lt{
							Column: clause.Column{Name: "at_rev", Table: "records"},
							Value:  clause.Column{Name: "at_rev", Table: "excluded"}},
					)}},
					DoUpdates: clause.AssignmentColumns([]string{"content", "at_rev"}),
					Columns:   []clause.Column{{Name: "repo"}, {Name: "collection"}, {Name: "rkey"}}}).
				Create(recs).Error
			if err != nil {
				return fmt.Errorf("inserting records into the database: %w", err)
			}
		}

		err = c.db.Model(&repo.Repo{}).Where(&repo.Repo{ID: repoInfo.ID}).
			Updates(&repo.Repo{
				LastFirehoseRev: payload.Rev,
			}).Error
		if err != nil {
			log.Error().Err(err).Msgf("Failed to update last_firehose_rev for %q: %s", repoInfo.DID, err)
		}

		if payload.TooBig {
			// Just trigger a re-index by resetting rev.
			err := c.db.Model(&repo.Repo{}).Where(&repo.Repo{ID: repoInfo.ID}).
				Updates(&repo.Repo{
					FirstCursorSinceReset: c.remote.FirstCursorSinceReset,
					FirstRevSinceReset:    payload.Rev,
				}).Error
			if err != nil {
				return fmt.Errorf("failed to update repo info after cursor reset: %w", err)
			}
		}

		if repoInfo.FirstCursorSinceReset != c.remote.FirstCursorSinceReset {
			err := c.db.Model(&repo.Repo{}).Debug().Where(&repo.Repo{ID: repoInfo.ID}).
				Updates(&repo.Repo{
					FirstCursorSinceReset: c.remote.FirstCursorSinceReset,
					FirstRevSinceReset:    payload.Rev,
				}).Error
			if err != nil {
				return fmt.Errorf("failed to update repo info after cursor reset: %w", err)
			}
		}

		if err := c.updateCursor(ctx, payload.Seq); err != nil {
			return err
		}
	case "#handle":
		payload := &comatproto.SyncSubscribeRepos_Handle{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		exportEventTimestamp(ctx, c.remote.Host, payload.Time)

		if c.remote.FirstCursorSinceReset == 0 {
			if err := c.resetCursor(ctx, payload.Seq); err != nil {
				return fmt.Errorf("handling cursor reset: %w", err)
			}
		}
		if first {
			if err := c.checkForCursorReset(ctx, payload.Seq); err != nil {
				return err
			}
		}
		// No-op, we don't store handles.
		if err := c.updateCursor(ctx, payload.Seq); err != nil {
			return err
		}
	case "#migrate":
		payload := &comatproto.SyncSubscribeRepos_Migrate{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		exportEventTimestamp(ctx, c.remote.Host, payload.Time)

		if c.remote.FirstCursorSinceReset == 0 {
			if err := c.resetCursor(ctx, payload.Seq); err != nil {
				return fmt.Errorf("handling cursor reset: %w", err)
			}
		}
		if first {
			if err := c.checkForCursorReset(ctx, payload.Seq); err != nil {
				return err
			}
		}

		log.Debug().Interface("payload", payload).Str("did", payload.Did).Msgf("MIGRATION")
		// TODO
		if err := c.updateCursor(ctx, payload.Seq); err != nil {
			return err
		}
	case "#tombstone":
		payload := &comatproto.SyncSubscribeRepos_Tombstone{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		exportEventTimestamp(ctx, c.remote.Host, payload.Time)

		if c.remote.FirstCursorSinceReset == 0 {
			if err := c.resetCursor(ctx, payload.Seq); err != nil {
				return fmt.Errorf("handling cursor reset: %w", err)
			}
		}
		if first {
			if err := c.checkForCursorReset(ctx, payload.Seq); err != nil {
				return err
			}
		}
		// TODO
		if err := c.updateCursor(ctx, payload.Seq); err != nil {
			return err
		}
	case "#info":
		payload := &comatproto.SyncSubscribeRepos_Info{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}
		switch payload.Name {
		case "OutdatedCursor":
			if !first {
				log.Warn().Msgf("Received cursor reset notification in the middle of a stream: %+v", payload)
			}
			c.remote.FirstCursorSinceReset = 0
		default:
			log.Error().Msgf("Unknown #info message %q: %+v", payload.Name, payload)
		}
	case "#identity":
		payload := &comatproto.SyncSubscribeRepos_Identity{}
		if err := payload.UnmarshalCBOR(r); err != nil {
			return fmt.Errorf("failed to unmarshal commit: %w", err)
		}

		exportEventTimestamp(ctx, c.remote.Host, payload.Time)
		log.Trace().Str("did", payload.Did).Str("type", typ).Int64("seq", payload.Seq).
			Msgf("#identity message: %s seq=%d time=%q", payload.Did, payload.Seq, payload.Time)

		resolver.Resolver.FlushCacheFor(payload.Did)

		// TODO: fetch DID doc and update PDS field?
	default:
		b, err := io.ReadAll(r)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to read message payload: %s", err)
		}
		log.Warn().Msgf("Unknown message type received: %s payload=%q", typ, string(b))
	}
	return nil
}

type Header struct {
	Op   int64
	Type string
}

func parseHeader(node datamodel.Node) (Header, error) {
	r := Header{}
	op, err := node.LookupByString("op")
	if err != nil {
		return r, fmt.Errorf("missing 'op': %w", err)
	}
	r.Op, err = op.AsInt()
	if err != nil {
		return r, fmt.Errorf("op.AsInt(): %w", err)
	}
	if r.Op == -1 {
		// Error frame, type should not be present
		return r, nil
	}
	t, err := node.LookupByString("t")
	if err != nil {
		return r, fmt.Errorf("missing 't': %w", err)
	}
	r.Type, err = t.AsString()
	if err != nil {
		return r, fmt.Errorf("t.AsString(): %w", err)
	}
	return r, nil
}

func parseError(node datamodel.Node) (xrpc.XRPCError, error) {
	r := xrpc.XRPCError{}
	e, err := node.LookupByString("error")
	if err != nil {
		return r, fmt.Errorf("missing 'error': %w", err)
	}
	r.ErrStr, err = e.AsString()
	if err != nil {
		return r, fmt.Errorf("error.AsString(): %w", err)
	}
	m, err := node.LookupByString("message")
	if err == nil {
		r.Message, err = m.AsString()
		if err != nil {
			return r, fmt.Errorf("message.AsString(): %w", err)
		}
	} else if !errors.Is(err, datamodel.ErrNotExists{}) {
		return r, fmt.Errorf("looking up 'message': %w", err)
	}

	return r, nil
}

func exportEventTimestamp(ctx context.Context, remote string, timestamp string) {
	if t, err := time.Parse(time.RFC3339, timestamp); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("pds", remote).Msgf("Failed to parse %q as a timestamp: %s", timestamp, err)
	} else {
		lastEventTimestamp.WithLabelValues(remote).Set(float64(t.Unix()))
	}
}
