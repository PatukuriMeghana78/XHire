package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/services"
)

type EmailHandler struct {
	service *services.EmailService
}

func NewEmailHandler(service *services.EmailService) *EmailHandler {
	return &EmailHandler{service: service}
}

// ===============================
// Gmail OAuth Start
// ===============================

func (h *EmailHandler) StartOAuth(c *gin.Context) {

	// Temporary hardcoded userID for testing
	userID := uint(1)

	state := fmt.Sprintf("%d", userID)

	authURL := h.service.GetAuthURL(state)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// ===============================
// Gmail OAuth Callback
// ===============================

func (h *EmailHandler) HandleOAuthCallback(c *gin.Context) {

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	userIDInt, err := strconv.Atoi(state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	token, err := h.service.ExchangeCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gmailToken := models.GmailToken{
		UserID:       uint(userIDInt),
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if err := h.service.DB().Create(&gmailToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create notification for successful Gmail connection
	notification := &models.Notification{
		UserID:    uint(userIDInt),
		Type:      "system",
		Title:     "Gmail Connected Successfully",
		Message:   "Your Gmail account has been connected. Email sync will run automatically every 3 minutes.",
		Read:      false,
		ActionURL: "/emails",
	}
	h.service.CreateConnectionNotification(notification)

	// Redirect back to the frontend Settings page
	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173/settings?gmail=connected")
}

// ===============================
// Existing Methods
// ===============================

func (h *EmailHandler) GetPendingSuggestions(c *gin.Context) {
	userID := c.GetUint("user_id")

	suggestions, err := h.service.GetPendingSuggestions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch suggestions"})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

func (h *EmailHandler) ConfirmSuggestion(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		EmailEventID  uint   `json:"email_event_id" binding:"required"`
		ApplicationID *uint  `json:"application_id"`
		Status        string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.service.ConfirmSuggestion(userID, req.EmailEventID, req.ApplicationID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

func (h *EmailHandler) IgnoreSuggestion(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		EmailEventID uint `json:"email_event_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.IgnoreSuggestion(userID, req.EmailEventID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "suggestion ignored"})
}

func (h *EmailHandler) SyncEmails(c *gin.Context) {
	userID := c.GetUint("user_id")

	count, err := h.service.SyncUserEmails(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sync completed", "processed": count})
}
