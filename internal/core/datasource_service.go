package core

import (
	"database/sql"
	"fmt"

	"Bridgo/internal/models"
)

// DataSourceService handles data source management operations
type DataSourceService struct {
	metaDB *sql.DB
}

// NewDataSourceService creates a new DataSourceService
func NewDataSourceService(metaDB *sql.DB) *DataSourceService {
	return &DataSourceService{metaDB: metaDB}
}

// GetUserDataSources retrieves all data sources for a user
func (dss *DataSourceService) GetUserDataSources(user_id string) ([]models.DataSource, error) {
	query := `
        SELECT id, user_id, source_name, db_type, host, port, database_name, db_username, description, created_at, updated_at, last_connection_status, last_connection_at
        FROM data_sources 
        WHERE user_id = ? 
        ORDER BY created_at DESC
    `

	rows, err := dss.metaDB.Query(query, user_id)
	if err != nil {
		return nil, fmt.Errorf("failed to query data sources: %w", err)
	}
	defer rows.Close()

	var data_sources []models.DataSource
	for rows.Next() {
		var ds models.DataSource
		var description sql.NullString
		var last_connection_status sql.NullString
		var last_connection_at sql.NullTime

		err = rows.Scan(&ds.ID, &ds.UserID, &ds.SourceName, &ds.DBType, &ds.Host, &ds.Port, &ds.DatabaseName, &ds.DBUsername, &description, &ds.CreatedAt, &ds.UpdatedAt, &last_connection_status, &last_connection_at)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data source: %w", err)
		}

		ds.Description = description
		ds.LastConnectionStatus = last_connection_status
		ds.LastConnectionAt = last_connection_at

		data_sources = append(data_sources, ds)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating data source rows: %w", err)
	}

	return data_sources, nil
}

// GetDataSourceSchema retrieves schema for a specific data source
func (dss *DataSourceService) GetDataSourceSchema(data_source_id string, user_id string) ([]models.DataSourceSchema, error) {
	// First verify the data source belongs to the user
	var count int
	err := dss.metaDB.QueryRow("SELECT COUNT(*) FROM data_sources WHERE id = ? AND user_id = ?", data_source_id, user_id).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to verify data source ownership: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("data source not found or access denied")
	}

	query := `
        SELECT id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, retrieved_at
        FROM data_source_schemas 
        WHERE data_source_id = ? 
        ORDER BY table_name, column_name
    `

	rows, err := dss.metaDB.Query(query, data_source_id)
	if err != nil {
		return nil, fmt.Errorf("failed to query data source schemas: %w", err)
	}
	defer rows.Close()

	var schemas []models.DataSourceSchema
	for rows.Next() {
		var schema models.DataSourceSchema
		err = rows.Scan(&schema.ID, &schema.DataSourceID, &schema.SchemaName, &schema.TableName, &schema.ColumnName, &schema.ColumnType, &schema.IsNullable, &schema.RetrievedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	return schemas, nil
}
