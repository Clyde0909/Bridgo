package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"Bridgo/internal/auth"
	"Bridgo/internal/core"
	"Bridgo/internal/models"
)

// dbConnectAndFetchSchemaAPIHandler handles connecting to a database and fetching its schema.
// This handler will be protected by the JWTMiddleware.
func (h *HandlerDependencies) dbConnectAndFetchSchemaAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	var input core.ConnectAndFetchSchemaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	input.UserID = claims.UserID // Set UserID from JWT claims

	if input.DBType == "" || input.SourceName == "" || input.Host == "" || input.Port == 0 || input.User == "" || input.DBName == "" {
		http.Error(w, "Missing required connection details (dbType, sourceName, host, port, user, dbName)", http.StatusBadRequest)
		return
	}

	savedSchema, err := h.CoreService.ConnectAndFetchSchema(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect, fetch, or save schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Data source and schema saved successfully.",
		"schema":  savedSchema, // Return the saved schema items, which now include their IDs
	})
}

// dbTestConnectionAPIHandler tests database connection without saving
func (h *HandlerDependencies) dbTestConnectionAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	var input core.ConnectAndFetchSchemaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	input.UserID = claims.UserID

	// Test connection and fetch schema without saving
	schema, err := h.CoreService.TestConnectionAndFetchSchema(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Connection test failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Connection test successful",
		"schema":  schema,
	})
}

// dbSaveDataSourceAPIHandler saves datasource after successful connection test
func (h *HandlerDependencies) dbSaveDataSourceAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	var request struct {
		ConnectionInput core.ConnectAndFetchSchemaInput `json:"connection"`
		Schema          []models.DataSourceSchema       `json:"schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	request.ConnectionInput.UserID = claims.UserID

	// Save the datasource
	savedDataSource, err := h.CoreService.SaveDataSource(request.ConnectionInput, request.Schema)
	if err != nil {
		http.Error(w, "Failed to save data source: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Data source saved successfully",
		"datasource": savedDataSource,
	})
}

// getUserDataSourcesAPIHandler retrieves all data sources for a user
func (h *HandlerDependencies) getUserDataSourcesAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	dataSources, err := h.CoreService.GetUserDataSources(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to retrieve data sources: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"datasources": dataSources,
	})
}

// getDataSourceSchemaAPIHandler retrieves schema for a specific data source
func (h *HandlerDependencies) getDataSourceSchemaAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	dataSourceID := r.URL.Query().Get("datasource_id")
	if dataSourceID == "" {
		http.Error(w, "datasource_id parameter is required", http.StatusBadRequest)
		return
	}

	schema, err := h.CoreService.GetDataSourceSchema(dataSourceID, claims.UserID)
	if err != nil {
		http.Error(w, "Failed to retrieve schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"schema":  schema,
	})
}
