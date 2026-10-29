package services

import (
	"errors"
	"strconv"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"

	"gorm.io/gorm"
)

type CandidateService struct {
	profiles *repositories.ProfileRepository
	users    *repositories.UserRepository
	jobs     *repositories.JobRepository
}

func NewCandidateService(profiles *repositories.ProfileRepository, users *repositories.UserRepository, jobs *repositories.JobRepository) *CandidateService {
	return &CandidateService{profiles: profiles, users: users, jobs: jobs}
}

func (s *CandidateService) GetOrCreateProfile(userID uint) (*models.CandidateProfile, error) {
	p, err := s.profiles.GetCandidateByUserID(userID)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	user, uerr := s.users.GetByID(userID)
	if uerr != nil {
		return nil, uerr
	}
	p = &models.CandidateProfile{
		UserID:   userID,
		FullName: user.FullName,
		Email:    user.Email,
		Skills:   models.StringSlice{},
	}
	if err := s.profiles.UpsertCandidate(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *CandidateService) GetByID(id uint) (*models.CandidateProfile, error) {
	return s.profiles.GetCandidateByID(id)
}

func (s *CandidateService) UpdateProfile(userID uint, in *models.CandidateProfile) (*models.CandidateProfile, error) {
	existing, err := s.GetOrCreateProfile(userID)
	if err != nil {
		return nil, err
	}
	if in.FullName != "" {
		existing.FullName = in.FullName
	}
	if in.Email != "" {
		existing.Email = in.Email
	}
	if in.Phone != "" {
		existing.Phone = in.Phone
	}
	if in.Headline != "" {
		existing.Headline = in.Headline
	}
	if in.Location != "" {
		existing.Location = in.Location
	}
	if in.ExperienceYears != nil {
		existing.ExperienceYears = in.ExperienceYears
	}
	if in.Skills != nil {
		existing.Skills = in.Skills
	}
	if in.Education != "" {
		existing.Education = in.Education
	}
	if in.About != "" {
		existing.About = in.About
	}
	if in.LinkedinURL != "" {
		existing.LinkedinURL = in.LinkedinURL
	}
	if in.GithubURL != "" {
		existing.GithubURL = in.GithubURL
	}
	if err := s.profiles.UpsertCandidate(existing); err != nil {
		return nil, err
	}
	if existing.FullName != "" {
		if user, err := s.users.GetByID(userID); err == nil {
			user.FullName = existing.FullName
			_ = s.users.Update(user)
		}
	}
	return existing, nil
}

func (s *CandidateService) SaveResume(userID uint, fileName, path, url string, sizeKb int) (*models.Resume, error) {
	resume := &models.Resume{
		CandidateID: userID,
		FileName:    fileName,
		FilePath:    path,
		FileURL:     url,
		SizeKb:      sizeKb,
	}
	if err := s.profiles.CreateResume(resume); err != nil {
		return nil, err
	}
	_ = s.profiles.SetCandidateResumeID(userID, resume.ID)
	if p, err := s.GetOrCreateProfile(userID); err == nil {
		p.ResumeID = &resume.ID
		_ = s.profiles.UpsertCandidate(p)
	}
	return resume, nil
}

func (s *CandidateService) GetResume(userID uint) (*models.Resume, error) {
	return s.profiles.LatestResume(userID)
}

func (s *CandidateService) GetResumeByID(id uint) (*models.Resume, error) {
	return s.profiles.GetResumeByID(id)
}

// AnalyzeResume returns an honest analysis result. The existing NLP service only
// parses job emails (company/role/status/health/skill-overlap); it has no
// PDF/DOCX text extraction or resume skill extractor, and adding one would
// require a new dependency. Rather than fabricate scores from the candidate's
// skill count (the previous behaviour), we report parsing as unavailable and
// return only real data: file metadata and the candidate's declared skills.
func (s *CandidateService) AnalyzeResume(resume *models.Resume, profile *models.CandidateProfile) map[string]interface{} {
	declared := []string{}
	if profile != nil && profile.Skills != nil {
		declared = profile.Skills
	}

	return map[string]interface{}{
		"resumeId": strconv.FormatUint(uint64(resume.ID), 10),
		"parsed":   false,
		"status":   "unavailable",
		"reason": "Resume content parsing is not available: the current NLP service only analyzes job emails and cannot extract text from PDF/DOCX resumes. No scores were generated to avoid fabricating results.",
		// Honest empties — nothing was extracted from the resume content.
		"overallScore":    0,
		"extractedSkills": []string{},
		"sections":        []interface{}{},
		"strengths":       []string{},
		"improvements":    []string{},
		// Real data we actually have.
		"declaredSkills": declared,
		"fileName":       resume.FileName,
		"sizeKb":         resume.SizeKb,
	}
}
