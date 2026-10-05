package repositories

import (
	"strings"

	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type JobFilters struct {
	Search         string
	Location       string
	JobType        string
	WorkMode       string
	MinExperience  *int
	RecruiterID    uint
	Status         string
}

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *models.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) Update(job *models.Job) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) Delete(id uint) error {
	return r.db.Delete(&models.Job{}, id).Error
}

func (r *JobRepository) GetByID(id uint) (*models.Job, error) {
	var job models.Job
	err := r.db.First(&job, id).Error
	return &job, err
}

func (r *JobRepository) List(filters JobFilters) ([]models.Job, error) {
	q := r.db.Model(&models.Job{})
	if filters.RecruiterID != 0 {
		q = q.Where("recruiter_id = ?", filters.RecruiterID)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	if filters.Location != "" {
		q = q.Where("LOWER(location) LIKE ?", "%"+strings.ToLower(filters.Location)+"%")
	}
	if filters.JobType != "" && filters.JobType != "all" {
		q = q.Where("job_type = ?", filters.JobType)
	}
	if filters.WorkMode != "" && filters.WorkMode != "all" {
		q = q.Where("work_mode = ?", filters.WorkMode)
	}
	if filters.MinExperience != nil {
		q = q.Where("experience_min IS NULL OR experience_min <= ?", *filters.MinExperience)
	}
	if term := strings.TrimSpace(filters.Search); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where("LOWER(title) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(required_skills) LIKE ?", like, like, like)
	}
	var jobs []models.Job
	err := q.Order("created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (r *JobRepository) CountApplicants(jobID uint) (int64, error) {
	var n int64
	err := r.db.Model(&models.Application{}).Where("job_id = ?", jobID).Count(&n).Error
	return n, err
}
