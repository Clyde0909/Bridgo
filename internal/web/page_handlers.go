package web

import (
	"net/http"
	"path/filepath"
)

func (h *HandlerDependencies) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "web/ui/index.html")
}

func (h *HandlerDependencies) registerPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/ui/register.html")
}

func (h *HandlerDependencies) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/ui/login.html")
}

func (h *HandlerDependencies) dashboardPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(".", "web", "ui", "dashboard.html"))
}

func (h *HandlerDependencies) testButtonPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(".", "web", "ui", "test_button.html"))
}
