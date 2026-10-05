package services

import (
	"errors"
	"strings"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
)

type JobService struct {
	repo        *repositories.JobRepository
	profileRepo *repositories.ProfileRepository
}

func NewJobService(repo *repositories.JobRepository, profileRepo *repositories.ProfileRepository) *JobService {
	return &JobService{repo: repo, profileRepo: profileRepo}
}

type JobInput struct {
	Title            string
	CompanyName      string
	Location         string
	WorkMode         string
	JobType          string
	SalaryMin        *int
	SalaryMax        *int
	ExperienceMin    *int
	Description      string
	Responsibilities []string
	RequiredSkills   []string
	Status           string
}

func (s *JobService) Create(recruiterID uint, in JobInput) (*models.Job, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("title is required")
	}
	company := in.CompanyName
	if company == "" && s.profileRepo != nil {
		if p, err := s.profileRepo.GetRecruiterByUserID(recruiterID); err == nil {
			company = p.CompanyName
		}
	}
	status := in.Status
	if status == "" {
		status = "open"
	}
	job := &models.Job{
		RecruiterID:      recruiterID,
		Title:            in.Title,
		CompanyName:      company,
		Location:         in.Location,
		WorkMode:         defaultString(in.WorkMode, "onsite"),
		JobType:          defaultString(in.JobType, "full-time"),
		SalaryMin:        in.SalaryMin,
		SalaryMax:        in.SalaryMax,
		ExperienceMin:    in.ExperienceMin,
		Description:      in.Description,
		Responsibilities: in.Responsibilities,
		RequiredSkills:   in.RequiredSkills,
		Status:           status,
	}
	if err := s.repo.Create(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) Update(recruiterID, jobID uint, in JobInput) (*models.Job, error) {
	job, err := s.repo.GetByID(jobID)
	if err != nil {
		return nil, err
	}
	if job.RecruiterID != recruiterID {
		return nil, errors.New("unauthorized")
	}
	if in.Title != "" {
		job.Title = in.Title
	}
	if in.CompanyName != "" {
		job.CompanyName = in.CompanyName
	}
	if in.Location != "" {
		job.Location = in.Location
	}
	if in.WorkMode != "" {
		job.WorkMode = in.WorkMode
	}
	if in.JobType != "" {
		job.JobType = in.JobType
	}
	if in.SalaryMin != nil {
		job.SalaryMin = in.SalaryMin
	}
	if in.SalaryMax != nil {
		job.SalaryMax = in.SalaryMax
	}
	if in.ExperienceMin != nil {
		job.ExperienceMin = in.ExperienceMin
	}
	if in.Description != "" {
		job.Description = in.Description
	}
	if in.Responsibilities != nil {
		job.Responsibilities = in.Responsibilities
	}
	if in.RequiredSkills != nil {
		job.RequiredSkills = in.RequiredSkills
	}
	if in.Status != "" {
		job.Status = in.Status
	}
	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) Delete(recruiterID, jobID uint) error {
	job, err := s.repo.GetByID(jobID)
	if err != nil {
		return err
	}
	if job.RecruiterID != recruiterID {
		return errors.New("unauthorized")
	}
	return s.repo.Delete(jobID)
}

func (s *JobService) Get(id uint) (*models.Job, error) {
	return s.repo.GetByID(id)
}

func (s *JobService) List(filters repositories.JobFilters) ([]models.Job, error) {
	return s.repo.List(filters)
}

func (s *JobService) ApplicantCount(jobID uint) int {
	n, _ := s.repo.CountApplicants(jobID)
	return int(n)
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
