package handlers

import (
	"net/http"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
)

// ApplicationV1Handler serves the shared XHire application detail endpoint,
// accessible by the owning candidate or the job's recruiter.
type ApplicationV1Handler struct {
	apps     *services.ApplicationService
	profiles *repositories.ProfileRepository
}

func NewApplicationV1Handler(
	apps *services.ApplicationService,
	profiles *repositories.ProfileRepository,
) *ApplicationV1Handler {
	return &ApplicationV1Handler{apps: apps, profiles: profiles}
}

// GetByID handles GET /api/v1/applications/:id
func (h *ApplicationV1Handler) GetByID(c *gin.Context) {
	requesterID := c.GetUint("user_id")
	role := c.GetString("role")

	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	app, err := h.apps.GetMarketplaceByID(id)
	if err != nil || app == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	authorized := false
	if role == models.RoleRecruiter {
		authorized = app.Job != nil && app.Job.RecruiterID == requesterID
	} else {
		authorized = app.CandidateID == requesterID || app.UserID == requesterID
	}
	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	cid := app.CandidateID
	if cid == 0 {
		cid = app.UserID
	}
	name := ""
	if p, perr := h.profiles.GetCandidateByUserID(cid); perr == nil && p != nil {
		name = p.FullName
	}

	c.JSON(http.StatusOK, dto.ApplicationFromModel(app, name))
}
