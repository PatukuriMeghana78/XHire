package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
)

type CandidateHandler struct {
	candidates *services.CandidateService
	apps       *services.ApplicationService
	match      *services.MatchService
	profiles   *repositories.ProfileRepository
}

func NewCandidateHandler(
	candidates *services.CandidateService,
	apps *services.ApplicationService,
	match *services.MatchService,
	profiles *repositories.ProfileRepository,
) *CandidateHandler {
	return &CandidateHandler{candidates: candidates, apps: apps, match: match, profiles: profiles}
}

const resumeUploadDir = "uploads/resumes"

// GetMyProfile handles GET /api/v1/candidate/profile
func (h *CandidateHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	p, err := h.candidates.GetOrCreateProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load profile"})
		return
	}
	c.JSON(http.StatusOK, dto.CandidateFromModel(p))
}

// UpdateMyProfile handles PUT /api/v1/candidate/profile
func (h *CandidateHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var in models.CandidateProfile
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.candidates.UpdateProfile(userID, &in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}
	c.JSON(http.StatusOK, dto.CandidateFromModel(p))
}

// GetCandidate handles GET /api/v1/candidates/:id (recruiter directory view).
func (h *CandidateHandler) GetCandidate(c *gin.Context) {
	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.profiles.GetCandidateByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "candidate not found"})
		return
	}
	c.JSON(http.StatusOK, dto.CandidateFromModel(p))
}

// GetMyResume handles GET /api/v1/candidate/resume
func (h *CandidateHandler) GetMyResume(c *gin.Context) {
	userID := c.GetUint("user_id")
	resume, err := h.candidates.GetResume(userID)
	if err != nil || resume == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, dto.ResumeFromModel(resume))
}

// UploadResume handles POST /api/v1/candidate/resume (multipart field "resume").
func (h *CandidateHandler) UploadResume(c *gin.Context) {
	userID := c.GetUint("user_id")
	fh, err := c.FormFile("resume")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resume file required"})
		return
	}

	if err := os.MkdirAll(resumeUploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare storage"})
		return
	}

	safeName := filepath.Base(fh.Filename)
	stored := fmt.Sprintf("%d_%d_%s", userID, time.Now().UnixNano(), safeName)
	dst := filepath.Join(resumeUploadDir, stored)
	if err := c.SaveUploadedFile(fh, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save resume"})
		return
	}

	url := "/uploads/resumes/" + stored
	sizeKb := int(fh.Size / 1024)
	resume, err := h.candidates.SaveResume(userID, safeName, dst, url, sizeKb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record resume"})
		return
	}
	c.JSON(http.StatusCreated, dto.ResumeFromModel(resume))
}

// GetCandidateResume handles GET /api/v1/candidates/:id/resume (recruiter view).
func (h *CandidateHandler) GetCandidateResume(c *gin.Context) {
	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	resume, err := h.profiles.LatestResume(id)
	if err != nil || resume == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume not found"})
		return
	}
	c.JSON(http.StatusOK, dto.ResumeFromModel(resume))
}

// GetResumeAnalysis handles GET /api/v1/resumes/:resumeId/analysis
func (h *CandidateHandler) GetResumeAnalysis(c *gin.Context) {
	resumeID, err := dto.ParseID(c.Param("resumeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}
	resume, err := h.candidates.GetResumeByID(resumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume not found"})
		return
	}
	profile, _ := h.profiles.GetCandidateByUserID(resume.CandidateID)
	c.JSON(http.StatusOK, h.candidates.AnalyzeResume(resume, profile))
}

// ListMyApplications handles GET /api/v1/candidate/applications
func (h *CandidateHandler) ListMyApplications(c *gin.Context) {
	userID := c.GetUint("user_id")
	apps, err := h.apps.GetCandidateMarketplaceApps(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list applications"})
		return
	}
	name := ""
	if p, perr := h.profiles.GetCandidateByUserID(userID); perr == nil && p != nil {
		name = p.FullName
	}
	c.JSON(http.StatusOK, applicationsToResponse(apps, name))
}

// SkillGaps handles GET /api/v1/candidate/skill-gaps?job_id=
func (h *CandidateHandler) SkillGaps(c *gin.Context) {
	userID := c.GetUint("user_id")
	var jobID uint
	if v := c.Query("job_id"); v != "" {
		if id, err := dto.ParseID(v); err == nil {
			jobID = id
		}
	}
	gaps, err := h.match.SkillGaps(userID, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute skill gaps"})
		return
	}
	c.JSON(http.StatusOK, gaps)
}

// JobMatch handles GET /api/v1/candidate/jobs/:jobId/match
func (h *CandidateHandler) JobMatch(c *gin.Context) {
	userID := c.GetUint("user_id")
	jobID, err := dto.ParseID(c.Param("jobId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	result, err := h.match.JobMatch(userID, jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// applicationsToResponse converts a slice of applications into DTO responses.
// If name is empty, each candidate's name is resolved individually.
func applicationsToResponse(apps []models.Application, name string) []dto.ApplicationResponse {
	out := make([]dto.ApplicationResponse, 0, len(apps))
	for i := range apps {
		out = append(out, dto.ApplicationFromModel(&apps[i], name))
	}
	return out
}
