package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewMemoryDB creates an in-memory SQLite database
// This works without CGO by using a pure Go implementation
func NewMemoryDB() *gorm.DB {
	// Use in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	log.Println("✅ Connected to in-memory database")
	return db
}
