package models

import "gorm.io/gorm"

type Job struct {
	gorm.Model
	RecruiterID      uint        `gorm:"index;not null" json:"recruiterId"`
	Title            string      `json:"title"`
	CompanyName      string      `json:"companyName"`
	Location         string      `json:"location"`
	WorkMode         string      `json:"workMode"`
	JobType          string      `json:"jobType"`
	SalaryMin        *int        `json:"salaryMin,omitempty"`
	SalaryMax        *int        `json:"salaryMax,omitempty"`
	ExperienceMin    *int        `json:"experienceMin,omitempty"`
	Description      string      `gorm:"type:text" json:"description"`
	Responsibilities StringSlice `gorm:"type:text" json:"responsibilities,omitempty"`
	RequiredSkills   StringSlice `gorm:"type:text" json:"requiredSkills"`
	Status           string      `gorm:"index;default:open" json:"status"`
	Recruiter        User        `gorm:"foreignKey:RecruiterID" json:"-"`
	Applications     []Application `gorm:"foreignKey:JobID" json:"-"`
}
