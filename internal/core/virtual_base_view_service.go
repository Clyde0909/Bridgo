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

// VirtualBaseViewService handles virtual base view operations
type VirtualBaseViewService struct {
	metaDB *sql.DB
}

// NewVirtualBaseViewService creates a new VirtualBaseViewService
func NewVirtualBaseViewService(metaDB *sql.DB) *VirtualBaseViewService {
	return &VirtualBaseViewService{metaDB: metaDB}
}

// CreateVirtualBaseView creates a new virtual base view for a single table
func (vbvs *VirtualBaseViewService) CreateVirtualBaseView(input models.CreateVirtualBaseViewInput) (*models.VirtualBaseView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("virtual base view name cannot be empty")
	}
	if input.DataSourceID == "" {
		return nil, fmt.Errorf("data source ID cannot be empty")
	}
	if input.TableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}
	if len(input.SelectedColumns) == 0 {
		return nil, fmt.Errorf("at least one column must be selected")
	}

	// Validate that all selected columns belong to the specified table and data source
	placeholders := make([]string, len(input.SelectedColumns))
	args := make([]interface{}, len(input.SelectedColumns)+3)
	for i, columnName := range input.SelectedColumns {
		placeholders[i] = "?"
		args[i] = columnName
	}
	args[len(input.SelectedColumns)] = input.DataSourceID
	args[len(input.SelectedColumns)+1] = input.TableName
	args[len(input.SelectedColumns)+2] = input.UserID

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM data_source_schemas dss 
		JOIN data_sources ds ON dss.data_source_id = ds.id 
		WHERE dss.column_name IN (%s) AND ds.id = ? AND dss.table_name = ? AND ds.user_id = ?
	`, strings.Join(placeholders, ","))

	var count int
	err := vbvs.metaDB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to validate column access: %w", err)
	}
	if count != len(input.SelectedColumns) {
		return nil, fmt.Errorf("some selected columns do not belong to the specified table or data source")
	}

	// Create definition
	definition := models.VirtualBaseViewDefinition{
		ColumnNames: input.SelectedColumns,
	}
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal virtual base view definition: %w", err)
	}

	now := time.Now().UTC()
	virtualBaseView := &models.VirtualBaseView{
		ID:              uuid.NewString(),
		UserID:          input.UserID,
		Name:            input.Name,
		Description:     input.Description,
		DataSourceID:    input.DataSourceID,
		TableName:       input.TableName,
		SelectedColumns: string(definitionJSON),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = vbvs.metaDB.Exec(`
        INSERT INTO virtual_base_views (id, user_id, name, description, data_source_id, table_name, selected_columns, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, virtualBaseView.ID, virtualBaseView.UserID, virtualBaseView.Name, virtualBaseView.Description,
		virtualBaseView.DataSourceID, virtualBaseView.TableName, virtualBaseView.SelectedColumns,
		virtualBaseView.CreatedAt, virtualBaseView.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to save virtual base view: %w. Ensure the name is unique for this user.", err)
	}

	return virtualBaseView, nil
}

// GetUserVirtualBaseViews retrieves all virtual base views for a user
func (vbvs *VirtualBaseViewService) GetUserVirtualBaseViews(userID string) ([]models.VirtualBaseView, error) {
	query := `
        SELECT id, user_id, name, description, data_source_id, table_name, selected_columns, created_at, updated_at, last_accessed_at
        FROM virtual_base_views 
        WHERE user_id = ? 
        ORDER BY created_at DESC
    `

	rows, err := vbvs.metaDB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual base views: %w", err)
	}
	defer rows.Close()

	var virtualBaseViews []models.VirtualBaseView
	for rows.Next() {
		var vbv models.VirtualBaseView
		var description sql.NullString
		var lastAccessedAt sql.NullTime

		err = rows.Scan(&vbv.ID, &vbv.UserID, &vbv.Name, &description, &vbv.DataSourceID,
			&vbv.TableName, &vbv.SelectedColumns, &vbv.CreatedAt, &vbv.UpdatedAt, &lastAccessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan virtual base view: %w", err)
		}

		if description.Valid {
			vbv.Description = &description.String
		}
		if lastAccessedAt.Valid {
			vbv.LastAccessedAt = &lastAccessedAt.Time
		}

		virtualBaseViews = append(virtualBaseViews, vbv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating virtual base view rows: %w", err)
	}

	return virtualBaseViews, nil
}

