package web

import (
	"encoding/json"
	"net/http"

	"Bridgo/internal/auth"
	"Bridgo/internal/core"
)

// getUserVirtualViewsAPIHandler retrieves all virtual views for a user
func (h *HandlerDependencies) getUserVirtualViewsAPIHandler(w http.ResponseWriter, r *http.Request) {
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

	virtualViews, err := h.CoreService.GetUserVirtualViews(claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual views: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"virtualviews": virtualViews,
	})
}

// createVirtualViewAPIHandler handles the creation of a new virtual view.
// This handler will be protected by the JWTMiddleware.
func (h *HandlerDependencies) createVirtualViewAPIHandler(w http.ResponseWriter, r *http.Request) {
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

	var input core.CreateVirtualViewInput
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
			"message": "Virtual view name is required",
		})
		return
	}
	if len(input.SelectedSchemaIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "At least one column must be selected for the virtual view",
		})
		return
	}

	virtualView, err := h.CoreService.CreateVirtualView(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create virtual view: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Virtual view created successfully",
		"virtualview": virtualView,
	})
}

// getVirtualViewSchemaAPIHandler retrieves schema for a specific virtual view
func (h *HandlerDependencies) getVirtualViewSchemaAPIHandler(w http.ResponseWriter, r *http.Request) {
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

	virtualViewID := r.URL.Query().Get("virtual_view_id")
	if virtualViewID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "virtual_view_id parameter is required",
		})
		return
	}

	schema, err := h.CoreService.GetVirtualViewSchema(virtualViewID, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual view schema: " + err.Error(),
		})
		return
	}

	// Convert sql.NullBool to proper JSON format for frontend (same as datasource handler)
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

// getVirtualViewSampleDataAPIHandler retrieves sample data for a specific virtual view
func (h *HandlerDependencies) getVirtualViewSampleDataAPIHandler(w http.ResponseWriter, r *http.Request) {
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

	virtualViewID := r.URL.Query().Get("virtual_view_id")
	if virtualViewID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "virtual_view_id parameter is required",
		})
		return
	}

	sampleData, err := h.CoreService.GetVirtualViewSampleData(virtualViewID, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve virtual view sample data: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    sampleData,
	})
}
