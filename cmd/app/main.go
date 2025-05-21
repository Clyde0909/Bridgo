package main

import (
	// "database/sql"
	"fmt"
	"log"
	"net"
	"net/http"

	"Bridgo/internal/auth" // Added for middleware
	"Bridgo/internal/metadata"
	"Bridgo/internal/server"
	"Bridgo/internal/web"
)

func main() {
	// Initialize DuckDB
	db, err := metadata.InitDB(".")
	if err != nil {
		log.Fatalf("Failed to initialize metadata database: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("Error closing database: %v", cerr)
		}
	}()

	// Initialize the central application which holds all services
	app := server.NewApp(db) // Pass db to NewApp

	// Create a new ServeMux (router)
	mux := http.NewServeMux()

	// Initialize web handlers/routes with necessary service dependencies
	handlerDeps := web.NewHandlers(app.UserService, app.CoreService)
	handlerDeps.RegisterRoutes(mux) // Register routes onto the new mux

	// Define public paths that do not require authentication
	publicPaths := []string{
		"/",
		"/login",
		"/register",
		"/api/login",
		"/api/register",
		"/static/",        // Static assets
		"/dashboard",      // Main dashboard page
		"/dashboard_home", // Dashboard iframe content
		"/db_connections", // Dashboard iframe content
		"/settings",       // Dashboard iframe content
		"/favicon.ico",    // Browser favicon request
	}

	// Wrap the mux with the JWT middleware
	protectedMux := auth.JWTMiddleware(mux, publicPaths)

	addr := "0.0.0.0:18080"
	fmt.Printf("Attempting to listen on tcp4 %s\n", addr)

	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatalf("Failed to listen on tcp4 %s: %v", addr, err)
	}

	fmt.Printf("Successfully listening on tcp4 %s. Access it at http://%s\n", listener.Addr().String(), addr)
	fmt.Println("Starting server...")

	// Use the protected mux for the HTTP server
	if err := http.Serve(listener, protectedMux); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		} else {
			log.Println("HTTP server shut down.")
		}
	}
}
