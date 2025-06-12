package models

import "time"

// VirtualBaseView represents the structure of the 'virtual_base_views' table.
// Each Virtual BaseView maps to exactly one table in a data source.
type VirtualBaseView struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	DataSourceID    string     `json:"data_source_id"`
	TableName       string     `json:"table_name"`
	SelectedColumns string     `json:"selected_columns"` // JSON array of column names
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastAccessedAt  *time.Time `json:"last_accessed_at,omitempty"`
}

// VirtualBaseViewDefinition defines the structure for the JSON 'selected_columns' field
type VirtualBaseViewDefinition struct {
	ColumnNames []string `json:"column_names"`
}

// CreateVirtualBaseViewInput defines the input for creating a virtual base view
type CreateVirtualBaseViewInput struct {
	UserID          string   `json:"-"` // Passed internally
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	DataSourceID    string   `json:"data_source_id"`
	TableName       string   `json:"table_name"`
	SelectedColumns []string `json:"selected_columns"`
}
