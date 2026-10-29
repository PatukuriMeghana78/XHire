package services

import (
	"errors"
	"time"

	"job_application_analysis/internal/config"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo        *repositories.UserRepository
	profileRepo *repositories.ProfileRepository
}

func NewAuthService(repo *repositories.UserRepository, profileRepo *repositories.ProfileRepository) *AuthService {
	return &AuthService{repo: repo, profileRepo: profileRepo}
}

func normalizeRole(role string) string {
	if role == models.RoleRecruiter {
		return models.RoleRecruiter
	}
	return models.RoleCandidate
}

func (s *AuthService) createToken(user *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    normalizeRole(user.Role),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(config.JWTSecret())
}

func (s *AuthService) seedProfile(user *models.User) {
	if s.profileRepo == nil {
		return
	}
	if user.Role == models.RoleRecruiter {
		_ = s.profileRepo.UpsertRecruiter(&models.RecruiterProfile{
			UserID:   user.ID,
			FullName: user.FullName,
			Email:    user.Email,
		})
		return
	}
	_ = s.profileRepo.UpsertCandidate(&models.CandidateProfile{
		UserID:   user.ID,
		FullName: user.FullName,
		Email:    user.Email,
		Skills:   models.StringSlice{},
	})
}

// SIGNUP (legacy Applylitics)
func (s *AuthService) Signup(email, password string) error {
	_, _, err := s.Register(email, password, "", models.RoleCandidate)
	return err
}

func (s *AuthService) Register(email, password, fullName, role string) (*models.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, "", err
	}
	user := &models.User{
		Email:    email,
		Password: string(hash),
		FullName: fullName,
		Role:     normalizeRole(role),
	}
	if err := s.repo.Create(user); err != nil {
		return nil, "", err
	}
	s.seedProfile(user)
	token, err := s.createToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) Login(email, password string) (string, error) {
	user, token, err := s.LoginUser(email, password)
	if err != nil {
		return "", err
	}
	_ = user
	return token, nil
}

func (s *AuthService) LoginUser(email, password string) (*models.User, string, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if user.Role == "" {
		user.Role = models.RoleCandidate
	}
	token, err := s.createToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) GetUser(id uint) (*models.User, error) {
	return s.repo.GetByID(id)
}
