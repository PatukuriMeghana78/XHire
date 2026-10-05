package database

import (
	"log"

	"job_application_analysis/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dbURL string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database: ", err)
	}

	// Auto migrate all models
	err = db.AutoMigrate(
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
		log.Fatal("❌ Failed to migrate database: ", err)
	}

	log.Println("✅ Database connected and migrated successfully")

	return db
}
