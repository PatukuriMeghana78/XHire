package services

import (
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
)

type NotificationService struct {
	repo *repositories.NotificationRepository
}

func NewNotificationService(repo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) CreateNotification(userID uint, notifType, title, message, actionURL string) error {
	notification := &models.Notification{
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		ActionURL: actionURL,
		Read:      false,
	}
	return s.repo.Create(notification)
}

func (s *NotificationService) GetNotifications(userID uint) ([]models.Notification, error) {
	return s.repo.FindByUserID(userID)
}

func (s *NotificationService) GetUnreadNotifications(userID uint) ([]models.Notification, error) {
	return s.repo.FindUnreadByUserID(userID)
}

func (s *NotificationService) MarkAsRead(id, userID uint) error {
	return s.repo.MarkAsRead(id, userID)
}

func (s *NotificationService) MarkAllAsRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}

func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	return s.repo.CountUnread(userID)
}
