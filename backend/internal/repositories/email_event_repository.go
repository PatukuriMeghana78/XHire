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

// FindRecentByUserID returns a user's detected email events, newest first, whether or
// not the user has confirmed them yet. Only the caller's own rows are ever reachable.
func (r *EmailEventRepository) FindRecentByUserID(userID uint, limit int) ([]models.EmailEvent, error) {
	var events []models.EmailEvent
	err := r.db.Where("user_id = ?", userID).
		Order("processed_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ExistsByEmailID reports whether an event has already been recorded for the
// given user and Gmail message ID. Used to avoid duplicate records and
// repeated notifications when an email is processed more than once.
func (r *EmailEventRepository) ExistsByEmailID(userID uint, emailID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.EmailEvent{}).
		Where("user_id = ? AND email_id = ?", userID, emailID).
		Count(&count).Error
	return count > 0, err
}

func (r *EmailEventRepository) Update(event *models.EmailEvent) error {
	return r.db.Save(event).Error
}
