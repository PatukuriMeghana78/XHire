package models

import "time"

type GmailToken struct {
	ID             uint   `gorm:"primaryKey"`
	UserID         uint   `gorm:"not null"`
	AccessToken    string `gorm:"type:text;not null"`
	RefreshToken   string `gorm:"type:text"`
	DetectedStatus string // ✅ Add this
	EmailBody      string `gorm:"type:text"` // ✅ Add this
	Expiry         time.Time
	CreatedAt      time.Time
}
