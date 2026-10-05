package services

import (
	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type SuggestionService struct {
	DB *gorm.DB
}

//
// STEP 3 — Create Suggestion
//
func (s *SuggestionService) CreateSuggestion(suggestion *models.SuggestedUpdate) error {
	return s.DB.Create(suggestion).Error
}

//
// STEP 4 — Get Unconfirmed Suggestions
//
func (s *SuggestionService) GetUnconfirmed(applicationID uint) ([]models.SuggestedUpdate, error) {

	var suggestions []models.SuggestedUpdate

	err := s.DB.
		Where("application_id = ? AND confirmed = ?", applicationID, false).
		Find(&suggestions).Error

	return suggestions, err
}

//
// STEP 5 — Confirm Suggestion (Transaction 🔥)
//
func (s *SuggestionService) ConfirmSuggestion(suggestionID uint) error {

	return s.DB.Transaction(func(tx *gorm.DB) error {

		var suggestion models.SuggestedUpdate

		// 1️⃣ Find suggestion
		if err := tx.First(&suggestion, suggestionID).Error; err != nil {
			return err
		}

		// 2️⃣ Update application status
		if err := tx.Model(&models.Application{}).
			Where("id = ?", suggestion.ApplicationID).
			Update("status", suggestion.SuggestedStatus).Error; err != nil {
			return err
		}

		// 3️⃣ Mark suggestion confirmed
		if err := tx.Model(&suggestion).
			Update("confirmed", true).Error; err != nil {
			return err
		}

		return nil
	})
}
