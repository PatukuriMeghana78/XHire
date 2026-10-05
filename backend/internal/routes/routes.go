package routes

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"job_application_analysis/internal/handlers"
	"job_application_analysis/internal/middlewares"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/services"
)

func Register(r *gin.Engine, db *gorm.DB) {

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Repositories
	appRepo := repositories.NewApplicationRepository(db)
	userRepo := repositories.NewUserRepository(db)
	emailRepo := repositories.NewEmailEventRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)
	jobRepo := repositories.NewJobRepository(db)
	profileRepo := repositories.NewProfileRepository(db)

	// NLP service URL — reads from .env, defaults to port 8001
	nlpURL := os.Getenv("NLP_SERVICE_URL")
	if nlpURL == "" {
		nlpURL = "http://localhost:8001" // ← Change to 8001
	}

	// Services
	appService := services.NewApplicationService(appRepo)
	authService := services.NewAuthService(userRepo, profileRepo)
	analyticsService := services.NewAnalyticsService(appRepo)
	notificationService := services.NewNotificationService(notificationRepo)

	// XHire services
	jobService := services.NewJobService(jobRepo, profileRepo)
	candidateService := services.NewCandidateService(profileRepo, userRepo, jobRepo)
	recruiterService := services.NewRecruiterService(profileRepo, userRepo)
	matchService := services.NewMatchService(jobRepo, profileRepo, appRepo)

	emailService := services.NewEmailService(
		db,
		appRepo,
		emailRepo,
		userRepo,
		notificationRepo,
		nlpURL,
	)

	// Handlers
	appHandler := handlers.NewApplicationHandler(appService, db, notificationRepo)
	authHandler := handlers.NewAuthHandler(authService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	emailHandler := handlers.NewEmailHandler(emailService)
	suggestionHandler := handlers.NewSuggestionHandler(db)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	settingsHandler := handlers.NewSettingsHandler(userRepo)

	// XHire handlers
	jobHandler := handlers.NewJobHandler(jobService, appService, matchService, profileRepo)
	candidateHandler := handlers.NewCandidateHandler(candidateService, appService, matchService, profileRepo)
	recruiterHandler := handlers.NewRecruiterHandler(recruiterService, appService, matchService, jobService, profileRepo)
	applicationV1Handler := handlers.NewApplicationV1Handler(appService, profileRepo)

	// Public routes
	r.POST("/api/auth/signup", authHandler.Signup)
	r.POST("/api/auth/login", authHandler.Login)

	r.GET("/api/email/oauth", emailHandler.StartOAuth)
	r.GET("/api/email/oauth/callback", emailHandler.HandleOAuthCallback)

	// Protected routes
	api := r.Group("/api")
	api.Use(middlewares.AuthRequired())
	{
		// Applications
		api.POST("/applications", appHandler.Create)
		api.GET("/applications", appHandler.GetAll)
		api.GET("/applications/:id", appHandler.GetByID)
		api.PUT("/applications/:id", appHandler.Update)
		api.PATCH("/applications/:id/status", appHandler.UpdateStatus)
		api.DELETE("/applications/:id", appHandler.Delete)

		// Health Score
		api.GET("/applications/:id/health-score", appHandler.GetHealthScore)

		// Analytics
		api.GET("/analytics/dashboard", analyticsHandler.GetDashboardStats)
		api.GET("/analytics/distribution", analyticsHandler.GetStatusDistribution)
		api.GET("/analytics/time-to-response", analyticsHandler.GetTimeToResponse)
		api.GET("/analytics/weekly-summary", analyticsHandler.GetWeeklySummary)

		// Email suggestions
		api.GET("/email/suggestions", emailHandler.GetPendingSuggestions)
		api.POST("/email/confirm", emailHandler.ConfirmSuggestion)
		api.POST("/email/ignore", emailHandler.IgnoreSuggestion)
		api.POST("/email/sync", emailHandler.SyncEmails)

		// AI Suggestions
		api.POST("/suggestions", suggestionHandler.CreateSuggestion)
		api.GET("/suggestions", suggestionHandler.GetSuggestions)
		api.POST("/suggestions/:id/confirm", suggestionHandler.ConfirmSuggestion)

		// Notifications
		api.GET("/notifications", notificationHandler.GetNotifications)
		api.GET("/notifications/unread", notificationHandler.GetUnreadNotifications)
		api.GET("/notifications/unread/count", notificationHandler.GetUnreadCount)
		api.POST("/notifications/:id/read", notificationHandler.MarkAsRead)
		api.POST("/notifications/read-all", notificationHandler.MarkAllAsRead)

		// Settings
		api.GET("/settings", settingsHandler.GetSettings)
		api.PUT("/settings/profile", settingsHandler.UpdateProfile)
		api.POST("/settings/gmail/disconnect", settingsHandler.DisconnectGmail)
	}

	registerV1(r, authHandler, jobHandler, candidateHandler, recruiterHandler, applicationV1Handler)
}

