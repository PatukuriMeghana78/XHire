package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/gorm"
)

func nlpServiceURL() string {
	if url := os.Getenv("NLP_SERVICE_URL"); url != "" {
		return url
	}
	return "http://localhost:8000" // default FastAPI port
}

// RefreshHealthScore calls the Python NLP /health-score endpoint,
// then updates the Application record in the DB.
func RefreshHealthScore(db *gorm.DB, app *models.Application, emailSummary string) error {
	// 1. Build request payload
	payload := models.HealthScoreRequest{
		ApplicationID: app.ID,
		Company:       app.Company,
		Role:          app.Role,
		Status:        app.Status,
		AppliedDate:   app.AppliedDate,
		LastUpdated:   app.LastUpdated,
		FollowUpSent:  app.FollowUpSent,
		EmailSummary:  emailSummary,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("health score: marshal error: %w", err)
	}

	// 2. Call Python NLP service
	url := nlpServiceURL() + "/health-score"
	log.Printf("🔍 Calling NLP at: %s", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("❌ NLP error: %v", err)
		return fmt.Errorf("health score: NLP service unreachable: %w", err)
	}
	log.Printf("✅ NLP response status: %d", resp.StatusCode)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health score: NLP service returned %d", resp.StatusCode)
	}

	// 3. Parse response
	var result models.HealthScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("health score: decode error: %w", err)
	}

	// 4. Serialize factors map → JSON string for storage
	factorsJSON, err := json.Marshal(result.Factors)
	if err != nil {
		factorsJSON = []byte("{}") // safe fallback
	}

	// 5. Update application record
	now := time.Now()
	if err := db.Model(app).Updates(map[string]interface{}{
		"health_score":      result.Score,
		"health_color":      result.Color,
		"health_factors":    string(factorsJSON),
		"health_updated_at": now,
	}).Error; err != nil {
		return fmt.Errorf("health score: DB update failed: %w", err)
	}

	// Keep in-memory struct in sync too
	app.HealthScore = result.Score
	app.HealthColor = result.Color
	app.HealthFactors = string(factorsJSON)
	app.HealthUpdatedAt = now

	return nil
}

// GetHealthColor returns badge color string from score (utility helper).
// Use this if you ever want to recompute color client-side.
func GetHealthColor(score int) string {
	switch {
	case score >= 70:
		return "green"
	case score >= 40:
		return "yellow"
	default:
		return "red"
	}
}

type ApplicationService struct {
	repo *repositories.ApplicationRepository
}

func NewApplicationService(repo *repositories.ApplicationRepository) *ApplicationService {
	return &ApplicationService{repo: repo}
}

func (s *ApplicationService) CreateApplication(app *models.Application) error {
	if app.Status == "" {
		app.Status = "Applied"
	}
	if app.CandidateID == 0 {
		app.CandidateID = app.UserID
	}
	app.LastUpdated = time.Now()

	// Create the application first
	if err := s.repo.Create(app); err != nil {
		return err
	}

	// Now app.ID should be set by the database
	// Add initial status history only if ID is set
	if app.ID > 0 {
		history := &models.StatusHistory{
			ApplicationID: app.ID,
			Status:        app.Status,
			ChangedAt:     app.AppliedDate,
			Source:        "Manual",
		}

		// Don't fail the whole operation if history fails
		if err := s.repo.AddStatusHistory(history); err != nil {
			// Log error but don't return it
			return nil
		}
	}

	return nil
}

func (s *ApplicationService) GetApplications() ([]models.Application, error) {
	return s.repo.GetAll()
}

func (s *ApplicationService) GetUserApplications(userID uint) ([]models.Application, error) {
	return s.repo.FindByUserID(userID)
}

func (s *ApplicationService) GetApplicationByID(userID, id uint) (*models.Application, error) {
	app, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if app.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return app, nil
}

