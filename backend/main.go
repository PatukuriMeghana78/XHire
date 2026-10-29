package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"job_application_analysis/internal/config"
	"job_application_analysis/internal/database"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
	"job_application_analysis/internal/routes"
	"job_application_analysis/internal/services"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system env")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL is not set")
	}

	// Fail safely if the JWT signing secret is missing or insecure.
	if err := config.ValidateJWTSecret(); err != nil {
		log.Fatal("❌ JWT configuration error:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to PostgreSQL
	log.Println("📊 Connecting to PostgreSQL database...")
	db := database.Connect(dbURL)
	log.Println("✅ Database connected successfully!")

	// Run migrations
	log.Println("🔄 Running database migrations...")
	err := db.AutoMigrate(
		&models.User{},
		&models.Application{},
		&models.StatusHistory{},
		&models.EmailEvent{},
		&models.SuggestedUpdate{},
		&models.GmailToken{},
		&models.Notification{},
		// XHire
		&models.Job{},
		&models.CandidateProfile{},
		&models.RecruiterProfile{},
		&models.Resume{},
	)
	if err != nil {
		log.Fatal("❌ Migration failed:", err)
	}
	log.Println("✅ Migrations complete!")

	r := gin.Default()

	// Repositories
	appRepo := repositories.NewApplicationRepository(db)
	userRepo := repositories.NewUserRepository(db)
	emailRepo := repositories.NewEmailEventRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)

	// Email Service
	emailService := services.NewEmailService(
		db,
		appRepo,
		emailRepo,
		userRepo,
		notificationRepo,
		os.Getenv("NLP_SERVICE_URL"),
	)

	// Background Gmail polling worker
	go func() {
		for {
			log.Println("🔄 Checking Gmail...")
			emailService.PollUnreadEmails()
			time.Sleep(3 * time.Minute)
		}
	}()

	// Register routes (pass notificationRepo)
	routes.Register(r, db)

	log.Println("🚀 Server running on port:", port)
	r.Run(":" + port)
}
