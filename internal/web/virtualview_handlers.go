package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"Bridgo/internal/auth"
	"Bridgo/internal/core"
)

// getUserVirtualViewsAPIHandler retrieves all virtual views for a user
func (h *HandlerDependencies) getUserVirtualViewsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	virtualViews, err := h.CoreService.GetUserVirtualViews(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to retrieve virtual views: "+err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.GetUserClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized: Missing user claims.", http.StatusUnauthorized)
		return
	}

	var input core.CreateVirtualViewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	input.UserID = claims.UserID // Set UserID from JWT claims

	if input.Name == "" {
		http.Error(w, "Virtual view name is required", http.StatusBadRequest)
		return
	}
	if len(input.SelectedSchemaIDs) == 0 {
		http.Error(w, "At least one column must be selected for the virtual view", http.StatusBadRequest)
		return
	}

	virtualView, err := h.CoreService.CreateVirtualView(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create virtual view: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(virtualView) // Return the created virtual view object
}