func (s *ApplicationService) UpdateApplication(
	userID uint,
	id uint,
	company string,
	role string,
	status string,
) (*models.Application, error) {

	app, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if app.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	oldStatus := app.Status

	if company != "" {
		app.Company = company
	}
	if role != "" {
		app.Role = role
	}
	if status != "" && status != oldStatus {
		app.Status = status
		app.LastUpdated = time.Now()

		// Add status history
		history := &models.StatusHistory{
			ApplicationID: app.ID,
			Status:        status,
			ChangedAt:     time.Now(),
			Source:        "Manual",
		}
		if err := s.repo.AddStatusHistory(history); err != nil {
			return nil, err
		}
	}

	err = s.repo.Update(app)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (s *ApplicationService) UpdateApplicationStatus(userID, id uint, status, source string) (*models.Application, error) {
	app, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if app.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	if app.Status == status {
		return app, nil
	}

	app.Status = status
	app.LastUpdated = time.Now()

	// Add status history
	history := &models.StatusHistory{
		ApplicationID: app.ID,
		Status:        status,
		ChangedAt:     time.Now(),
		Source:        source,
	}

	if err := s.repo.Update(app); err != nil {
		return nil, err
	}

	if err := s.repo.AddStatusHistory(history); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *ApplicationService) DeleteApplication(userID, id uint) error {
	app, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	if app.UserID != userID {
		return errors.New("unauthorized")
	}

	return s.repo.Delete(id)
}

func (s *ApplicationService) ApplyToJob(candidateID uint, job *models.Job, coverLetter string, resumeID *uint, matchScore int) (*models.Application, error) {
	if existing, err := s.repo.FindByJobAndCandidate(job.ID, candidateID); err == nil && existing.ID != 0 {
		return existing, errors.New("already applied")
	}

	now := time.Now()
	jobID := job.ID
	app := &models.Application{
		UserID:      candidateID,
		CandidateID: candidateID,
		JobID:       &jobID,
		Company:     job.CompanyName,
		Role:        job.Title,
		AppliedDate: now,
		Source:      "XHire",
		Status:      "applied",
		LastUpdated: now,
		CoverLetter: coverLetter,
		ResumeID:    resumeID,
		MatchScore:  matchScore,
	}
	if err := s.CreateApplication(app); err != nil {
		return nil, err
	}
	loaded, err := s.repo.FindByIDWithJob(app.ID)
	if err != nil {
		return app, nil
	}
	return loaded, nil
}

func (s *ApplicationService) GetCandidateMarketplaceApps(candidateID uint) ([]models.Application, error) {
	return s.repo.FindByCandidateID(candidateID)
}

func (s *ApplicationService) GetRecruiterMarketplaceApps(recruiterID uint) ([]models.Application, error) {
	return s.repo.FindByRecruiterID(recruiterID)
}

func (s *ApplicationService) GetJobApplications(jobID uint) ([]models.Application, error) {
	return s.repo.FindByJobID(jobID)
}

func (s *ApplicationService) RecruiterUpdateStatus(recruiterID, applicationID uint, status string) (*models.Application, error) {
	app, err := s.repo.FindByIDWithJob(applicationID)
	if err != nil {
		return nil, err
	}
	if app.Job == nil || app.Job.RecruiterID != recruiterID {
		return nil, errors.New("unauthorized")
	}
	if app.Status == status {
		return app, nil
	}
	app.Status = status
	app.LastUpdated = time.Now()
	if err := s.repo.Update(app); err != nil {
		return nil, err
	}
	_ = s.repo.AddStatusHistory(&models.StatusHistory{
		ApplicationID: app.ID,
		Status:        status,
		ChangedAt:     time.Now(),
		Source:        "Recruiter",
	})
	return s.repo.FindByIDWithJob(app.ID)
}

func (s *ApplicationService) GetMarketplaceByID(id uint) (*models.Application, error) {
	return s.repo.FindByIDWithJob(id)
}

const ScoreStaleDuration = 24 * time.Hour

// isScoreStale returns true if the health score needs recalculating.
// Reasons it might be stale:
//   - Never calculated (zero time)
//   - Older than ScoreStaleDuration
//   - Application was updated more recently than the score
func isScoreStale(app *models.Application) bool {
	// Never been calculated
	if app.HealthUpdatedAt.IsZero() {
		return true
	}

	// Score is older than 24 hours
	if time.Since(app.HealthUpdatedAt) > ScoreStaleDuration {
		return true
	}

	// Application status/data changed AFTER the score was last calculated
	if app.LastUpdated.After(app.HealthUpdatedAt) {
		return true
	}

	return false
}

// GetApplicationWithFreshScore fetches a single application by ID,
// recalculates its health score if stale, and returns the up-to-date record.
//
// Usage in your handler:
//
//	app, err := services.GetApplicationWithFreshScore(db, appID, "")
func GetApplicationWithFreshScore(db *gorm.DB, appID uint, emailSummary string) (*models.Application, error) {
	var app models.Application

	// Fetch with status history preloaded
	if err := db.Preload("StatusHistory").First(&app, appID).Error; err != nil {
		return nil, err
	}

	// Recalculate score if stale — non-fatal if NLP service is down
	if isScoreStale(&app) {
		_ = RefreshHealthScore(db, &app, emailSummary)
		// We intentionally ignore the error here:
		// if the NLP service is temporarily down, we still return
		// the application with the last known score rather than failing.
	}

	return &app, nil
}

// GetApplicationsWithFreshScores fetches ALL applications for a user,
// recalculating any stale scores. Refreshes are done concurrently
// to avoid slow sequential NLP calls.
//
// Usage in your handler:
//
//	apps, err := services.GetApplicationsWithFreshScores(db, userID, nil)
func GetApplicationsWithFreshScores(
	db *gorm.DB,
	userID uint,
	emailSummaries map[uint]string, // optional: map[applicationID]emailText
) ([]models.Application, error) {
	var apps []models.Application

	if err := db.Where("user_id = ?", userID).
		Preload("StatusHistory").
		Find(&apps).Error; err != nil {
		return nil, err
	}

	// Refresh stale scores concurrently
	type result struct {
		index int
		err   error
	}

	results := make(chan result, len(apps))

	for i := range apps {
		app := &apps[i] // capture pointer to slice element

		if !isScoreStale(app) {
			results <- result{index: i, err: nil}
			continue
		}

		go func(idx int, a *models.Application) {
			summary := ""
			if emailSummaries != nil {
				summary = emailSummaries[a.ID]
			}
			err := RefreshHealthScore(db, a, summary)
			results <- result{index: idx, err: err}
		}(i, app)
	}

	// Wait for all goroutines (stale + non-stale) to report
	for range apps {
		<-results
	}

	return apps, nil
}
