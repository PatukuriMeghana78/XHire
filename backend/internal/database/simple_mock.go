package database

import (
	"log"
	"sync"
	"time"

	"job_application_analysis/internal/models"
	"gorm.io/gorm"
)

// SimpleMockDB wraps MockDB to provide GORM-like interface
type SimpleMockDB struct {
	*MockDB
}

// NewSimpleMockDB creates a new simple mock database that works without CGO
func NewSimpleMockDB() *gorm.DB {
	log.Println("📦 Using in-memory mock database (no CGO required)")
	log.Println("⚠️  Data will not persist between restarts")
	
	mockDB := InitMockDB()
	
	// Create a fake GORM DB - we'll intercept calls in repositories
	// This is a workaround to avoid CGO compilation issues on Windows
	db := &gorm.DB{
		Config: &gorm.Config{},
	}
	
	// Store mock DB reference globally
	globalMockDB = mockDB
	
	return db
}

var globalMockDB *MockDB
var mockMutex sync.RWMutex

// GetGlobalMockDB returns the global mock database instance
func GetGlobalMockDB() *MockDB {
	mockMutex.RLock()
	defer mockMutex.RUnlock()
	return globalMockDB
}

// Helper functions for repositories to use

// MockCreate creates a record in mock DB
func MockCreate(model interface{}) error {
	db := GetGlobalMockDB()
	if db == nil {
		return nil
	}
	
	switch v := model.(type) {
	case *models.User:
		return db.CreateUser(v)
	case *models.Application:
		return db.CreateApplication(v)
	case *models.StatusHistory:
		return db.CreateStatusHistory(v)
	}
	return nil
}

// MockFindUserByEmail finds user by email
func MockFindUserByEmail(email string) (*models.User, error) {
	db := GetGlobalMockDB()
	if db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return db.FindUserByEmail(email)
}

// MockFindApplicationsByUserID finds applications by user ID
func MockFindApplicationsByUserID(userID uint) ([]*models.Application, error) {
	db := GetGlobalMockDB()
	if db == nil {
		return nil, nil
	}
	return db.FindApplicationsByUserID(userID)
}

// MockFindApplicationByID finds application by ID
func MockFindApplicationByID(id uint) (*models.Application, error) {
	db := GetGlobalMockDB()
	if db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return db.FindApplicationByID(id)
}

// MockUpdateApplication updates an application
func MockUpdateApplication(app *models.Application) error {
	db := GetGlobalMockDB()
	if db == nil {
		return nil
	}
	return db.UpdateApplication(app)
}

// MockDeleteApplication deletes an application
func MockDeleteApplication(id uint) error {
	db := GetGlobalMockDB()
	if db == nil {
		return nil
	}
	return db.DeleteApplication(id)
}

// MockAutoMigrate does nothing for mock DB
func MockAutoMigrate(db *gorm.DB, models ...interface{}) error {
	log.Println("✅ Mock database initialized (no migrations needed)")
	return nil
}

// MockGetAnalytics returns mock analytics data
func MockGetAnalytics(userID uint) map[string]interface{} {
	db := GetGlobalMockDB()
	if db == nil {
		return map[string]interface{}{
			"total":      0,
			"applied":    0,
			"interview":  0,
			"offer":      0,
			"rejected":   0,
		}
	}
	
	apps, _ := db.FindApplicationsByUserID(userID)
	
	stats := map[string]int{
		"applied":   0,
		"interview": 0,
		"offer":     0,
		"rejected":  0,
	}
	
	for _, app := range apps {
		switch app.Status {
		case "applied":
			stats["applied"]++
		case "interview":
			stats["interview"]++
		case "offer":
			stats["offer"]++
		case "rejected":
			stats["rejected"]++
		}
	}
	
	return map[string]interface{}{
		"total":      len(apps),
		"applied":    stats["applied"],
		"interview":  stats["interview"],
		"offer":      stats["offer"],
		"rejected":   stats["rejected"],
		"updated_at": time.Now(),
	}
}
