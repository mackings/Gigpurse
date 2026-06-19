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
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok1 := claims["user_id"].(string)
		userRole, ok2 := claims["role"].(string)
		if !ok1 || !ok2 {
			http.Error(w, "invalid token claims fields", http.StatusUnauthorized)
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
