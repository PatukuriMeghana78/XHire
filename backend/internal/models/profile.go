package models

import "gorm.io/gorm"

type CandidateProfile struct {
	gorm.Model
	UserID          uint        `gorm:"uniqueIndex;not null" json:"userId"`
	FullName        string      `json:"fullName"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone,omitempty"`
	Headline        string      `json:"headline,omitempty"`
	Location        string      `json:"location,omitempty"`
	ExperienceYears *int        `json:"experienceYears,omitempty"`
	Skills          StringSlice `gorm:"type:text" json:"skills"`
	Education       string      `json:"education,omitempty"`
	About           string      `gorm:"type:text" json:"about,omitempty"`
	LinkedinURL     string      `json:"linkedinUrl,omitempty"`
	GithubURL       string      `json:"githubUrl,omitempty"`
	ResumeID        *uint       `json:"resumeId,omitempty"`
}

type RecruiterProfile struct {
	gorm.Model
	UserID         uint   `gorm:"uniqueIndex;not null" json:"userId"`
	FullName       string `json:"fullName"`
	Email          string `json:"email"`
	CompanyName    string `json:"companyName"`
	CompanyWebsite string `json:"companyWebsite,omitempty"`
	CompanySize    string `json:"companySize,omitempty"`
	Industry       string `json:"industry,omitempty"`
	Location       string `json:"location,omitempty"`
	About          string `gorm:"type:text" json:"about,omitempty"`
}

type Resume struct {
	gorm.Model
	CandidateID uint   `gorm:"index;not null" json:"candidateId"`
	FileName    string `json:"fileName"`
	FilePath    string `json:"-"`
	FileURL     string `json:"fileUrl,omitempty"`
	SizeKb      int    `json:"sizeKb,omitempty"`
}
