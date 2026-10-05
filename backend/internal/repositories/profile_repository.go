package repositories

import (
	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) UpsertCandidate(p *models.CandidateProfile) error {
	var existing models.CandidateProfile
	err := r.db.Where("user_id = ?", p.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(p).Error
	}
	if err != nil {
		return err
	}
	p.ID = existing.ID
	return r.db.Save(p).Error
}

func (r *ProfileRepository) GetCandidateByUserID(userID uint) (*models.CandidateProfile, error) {
	var p models.CandidateProfile
	err := r.db.Where("user_id = ?", userID).First(&p).Error
	return &p, err
}

func (r *ProfileRepository) GetCandidateByID(id uint) (*models.CandidateProfile, error) {
	var p models.CandidateProfile
	err := r.db.First(&p, id).Error
	if err == nil {
		return &p, nil
	}
	return r.GetCandidateByUserID(id)
}

func (r *ProfileRepository) UpsertRecruiter(p *models.RecruiterProfile) error {
	var existing models.RecruiterProfile
	err := r.db.Where("user_id = ?", p.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(p).Error
	}
	if err != nil {
		return err
	}
	p.ID = existing.ID
	return r.db.Save(p).Error
}

func (r *ProfileRepository) GetRecruiterByUserID(userID uint) (*models.RecruiterProfile, error) {
	var p models.RecruiterProfile
	err := r.db.Where("user_id = ?", userID).First(&p).Error
	return &p, err
}

func (r *ProfileRepository) CreateResume(resume *models.Resume) error {
	return r.db.Create(resume).Error
}

func (r *ProfileRepository) GetResumeByID(id uint) (*models.Resume, error) {
	var resume models.Resume
	err := r.db.First(&resume, id).Error
	return &resume, err
}

func (r *ProfileRepository) LatestResume(candidateID uint) (*models.Resume, error) {
	var resume models.Resume
	err := r.db.Where("candidate_id = ?", candidateID).Order("created_at DESC").First(&resume).Error
	return &resume, err
}

func (r *ProfileRepository) SetCandidateResumeID(userID, resumeID uint) error {
	return r.db.Model(&models.CandidateProfile{}).
		Where("user_id = ?", userID).
		Update("resume_id", resumeID).Error
}