// GetVirtualBaseViewSchema retrieves schema information for a virtual base view
func (vbvs *VirtualBaseViewService) GetVirtualBaseViewSchema(virtualBaseViewID string, userID string) ([]models.DataSourceSchema, error) {
	// First verify the virtual base view belongs to the user and get its definition
	var selectedColumnsJSON, dataSourceID, tableName string
	err := vbvs.metaDB.QueryRow("SELECT selected_columns, data_source_id, table_name FROM virtual_base_views WHERE id = ? AND user_id = ?", virtualBaseViewID, userID).Scan(&selectedColumnsJSON, &dataSourceID, &tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual base view not found or access denied")
		}
		return nil, fmt.Errorf("failed to get virtual base view definition: %w", err)
	}

	// Parse the definition to get selected column names
	var definition models.VirtualBaseViewDefinition
	err = json.Unmarshal([]byte(selectedColumnsJSON), &definition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse virtual base view definition: %w", err)
	}

	if len(definition.ColumnNames) == 0 {
		return nil, fmt.Errorf("virtual base view has no selected columns")
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(definition.ColumnNames))
	args := make([]interface{}, len(definition.ColumnNames)+2)
	for i, colName := range definition.ColumnNames {
		placeholders[i] = "?"
		args[i] = colName
	}
	args[len(definition.ColumnNames)] = dataSourceID
	args[len(definition.ColumnNames)+1] = tableName

	// Get schema information for the selected columns
	query := fmt.Sprintf(`
		SELECT id, data_source_id, schema_name, table_name, column_name, column_type, is_nullable, is_primary_key, retrieved_at
		FROM data_source_schemas 
		WHERE column_name IN (%s) AND data_source_id = ? AND table_name = ?
		ORDER BY column_name
	`, strings.Join(placeholders, ","))

	rows, err := vbvs.metaDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual base view schema: %w", err)
	}
	defer rows.Close()

	var schemas []models.DataSourceSchema
	for rows.Next() {
		var schema models.DataSourceSchema
		err = rows.Scan(&schema.ID, &schema.DataSourceID, &schema.SchemaName, &schema.TableName,
			&schema.ColumnName, &schema.ColumnType, &schema.IsNullable, &schema.IsPrimaryKey, &schema.RetrievedAt)
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

// GetVirtualBaseViewSampleData retrieves sample data (5 rows) from a virtual base view
func (vbvs *VirtualBaseViewService) GetVirtualBaseViewSampleData(virtualBaseViewID string, userID string) (map[string]interface{}, error) {
	// Get virtual base view details
	var dataSourceID, tableName, selectedColumnsJSON string
	err := vbvs.metaDB.QueryRow(`
		SELECT data_source_id, table_name, selected_columns 
		FROM virtual_base_views 
		WHERE id = ? AND user_id = ?
	`, virtualBaseViewID, userID).Scan(&dataSourceID, &tableName, &selectedColumnsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual base view not found or access denied")
		}
		return nil, fmt.Errorf("failed to get virtual base view: %w", err)
	}

	// Parse selected columns
	var definition models.VirtualBaseViewDefinition
	err = json.Unmarshal([]byte(selectedColumnsJSON), &definition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse virtual base view definition: %w", err)
	}

	// Get data source connection info
	var dbType string
	var host sql.NullString
	var port sql.NullInt64
	var databaseName sql.NullString
	var dbUsername sql.NullString
	var passwordEncrypted sql.NullString

	err = vbvs.metaDB.QueryRow(`
		SELECT db_type, host, port, database_name, db_username, password_encrypted
		FROM data_sources 
		WHERE id = ? AND user_id = ?
	`, dataSourceID, userID).Scan(&dbType, &host, &port, &databaseName, &dbUsername, &passwordEncrypted)

	if err != nil {
		return nil, fmt.Errorf("failed to get data source info: %w", err)
	}

	// Use the selected column names directly
	columnNames := definition.ColumnNames

	// Connect to external database
	var connectionString string
	var driverName string

	switch dbType {
	case "postgresql":
		driverName = "postgres"
		connectionString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host.String, port.Int64, dbUsername.String, passwordEncrypted.String, databaseName.String)
	case "mysql":
		driverName = "mysql"
		connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			dbUsername.String, passwordEncrypted.String, host.String, port.Int64, databaseName.String)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	extDB, err := sql.Open(driverName, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to external database: %w", err)
	}
	defer extDB.Close()

	err = extDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping external database: %w", err)
	}

	// Build SELECT query for the single table
	selectQuery := fmt.Sprintf("SELECT %s FROM %s LIMIT 5", strings.Join(columnNames, ", "), tableName)

	dataRows, err := extDB.Query(selectQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sample data query: %w", err)
	}
	defer dataRows.Close()

	// Get column types
	columnTypes, err := dataRows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	result := map[string]interface{}{
		"columns": columnNames,
		"rows":    []map[string]interface{}{},
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

		// Convert values to a map for JSON serialization
		row := make(map[string]interface{})
		for i, val := range values {
			if val == nil {
				row[columnNames[i]] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					row[columnNames[i]] = string(v)
				default:
					row[columnNames[i]] = v
				}
			}
		}

		result["rows"] = append(result["rows"].([]map[string]interface{}), row)
	}

	if err = dataRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating data rows: %w", err)
	}

	return result, nil
}
