package pds

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	Disabled              bool `gorm:"default:false"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PDS{})
}

func NormalizeHost(host string) string {
	return strings.TrimRight(host, "/")
}

func EnsureExists(ctx context.Context, db *gorm.DB, host string) (*PDS, error) {
	host = NormalizeHost(host)
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
	host = NormalizeHost(host)
	for _, p := range whitelist {
		if match, _ := filepath.Match(p, host); match {
			return true
		}
	}
	return false
}
