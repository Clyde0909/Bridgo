package web

import (
	"encoding/json"
	"net/http"

	"Bridgo/internal/auth"
	"Bridgo/internal/models"
)

// getUserVirtualBaseViewsAPIHandler retrieves all virtual base views for a user
func (h *HandlerDependencies) getUserVirtualBaseViewsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Only GET method is allowed",
		})
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Unauthorized: Missing user claims",
		})
		return
	}

	virtualBaseViews, err := h.CoreService.GetUserVirtualBaseViews(claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual base views: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":            true,
		"virtual_base_views": virtualBaseViews,
	})
}

// createVirtualBaseViewAPIHandler handles the creation of a new virtual base view
func (h *HandlerDependencies) createVirtualBaseViewAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Only POST method is allowed",
		})
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Unauthorized: Missing user claims",
		})
		return
	}

	var input models.CreateVirtualBaseViewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}
	input.UserID = claims.UserID // Set UserID from JWT claims

	if input.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Virtual base view name is required",
		})
		return
	}
	if input.DataSourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Data source ID is required",
		})
		return
	}
	if input.TableName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Table name is required",
		})
		return
	}
	if len(input.SelectedColumns) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "At least one column must be selected for the virtual base view",
		})
		return
	}

	virtualBaseView, err := h.CoreService.CreateVirtualBaseView(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create virtual base view: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":           true,
		"message":           "Virtual base view created successfully",
		"virtual_base_view": virtualBaseView,
	})
}

// getVirtualBaseViewSchemaAPIHandler retrieves schema for a specific virtual base view
func (h *HandlerDependencies) getVirtualBaseViewSchemaAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Only GET method is allowed",
		})
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Unauthorized: Missing user claims",
		})
		return
	}

	virtualBaseViewID := r.URL.Query().Get("virtual_base_view_id")
	if virtualBaseViewID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "virtual_base_view_id parameter is required",
		})
		return
	}

	schema, err := h.CoreService.GetVirtualBaseViewSchema(virtualBaseViewID, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual base view schema: " + err.Error(),
		})
		return
	}

	// Convert sql.NullBool to proper JSON format for frontend
	var schemaResponses []SchemaResponse
	for _, s := range schema {
		schemaResponse := SchemaResponse{
			ID:           s.ID,
			DataSourceID: s.DataSourceID,
			SchemaName:   s.SchemaName.String,
			TableName:    s.TableName,
			ColumnName:   s.ColumnName,
			ColumnType:   s.ColumnType,
			IsNullable:   s.IsNullable.Valid && s.IsNullable.Bool,
			IsPrimaryKey: s.IsPrimaryKey.Valid && s.IsPrimaryKey.Bool,
			RetrievedAt:  s.RetrievedAt.Format("2006-01-02T15:04:05Z"),
		}
		schemaResponses = append(schemaResponses, schemaResponse)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"schema":  schemaResponses,
	})
}

// getVirtualBaseViewSampleDataAPIHandler retrieves sample data for a specific virtual base view
func (h *HandlerDependencies) getVirtualBaseViewSampleDataAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Only GET method is allowed",
		})
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Unauthorized: Missing user claims",
		})
		return
	}

	virtualBaseViewID := r.URL.Query().Get("virtual_base_view_id")
	if virtualBaseViewID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "virtual_base_view_id parameter is required",
		})
		return
	}

	sampleData, err := h.CoreService.GetVirtualBaseViewSampleData(virtualBaseViewID, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual base view sample data: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    sampleData,
	})
}
