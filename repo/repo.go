package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/uabluerail/indexer/models"
	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/util/resolver"
)

type Repo struct {
	ID                    models.ID `gorm:"primarykey"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	PDS                   models.ID `gorm:"default:0;index:rev_state_index,priority:2;index:was_indexed,priority:2;index:indexed_count,priority:1,where:((failed_attempts < 3) AND (last_indexed_rev <> ''::text) AND ((last_indexed_rev >= first_rev_since_reset) OR (first_rev_since_reset IS NULL) OR (first_rev_since_reset = ''::text)))"`
	DID                   string    `gorm:"uniqueIndex;column:did"`
	LastIndexedRev        string    `gorm:"index:rev_state_index,expression:(last_indexed_rev < first_rev_since_reset),priority:1;index:was_indexed,expression:(last_indexed_rev is null OR last_indexed_rev = ''),priority:1"`
	FirstRevSinceReset    string
	LastFirehoseRev       string
	FirstCursorSinceReset int64 `gorm:"index:indexed_count,priority:2"`
	TombstonedAt          time.Time
	LastIndexAttempt      time.Time
	LastError             string
	FailedAttempts        int `gorm:"default:0"`
	LastKnownKey          string
}

type Record struct {
	ID         models.ID `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time       `gorm:"autoUpdateTime:false"`
	Repo       models.ID       `gorm:"index:idx_repo_record_key,unique,priority:1;not null;index:idx_repo_rev"`
	Collection string          `gorm:"index:idx_repo_record_key,unique,priority:2;not null"`
	Rkey       string          `gorm:"index:idx_repo_record_key,unique,priority:3"`
	AtRev      string          `gorm:"index:idx_repo_rev"`
	Content    json.RawMessage `gorm:"type:JSONB"`
	Deleted    bool            `gorm:"default:false"`
}

type BadRecord struct {
	ID        models.ID `gorm:"primarykey"`
	CreatedAt time.Time
	PDS       models.ID `gorm:"index"`
	Cursor    int64
	Error     string
	Content   []byte
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Repo{}, &Record{}, &BadRecord{})
}

func EnsureExists(ctx context.Context, db *gorm.DB, did string) (*Repo, bool, error) {
	r := Repo{}
	if err := db.Model(&r).Where(&Repo{DID: did}).Take(&r).Error; err == nil {
		// Already have a row, just return it.
		return &r, false, nil
	} else {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, fmt.Errorf("querying DB: %w", err)
		}
	}

	// No row yet, so we need to create one (keeping in mind that it can be created
	// concurrently by someone else).
	// 1) resolve did (i.e., query PLC)
	// 2) get PDS address from didDoc and ensure we have a record for it
	// 3) in a transaction, check if we have a record for the repo
	//     if we don't - just create a record
	//     if we do - compare PDS IDs
	//        if they don't match - also reset FirstRevSinceReset

	u, pubKey, err := resolver.GetPDSEndpointAndPublicKey(ctx, did)
	if err != nil {
		return nil, false, fmt.Errorf("fetching DID Document: %w", err)
	}

	remote, err := pds.EnsureExists(ctx, db, u.String())
	if err != nil {
		return nil, false, fmt.Errorf("failed to get PDS record from DB for %q: %w", u.String(), err)
	}
	r = Repo{
		DID:          did,
		PDS:          models.ID(remote.ID),
		LastKnownKey: pubKey,
	}
	created := false
	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&r).Where(&Repo{DID: r.DID}).FirstOrCreate(&r)
		if err := result.Error; err != nil {
			return fmt.Errorf("looking for repo: %w", err)
		}
		if r.PDS != models.ID(remote.ID) {
			return tx.Model(&r).Select("FirstRevSinceReset").Updates(&Repo{FirstRevSinceReset: ""}).Error
		}
		created = result.RowsAffected > 0
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("upserting repo record: %w", err)
	}
	return &r, created, nil
}
