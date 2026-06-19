package http

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "gigpurse-fallback-secret-key-12345"
	}
	return []byte(secret)
}

func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "authorization_required", "authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "invalid_authorization_format", "invalid authorization format")
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			respondError(w, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondError(w, http.StatusUnauthorized, "invalid_token_claims", "invalid token claims")
			return
		}

		userID, ok1 := claims["user_id"].(string)
		userRole, ok2 := claims["role"].(string)
		if !ok1 || !ok2 {
			respondError(w, http.StatusUnauthorized, "invalid_token_claims", "invalid token claims fields")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserRoleKey, userRole)
		next(w, r.WithContext(ctx))
	}
}

// GetUserFromContext retrieves UserID and Role from request context
func GetUserFromContext(ctx context.Context) (string, string, bool) {
	userID, ok1 := ctx.Value(UserIDKey).(string)
	userRole, ok2 := ctx.Value(UserRoleKey).(string)
	return userID, userRole, ok1 && ok2
}
