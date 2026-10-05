package repositories

import (
	"job_application_analysis/internal/database"
	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func (r *ApplicationRepository) Create(app *models.Application) error {
	if database.GetGlobalMockDB() != nil {
		return database.MockCreate(app)
	}
	return r.db.Create(app).Error
}

func (r *ApplicationRepository) GetAll() ([]models.Application, error) {
	if database.GetGlobalMockDB() != nil {
		return []models.Application{}, nil
	}
	var apps []models.Application
	err := r.db.Find(&apps).Error
	return apps, err
}

func (r *ApplicationRepository) GetByID(id uint) (*models.Application, error) {
	if database.GetGlobalMockDB() != nil {
		return database.MockFindApplicationByID(id)
	}
	var app models.Application
	err := r.db.First(&app, id).Error
	return &app, err
}

func (r *ApplicationRepository) Update(app *models.Application) error {
	if database.GetGlobalMockDB() != nil {
		return database.MockUpdateApplication(app)
	}
	return r.db.Save(app).Error
}

func (r *ApplicationRepository) Delete(id uint) error {
	if database.GetGlobalMockDB() != nil {
		return database.MockDeleteApplication(id)
	}
	return r.db.Delete(&models.Application{}, id).Error
}

func (r *ApplicationRepository) FindByID(id uint) (*models.Application, error) {
	if database.GetGlobalMockDB() != nil {
		return database.MockFindApplicationByID(id)
	}
	var app models.Application
	err := r.db.Preload("StatusHistory").First(&app, id).Error
	return &app, err
}

func (r *ApplicationRepository) FindByUserID(userID uint) ([]models.Application, error) {
	if database.GetGlobalMockDB() != nil {
		apps, err := database.MockFindApplicationsByUserID(userID)
		if err != nil {
			return []models.Application{}, err
		}
		result := make([]models.Application, len(apps))
		for i, app := range apps {
			result[i] = *app
		}
		return result, nil
	}

	var apps []models.Application
	err := r.db.Where("user_id = ?", userID).
		Preload("StatusHistory").
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *ApplicationRepository) AddStatusHistory(history *models.StatusHistory) error {
	if database.GetGlobalMockDB() != nil {
		return database.MockCreate(history)
	}
	return r.db.Create(history).Error
}

// UpdateStatus updates the status of an application
func (r *ApplicationRepository) UpdateStatus(applicationID uint, userID uint, newStatus string) error {
	return r.db.Model(&models.Application{}).
		Where("id = ? AND user_id = ?", applicationID, userID).
		Update("status", newStatus).Error
}

func (r *ApplicationRepository) FindByJobAndCandidate(jobID, candidateID uint) (*models.Application, error) {
	var app models.Application
	err := r.db.Where("job_id = ? AND (candidate_id = ? OR user_id = ?)", jobID, candidateID, candidateID).
		First(&app).Error
	return &app, err
}

func (r *ApplicationRepository) FindByJobID(jobID uint) ([]models.Application, error) {
	var apps []models.Application
	err := r.db.Where("job_id = ?", jobID).
		Preload("StatusHistory").
		Preload("Job").
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *ApplicationRepository) FindByCandidateID(candidateID uint) ([]models.Application, error) {
	var apps []models.Application
	err := r.db.Where("candidate_id = ? OR user_id = ?", candidateID, candidateID).
		Preload("StatusHistory").
		Preload("Job").
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *ApplicationRepository) FindByRecruiterID(recruiterID uint) ([]models.Application, error) {
	var apps []models.Application
	err := r.db.Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.recruiter_id = ?", recruiterID).
		Preload("StatusHistory").
		Preload("Job").
		Order("applications.created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *ApplicationRepository) FindByIDWithJob(id uint) (*models.Application, error) {
	var app models.Application
	err := r.db.Preload("StatusHistory").Preload("Job").First(&app, id).Error
	return &app, err
}

func (r *ApplicationRepository) UpdateMatchScore(id uint, score int) error {
	return r.db.Model(&models.Application{}).Where("id = ?", id).Update("match_score", score).Error
}
