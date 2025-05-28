package core

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"Bridgo/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionService handles database connection and schema operations
type ConnectionService struct {
	metaDB *sql.DB
}

// NewConnectionService creates a new ConnectionService
func NewConnectionService(metaDB *sql.DB) *ConnectionService {
	return &ConnectionService{metaDB: metaDB}
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
func (cs *ConnectionService) ConnectAndFetchSchema(input ConnectAndFetchSchemaInput) ([]models.DataSourceSchema, error) {
	var dsn string
	var driver_name string

	switch input.DBType {
	case "postgresql":
		driver_name = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			input.Host, input.Port, input.User, input.Password, input.DBName)
	case "mysql":
		driver_name = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			input.User, input.Password, input.Host, input.Port, input.DBName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", input.DBType)
	}

	ext_db, err := sql.Open(driver_name, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open external database connection: %w", err)
	}
	defer ext_db.Close()

	ping_err := ext_db.Ping()
	now := time.Now().UTC()

	// Begin transaction for metadata updates
	tx, err := cs.metaDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin metadata transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Save Data Source
	data_source_id := uuid.NewString()
	// For simplicity, storing password as is. In production, encrypt it!
	// TODO: Implement proper encryption for password
	password_encrypted := sql.NullString{String: input.Password, Valid: input.Password != ""}

	_, err = tx.Exec(`
        INSERT INTO data_sources (id, user_id, source_name, db_type, host, port, database_name, db_username, password_encrypted, created_at, updated_at, last_connection_status, last_connection_at, last_error_message)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, data_source_id, input.UserID, input.SourceName, input.DBType, input.Host, input.Port, input.DBName, input.User, password_encrypted, now, now,
		func() string {
			if ping_err == nil {
				return "connected"
			} else {
				return "failed"
			}
		}(),
		func() sql.NullTime {
			if ping_err == nil {
				return sql.NullTime{Time: now, Valid: true}
			} else {
				return sql.NullTime{Valid: false}
			}
		}(),
		func() sql.NullString {
			if ping_err != nil {
				return sql.NullString{String: ping_err.Error(), Valid: true}
			} else {
				return sql.NullString{Valid: false}
			}
		}(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save data source: %w", err)
	}

	if ping_err != nil {
		_ = tx.Commit() // Commit the data source with failed status
		return nil, fmt.Errorf("failed to ping external database: %w", ping_err)
	}

	log.Printf("Successfully connected to %s database: %s for user: %s, source: %s\n", input.DBType, input.DBName, input.UserID, input.SourceName)

	// Fetch and Save Schema
	fetched_schema, err := cs.fetchSchemaFromDatabase(ext_db, input, data_source_id, now)
	if err != nil {
		return nil, err
	}

	// Save schema to metadata database
	for _, schemaItem := range fetched_schema {
		_, err = tx.Exec(`
            INSERT INTO data_source_schemas (id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, is_primary_key, retrieved_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `, schemaItem.ID, schemaItem.DataSourceID, schemaItem.SchemaName, schemaItem.TableName, schemaItem.ColumnName, schemaItem.ColumnType, schemaItem.IsNullable, schemaItem.IsPrimaryKey, schemaItem.RetrievedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save data source schema item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit metadata transaction: %w", err)
	}

	return fetched_schema, nil
}

// TestConnectionAndFetchSchema connects to a database and fetches its schema
// without saving to metadata. Returns schema for preview.
func (cs *ConnectionService) TestConnectionAndFetchSchema(input ConnectAndFetchSchemaInput) ([]models.DataSourceSchema, error) {
	var dsn string
	var driver_name string

	switch input.DBType {
	case "postgresql":
		driver_name = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			input.Host, input.Port, input.User, input.Password, input.DBName)
	case "mysql":
		driver_name = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			input.User, input.Password, input.Host, input.Port, input.DBName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", input.DBType)
	}

	ext_db, err := sql.Open(driver_name, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open external database connection: %w", err)
	}
	defer ext_db.Close()

	ping_err := ext_db.Ping()
	if ping_err != nil {
		return nil, fmt.Errorf("failed to ping external database: %w", ping_err)
	}

	log.Printf("Successfully tested connection to %s database: %s\n", input.DBType, input.DBName)

	// Fetch Schema (without saving)
	now := time.Now().UTC()
	return cs.fetchSchemaFromDatabase(ext_db, input, "", now)
}

// SaveDataSource saves a data source and its schema to metadata after successful testing
func (cs *ConnectionService) SaveDataSource(input ConnectAndFetchSchemaInput, schema []models.DataSourceSchema) (*models.DataSource, error) {
	now := time.Now().UTC()

	// Begin transaction for metadata updates
	tx, err := cs.metaDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin metadata transaction: %w", err)
	}
	defer tx.Rollback()

	// Save Data Source
	data_source_id := uuid.NewString()
	password_encrypted := sql.NullString{String: input.Password, Valid: input.Password != ""}

	_, err = tx.Exec(`
        INSERT INTO data_sources (id, user_id, source_name, db_type, host, port, database_name, db_username, password_encrypted, created_at, updated_at, last_connection_status, last_connection_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, data_source_id, input.UserID, input.SourceName, input.DBType, input.Host, input.Port, input.DBName, input.User, password_encrypted, now, now, "connected", sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to save data source: %w", err)
	}

	// Save Schema with new IDs
	for _, schemaItem := range schema {
		new_schema_id := uuid.NewString()
		_, err = tx.Exec(`
            INSERT INTO data_source_schemas (id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, is_primary_key, retrieved_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `, new_schema_id, data_source_id, schemaItem.SchemaName, schemaItem.TableName, schemaItem.ColumnName, schemaItem.ColumnType, schemaItem.IsNullable, schemaItem.IsPrimaryKey, schemaItem.RetrievedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save data source schema item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit metadata transaction: %w", err)
	}

	// Return the saved data source
	saved_data_source := &models.DataSource{
		ID:           data_source_id,
		UserID:       input.UserID,
		SourceName:   input.SourceName,
		DBType:       input.DBType,
		Host:         sql.NullString{String: input.Host, Valid: true},
		Port:         sql.NullInt64{Int64: int64(input.Port), Valid: true},
		DatabaseName: sql.NullString{String: input.DBName, Valid: true},
		DBUsername:   sql.NullString{String: input.User, Valid: true},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	log.Printf("Successfully saved data source: %s for user: %s\n", input.SourceName, input.UserID)
	return saved_data_source, nil
}

// fetchSchemaFromDatabase is a helper function to fetch schema from a database
func (cs *ConnectionService) fetchSchemaFromDatabase(ext_db *sql.DB, input ConnectAndFetchSchemaInput, data_source_id string, now time.Time) ([]models.DataSourceSchema, error) {
	var fetched_schema []models.DataSourceSchema

	schema_query := ""
	switch input.DBType {
	case "postgresql":
		// For PostgreSQL, using a LEFT JOIN approach for PK detection.
		// Assumes tables are in the 'public' schema. If different, this needs adjustment.
		schema_query = `
            SELECT
                c.table_schema,
                c.table_name,
                c.column_name,
                c.data_type,
                c.is_nullable,
                CASE
                    WHEN pk_tc.is_primary IS NOT NULL THEN 'YES'
                    ELSE 'NO'
                END AS is_primary_key
            FROM information_schema.columns c
            LEFT JOIN (
                SELECT
                    kcu.table_schema,
                    kcu.table_name,
                    kcu.column_name,
                    true AS is_primary
                FROM information_schema.key_column_usage kcu
                JOIN information_schema.table_constraints tc
                  ON kcu.constraint_schema = tc.constraint_schema
                 AND kcu.constraint_name = tc.constraint_name
                 AND tc.constraint_type = 'PRIMARY KEY'
            ) pk_tc
              ON c.table_schema = pk_tc.table_schema
             AND c.table_name = pk_tc.table_name
             AND c.column_name = pk_tc.column_name
            WHERE c.table_schema = 'public' -- Consider making schema configurable if not always 'public'
            ORDER BY c.table_schema, c.table_name, c.ordinal_position;
        `
	case "mysql":
		// For MySQL, TABLE_SCHEMA is often the database name itself.
		schema_query = fmt.Sprintf(`
            SELECT 
                c.TABLE_SCHEMA, 
                c.TABLE_NAME, 
                c.COLUMN_NAME, 
                c.DATA_TYPE, 
                c.IS_NULLABLE,
                CASE 
                    WHEN k.CONSTRAINT_NAME = 'PRIMARY' THEN 'YES' 
                    ELSE 'NO' 
                END AS is_primary_key
            FROM INFORMATION_SCHEMA.COLUMNS c
            LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE k 
                ON c.TABLE_SCHEMA = k.TABLE_SCHEMA
                AND c.TABLE_NAME = k.TABLE_NAME
                AND c.COLUMN_NAME = k.COLUMN_NAME
                AND k.CONSTRAINT_NAME = 'PRIMARY'
            WHERE c.TABLE_SCHEMA = '%s' 
            ORDER BY c.TABLE_SCHEMA, c.TABLE_NAME, c.ORDINAL_POSITION;
        `, input.DBName)
	default:
		return nil, fmt.Errorf("schema fetching for %s not implemented", input.DBType)
	}

	rows, err := ext_db.Query(schema_query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema information: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ds_schema models.DataSourceSchema
		var schema_name sql.NullString
		var is_nullable_str string    // Read IS_NULLABLE as string first for flexibility
		var is_primary_key_str string // Read IS_PRIMARY_KEY as string

		if input.DBType == "postgresql" {
			err = rows.Scan(&schema_name, &ds_schema.TableName, &ds_schema.ColumnName, &ds_schema.ColumnType, &is_nullable_str, &is_primary_key_str)
		} else if input.DBType == "mysql" {
			err = rows.Scan(&schema_name, &ds_schema.TableName, &ds_schema.ColumnName, &ds_schema.ColumnType, &is_nullable_str, &is_primary_key_str)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		ds_schema.ID = uuid.NewString()
		ds_schema.DataSourceID = data_source_id
		ds_schema.SchemaName = schema_name
		ds_schema.RetrievedAt = now
		if is_nullable_str == "YES" || is_nullable_str == "TRUE" {
			ds_schema.IsNullable = sql.NullBool{Bool: true, Valid: true}
		} else if is_nullable_str == "NO" || is_nullable_str == "FALSE" {
			ds_schema.IsNullable = sql.NullBool{Bool: false, Valid: true}
		} else {
			ds_schema.IsNullable = sql.NullBool{Valid: false}
		}

		if is_primary_key_str == "YES" {
			ds_schema.IsPrimaryKey = sql.NullBool{Bool: true, Valid: true}
		} else {
			ds_schema.IsPrimaryKey = sql.NullBool{Bool: false, Valid: true}
		}

		fetched_schema = append(fetched_schema, ds_schema)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	return fetched_schema, nil
}
