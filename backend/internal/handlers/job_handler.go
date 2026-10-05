package handlers

import (
	"net/http"
	"strconv"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobs     *services.JobService
	apps     *services.ApplicationService
	match    *services.MatchService
	profiles *repositories.ProfileRepository
}

func NewJobHandler(
	jobs *services.JobService,
	apps *services.ApplicationService,
	match *services.MatchService,
	profiles *repositories.ProfileRepository,
) *JobHandler {
	return &JobHandler{jobs: jobs, apps: apps, match: match, profiles: profiles}
}

type jobRequest struct {
	Title            string   `json:"title"`
	CompanyName      string   `json:"companyName"`
	Location         string   `json:"location"`
	WorkMode         string   `json:"workMode"`
	JobType          string   `json:"jobType"`
	SalaryMin        *int     `json:"salaryMin"`
	SalaryMax        *int     `json:"salaryMax"`
	ExperienceMin    *int     `json:"experienceMin"`
	Description      string   `json:"description"`
	Responsibilities []string `json:"responsibilities"`
	RequiredSkills   []string `json:"requiredSkills"`
	Status           string   `json:"status"`
}

func (r jobRequest) toInput() services.JobInput {
	return services.JobInput{
		Title:            r.Title,
		CompanyName:      r.CompanyName,
		Location:         r.Location,
		WorkMode:         r.WorkMode,
		JobType:          r.JobType,
		SalaryMin:        r.SalaryMin,
		SalaryMax:        r.SalaryMax,
		ExperienceMin:    r.ExperienceMin,
		Description:      r.Description,
		Responsibilities: r.Responsibilities,
		RequiredSkills:   r.RequiredSkills,
		Status:           r.Status,
	}
}

func (h *JobHandler) toResponse(j *models.Job) dto.JobResponse {
	return dto.JobFromModel(j, h.jobs.ApplicantCount(j.ID))
}

func (h *JobHandler) toResponseList(jobs []models.Job) []dto.JobResponse {
	out := make([]dto.JobResponse, 0, len(jobs))
	for i := range jobs {
		out = append(out, h.toResponse(&jobs[i]))
	}
	return out
}

// List handles GET /api/v1/jobs (open jobs for candidates).
func (h *JobHandler) List(c *gin.Context) {
	filters := repositories.JobFilters{
		Search:   c.Query("search"),
		Location: c.Query("location"),
		JobType:  c.Query("job_type"),
		WorkMode: c.Query("work_mode"),
		Status:   "open",
	}
	if v := c.Query("min_experience"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filters.MinExperience = &n
		}
	}
	jobs, err := h.jobs.List(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}
	c.JSON(http.StatusOK, h.toResponseList(jobs))
}

// Get handles GET /api/v1/jobs/:id
func (h *JobHandler) Get(c *gin.Context) {
	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	job, err := h.jobs.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	resp := h.toResponse(job)
	c.JSON(http.StatusOK, resp)
}

// Create handles POST /api/v1/jobs (recruiter).
func (h *JobHandler) Create(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := h.jobs.Create(recruiterID, req.toInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, h.toResponse(job))
}

// Update handles PUT /api/v1/jobs/:id (recruiter).
func (h *JobHandler) Update(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := h.jobs.Update(recruiterID, id, req.toInput())
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, h.toResponse(job))
}

// Delete handles DELETE /api/v1/jobs/:id (recruiter).
func (h *JobHandler) Delete(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	id, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.jobs.Delete(recruiterID, id); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// MyJobs handles GET /api/v1/recruiter/jobs
func (h *JobHandler) MyJobs(c *gin.Context) {
	recruiterID := c.GetUint("user_id")
	jobs, err := h.jobs.List(repositories.JobFilters{RecruiterID: recruiterID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}
	c.JSON(http.StatusOK, h.toResponseList(jobs))
}

// Apply handles POST /api/v1/jobs/:id/apply (candidate).
func (h *JobHandler) Apply(c *gin.Context) {
	candidateID := c.GetUint("user_id")
	jobID, err := dto.ParseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	job, err := h.jobs.Get(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	var req struct {
		CoverLetter string `json:"coverLetter"`
		ResumeID    string `json:"resumeId"`
	}
	_ = c.ShouldBindJSON(&req)

	var resumeID *uint
	if req.ResumeID != "" {
		if rid, perr := dto.ParseID(req.ResumeID); perr == nil {
			resumeID = &rid
		}
	}

	matchScore := 0
	skills := []string{}
	if profile, perr := h.profiles.GetCandidateByUserID(candidateID); perr == nil && profile != nil {
		skills = profile.Skills
	}
	matchScore = h.match.Score(job.RequiredSkills, skills).MatchScore

	app, err := h.apps.ApplyToJob(candidateID, job, req.CoverLetter, resumeID, matchScore)
	if err != nil {
		if err.Error() == "already applied" {
			c.JSON(http.StatusConflict, gin.H{"error": "already applied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not apply"})
		return
	}

	name := ""
	if profile, perr := h.profiles.GetCandidateByUserID(candidateID); perr == nil && profile != nil {
		name = profile.FullName
	}
	c.JSON(http.StatusCreated, dto.ApplicationFromModel(app, name))
}
