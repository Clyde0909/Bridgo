// Package web contains the HTTP handlers and routing for the Bridgo application.
// This file serves as the main entry point for all handlers.
// The actual handler implementations are split across multiple files:
// - dependencies.go: HandlerDependencies struct and constructor
// - routes.go: Route registration
// - page_handlers.go: Page serving handlers
// - auth_handlers.go: Authentication API handlers
// - datasource_handlers.go: Data source API handlers
// - virtualview_handlers.go: Virtual view API handlers
package web
