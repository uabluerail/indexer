package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	PDS                   models.ID `gorm:"index:rev_state_index,priority:2;index:was_indexed,priority:2"`
	DID                   string    `gorm:"uniqueIndex;column:did"`
	LastIndexedRev        string    `gorm:"index:rev_state_index,expression:(last_indexed_rev < first_rev_since_reset),priority:1;index:was_indexed,expression:(last_indexed_rev is null OR last_indexed_rev = ''),priority:1"`
	FirstRevSinceReset    string
	FirstCursorSinceReset int64
	TombstonedAt          time.Time
	LastIndexAttempt      time.Time
	LastError             string
	FailedAttempts        int `gorm:"default:0"`
}

type Record struct {
	ID         models.ID `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Repo       models.ID       `gorm:"index:idx_repo_record_key,unique,priority:1;not null;index:idx_repo_rev"`
	Collection string          `gorm:"index:idx_repo_record_key,unique,priority:2;not null"`
	Rkey       string          `gorm:"index:idx_repo_record_key,unique,priority:3"`
	AtRev      string          `gorm:"index:idx_repo_rev"`
	Content    json.RawMessage `gorm:"type:JSONB"`
	Deleted    bool
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Repo{}, &Record{})
}

func EnsureExists(ctx context.Context, db *gorm.DB, did string) (*Repo, error) {
	r := Repo{}
	if err := db.Model(&r).Where(&Repo{DID: did}).Take(&r).Error; err == nil {
		// Already have a row, just return it.
		return &r, nil
	} else {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("querying DB: %w", err)
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

	doc, err := resolver.GetDocument(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("fetching DID Document: %w", err)
	}

	pdsHost := ""
	for _, srv := range doc.Service {
		if srv.Type != "AtprotoPersonalDataServer" {
			continue
		}
		pdsHost = srv.ServiceEndpoint
	}
	if pdsHost == "" {
		return nil, fmt.Errorf("did not find any PDS in DID Document")
	}
	u, err := url.Parse(pdsHost)
	if err != nil {
		return nil, fmt.Errorf("PDS endpoint (%q) is an invalid URL: %w", pdsHost, err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("PDS endpoint (%q) doesn't have a host part", pdsHost)
	}
	remote := pds.PDS{Host: u.String()}
	if err := db.Model(&remote).Where(&pds.PDS{Host: remote.Host}).FirstOrCreate(&remote).Error; err != nil {
		return nil, fmt.Errorf("failed to get PDS record from DB for %q: %w", remote.Host, err)
	}
	r = Repo{DID: did, PDS: models.ID(remote.ID)}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&r).Where(&Repo{DID: r.DID}).FirstOrCreate(&r).Error; err != nil {
			return fmt.Errorf("looking for repo: %w", err)
		}
		if r.PDS != models.ID(remote.ID) {
			return tx.Model(&r).Select("FirstRevSinceReset").Updates(&Repo{FirstRevSinceReset: ""}).Error
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("upserting repo record: %w", err)
	}
	return &r, nil
}
