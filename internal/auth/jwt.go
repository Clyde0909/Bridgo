package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecretKey is used to sign and verify JWT tokens.
// In a production environment, this should be loaded from a secure configuration.
var jwtSecretKey = []byte("your-secret-key-please-change") // TODO: Change this and load from config

// Claims defines the structure of the JWT claims.
type Claims struct {
	Username string `json:"username"`
	UserID   string `json:"userID"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a new JWT for a given username and userID.
func GenerateJWT(username, userID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token expires in 24 hours
	claims := &Claims{
		Username: username,
		UserID:   userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "Bridgo",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ValidateJWT validates a JWT string and returns the claims if the token is valid.
func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
