package web

import (
	"Bridgo/internal/core"
	"Bridgo/internal/users"
)

// HandlerDependencies holds the services that handlers will need.
type HandlerDependencies struct {
	UserService *users.Service
	CoreService *core.CoreService // Will be used for data-related APIs later
}

// NewHandlers creates a new HandlerDependencies struct.
func NewHandlers(us *users.Service, cs *core.CoreService) *HandlerDependencies {
	return &HandlerDependencies{
		UserService: us,
		CoreService: cs,
	}
}
