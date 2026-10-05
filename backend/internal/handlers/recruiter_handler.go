package handlers

import (
	"net/http"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
)

type RecruiterHandler struct {
	recruiters *services.RecruiterService
	apps       *services.ApplicationService
	match      *services.MatchService
	jobs       *services.JobService
	profiles   *repositories.ProfileRepository
}

func NewRecruiterHandler(
	recruiters *services.RecruiterService,
	apps *services.ApplicationService,
	match *services.MatchService,
	jobs *services.JobService,
	profiles *repositories.ProfileRepository,
) *RecruiterHandler {
	return &RecruiterHandler{
		recruiters: recruiters,
		apps:       apps,
		match:      match,
		jobs:       jobs,
		profiles:   profiles,
	}
}

func (h *RecruiterHandler) candidateName(app *models.Application) string {
	cid := app.CandidateID
	if cid == 0 {
		cid = app.UserID
	}
	if p, err := h.profiles.GetCandidateByUserID(cid); err == nil && p != nil {
		return p.FullName
	}
	return ""
}

func (h *RecruiterHandler) toResponses(apps []models.Application) []dto.ApplicationResponse {
	out := make([]dto.ApplicationResponse, 0, len(apps))
	for i := range apps {
		out = append(out, dto.ApplicationFromModel(&apps[i], h.candidateName(&apps[i])))
	}
	return out
}

// ownsJob verifies the job exists and belongs to the recruiter.
func (h *RecruiterHandler) ownsJob(recruiterID, jobID uint) (*models.Job, bool) {
	job, err := h.jobs.Get(jobID)
	if err != nil || job == nil {
		return nil, false
	}
	return job, job.RecruiterID == recruiterID
}

// GetMyProfile handles GET /api/v1/recruiter/profile
func (h *RecruiterHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	p, err := h.recruiters.GetOrCreateProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load profile"})
		return
	}
	c.JSON(http.StatusOK, dto.RecruiterFromModel(p))
}

// UpdateMyProfile handles PUT /api/v1/recruiter/profile
func (h *RecruiterHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var in models.RecruiterProfile
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.recruiters.UpdateProfile(userID, &in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}
	c.JSON(http.StatusOK, dto.RecruiterFromModel(p))
}

// ListApplications handles GET /api/v1/recruiter/applications
func (h *RecruiterHandler) ListApplications(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	apps, err := h.apps.GetRecruiterMarketplaceApps(recruiterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list applications"})
		return
	}
	c.JSON(http.StatusOK, h.toResponses(apps))
}

// ListJobApplications handles GET /api/v1/recruiter/jobs/:jobId/applications
func (h *RecruiterHandler) ListJobApplications(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	jobID, err := dto.ParseID(c.Param("jobId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	if _, ok := h.ownsJob(recruiterID, jobID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	apps, err := h.apps.GetJobApplications(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list applications"})
		return
	}
	c.JSON(http.StatusOK, h.toResponses(apps))
}

// UpdateStatus handles PATCH /api/v1/recruiter/applications/:id/status
func (h *RecruiterHandler) UpdateStatus(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	appID, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	app, err := h.apps.RecruiterUpdateStatus(recruiterID, appID, req.Status)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	c.JSON(http.StatusOK, dto.ApplicationFromModel(app, h.candidateName(app)))
}

// Ranking handles GET /api/v1/recruiter/jobs/:jobId/ranking
func (h *RecruiterHandler) Ranking(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	jobID, err := dto.ParseID(c.Param("jobId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	if _, ok := h.ownsJob(recruiterID, jobID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	ranked, err := h.match.RankJob(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rank candidates"})
		return
	}
	c.JSON(http.StatusOK, ranked)
}

// RankingExplanation handles GET /api/v1/recruiter/jobs/:jobId/candidates/:candidateId/explanation
func (h *RecruiterHandler) RankingExplanation(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	jobID, err := dto.ParseID(c.Param("jobId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	candidateID, err := dto.ParseID(c.Param("candidateId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid candidate id"})
		return
	}
	if _, ok := h.ownsJob(recruiterID, jobID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.match.JobMatch(candidateID, jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "match unavailable"})
		return
	}
	c.JSON(http.StatusOK, result)
}