// registerV1 wires the XHire /api/v1 surface used by the React frontend.
func registerV1(
	r *gin.Engine,
	authHandler *handlers.AuthHandler,
	jobHandler *handlers.JobHandler,
	candidateHandler *handlers.CandidateHandler,
	recruiterHandler *handlers.RecruiterHandler,
	applicationV1Handler *handlers.ApplicationV1Handler,
) {
	v1 := r.Group("/api/v1")

	// Public auth
	v1.POST("/auth/login", authHandler.LoginV1)
	v1.POST("/auth/register", authHandler.Register)

	// Authenticated (any role)
	authed := v1.Group("")
	authed.Use(middlewares.AuthRequired())
	{
		authed.GET("/auth/me", authHandler.Me)
		authed.POST("/auth/logout", authHandler.Logout)

		authed.GET("/jobs", jobHandler.List)
		authed.GET("/jobs/:id", jobHandler.Get)

		authed.GET("/applications/:id", applicationV1Handler.GetByID)
	}

	// Candidate-only
	candidate := v1.Group("")
	candidate.Use(middlewares.AuthRequired(), middlewares.RequireRole(models.RoleCandidate))
	{
		candidate.GET("/candidate/profile", candidateHandler.GetMyProfile)
		candidate.PUT("/candidate/profile", candidateHandler.UpdateMyProfile)

		candidate.GET("/candidate/resume", candidateHandler.GetMyResume)
		candidate.POST("/candidate/resume", candidateHandler.UploadResume)
		candidate.GET("/resumes/:resumeId/analysis", candidateHandler.GetResumeAnalysis)

		candidate.GET("/candidate/applications", candidateHandler.ListMyApplications)
		candidate.GET("/candidate/skill-gaps", candidateHandler.SkillGaps)
		candidate.GET("/candidate/jobs/:jobId/match", candidateHandler.JobMatch)

		candidate.POST("/jobs/:id/apply", jobHandler.Apply)
	}

	// Recruiter-only
	recruiter := v1.Group("")
	recruiter.Use(middlewares.AuthRequired(), middlewares.RequireRole(models.RoleRecruiter))
	{
		recruiter.GET("/recruiter/profile", recruiterHandler.GetMyProfile)
		recruiter.PUT("/recruiter/profile", recruiterHandler.UpdateMyProfile)

		// Job CRUD
		recruiter.POST("/jobs", jobHandler.Create)
		recruiter.PUT("/jobs/:id", jobHandler.Update)
		recruiter.DELETE("/jobs/:id", jobHandler.Delete)
		recruiter.GET("/recruiter/jobs", jobHandler.MyJobs)

		// Applications
		recruiter.GET("/recruiter/applications", recruiterHandler.ListApplications)
		recruiter.GET("/recruiter/jobs/:jobId/applications", recruiterHandler.ListJobApplications)
		recruiter.PATCH("/recruiter/applications/:id/status", recruiterHandler.UpdateStatus)

		// Candidate directory + AI ranking
		recruiter.GET("/candidates/:id", candidateHandler.GetCandidate)
		recruiter.GET("/candidates/:id/resume", candidateHandler.GetCandidateResume)
		recruiter.GET("/recruiter/jobs/:jobId/ranking", recruiterHandler.Ranking)
		recruiter.GET("/recruiter/jobs/:jobId/candidates/:candidateId/explanation", recruiterHandler.RankingExplanation)
	}
}
