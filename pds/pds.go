package pds

import (
	"time"

	"gorm.io/gorm"
)

type PDS struct {
	gorm.Model
	Host                  string `gorm:"uniqueIndex"`
	Cursor                int64
	FirstCursorSinceReset int64
	LastList              time.Time
	CrawlLimit            int
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PDS{})
}
