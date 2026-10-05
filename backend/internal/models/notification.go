package models

import "time"

type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Type      string    `json:"type" gorm:"not null"` // email, application, system
	Title     string    `json:"title" gorm:"not null"`
	Message   string    `json:"message" gorm:"not null"`
	Read      bool      `json:"read" gorm:"default:false"`
	ActionURL string    `json:"action_url"`
	CreatedAt time.Time `json:"created_at"`
}
