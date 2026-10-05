//go:build ignore

package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"job_application_analysis/internal/database"
	"job_application_analysis/internal/models"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL is not set")
	}

	// Connect to database
	log.Println("📊 Connecting to database...")
	db := database.Connect(dbURL)
	log.Println("✅ Connected!")

	// Get first user (or create one for testing)
	var user models.User
	if err := db.First(&user).Error; err != nil {
		log.Println("⚠️  No users found. Please signup first!")
		return
	}

	log.Println("👤 Found user:", user.Email)

	// Create sample notifications
	notifications := []models.Notification{
		{
			UserID:    user.ID,
			Type:      "email",
			Title:     "New Email from Google",
			Message:   "Interview invitation received for Software Engineer position",
			Read:      false,
			ActionURL: "/emails",
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			UserID:    user.ID,
			Type:      "application",
			Title:     "Application Status Updated",
			Message:   "Your application at Microsoft moved to Interview stage",
			Read:      false,
			ActionURL: "/applications",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			UserID:    user.ID,
			Type:      "system",
			Title:     "Weekly Summary Ready",
			Message:   "Your weekly job search summary is now available",
			Read:      true,
			ActionURL: "/dashboard",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			UserID:    user.ID,
			Type:      "email",
			Title:     "Follow-up Reminder",
			Message:   "Time to follow up on your application at Amazon (7 days old)",
			Read:      true,
			ActionURL: "/applications",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			UserID:    user.ID,
			Type:      "application",
			Title:     "New Application Added",
			Message:   "Successfully added application for Senior Developer at Netflix",
			Read:      false,
			ActionURL: "/applications",
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}

	for _, notif := range notifications {
		if err := db.Create(&notif).Error; err != nil {
			log.Println("❌ Failed to create notification:", err)
		} else {
			log.Println("✅ Created:", notif.Title)
		}
	}

	log.Println("🎉 Seed complete! Check your notifications page.")
}
