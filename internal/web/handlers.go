package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"Bridgo/internal/auth"
	"Bridgo/internal/core"
	"Bridgo/internal/users"
)

// HandlerDependencies holds the services that handlers will need.
type HandlerDependencies struct {
	UserService *users.Service
	CoreService *core.Service // Will be used for data-related APIs later
}

// NewHandlers creates a new HandlerDependencies struct.
func NewHandlers(us *users.Service, cs *core.Service) *HandlerDependencies {
	return &HandlerDependencies{
		UserService: us,
		CoreService: cs,
	}
}

// RegisterRoutes sets up the HTTP routes using methods of HandlerDependencies.
// The function name is changed from RegisterHandlers to RegisterRoutes for clarity.
func (h *HandlerDependencies) RegisterRoutes(mux *http.ServeMux) {
	// Page serving handlers
	mux.HandleFunc("/", h.homeHandler)
	mux.HandleFunc("/register", h.registerPageHandler)
	mux.HandleFunc("/login", h.loginPageHandler)
	mux.HandleFunc("/dashboard", h.dashboardPageHandler) // 대시보드 메인 페이지

	// API handlers
	mux.HandleFunc("/api/register", h.registerAPIHandler)
	mux.HandleFunc("/api/login", h.loginAPIHandler)
	mux.HandleFunc("/api/db/connect-and-fetch-schema", h.dbConnectAndFetchSchemaAPIHandler)
	mux.HandleFunc("/api/virtual-views/create", h.createVirtualViewAPIHandler) // New API endpoint for virtual views

	// Static files (CSS, JS, images etc.) from web/ui directory served under /static/ path
	// e.g., /static/css/style.css will serve web/ui/css/style.css
	staticDir := http.Dir(filepath.Join(".", "web", "ui"))
	fileServer := http.FileServer(staticDir)
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Handlers for pages loaded within the dashboard iframe
	// These ensure that direct navigation or refresh within iframe content works
	// and also allows for cleaner URLs.
	mux.HandleFunc("/dashboard_home", func(w http.ResponseWriter, r *http.Request) {
		// Here, you might want to add authentication check before serving the file
		// For now, just serving. This will be protected by middleware later.
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "dashboard_home.html"))
	})
	mux.HandleFunc("/db_connections", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "db_connections.html"))
	})
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "settings.html"))
	})

	fmt.Println("Registered web routes")
}

func (h *HandlerDependencies) homeHandler(w http.ResponseWriter, r *http.Request) {
	// For now, just a simple message. Later, this will serve an HTML file.
	// Check if user is logged in (future enhancement)
	// For now, link to login/register
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Serve the main index.html which will have links to login/register
	http.ServeFile(w, r, "web/ui/index.html")
}

func (h *HandlerDependencies) registerPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/ui/register.html")
}

func (h *HandlerDependencies) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/ui/login.html")
}

func (h *HandlerDependencies) dashboardPageHandler(w http.ResponseWriter, r *http.Request) {
	// This page should require authentication.
	// For now, we serve it directly. Authentication will be handled by middleware.
	http.ServeFile(w, r, filepath.Join(".", "web", "ui", "dashboard.html"))
}

// registerAPIHandler handles new user registration.
func (h *HandlerDependencies) registerAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username string `json:"username"`
		Email    string `json:"email"` // Optional, but good to have
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Use the injected UserService
	user, err := h.UserService.AddUser(creds.Username, creds.Email, creds.Password)
	if err != nil {
		// Check if the error is due to username already existing
		if err.Error() == "username already exists" { // This check could be more robust
			http.Error(w, "Username already taken", http.StatusConflict)
		} else {
			http.Error(w, "Failed to register user: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully", "userID": user.ID})
}

// loginAPIHandler handles user login.
func (h *HandlerDependencies) loginAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Use the injected UserService
	user, err := h.UserService.ValidatePassword(creds.Username, creds.Password)
	if err != nil {
		// Differentiate between "user not found" and "invalid password"
		// For security, often a generic message is better for login failures.
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	tokenString, err := auth.GenerateJWT(user.Username, user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"token":   tokenString,
		"userID":  user.ID,
	})
}

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
	fmt.Printf("User %s (ID: %s) attempting to connect to DB and fetch schema.\n", claims.Username, claims.UserID)

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

/*
func apiDataHandler(w http.ResponseWriter, r *http.Request) {
	// This handler would interact with the internal/server logic
	// to fetch or process data and return it, likely as JSON.
	// Example:
	// data, err := server.GlobalServer.GetData(r.URL.Query().Get("param"))
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// json.NewEncoder(w).Encode(data)
}
*/
