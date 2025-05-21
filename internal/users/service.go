package users

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"Bridgo/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service handles user-related operations using a database.
type Service struct {
	db *sql.DB // Database connection pool
}

// NewService creates and returns a new UserService instance.
func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

// AddUser adds a new user to the DuckDB database.
func (s *Service) AddUser(username, email, password string) (models.User, error) {
	// Check if username or email already exists
	var existingUserID string
	err := s.db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ? LIMIT 1", username, email).Scan(&existingUserID)
	if err != nil && err != sql.ErrNoRows {
		return models.User{}, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if err != sql.ErrNoRows {
		// Determine if it was username or email conflict for a more specific error
		var tempUsername string
		s.db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&tempUsername)
		if tempUsername == username {
			return models.User{}, errors.New("username already exists")
		}
		return models.User{}, errors.New("email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.NewString()
	now := time.Now().UTC() // Use UTC for timestamps

	newUser := models.User{
		ID:       userID,
		Username: username,
		Email:    email,
		Password: string(hashedPassword), // This is the hash
		IsActive: true,                   // New users are active by default
		CreatedAt: now,                  // Set Go time, DB will also set its default
		UpdatedAt: now,
	}

	_, err = s.db.Exec(
		"INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		newUser.ID, newUser.Username, newUser.Email, newUser.Password, newUser.IsActive, newUser.CreatedAt, newUser.UpdatedAt,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	// Return the user object as it was prepared for insertion (or fetch from DB for full accuracy)
	// For now, returning newUser is sufficient as DB defaults match what we set.
	return newUser, nil
}

// GetUserByUsername retrieves a user by their username from DuckDB.
func (s *Service) GetUserByUsername(username string) (models.User, error) {
	var user models.User
	var createdAtStr, updatedAtStr string // Read as string then parse

	err := s.db.QueryRow(
		"SELECT id, username, email, password_hash, is_active, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.IsActive, &createdAtStr, &updatedAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, errors.New("user not found")
		}
		return models.User{}, fmt.Errorf("failed to get user by username '%s': %w", username, err)
	}

	// Attempt to parse timestamps (example, adjust format if needed)
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr) // Or other appropriate format DuckDB uses
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

	return user, nil
}

// GetUserByID retrieves a user by their ID from DuckDB.
func (s *Service) GetUserByID(id string) (models.User, error) {
	var user models.User
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRow(
		"SELECT id, username, email, password_hash, is_active, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.IsActive, &createdAtStr, &updatedAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, errors.New("user not found")
		}
		return models.User{}, fmt.Errorf("failed to get user by id '%s': %w", id, err)
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
	return user, nil
}

// ValidatePassword checks if the provided password matches the stored hashed password in DuckDB.
func (s *Service) ValidatePassword(username, password string) (models.User, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return models.User{}, err // User not found or other DB error
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return models.User{}, errors.New("invalid password") // Password does not match
	}
	return user, nil // Validation successful
}
