package services

import (
	"errors"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"

	"gorm.io/gorm"
)

type RecruiterService struct {
	profiles *repositories.ProfileRepository
	users    *repositories.UserRepository
}

func NewRecruiterService(profiles *repositories.ProfileRepository, users *repositories.UserRepository) *RecruiterService {
	return &RecruiterService{profiles: profiles, users: users}
}

func (s *RecruiterService) GetOrCreateProfile(userID uint) (*models.RecruiterProfile, error) {
	p, err := s.profiles.GetRecruiterByUserID(userID)
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
	p = &models.RecruiterProfile{
		UserID:   userID,
		FullName: user.FullName,
		Email:    user.Email,
	}
	if err := s.profiles.UpsertRecruiter(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *RecruiterService) UpdateProfile(userID uint, in *models.RecruiterProfile) (*models.RecruiterProfile, error) {
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
	if in.CompanyName != "" {
		existing.CompanyName = in.CompanyName
	}
	if in.CompanyWebsite != "" {
		existing.CompanyWebsite = in.CompanyWebsite
	}
	if in.CompanySize != "" {
		existing.CompanySize = in.CompanySize
	}
	if in.Industry != "" {
		existing.Industry = in.Industry
	}
	if in.Location != "" {
		existing.Location = in.Location
	}
	if in.About != "" {
		existing.About = in.About
	}
	if err := s.profiles.UpsertRecruiter(existing); err != nil {
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
