package handlers

import (
	"net/http"
	"strconv"
	"time"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApplicationHandler struct {
	service          *services.ApplicationService
	db               *gorm.DB
	notificationRepo *repositories.NotificationRepository
}

func NewApplicationHandler(service *services.ApplicationService, db *gorm.DB, notificationRepo *repositories.NotificationRepository) *ApplicationHandler {
	return &ApplicationHandler{
		service:          service,
		db:               db,
		notificationRepo: notificationRepo,
	}
}

func (h *ApplicationHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Company      string `json:"company" binding:"required"`
		Role         string `json:"role" binding:"required"`
		AppliedDate  string `json:"applied_date"`
		Source       string `json:"source"`
		Status       string `json:"status"`
		FollowUpSent bool   `json:"follow_up_sent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status == "" {
		req.Status = "Applied"
	}

	if req.Source == "" {
		req.Source = "Manual"
	}

	var appliedDate time.Time
	if req.AppliedDate != "" {
		parsedDate, err := time.Parse(time.RFC3339, req.AppliedDate)
		if err != nil {
			parsedDate, err = time.Parse("2006-01-02", req.AppliedDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
				return
			}
		}
		appliedDate = parsedDate
	} else {
		appliedDate = time.Now()
	}

	app := &models.Application{
		UserID:       userID,
		Company:      req.Company,
		Role:         req.Role,
		AppliedDate:  appliedDate,
		Source:       req.Source,
		Status:       req.Status,
		LastUpdated:  time.Now(),
		FollowUpSent: req.FollowUpSent,
	}

	if err := h.service.CreateApplication(app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create application: " + err.Error()})
		return
	}

	// Calculate health score immediately after creation
	_ = services.RefreshHealthScore(h.db, app, "")

	c.JSON(http.StatusCreated, app)
}

func (h *ApplicationHandler) GetAll(c *gin.Context) {
	userID := c.GetUint("user_id")

	apps, err := services.GetApplicationsWithFreshScores(h.db, userID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch"})
		return
	}
	c.JSON(http.StatusOK, apps)
}

func (h *ApplicationHandler) GetByID(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	app, err := services.GetApplicationWithFreshScore(h.db, uint(id), "")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	if app.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, app)
}

// GetHealthScore manually triggers a health score refresh for one application.
// Route: GET /api/applications/:id/health-score
func (h *ApplicationHandler) GetHealthScore(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Fetch application
	app, err := h.service.GetApplicationByID(userID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// Force refresh (ignore stale check — user explicitly requested it)
	if err := services.RefreshHealthScore(h.db, app, ""); err != nil {
		// Return last known score if NLP is down rather than failing
		c.JSON(http.StatusOK, gin.H{
			"health_score":      app.HealthScore,
			"health_color":      app.HealthColor,
			"health_factors":    app.HealthFactors,
			"health_updated_at": app.HealthUpdatedAt,
			"warning":           "NLP service unavailable, showing last known score",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"health_score":      app.HealthScore,
		"health_color":      app.HealthColor,
		"health_factors":    app.HealthFactors,
		"health_updated_at": app.HealthUpdatedAt,
	})
}

func (h *ApplicationHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Company string `json:"company"`
		Role    string `json:"role"`
		Status  string `json:"status"`
		Source  string `json:"source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.service.UpdateApplication(userID, uint(id), req.Company, req.Role, req.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// Refresh health score after update (status may have changed)
	_ = services.RefreshHealthScore(h.db, app, "")

	c.JSON(http.StatusOK, app)
}

func (h *ApplicationHandler) UpdateStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Source string `json:"source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !dto.ValidStatusUpdate(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	if req.Source == "" {
		req.Source = "Manual"
	}

	app, err := h.service.UpdateApplicationStatus(userID, uint(id), req.Status, req.Source)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Refresh health score after status change
	_ = services.RefreshHealthScore(h.db, app, "")

	// Create notification for status change
	notification := &models.Notification{
		UserID:    userID,
		Type:      "application",
		Title:     "Application Status Updated",
		Message:   "Your application at " + app.Company + " moved to " + req.Status + " stage",
		Read:      false,
		ActionURL: "/applications",
	}
	_ = h.notificationRepo.Create(notification)

	c.JSON(http.StatusOK, app)
}

func (h *ApplicationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.DeleteApplication(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application deleted"})
}
