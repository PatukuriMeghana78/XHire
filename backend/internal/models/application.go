package models

import (
	"time"

	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	UserID      uint  `json:"user_id" gorm:"index"`
	CandidateID uint  `json:"candidate_id" gorm:"index"`
	JobID       *uint `json:"job_id,omitempty" gorm:"index"`

	Company     string    `json:"company"`
	Role        string    `json:"role"`
	AppliedDate time.Time `json:"applied_date"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
	FollowUpSent bool     `json:"follow_up_sent"`

	CoverLetter string `gorm:"type:text" json:"cover_letter,omitempty"`
	ResumeID    *uint  `json:"resume_id,omitempty"`
	MatchScore  int    `json:"match_score"`

	StatusHistory []StatusHistory `json:"history,omitempty" gorm:"foreignKey:ApplicationID"`
	Job           *Job            `gorm:"foreignKey:JobID" json:"-"`

	HealthScore     int       `json:"health_score"`
	HealthColor     string    `json:"health_color"`
	HealthFactors   string    `json:"health_factors"`
	HealthUpdatedAt time.Time `json:"health_updated_at"`
}

type StatusHistory struct {
	gorm.Model
	ApplicationID uint      `json:"application_id"`
	Status        string    `json:"status"`
	ChangedAt     time.Time `json:"date"`
	Source        string    `json:"source"`
}

type EmailEvent struct {
	gorm.Model
	UserID          uint      `json:"user_id"`
	EmailID         string    `json:"email_id"`
	DetectedCompany string    `json:"detected_company"`
	DetectedRole    string    `json:"detected_role"`
	SuggestedStatus string    `json:"suggested_status"`
	Confidence      int       `json:"confidence"`
	ProcessedAt     time.Time `json:"processed_at"`
	Confirmed       bool      `json:"confirmed"`
	ApplicationID   *uint     `json:"application_id,omitempty"`
}

type HealthScoreRequest struct {
	ApplicationID uint      `json:"application_id"`
	Company       string    `json:"company"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	AppliedDate   time.Time `json:"applied_date"`
	LastUpdated   time.Time `json:"last_updated"`
	FollowUpSent  bool      `json:"follow_up_sent"`
	EmailSummary  string    `json:"email_summary"`
}

type HealthScoreResponse struct {
	Score   int            `json:"score"`
	Color   string         `json:"color"`
	Factors map[string]int `json:"factors"`
	Reason  string         `json:"reason"`
}
