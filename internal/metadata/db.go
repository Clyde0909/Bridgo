package metadata

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb" // DuckDB driver
	"golang.org/x/crypto/bcrypt"
)

const dbFileName = "bridgo_meta.db"

// InitDB initializes the DuckDB database connection and creates tables if they don't exist.
// It returns the database connection pool.
func InitDB(basePath string) (*sql.DB, error) {
	dbPath := filepath.Join(basePath, dbFileName)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Successfully connected to DuckDB at:", dbPath)

	if err = createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Ensure admin user exists
	if err = ensureAdminUserExists(db); err != nil {
		return nil, fmt.Errorf("failed to ensure admin user: %w", err)
	}

	return db, nil
}

// createSchema creates all necessary tables if they do not already exist.
func createSchema(db *sql.DB) error {
	schemaSQL := `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    role_name TEXT UNIQUE NOT NULL,
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY,
    permission_name TEXT UNIQUE NOT NULL,
    description TEXT,
    category TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id)
);

CREATE TABLE IF NOT EXISTS data_sources (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    source_name TEXT NOT NULL,
    db_type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT,
    db_username TEXT,
    password_encrypted TEXT,
    ssl_mode TEXT,
    additional_params TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_connection_status TEXT,
    last_connection_at TIMESTAMP,
    last_error_message TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS data_source_schemas (
    id TEXT PRIMARY KEY,
    data_source_id TEXT NOT NULL,
    schema_name TEXT,
    table_name TEXT NOT NULL,
    column_name TEXT NOT NULL,
    column_type TEXT NOT NULL,
    is_nullable BOOLEAN,
    retrieved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

CREATE TABLE IF NOT EXISTS user_datasource_privileges (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    data_source_id TEXT NOT NULL,
    privilege_type TEXT NOT NULL,
    can_grant BOOLEAN DEFAULT FALSE NOT NULL,
    granted_by_user_id TEXT,
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id),
    FOREIGN KEY (granted_by_user_id) REFERENCES users(id),
    UNIQUE (user_id, data_source_id, privilege_type)
);

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    preference_value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, preference_key),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    action_type TEXT NOT NULL,
    target_resource_id TEXT,
    details TEXT,
    ip_address TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS saved_queries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    query_name TEXT NOT NULL,
    query_text TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS virtual_views (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    definition TEXT NOT NULL, -- JSON string detailing the view structure
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (user_id, name) -- A user cannot have two virtual views with the same name
);
	`

	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema creation SQL: %w", err)
	}

	fmt.Println("Database schema checked/created successfully.")
	return nil
}

// ensureAdminUserExists checks if an admin user exists and creates one if not.
func ensureAdminUserExists(db *sql.DB) error {
	var userID string
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", "admin").Scan(&userID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for admin user: %w", err)
	}

	if err == sql.ErrNoRows {
		// Admin user does not exist, create it
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		adminUserID := uuid.NewString()
		adminEmail := "admin@admin.com" // Using a valid email format
		now := time.Now().UTC()

		_, err = db.Exec(
			"INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			adminUserID, "admin", adminEmail, string(hashedPassword), true, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert admin user: %w", err)
		}
		fmt.Println("Admin user created successfully.")
	} else {
		fmt.Println("Admin user already exists.")
	}
	return nil
}

// Helper function to get the project root (if needed, for now db is in root)
func getProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}
