package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// maxResumeSizeBytes caps uploads at 5 MB (matches the frontend limit).
const maxResumeSizeBytes = 5 << 20

// allowedResumeExts is the whitelist of accepted resume file extensions.
var allowedResumeExts = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".txt":  true,
	".md":   true,
	".rtf":  true,
}

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

	if fh.Size > maxResumeSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resume too large (max 5 MB)"})
		return
	}

	// Keep the original name only for display; never let it reach the filesystem.
	original := filepath.Base(fh.Filename)
	ext := strings.ToLower(filepath.Ext(original))
	if !allowedResumeExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
		return
	}

	if err := os.MkdirAll(resumeUploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare storage"})
		return
	}

	// Server-generated name from trusted values only (user id + timestamp + whitelisted ext).
	stored := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
	dst := filepath.Join(resumeUploadDir, stored)
	if err := c.SaveUploadedFile(fh, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save resume"})
		return
	}

	sizeKb := int(fh.Size / 1024)
	resume, err := h.candidates.SaveResume(userID, original, dst, "", sizeKb)
	if err != nil {
		_ = os.Remove(dst) // don't leave an orphaned file if the DB record failed
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record resume"})
		return
	}
	c.JSON(http.StatusCreated, dto.ResumeFromModel(resume))
}

// DownloadResume handles GET /api/v1/resumes/:resumeId/download.
// Serves the stored file only to its owning candidate or to a recruiter.
func (h *CandidateHandler) DownloadResume(c *gin.Context) {
	resumeID, err := dto.ParseID(c.Param("resumeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume id"})
		return
	}

	resume, err := h.candidates.GetResumeByID(resumeID)
	if err != nil || resume == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume not found"})
		return
	}

	// Authorization: the owning candidate, or any recruiter (recruiters review candidates).
	userID := c.GetUint("user_id")
	role := c.GetString("role")
	if role != models.RoleRecruiter && resume.CandidateID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// Confine the served path to the upload directory (defense-in-depth against
	// any tampered/legacy FilePath value).
	absPath, ok := safeResumePath(resume.FilePath)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume file unavailable"})
		return
	}
	if _, statErr := os.Stat(absPath); statErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume file unavailable"})
		return
	}

	c.FileAttachment(absPath, sanitizeDownloadName(resume.FileName))
}

// resumeOwnedBy reports whether a resume record belongs to the given user.
func resumeOwnedBy(resume *models.Resume, userID uint) bool {
	return resume != nil && userID != 0 && resume.CandidateID == userID
}

// safeResumePath resolves a stored resume path and verifies it stays inside the
// upload directory. Returns ok=false if the path escapes or is empty.
func safeResumePath(stored string) (string, bool) {
	if stored == "" {
		return "", false
	}
	baseAbs, err := filepath.Abs(resumeUploadDir)
	if err != nil {
		return "", false
	}
	abs, err := filepath.Abs(stored)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(baseAbs, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return abs, true
}

// sanitizeDownloadName strips path separators and header-injection characters
// from the display filename used in Content-Disposition.
func sanitizeDownloadName(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\r', '\n', 0:
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "resume"
	}
	return name
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
	if err != nil || resume == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume not found"})
		return
	}

	// This route is candidate-only, so the caller is a candidate: only the owner
	// may read the analysis of their own resume (otherwise IDs are enumerable).
	if !resumeOwnedBy(resume, c.GetUint("user_id")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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
