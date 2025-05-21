package models

import (
	"database/sql"
	"time"
)

// DataSource represents the structure of the 'data_sources' table.
type DataSource struct {
	ID                   string         `json:"id"`
	UserID               string         `json:"user_id"`
	SourceName           string         `json:"source_name"`
	DBType               string         `json:"db_type"`
	Host                 sql.NullString `json:"host"`
	Port                 sql.NullInt64  `json:"port"`
	DatabaseName         sql.NullString `json:"database_name"`
	DBUsername           sql.NullString `json:"db_username"`
	PasswordEncrypted    sql.NullString `json:"password_encrypted"` // Store securely, e.g., encrypted
	SSLMode              sql.NullString `json:"ssl_mode"`
	AdditionalParams     sql.NullString `json:"additional_params"`
	Description          sql.NullString `json:"description"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	LastConnectionStatus sql.NullString `json:"last_connection_status"`
	LastConnectionAt     sql.NullTime   `json:"last_connection_at"`
	LastErrorMessage     sql.NullString `json:"last_error_message"`
}

// DataSourceSchema represents the structure of the 'data_source_schemas' table.
type DataSourceSchema struct {
	ID           string         `json:"id"`
	DataSourceID string         `json:"data_source_id"`
	SchemaName   sql.NullString `json:"schema_name"`
	TableName    string         `json:"table_name"`
	ColumnName   string         `json:"column_name"`
	ColumnType   string         `json:"column_type"`
	IsNullable   sql.NullBool   `json:"is_nullable"`
	RetrievedAt  time.Time      `json:"retrieved_at"`
}
