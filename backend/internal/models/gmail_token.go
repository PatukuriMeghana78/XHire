package models

import "time"

type GmailToken struct {
	ID             uint   `gorm:"primaryKey"`
	UserID         uint   `gorm:"not null"`
	AccessToken    string `gorm:"type:text;not null" json:"-"`
	RefreshToken   string `gorm:"type:text" json:"-"`
	DetectedStatus string // ✅ Add this
	EmailBody      string `gorm:"type:text"` // ✅ Add this
	Expiry         time.Time
	CreatedAt      time.Time
	// NeedsReconnect records that Google no longer accepts the stored grant (revoked,
	// expired, or issued without a refresh token). It is persisted so the poller stays
	// silent across restarts instead of retrying a dead grant forever.
	NeedsReconnect  bool   `gorm:"not null;default:false"`
	ReconnectReason string `gorm:"type:text"`
}
