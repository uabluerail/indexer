package pds

import (
	"time"

	"gorm.io/gorm"

	"github.com/uabluerail/indexer/models"
)

type PDS struct {
	ID                    models.ID `gorm:"primarykey"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Host                  string `gorm:"uniqueIndex"`
	Cursor                int64
	FirstCursorSinceReset int64
	LastList              time.Time
	CrawlLimit            int
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PDS{})
}
