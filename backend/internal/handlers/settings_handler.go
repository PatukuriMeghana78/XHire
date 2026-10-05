package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"job_application_analysis/internal/repositories"
)

type SettingsHandler struct {
	userRepo *repositories.UserRepository
}

func NewSettingsHandler(userRepo *repositories.UserRepository) *SettingsHandler {
	return &SettingsHandler{userRepo: userRepo}
}

// GetSettings returns user settings
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":           user.Email,
		"gmail_connected": user.GmailConnected,
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

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.GmailToken = ""
	user.GmailConnected = false

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect Gmail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gmail disconnected successfully"})
}
