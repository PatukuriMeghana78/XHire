package dto

import (
	"strconv"
	"strings"
	"time"

	"job_application_analysis/internal/models"
)

func ID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func ParseID(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	return uint(n), err
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func UserFromModel(u *models.User) UserResponse {
	role := u.Role
	if role == "" {
		role = models.RoleCandidate
	}
	return UserResponse{
		ID:       ID(u.ID),
		Email:    u.Email,
		FullName: u.FullName,
		Role:     role,
	}
}

type JobResponse struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	CompanyName      string   `json:"companyName"`
	Location         string   `json:"location"`
	WorkMode         string   `json:"workMode"`
	JobType          string   `json:"jobType"`
	SalaryMin        *int     `json:"salaryMin,omitempty"`
	SalaryMax        *int     `json:"salaryMax,omitempty"`
	ExperienceMin    *int     `json:"experienceMin,omitempty"`
	Description      string   `json:"description"`
	Responsibilities []string `json:"responsibilities,omitempty"`
	RequiredSkills   []string `json:"requiredSkills"`
	PostedAt         string   `json:"postedAt"`
	Status           string   `json:"status"`
	ApplicantCount   int      `json:"applicantCount,omitempty"`
	RecruiterID      string   `json:"recruiterId,omitempty"`
}

func JobFromModel(j *models.Job, applicantCount int) JobResponse {
	skills := []string(j.RequiredSkills)
	if skills == nil {
		skills = []string{}
	}
	return JobResponse{
		ID:               ID(j.ID),
		Title:            j.Title,
		CompanyName:      j.CompanyName,
		Location:         j.Location,
		WorkMode:         j.WorkMode,
		JobType:          j.JobType,
		SalaryMin:        j.SalaryMin,
		SalaryMax:        j.SalaryMax,
		ExperienceMin:    j.ExperienceMin,
		Description:      j.Description,
		Responsibilities: j.Responsibilities,
		RequiredSkills:   skills,
		PostedAt:         j.CreatedAt.UTC().Format(time.RFC3339),
		Status:           j.Status,
		ApplicantCount:   applicantCount,
		RecruiterID:      ID(j.RecruiterID),
	}
}

type TimelineItem struct {
	Status string `json:"status"`
	At     string `json:"at"`
	Note   string `json:"note,omitempty"`
}

type ApplicationResponse struct {
	ID            string         `json:"id"`
	JobID         string         `json:"jobId"`
	JobTitle      string         `json:"jobTitle"`
	CompanyName   string         `json:"companyName"`
	CandidateID   string         `json:"candidateId"`
	CandidateName string         `json:"candidateName,omitempty"`
	Status        string         `json:"status"`
	AppliedAt     string         `json:"appliedAt"`
	UpdatedAt     string         `json:"updatedAt,omitempty"`
	MatchScore    int            `json:"matchScore,omitempty"`
	Timeline      []TimelineItem `json:"timeline,omitempty"`
}

func NormalizeStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	switch s {
	case "underreview", "review":
		return "under-review"
	case "offer":
		return "hired"
	default:
		return s
	}
}

// ValidStatusUpdate reports whether an incoming application status is one of
// the platform's stages. Values are stored as sent, so this only gates writes;
// free-form strings would otherwise leak into status analytics.
func ValidStatusUpdate(status string) bool {
	switch NormalizeStatus(status) {
	case "applied", "under-review", "shortlisted", "interview", "rejected", "hired":
		return true
	default:
		return false
	}
}

func ApplicationFromModel(app *models.Application, candidateName string) ApplicationResponse {
	jobID := ""
	jobTitle := app.Role
	company := app.Company
	if app.JobID != nil {
		jobID = ID(*app.JobID)
	}
	if app.Job != nil {
		jobTitle = app.Job.Title
		company = app.Job.CompanyName
		jobID = ID(app.Job.ID)
	}
	candidateID := app.CandidateID
	if candidateID == 0 {
		candidateID = app.UserID
	}
	timeline := make([]TimelineItem, 0, len(app.StatusHistory))
	for _, h := range app.StatusHistory {
		timeline = append(timeline, TimelineItem{
			Status: NormalizeStatus(h.Status),
			At:     h.ChangedAt.UTC().Format(time.RFC3339),
			Note:   h.Source,
		})
	}
	return ApplicationResponse{
		ID:            ID(app.ID),
		JobID:         jobID,
		JobTitle:      jobTitle,
		CompanyName:   company,
		CandidateID:   ID(candidateID),
		CandidateName: candidateName,
		Status:        NormalizeStatus(app.Status),
		AppliedAt:     app.AppliedDate.UTC().Format(time.RFC3339),
		UpdatedAt:     app.LastUpdated.UTC().Format(time.RFC3339),
		MatchScore:    app.MatchScore,
		Timeline:      timeline,
	}
}

