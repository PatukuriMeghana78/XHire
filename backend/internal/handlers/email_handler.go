package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"job_application_analysis/internal/dto"
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

	// Use the authenticated user's ID so each Gmail connection binds to the
	// correct account (previously hardcoded to user 1, causing mix-ups).
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Issue a cryptographically random, single-use, expiring state bound to this
	// user. The state carries no secrets and is validated on callback (CSRF).
	state, err := h.service.IssueOAuthState(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start OAuth"})
		return
	}

	authURL := h.service.GetAuthURL(state)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// ===============================
// Gmail OAuth Callback
// ===============================

func (h *EmailHandler) HandleOAuthCallback(c *gin.Context) {

	// Google returns ?error=... when the user declines consent or the request itself
	// is rejected. There is no code to exchange, so report it and stop.
	if googleErr := c.Query("error"); googleErr != "" {
		log.Println("⚠️ Gmail OAuth callback rejected by Google:", googleErr)
		reason := services.OAuthFailureReason(googleErr)
		c.Redirect(http.StatusTemporaryRedirect, services.OAuthReturnURL(false, reason))
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect,
			services.OAuthReturnURL(false, "Gmail connection failed: no authorization code was returned."))
		return
	}

	// Validate and consume the state. Rejects missing, unknown, expired, or replayed
	// states, and recovers the user the flow was started for.
	userID, ok := h.service.ValidateOAuthState(state)
	if !ok {
		log.Println("⚠️ Gmail OAuth callback rejected: invalid or expired state")
		c.Redirect(http.StatusTemporaryRedirect,
			services.OAuthReturnURL(false, "Gmail connection failed: the request expired or was already used. Start again."))
		return
	}

	token, err := h.service.ExchangeCode(code)
	if err != nil {
		// The exchange error can mention Google's response text; log it here and send
		// the user a stable reason instead.
		log.Println("❌ Gmail code exchange failed for user", userID, ":", services.OAuthFailureReason("invalid_grant"))
		c.Redirect(http.StatusTemporaryRedirect,
			services.OAuthReturnURL(false, services.OAuthFailureReason("invalid_grant")))
		return
	}

	if token.RefreshToken == "" {
		// Google only sends a refresh token with the first consent. Without one the
		// connection dies an hour later, so say so instead of failing silently.
		log.Printf("⚠️ Gmail grant for user %d has no refresh token", userID)
	}

	// Storage is an upsert keyed on this user, and an empty refresh token never
	// overwrites a stored one, so a reconnect can actually repair the grant.
	if err := h.service.PersistGmailGrant(userID, token); err != nil {
		log.Println("❌ Failed to store Gmail grant for user", userID)
		c.Redirect(http.StatusTemporaryRedirect,
			services.OAuthReturnURL(false, "Gmail connected, but the session could not be saved. Try again."))
		return
	}

	// Create notification for successful Gmail connection
	notification := &models.Notification{
		UserID:    userID,
		Type:      "system",
		Title:     "Gmail Connected Successfully",
		Message:   "Your Gmail account has been connected. Email sync will run automatically every 3 minutes.",
		Read:      false,
		ActionURL: "/candidate",
	}
	h.service.CreateConnectionNotification(notification)

	// Redirect back to the candidate dashboard, which is where the Gmail card lives.
	c.Redirect(http.StatusTemporaryRedirect, services.OAuthReturnURL(true, ""))
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

	if !dto.ValidStatusUpdate(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	app, err := h.service.ConfirmSuggestion(userID, req.EmailEventID, req.ApplicationID, req.Status)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm suggestion"})
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
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ignore suggestion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "suggestion ignored"})
}

func (h *EmailHandler) SyncEmails(c *gin.Context) {
	userID := c.GetUint("user_id")

	count, err := h.service.SyncUserEmails(userID)
	if err != nil {
		// Never echo the raw Gmail/oauth2 error text to the client.
		writeGmailError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sync completed", "processed": count})
}
