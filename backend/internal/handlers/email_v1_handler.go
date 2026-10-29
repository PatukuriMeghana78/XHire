package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/services"
)

// EmailV1Handler exposes the Gmail surface the candidate dashboard needs: connection
// state, the emails already imported, an on-demand sync, and how to (re)authorize.
// Everything is scoped to the authenticated user id taken from the JWT.
type EmailV1Handler struct {
	email *services.EmailService
}

func NewEmailV1Handler(email *services.EmailService) *EmailV1Handler {
	return &EmailV1Handler{email: email}
}

// Status handles GET /api/v1/candidate/email/status
func (h *EmailV1Handler) Status(c *gin.Context) {
	userID := c.GetUint("user_id")
	status, err := h.email.ConnectionStatus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read gmail status"})
		return
	}
	c.JSON(http.StatusOK, dto.GmailStatusResponse{
		Connected:       status.Connected,
		NeedsReconnect:  status.NeedsReconnect,
		HasRefreshToken: status.HasRefreshToken,
		Reason:          status.Reason,
	})
}

// Messages handles GET /api/v1/candidate/email/messages
func (h *EmailV1Handler) Messages(c *gin.Context) {
	userID := c.GetUint("user_id")
	events, err := h.email.RecentEmails(userID, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list emails"})
		return
	}
	out := make([]dto.EmailEventResponse, 0, len(events))
	for i := range events {
		out = append(out, dto.EmailEventFromModel(&events[i]))
	}
	c.JSON(http.StatusOK, out)
}

// ConnectURL handles GET /api/v1/candidate/email/connect-url.
// The browser cannot attach a bearer token to a top-level navigation, so the UI asks
// for the authorization URL first and then navigates to it.
func (h *EmailV1Handler) ConnectURL(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	url, err := h.email.ConnectURL(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start Gmail connection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// Sync handles POST /api/v1/candidate/email/sync
func (h *EmailV1Handler) Sync(c *gin.Context) {
	userID := c.GetUint("user_id")
	processed, err := h.email.SyncUserEmails(userID)
	if err != nil {
		writeGmailError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"processed": processed})
}

// Disconnect handles POST /api/v1/candidate/email/disconnect
func (h *EmailV1Handler) Disconnect(c *gin.Context) {
	userID := c.GetUint("user_id")
	if err := h.email.DisconnectGmail(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect Gmail"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Gmail disconnected"})
}

// writeGmailError turns a Gmail failure into a stable machine code plus a message the
// UI can show. The internal error text stays in the server log, never in the response.
func writeGmailError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrGmailNotConnected):
		c.JSON(http.StatusConflict, gin.H{
			"error":   "gmail_not_connected",
			"message": "Connect Gmail to import your recruitment emails.",
		})
	case errors.Is(err, services.ErrGmailReconnect):
		c.JSON(http.StatusConflict, gin.H{
			"error":   "gmail_reconnect_required",
			"message": "Your Gmail authorization no longer works. Reconnect Gmail to keep importing emails.",
		})
	case errors.Is(err, services.ErrGmailMisconfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "gmail_misconfigured",
			"message": "Gmail is not configured correctly on the XHire server, so no mailbox can be read yet.",
		})
	default:
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "gmail_sync_failed",
			"message": "Gmail could not be reached right now. Try again in a few minutes.",
		})
	}
}
