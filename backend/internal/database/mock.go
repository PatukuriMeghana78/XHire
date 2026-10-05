package database

import (
	"errors"
	"sync"
	"time"

	"job_application_analysis/internal/models"
)

// MockDB is an in-memory database for development without PostgreSQL
type MockDB struct {
	users        map[uint]*models.User
	applications map[uint]*models.Application
	histories    map[uint]*models.StatusHistory
	emails       map[uint]*models.EmailEvent
	userID       uint
	appID        uint
	historyID    uint
	emailID      uint
	mu           sync.RWMutex
}

var mockDB *MockDB

// InitMockDB initializes the mock database
func InitMockDB() *MockDB {
	mockDB = &MockDB{
		users:        make(map[uint]*models.User),
		applications: make(map[uint]*models.Application),
		histories:    make(map[uint]*models.StatusHistory),
		emails:       make(map[uint]*models.EmailEvent),
		userID:       1,
		appID:        1,
		historyID:    1,
		emailID:      1,
	}
	return mockDB
}

// CreateUser creates a new user
func (db *MockDB) CreateUser(user *models.User) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	user.ID = db.userID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	db.users[user.ID] = user
	db.userID++
	return nil
}

// FindUserByEmail finds a user by email
func (db *MockDB) FindUserByEmail(email string) (*models.User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	for _, user := range db.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// CreateApplication creates a new application
func (db *MockDB) CreateApplication(app *models.Application) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	app.ID = db.appID
	app.CreatedAt = time.Now()
	app.UpdatedAt = time.Now()
	db.applications[app.ID] = app
	db.appID++
	return nil
}

// FindApplicationsByUserID finds all applications for a user
func (db *MockDB) FindApplicationsByUserID(userID uint) ([]*models.Application, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	var apps []*models.Application
	for _, app := range db.applications {
		if app.UserID == userID {
			// Load history
			var history []models.StatusHistory
			for _, h := range db.histories {
				if h.ApplicationID == app.ID {
					history = append(history, *h)
				}
			}
			app.StatusHistory = history
			apps = append(apps, app)
		}
	}
	return apps, nil
}

// FindApplicationByID finds an application by ID
func (db *MockDB) FindApplicationByID(id uint) (*models.Application, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	app, exists := db.applications[id]
	if !exists {
		return nil, errors.New("application not found")
	}
	
	// Load history
	var history []models.StatusHistory
	for _, h := range db.histories {
		if h.ApplicationID == app.ID {
			history = append(history, *h)
		}
	}
	app.StatusHistory = history
	
	return app, nil
}

// UpdateApplication updates an application
func (db *MockDB) UpdateApplication(app *models.Application) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	if _, exists := db.applications[app.ID]; !exists {
		return errors.New("application not found")
	}
	
	app.UpdatedAt = time.Now()
	db.applications[app.ID] = app
	return nil
}

// DeleteApplication deletes an application
func (db *MockDB) DeleteApplication(id uint) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	delete(db.applications, id)
	
	// Delete associated history
	for hid, h := range db.histories {
		if h.ApplicationID == id {
			delete(db.histories, hid)
		}
	}
	
	return nil
}

// CreateStatusHistory creates a status history entry
func (db *MockDB) CreateStatusHistory(history *models.StatusHistory) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	history.ID = db.historyID
	history.CreatedAt = time.Now()
	history.UpdatedAt = time.Now()
	db.histories[history.ID] = history
	db.historyID++
	return nil
}

// GetMockDB returns the mock database instance
func GetMockDB() *MockDB {
	if mockDB == nil {
		return InitMockDB()
	}
	return mockDB
}
