package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"Bridgo/internal/models"

	"github.com/google/uuid"
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

	// TODO: Validate that all SelectedSchemaIDs belong to data sources accessible by UserID
	// This requires joining data_source_schemas with data_sources and checking user_id.
	// For now, we assume they are valid and accessible.

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
