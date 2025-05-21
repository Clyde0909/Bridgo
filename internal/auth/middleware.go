package auth

import (
	"context"
	"net/http"
	"strings"
)

// MiddlewareKey is a custom type for context keys to avoid collisions.
type MiddlewareKey string

// UserContextKey is the key used to store user claims in the request context.
const UserContextKey MiddlewareKey = "userClaims"

// JWTMiddleware validates the JWT token from the Authorization header,
// skipping authentication for specified public paths.
func JWTMiddleware(next http.Handler, publicPaths []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the current request path is one of the public paths
		var isPublicPath bool
		for _, publicPath := range publicPaths {
			// Treat paths ending with "/" (and longer than just "/") as prefixes.
			// Treat "/" and other paths not ending with "/" as exact matches.
			if strings.HasSuffix(publicPath, "/") && len(publicPath) > 1 {
				if strings.HasPrefix(r.URL.Path, publicPath) {
					isPublicPath = true
					break
				}
			} else {
				if r.URL.Path == publicPath {
					isPublicPath = true
					break
				}
			}
		}

		if isPublicPath {
			next.ServeHTTP(w, r) // Serve without auth check
			return
		}

		// If not a public path, proceed with token validation
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Token is valid, add claims to context for downstream handlers
		ctx := context.WithValue(r.Context(), UserContextKey, claims) // Storing the whole claims struct
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserClaimsFromContext retrieves user claims from the request context.
// This can be used by handlers protected by the JWTMiddleware.
func GetUserClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	return claims, ok
}
