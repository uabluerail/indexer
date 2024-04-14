package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/util/plc"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PLCLogEntry struct {
	ID        models.ID `gorm:"primarykey"`
	CreatedAt time.Time

	DID          string        `gorm:"column:did;index:did_timestamp;uniqueIndex:did_cid"`
	CID          string        `gorm:"column:cid;uniqueIndex:did_cid"`
	PLCTimestamp string        `gorm:"column:plc_timestamp;index:did_timestamp,sort:desc;index:,sort:desc"`
	Nullified    bool          `gorm:"default:false"`
	Operation    plc.Operation `gorm:"type:JSONB;serializer:json"`
}

type Mirror struct {
	db       *gorm.DB
	upstream *url.URL
	limiter  *rate.Limiter

	mu                   sync.RWMutex
	lastSuccessTimestamp time.Time
}

func NewMirror(ctx context.Context, upstream string, db *gorm.DB) (*Mirror, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	u.Path, err = url.JoinPath(u.Path, "export")
	if err != nil {
		return nil, err
	}
	return &Mirror{
		db:       db,
		upstream: u,
		// Current rate limit is `500 per five minutes`, lets stay a bit under it.
		limiter: rate.NewLimiter(rate.Limit(450.0/300), 4),
	}, nil
}

func (m *Mirror) Start(ctx context.Context) error {
	go m.run(ctx)
	return nil
}

func (m *Mirror) run(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("module", "mirror").Logger()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("PLC mirror stopped")
			return
		default:
			if err := m.runOnce(ctx); err != nil {
				if ctx.Err() == nil {
					log.Error().Err(err).Msgf("Failed to get new log entries from PLC: %s", err)
				}
			} else {
				now := time.Now()
				m.mu.Lock()
				m.lastSuccessTimestamp = now
				m.mu.Unlock()
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func (m *Mirror) LastSuccess() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSuccessTimestamp
}

func (m *Mirror) LastRecordTimestamp(ctx context.Context) (string, error) {
	ts := ""
	err := m.db.WithContext(ctx).Model(&PLCLogEntry{}).Select("plc_timestamp").Order("plc_timestamp desc").Limit(1).Take(&ts).Error
	return ts, err
}

func (m *Mirror) runOnce(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	cursor := ""
	err := m.db.Model(&PLCLogEntry{}).Select("plc_timestamp").Order("plc_timestamp desc").Limit(1).Take(&cursor).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get the cursor: %w", err)
	}

	u := *m.upstream

	for {
		params := u.Query()
		params.Set("count", "1000")
		if cursor != "" {
			params.Set("after", cursor)
		}
		u.RawQuery = params.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return fmt.Errorf("constructing request: %w", err)
		}

		_ = m.limiter.Wait(ctx)
		log.Info().Msgf("Listing PLC log entries with cursor %q...", cursor)
		log.Debug().Msgf("Request URL: %s", u.String())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		newEntries := []PLCLogEntry{}
		decoder := json.NewDecoder(resp.Body)
		oldCursor := cursor

		for {
			var entry plc.OperationLogEntry
			err := decoder.Decode(&entry)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("parsing log entry: %w", err)
			}

			cursor = entry.CreatedAt
			newEntries = append(newEntries, *FromOperationLogEntry(entry))
		}

		if len(newEntries) == 0 || cursor == oldCursor {
			break
		}

		err = m.db.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "did"}, {Name: "cid"}},
				DoNothing: true,
			},
		).Create(newEntries).Error
		if err != nil {
			return fmt.Errorf("inserting log entry into database: %w", err)
		}

		log.Info().Msgf("Got %d log entries. New cursor: %q", len(newEntries), cursor)
	}
	return nil
}

func FromOperationLogEntry(op plc.OperationLogEntry) *PLCLogEntry {
	return &PLCLogEntry{
		DID:          op.DID,
		CID:          op.CID,
		PLCTimestamp: op.CreatedAt,
		Nullified:    op.Nullified,
		Operation:    op.Operation,
	}
}
