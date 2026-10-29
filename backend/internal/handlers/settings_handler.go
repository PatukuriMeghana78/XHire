package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"
)

type SettingsHandler struct {
	userRepo *repositories.UserRepository
	email    *services.EmailService
}

func NewSettingsHandler(userRepo *repositories.UserRepository, email *services.EmailService) *SettingsHandler {
	return &SettingsHandler{userRepo: userRepo, email: email}
}

// GetSettings returns user settings
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// The stored grant, not the legacy flag, decides the answer: a grant Google has
	// already refused must not be reported as connected.
	connected := user.GmailConnected
	if status, statusErr := h.email.ConnectionStatus(userID); statusErr == nil {
		connected = status.Connected
	}

	c.JSON(http.StatusOK, gin.H{
		"email":           user.Email,
		"gmail_connected": connected,
	})
}

// UpdateProfile updates user profile information
func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}

// DisconnectGmail disconnects Gmail integration
func (h *SettingsHandler) DisconnectGmail(c *gin.Context) {
	userID := c.GetUint("user_id")

	// The previous version only cleared an unused legacy column, so the background
	// poller kept reading the live grant from gmail_tokens and "disconnect" did
	// nothing. Clearing the grant is what actually stops the syncing.
	if err := h.email.DisconnectGmail(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect Gmail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gmail disconnected successfully"})
}
