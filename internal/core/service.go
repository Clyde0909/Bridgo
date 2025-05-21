package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"Bridgo/internal/models"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
	// Add other drivers as needed
)

// Service manages core data virtualization logic.
type Service struct {
	metaDB *sql.DB // For storing metadata about connected sources, schemas, etc.
}

// NewService creates a new core Service.
func NewService(mdb *sql.DB) *Service {
	return &Service{metaDB: mdb}
}

// ConnectAndFetchSchemaInput defines the input for ConnectAndFetchSchema
type ConnectAndFetchSchemaInput struct {
	SourceName string `json:"sourceName"`
	DBType     string `json:"dbType"`
	Host       string `json:"dbHost"`
	Port       int    `json:"dbPort"`
	User       string `json:"dbUser"`
	Password   string `json:"dbPassword"`
	DBName     string `json:"dbName"`
	UserID     string `json:"-"` // UserID is passed internally, not from JSON request
}

// ConnectAndFetchSchema connects to a given database, fetches its schema,
// saves the data source and its schema, and returns the schema.
func (s *Service) ConnectAndFetchSchema(input ConnectAndFetchSchemaInput) ([]models.DataSourceSchema, error) {
	var dsn string
	var driverName string

	switch input.DBType {
	case "postgresql":
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			input.Host, input.Port, input.User, input.Password, input.DBName)
	case "mysql":
		driverName = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			input.User, input.Password, input.Host, input.Port, input.DBName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", input.DBType)
	}

	extDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open external database connection: %w", err)
	}
	defer extDB.Close()

	pingErr := extDB.Ping()
	now := time.Now().UTC()

	// Begin transaction for metadata updates
	tx, err := s.metaDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin metadata transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Save Data Source
	dataSourceID := uuid.NewString()
	// For simplicity, storing password as is. In production, encrypt it!
	// TODO: Implement proper encryption for password
	passwordEncrypted := sql.NullString{String: input.Password, Valid: input.Password != ""}

	_, err = tx.Exec(`
        INSERT INTO data_sources (id, user_id, source_name, db_type, host, port, database_name, db_username, password_encrypted, created_at, updated_at, last_connection_status, last_connection_at, last_error_message)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, dataSourceID, input.UserID, input.SourceName, input.DBType, input.Host, input.Port, input.DBName, input.User, passwordEncrypted, now, now,
		func() string {
			if pingErr == nil {
				return "connected"
			} else {
				return "failed"
			}
		}(),
		func() sql.NullTime {
			if pingErr == nil {
				return sql.NullTime{Time: now, Valid: true}
			} else {
				return sql.NullTime{Valid: false}
			}
		}(),
		func() sql.NullString {
			if pingErr != nil {
				return sql.NullString{String: pingErr.Error(), Valid: true}
			} else {
				return sql.NullString{Valid: false}
			}
		}(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save data source: %w", err)
	}

	if pingErr != nil {
		_ = tx.Commit() // Commit the data source with failed status
		return nil, fmt.Errorf("failed to ping external database: %w", pingErr)
	}

	log.Printf("Successfully connected to %s database: %s for user: %s, source: %s\n", input.DBType, input.DBName, input.UserID, input.SourceName)

	// Fetch and Save Schema
	var fetchedSchema []models.DataSourceSchema

	schemaQuery := ""
	switch input.DBType {
	case "postgresql":
		schemaQuery = `
            SELECT table_schema, table_name, column_name, data_type, is_nullable 
            FROM information_schema.columns 
            WHERE table_schema = 'public' -- Or use input.DBName if schema can be different
            ORDER BY table_schema, table_name, ordinal_position;
        `
	case "mysql":
		// For MySQL, TABLE_SCHEMA is often the database name itself.
		schemaQuery = fmt.Sprintf(`
            SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, DATA_TYPE, IS_NULLABLE
            FROM INFORMATION_SCHEMA.COLUMNS
            WHERE TABLE_SCHEMA = '%s' 
            ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION;
        `, input.DBName)
	default:
		_ = tx.Commit() // Commit data source even if schema fetch is not supported
		return nil, fmt.Errorf("schema fetching for %s not implemented", input.DBType)
	}

	rows, err := extDB.Query(schemaQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema information: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dsSchema models.DataSourceSchema
		var schemaName sql.NullString
		var isNullableStr string // Read IS_NULLABLE as string first for flexibility

		if input.DBType == "postgresql" {
			err = rows.Scan(&schemaName, &dsSchema.TableName, &dsSchema.ColumnName, &dsSchema.ColumnType, &isNullableStr)
		} else if input.DBType == "mysql" {
			err = rows.Scan(&schemaName, &dsSchema.TableName, &dsSchema.ColumnName, &dsSchema.ColumnType, &isNullableStr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		dsSchema.ID = uuid.NewString()
		dsSchema.DataSourceID = dataSourceID
		dsSchema.SchemaName = schemaName
		dsSchema.RetrievedAt = now
		if isNullableStr == "YES" || isNullableStr == "TRUE" { // Handle variations
			dsSchema.IsNullable = sql.NullBool{Bool: true, Valid: true}
		} else if isNullableStr == "NO" || isNullableStr == "FALSE" {
			dsSchema.IsNullable = sql.NullBool{Bool: false, Valid: true}
		} else {
			dsSchema.IsNullable = sql.NullBool{Valid: false} // Unknown or NULL
		}

		_, err = tx.Exec(`
            INSERT INTO data_source_schemas (id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, retrieved_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        `, dsSchema.ID, dsSchema.DataSourceID, dsSchema.SchemaName, dsSchema.TableName, dsSchema.ColumnName, dsSchema.ColumnType, dsSchema.IsNullable, dsSchema.RetrievedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save data source schema item: %w", err)
		}
		fetchedSchema = append(fetchedSchema, dsSchema)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit metadata transaction: %w", err)
	}

	return fetchedSchema, nil
}

// CreateVirtualViewInput defines the input for creating a virtual view.
type CreateVirtualViewInput struct {
	UserID            string   `json:"-"` // Passed internally
	Name              string   `json:"name"`
	Description       *string  `json:"description"`
	SelectedSchemaIDs []string `json:"selected_schema_ids"`
}

// CreateVirtualView creates a new virtual view based on selected schema elements.
func (s *Service) CreateVirtualView(input CreateVirtualViewInput) (*models.VirtualView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("virtual view name cannot be empty")
	}
	if len(input.SelectedSchemaIDs) == 0 {
		return nil, fmt.Errorf("at least one schema column must be selected")
	}

	// TODO: Validate that all SelectedSchemaIDs belong to data sources accessible by UserID
	// This requires joining data_source_schemas with data_sources and checking user_id.
	// For now, we assume they are valid and accessible.

	definition := models.VirtualViewDefinition{
		SelectedColumns: make([]models.SelectedColumn, len(input.SelectedSchemaIDs)),
	}
	for i, schemaID := range input.SelectedSchemaIDs {
		definition.SelectedColumns[i] = models.SelectedColumn{DataSourceSchemaID: schemaID}
	}

	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal virtual view definition: %w", err)
	}

	now := time.Now().UTC()
	virtualView := &models.VirtualView{
		ID:          uuid.NewString(),
		UserID:      input.UserID,
		Name:        input.Name,
		Description: input.Description,
		Definition:  string(definitionJSON),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = s.metaDB.Exec(`
        INSERT INTO virtual_views (id, user_id, name, description, definition, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, virtualView.ID, virtualView.UserID, virtualView.Name, virtualView.Description, virtualView.Definition, virtualView.CreatedAt, virtualView.UpdatedAt)

	if err != nil {
		// Handle potential unique constraint violation (user_id, name)
		// This might require checking the error type or message if the DB driver provides it.
		// For DuckDB, a generic error might be returned. A more robust check would be needed.
		// Example: if strings.Contains(err.Error(), "UNIQUE constraint failed: virtual_views.user_id, virtual_views.name") { ... }
		return nil, fmt.Errorf("failed to save virtual view: %w. Ensure the name is unique for this user.", err)
	}

	return virtualView, nil
}

// Placeholder for querying data through a virtualized connection
func (s *Service) QueryData(userID string, dataSourceID string, query string) (interface{}, error) {
	// 1. Look up dataSourceID in metaDB to get connection details.
	// 2. Connect to the actual database.
	// 3. Execute the query.
	// 4. Return results.
	return nil, fmt.Errorf("QueryData not yet implemented")
}
