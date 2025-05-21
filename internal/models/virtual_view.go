package models

import "time"

// VirtualView represents the structure of the 'virtual_views' table.
type VirtualView struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	// Definition is a JSON string that details the structure of the virtual view,
	// referencing columns from 'data_source_schemas'.
	Definition     string     `json:"definition"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
}

// VirtualViewDefinition defines the structure for the JSON 'definition' field
// in the VirtualView model. It lists the selected columns that make up this view.
type VirtualViewDefinition struct {
	SelectedColumns []SelectedColumn `json:"selected_columns"`
	// Future enhancements:
	// Filters         []FilterCondition `json:"filters,omitempty"`
	// Joins           []JoinCondition   `json:"joins,omitempty"`
	// GroupByColumns  []string          `json:"group_by_columns,omitempty"`
	// OrderBy         []OrderByClause   `json:"order_by,omitempty"`
}

// SelectedColumn represents a single column chosen for the virtual view.
// It references a specific column in a specific data source's schema.
type SelectedColumn struct {
	DataSourceSchemaID string  `json:"data_source_schema_id"` // Foreign key to data_source_schemas.id
	Alias              *string `json:"alias,omitempty"`       // Optional alias for the column in the virtual view
	// Future enhancements:
	// TransformationFunction *string `json:"transformation_function,omitempty"` // e.g., UPPER, CONCAT, etc.
}

/*
// Example for future enhancements:
type FilterCondition struct {
	DataSourceSchemaID string `json:"data_source_schema_id"`
	Operator           string `json:"operator"` // e.g., "=", ">", "LIKE"
	Value              string `json:"value"`
}

type JoinCondition struct {
	LeftDataSourceSchemaID  string `json:"left_data_source_schema_id"`
	RightDataSourceSchemaID string `json:"right_data_source_schema_id"`
	JoinType                string `json:"join_type"` // e.g., "INNER", "LEFT"
	Operator                string `json:"operator"`  // e.g., "="
}

type OrderByClause struct {
	DataSourceSchemaID string `json:"data_source_schema_id"`
	Direction          string `json:"direction"` // "ASC" or "DESC"
}
*/
