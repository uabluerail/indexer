package pds

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/uabluerail/indexer/models"
)

const Unknown models.ID = 0

var whitelist []string = []string{
	"https://bsky.social",
	"https://*.bsky.network",
	"https://*",
}

type PDS struct {
	ID                    models.ID `gorm:"primarykey"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Host                  string `gorm:"uniqueIndex"`
	Cursor                int64
	FirstCursorSinceReset int64
	LastList              time.Time
	CrawlLimit            int
	Disabled              bool
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PDS{})
}

func EnsureExists(ctx context.Context, db *gorm.DB, host string) (*PDS, error) {
	if !IsWhitelisted(host) {
		return nil, fmt.Errorf("host %q is not whitelisted", host)
	}
	remote := PDS{Host: host}
	if err := db.Model(&remote).Where(&PDS{Host: host}).FirstOrCreate(&remote).Error; err != nil {
		return nil, fmt.Errorf("failed to get PDS record from DB for %q: %w", remote.Host, err)
	}
	return &remote, nil
}

func IsWhitelisted(host string) bool {
	for _, p := range whitelist {
		if match, _ := filepath.Match(p, host); match {
			return true
		}
	}
	return false
}
