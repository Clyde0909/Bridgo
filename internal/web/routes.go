package web

import (
	"fmt"
	"net/http"
	"path/filepath"
)

// RegisterRoutes sets up the HTTP routes using methods of HandlerDependencies.
func (h *HandlerDependencies) RegisterRoutes(mux *http.ServeMux) {
	// Page serving handlers
	mux.HandleFunc("/", h.homeHandler)
	mux.HandleFunc("/register", h.registerPageHandler)
	mux.HandleFunc("/login", h.loginPageHandler)
	mux.HandleFunc("/dashboard", h.dashboardPageHandler)

	// API handlers
	mux.HandleFunc("/api/register", h.registerAPIHandler)
	mux.HandleFunc("/api/login", h.loginAPIHandler)
	mux.HandleFunc("/api/db/test-connection", h.dbTestConnectionAPIHandler)
	mux.HandleFunc("/api/db/save-datasource", h.dbSaveDataSourceAPIHandler)
	mux.HandleFunc("/api/datasources", h.getUserDataSourcesAPIHandler)
	mux.HandleFunc("/api/datasources/schema", h.getDataSourceSchemaAPIHandler)
	mux.HandleFunc("/api/virtual-views", h.getUserVirtualViewsAPIHandler)
	mux.HandleFunc("/api/db/connect-and-fetch-schema", h.dbConnectAndFetchSchemaAPIHandler)
	mux.HandleFunc("/api/virtual-views/create", h.createVirtualViewAPIHandler)

	// Static files (CSS, JS, images etc.) from web/ui directory served under /static/ path
	// e.g., /static/css/style.css will serve web/ui/css/style.css
	staticDir := http.Dir(filepath.Join(".", "web", "ui"))
	fileServer := http.FileServer(staticDir)
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Dashboard iframe pages
	mux.HandleFunc("/dashboard_home", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "dashboard_home.html"))
	})
	mux.HandleFunc("/db_connections", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "db_connections.html"))
	})
	mux.HandleFunc("/virtual_views", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "virtual_views.html"))
	})
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(".", "web", "ui", "settings.html"))
	})
	mux.HandleFunc("/test_button", h.testButtonPageHandler)

	fmt.Println("Registered web routes")
}
