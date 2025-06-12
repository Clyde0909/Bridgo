package web

import (
	"encoding/json"
	"log"
	"net/http"

	"Bridgo/internal/auth"
)

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

	// Debug: Log the user information during login
	log.Printf("User login successful - Username: %s, UserID: %s", user.Username, user.ID)

	// Generate JWT token
	tokenString, err := auth.GenerateJWT(user.Username, user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	log.Printf("JWT token generated for user: %s", user.Username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"token":   tokenString,
		"userID":  user.ID,
	})
}
