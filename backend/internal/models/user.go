package models

import "gorm.io/gorm"

const (
	RoleCandidate = "candidate"
	RoleRecruiter = "recruiter"
)

type User struct {
	gorm.Model
	Email          string `gorm:"uniqueIndex" json:"email"`
	Password       string `json:"-"`
	FullName       string `json:"fullName"`
	Role           string `gorm:"index;default:candidate" json:"role"`
	GmailToken     string `json:"-"`
	GmailConnected bool   `json:"gmail_connected"`
}

func (u User) IsCandidate() bool {
	return u.Role == RoleCandidate || u.Role == ""
}

func (u User) IsRecruiter() bool {
	return u.Role == RoleRecruiter
}
