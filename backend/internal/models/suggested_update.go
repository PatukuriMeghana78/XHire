package models

import "time"

type SuggestedUpdate struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ApplicationID   uint      `json:"application_id"`
	SuggestedStatus string    `json:"suggested_status"`
	ConfidenceScore float64   `json:"confidence_score"`
	Reason          string    `json:"reason"`
	Confirmed       bool      `json:"confirmed" gorm:"default:false"`
	CreatedAt       time.Time `json:"created_at"`
}
