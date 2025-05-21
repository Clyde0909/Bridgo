package server

import (
	"database/sql" // Added import

	"Bridgo/internal/core"
	"Bridgo/internal/users"
)

// App represents the central application structure, holding references to all services.
// It was formerly named Server, renamed to App for clarity as it now manages services.
type App struct {
	UserService *users.Service
	CoreService *core.Service
	// Add other services here as the application grows
}

// NewApp creates and returns a new App instance, initializing all its services.
// It was formerly NewServer, renamed to NewApp.
func NewApp(db *sql.DB) *App { // Modified to accept *sql.DB
	userService := users.NewService(db) // Pass db to users.NewService
	coreService := core.NewService(db)  // Pass db to core.NewService

	return &App{
		UserService: userService,
		CoreService: coreService,
	}
}

// The user management methods (AddUser, GetUserByUsername, ValidatePassword)
// have been moved to internal/users/service.go.

// The core data management methods will be part of internal/core/service.go.
