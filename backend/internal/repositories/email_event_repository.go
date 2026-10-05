package repositories

import (
	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type EmailEventRepository struct {
	db *gorm.DB
}

func NewEmailEventRepository(db *gorm.DB) *EmailEventRepository {
	return &EmailEventRepository{db: db}
}

func (r *EmailEventRepository) Create(event *models.EmailEvent) error {
	return r.db.Create(event).Error
}

func (r *EmailEventRepository) FindByID(id uint) (*models.EmailEvent, error) {
	var event models.EmailEvent
	err := r.db.First(&event, id).Error
	return &event, err
}

func (r *EmailEventRepository) FindPendingByUserID(userID uint) ([]models.EmailEvent, error) {
	var events []models.EmailEvent
	err := r.db.Where("user_id = ? AND confirmed = ?", userID, false).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *EmailEventRepository) Update(event *models.EmailEvent) error {
	return r.db.Save(event).Error
}
