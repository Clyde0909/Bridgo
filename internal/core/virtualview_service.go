package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"Bridgo/internal/models"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// VirtualViewService handles virtual view operations
type VirtualViewService struct {
	metaDB *sql.DB
}

// NewVirtualViewService creates a new VirtualViewService
func NewVirtualViewService(metaDB *sql.DB) *VirtualViewService {
	return &VirtualViewService{metaDB: metaDB}
}

// CreateVirtualViewInput defines the input for creating a virtual view.
type CreateVirtualViewInput struct {
	UserID            string   `json:"-"` // Passed internally
	Name              string   `json:"name"`
	Description       *string  `json:"description"`
	SelectedSchemaIDs []string `json:"selected_schema_ids"`
}

// CreateVirtualView creates a new virtual view based on selected schema elements.
func (vvs *VirtualViewService) CreateVirtualView(input CreateVirtualViewInput) (*models.VirtualView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("virtual view name cannot be empty")
	}
	if len(input.SelectedSchemaIDs) == 0 {
		return nil, fmt.Errorf("at least one schema column must be selected")
	}

	// Validate that all SelectedSchemaIDs belong to data sources accessible by UserID
	for _, schema_id := range input.SelectedSchemaIDs {
		var count int
		err := vvs.metaDB.QueryRow(`
			SELECT COUNT(*) 
			FROM data_source_schemas dss 
			JOIN data_sources ds ON dss.data_source_id = ds.id 
			WHERE dss.id = ? AND ds.user_id = ?
		`, schema_id, input.UserID).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to validate schema access: %w", err)
		}
		if count == 0 {
			return nil, fmt.Errorf("schema with ID %s not found or access denied", schema_id)
		}
	}

	definition := models.VirtualViewDefinition{
		SelectedColumns: make([]models.SelectedColumn, len(input.SelectedSchemaIDs)),
	}
	for i, schema_id := range input.SelectedSchemaIDs {
		definition.SelectedColumns[i] = models.SelectedColumn{DataSourceSchemaID: schema_id}
	}

	definition_json, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal virtual view definition: %w", err)
	}

	now := time.Now().UTC()
	virtual_view := &models.VirtualView{
		ID:          uuid.NewString(),
		UserID:      input.UserID,
		Name:        input.Name,
		Description: input.Description,
		Definition:  string(definition_json),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = vvs.metaDB.Exec(`
        INSERT INTO virtual_views (id, user_id, name, description, definition, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, virtual_view.ID, virtual_view.UserID, virtual_view.Name, virtual_view.Description, virtual_view.Definition, virtual_view.CreatedAt, virtual_view.UpdatedAt)

	if err != nil {
		// Handle potential unique constraint violation (user_id, name)
		// This might require checking the error type or message if the DB driver provides it.
		// For DuckDB, a generic error might be returned. A more robust check would be needed.
		// Example: if strings.Contains(err.Error(), "UNIQUE constraint failed: virtual_views.user_id, virtual_views.name") { ... }
		return nil, fmt.Errorf("failed to save virtual view: %w. Ensure the name is unique for this user.", err)
	}

	return virtual_view, nil
}

// GetUserVirtualViews retrieves all virtual views for a user
func (vvs *VirtualViewService) GetUserVirtualViews(user_id string) ([]models.VirtualView, error) {
	query := `
        SELECT id, user_id, name, description, definition, created_at, updated_at, last_accessed_at
        FROM virtual_views 
        WHERE user_id = ? 
        ORDER BY created_at DESC
    `

	rows, err := vvs.metaDB.Query(query, user_id)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual views: %w", err)
	}
	defer rows.Close()

	var virtual_views []models.VirtualView
	for rows.Next() {
		var vv models.VirtualView
		var description sql.NullString
		var last_accessed_at sql.NullTime

		err = rows.Scan(&vv.ID, &vv.UserID, &vv.Name, &description, &vv.Definition, &vv.CreatedAt, &vv.UpdatedAt, &last_accessed_at)
		if err != nil {
			return nil, fmt.Errorf("failed to scan virtual view: %w", err)
		}

		if description.Valid {
			vv.Description = &description.String
		}
		if last_accessed_at.Valid {
			vv.LastAccessedAt = &last_accessed_at.Time
		}

		virtual_views = append(virtual_views, vv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating virtual view rows: %w", err)
	}

	return virtual_views, nil
}

// GetVirtualViewSchema retrieves schema information for a virtual view
func (vvs *VirtualViewService) GetVirtualViewSchema(virtual_view_id string, user_id string) ([]models.DataSourceSchema, error) {
	// First verify the virtual view belongs to the user
	var definition_json string
	err := vvs.metaDB.QueryRow("SELECT definition FROM virtual_views WHERE id = ? AND user_id = ?", virtual_view_id, user_id).Scan(&definition_json)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual view not found or access denied")
		}
		return nil, fmt.Errorf("failed to get virtual view definition: %w", err)
	}

	// Parse the definition to get selected column IDs
	var definition models.VirtualViewDefinition
	err = json.Unmarshal([]byte(definition_json), &definition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse virtual view definition: %w", err)
	}

	if len(definition.SelectedColumns) == 0 {
		return nil, fmt.Errorf("virtual view has no selected columns")
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(definition.SelectedColumns))
	args := make([]interface{}, len(definition.SelectedColumns))
	for i, col := range definition.SelectedColumns {
		placeholders[i] = "?"
		args[i] = col.DataSourceSchemaID
	}

	// Get schema information for the selected columns
	query := fmt.Sprintf(`
		SELECT id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, is_primary_key, retrieved_at
		FROM data_source_schemas 
		WHERE id IN (%s)
		ORDER BY table_name, column_name
	`, strings.Join(placeholders, ","))

	rows, err := vvs.metaDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual view schema: %w", err)
	}
	defer rows.Close()

	var schemas []models.DataSourceSchema
	for rows.Next() {
		var schema models.DataSourceSchema
		err = rows.Scan(&schema.ID, &schema.DataSourceID, &schema.SchemaName, &schema.TableName, &schema.ColumnName, &schema.ColumnType, &schema.IsNullable, &schema.IsPrimaryKey, &schema.RetrievedAt)
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

// GetVirtualViewSampleData retrieves sample data (5 rows) from a virtual view
func (vvs *VirtualViewService) GetVirtualViewSampleData(virtual_view_id string, user_id string) (map[string]interface{}, error) {
	// First verify the virtual view belongs to the user and get its definition
	var definition_json string
	err := vvs.metaDB.QueryRow("SELECT definition FROM virtual_views WHERE id = ? AND user_id = ?", virtual_view_id, user_id).Scan(&definition_json)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual view not found or access denied")
		}
		return nil, fmt.Errorf("failed to get virtual view definition: %w", err)
	}

	// Parse the definition to get selected column IDs
	var definition models.VirtualViewDefinition
	err = json.Unmarshal([]byte(definition_json), &definition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse virtual view definition: %w", err)
	}

	if len(definition.SelectedColumns) == 0 {
		return nil, fmt.Errorf("virtual view has no selected columns")
	}

	// Get schema information for the selected columns
	placeholders := make([]string, len(definition.SelectedColumns))
	args := make([]interface{}, len(definition.SelectedColumns))
	for i, col := range definition.SelectedColumns {
		placeholders[i] = "?"
		args[i] = col.DataSourceSchemaID
	}

	query := fmt.Sprintf(`
		SELECT dss.id, dss.table_name, dss.column_name, ds.id as datasource_id, ds.db_type, ds.host, ds.port, ds.database_name, ds.db_username, ds.password_encrypted
		FROM data_source_schemas dss 
		JOIN data_sources ds ON dss.data_source_id = ds.id
		WHERE dss.id IN (%s) AND ds.user_id = ?
		ORDER BY dss.table_name, dss.column_name
	`, strings.Join(placeholders, ","))

	args = append(args, user_id)

	rows, err := vvs.metaDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual view schema with datasource info: %w", err)
	}
	defer rows.Close()

	// Group columns by data source and table
	type ColumnInfo struct {
		ID         string
		TableName  string
		ColumnName string
	}

	type DataSourceInfo struct {
		ID                string
		DBType            string
		Host              sql.NullString
		Port              sql.NullInt64
		DatabaseName      sql.NullString
		DBUsername        sql.NullString
		PasswordEncrypted sql.NullString
		Tables            map[string][]ColumnInfo
	}

	dataSources := make(map[string]*DataSourceInfo)

	for rows.Next() {
		var col ColumnInfo
		var dsInfo DataSourceInfo
		err = rows.Scan(&col.ID, &col.TableName, &col.ColumnName, &dsInfo.ID, &dsInfo.DBType, &dsInfo.Host, &dsInfo.Port, &dsInfo.DatabaseName, &dsInfo.DBUsername, &dsInfo.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		if dataSources[dsInfo.ID] == nil {
			dsInfo.Tables = make(map[string][]ColumnInfo)
			dataSources[dsInfo.ID] = &dsInfo
		}

		ds := dataSources[dsInfo.ID]
		if ds.Tables[col.TableName] == nil {
			ds.Tables[col.TableName] = []ColumnInfo{}
		}
		ds.Tables[col.TableName] = append(ds.Tables[col.TableName], col)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	// Execute queries against each data source
	result := map[string]interface{}{
		"columns": []string{},
		"rows":    [][]interface{}{},
	}

	// For simplicity, we'll handle only single data source virtual views for now
	// In a real implementation, you might want to support JOINs across data sources
	if len(dataSources) != 1 {
		return nil, fmt.Errorf("virtual views spanning multiple data sources are not supported yet")
	}

	for _, dsInfo := range dataSources {
		// Connect to the external database
		var connectionString string
		var driverName string

		switch dsInfo.DBType {
		case "postgresql":
			driverName = "postgres" // Correct driver name for github.com/lib/pq
			connectionString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				dsInfo.Host.String, dsInfo.Port.Int64, dsInfo.DBUsername.String, dsInfo.PasswordEncrypted.String, dsInfo.DatabaseName.String)
		case "mysql":
			driverName = "mysql" // Correct driver name for github.com/go-sql-driver/mysql
			connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				dsInfo.DBUsername.String, dsInfo.PasswordEncrypted.String, dsInfo.Host.String, dsInfo.Port.Int64, dsInfo.DatabaseName.String)
		default:
			return nil, fmt.Errorf("unsupported database type: %s", dsInfo.DBType)
		}

		ext_db, err := sql.Open(driverName, connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external database: %w", err)
		}
		defer ext_db.Close()

		err = ext_db.Ping()
		if err != nil {
			return nil, fmt.Errorf("failed to ping external database: %w", err)
		}

		// Build SELECT query
		var columns []string
		var tableNames []string

		for tableName, cols := range dsInfo.Tables {
			tableNames = append(tableNames, tableName)
			for _, col := range cols {
				columns = append(columns, fmt.Sprintf("%s.%s", tableName, col.ColumnName))
				result["columns"] = append(result["columns"].([]string), fmt.Sprintf("%s.%s", tableName, col.ColumnName))
			}
		}

		// For now, support only single table queries
		// In a real implementation, you'd need to handle JOINs
		if len(tableNames) > 1 {
			return nil, fmt.Errorf("virtual views with multiple tables require JOIN logic which is not implemented yet")
		}

		selectQuery := fmt.Sprintf("SELECT %s FROM %s LIMIT 5", strings.Join(columns, ", "), tableNames[0])

		dataRows, err := ext_db.Query(selectQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to execute sample data query: %w", err)
		}
		defer dataRows.Close()

		// Get column types
		columnTypes, err := dataRows.ColumnTypes()
		if err != nil {
			return nil, fmt.Errorf("failed to get column types: %w", err)
		}

		for dataRows.Next() {
			// Create a slice of interface{} to hold the values
			values := make([]interface{}, len(columnTypes))
			valuePtrs := make([]interface{}, len(columnTypes))

			for i := range values {
				valuePtrs[i] = &values[i]
			}

			err = dataRows.Scan(valuePtrs...)
			if err != nil {
				return nil, fmt.Errorf("failed to scan data row: %w", err)
			}

			// Convert values to appropriate types for JSON serialization
			row := make([]interface{}, len(values))
			for i, val := range values {
				if val == nil {
					row[i] = nil
				} else {
					switch v := val.(type) {
					case []byte:
						row[i] = string(v)
					default:
						row[i] = v
					}
				}
			}

			result["rows"] = append(result["rows"].([][]interface{}), row)
		}

		if err = dataRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating data rows: %w", err)
		}
	}

	return result, nil
}