type CandidateProfileResponse struct {
	ID              string   `json:"id"`
	UserID          string   `json:"userId"`
	FullName        string   `json:"fullName"`
	Email           string   `json:"email"`
	Phone           string   `json:"phone,omitempty"`
	Headline        string   `json:"headline,omitempty"`
	Location        string   `json:"location,omitempty"`
	ExperienceYears *int     `json:"experienceYears,omitempty"`
	Skills          []string `json:"skills"`
	Education       string   `json:"education,omitempty"`
	About           string   `json:"about,omitempty"`
	LinkedinURL     string   `json:"linkedinUrl,omitempty"`
	GithubURL       string   `json:"githubUrl,omitempty"`
	ResumeID        string   `json:"resumeId,omitempty"`
}

func CandidateFromModel(p *models.CandidateProfile) CandidateProfileResponse {
	skills := []string(p.Skills)
	if skills == nil {
		skills = []string{}
	}
	resumeID := ""
	if p.ResumeID != nil {
		resumeID = ID(*p.ResumeID)
	}
	return CandidateProfileResponse{
		ID:              ID(p.ID),
		UserID:          ID(p.UserID),
		FullName:        p.FullName,
		Email:           p.Email,
		Phone:           p.Phone,
		Headline:        p.Headline,
		Location:        p.Location,
		ExperienceYears: p.ExperienceYears,
		Skills:          skills,
		Education:       p.Education,
		About:           p.About,
		LinkedinURL:     p.LinkedinURL,
		GithubURL:       p.GithubURL,
		ResumeID:        resumeID,
	}
}

type RecruiterProfileResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	FullName       string `json:"fullName"`
	Email          string `json:"email"`
	CompanyName    string `json:"companyName"`
	CompanyWebsite string `json:"companyWebsite,omitempty"`
	CompanySize    string `json:"companySize,omitempty"`
	Industry       string `json:"industry,omitempty"`
	Location       string `json:"location,omitempty"`
	About          string `json:"about,omitempty"`
}

func RecruiterFromModel(p *models.RecruiterProfile) RecruiterProfileResponse {
	return RecruiterProfileResponse{
		ID:             ID(p.ID),
		UserID:         ID(p.UserID),
		FullName:       p.FullName,
		Email:          p.Email,
		CompanyName:    p.CompanyName,
		CompanyWebsite: p.CompanyWebsite,
		CompanySize:    p.CompanySize,
		Industry:       p.Industry,
		Location:       p.Location,
		About:          p.About,
	}
}

type ResumeResponse struct {
	ID          string `json:"id"`
	CandidateID string `json:"candidateId"`
	FileName    string `json:"fileName"`
	FileURL     string `json:"fileUrl,omitempty"`
	UploadedAt  string `json:"uploadedAt"`
	SizeKb      int    `json:"sizeKb,omitempty"`
}

func ResumeFromModel(r *models.Resume) ResumeResponse {
	// Expose the authenticated download endpoint rather than a raw file path.
	// The old "/uploads/resumes/..." URL was never served and leaked storage layout.
	fileURL := r.FileURL
	if r.FilePath != "" {
		fileURL = "/api/v1/resumes/" + ID(r.ID) + "/download"
	}
	return ResumeResponse{
		ID:          ID(r.ID),
		CandidateID: ID(r.CandidateID),
		FileName:    r.FileName,
		FileURL:     fileURL,
		UploadedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
		SizeKb:      r.SizeKb,
	}
}

// GmailStatusResponse reports what the dashboard needs to know about a user's Gmail
// connection. It carries no token material and no mailbox contents.
type GmailStatusResponse struct {
	Connected       bool   `json:"connected"`
	NeedsReconnect  bool   `json:"needsReconnect"`
	HasRefreshToken bool   `json:"hasRefreshToken"`
	Reason          string `json:"reason,omitempty"`
}

// EmailEventResponse is one detected recruitment email. The subject and body are
// never stored by the poller, so only the classified fields are returned.
type EmailEventResponse struct {
	ID                   string `json:"id"`
	DetectedCompany      string `json:"detectedCompany"`
	DetectedRole         string `json:"detectedRole"`
	SuggestedStatus      string `json:"suggestedStatus"`
	Confidence           int    `json:"confidence"`
	ClassificationSource string `json:"classificationSource,omitempty"`
	DetectedAt           string `json:"detectedAt"`
	Confirmed            bool   `json:"confirmed"`
	ApplicationID        string `json:"applicationId,omitempty"`
}

func EmailEventFromModel(e *models.EmailEvent) EmailEventResponse {
	applicationID := ""
	if e.ApplicationID != nil {
		applicationID = ID(*e.ApplicationID)
	}
	return EmailEventResponse{
		ID:                   ID(e.ID),
		DetectedCompany:      e.DetectedCompany,
		DetectedRole:         e.DetectedRole,
		SuggestedStatus:      NormalizeStatus(e.SuggestedStatus),
		Confidence:           e.Confidence,
		ClassificationSource: e.ClassificationSource,
		DetectedAt:           e.ProcessedAt.UTC().Format(time.RFC3339),
		Confirmed:            e.Confirmed,
		ApplicationID:        applicationID,
	}
}
