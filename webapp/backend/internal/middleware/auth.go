package middleware

import (
	"context"
	"log"
	"net/http"
	"os"

	"backend/internal/repository"
)

type contextKey string

const userContextKey contextKey = "user"

func UserAuthMiddleware(sessionRepo *repository.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health check endpoint
			if r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("session_id")
			if err != nil {
				// Reduce logging for performance - only log in debug mode
				if os.Getenv("DEBUG") == "true" {
					log.Printf("Error retrieving session cookie: %v", err)
				}
				http.Error(w, "Unauthorized: No session cookie", http.StatusUnauthorized)
				return
			}
			sessionID := cookie.Value

			userID, err := sessionRepo.FindUserBySessionID(r.Context(), sessionID)
			if err != nil {
				// Reduce logging for performance - only log in debug mode
				if os.Getenv("DEBUG") == "true" {
					log.Printf("Error finding user by session ID: %v", err)
				}
				http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RobotAuthMiddleware(validAPIKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-KEY")

			if apiKey == "" || apiKey != validAPIKey {
				http.Error(w, "Forbidden: Invalid or missing API key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// コンテキストからユーザー情報を取得
// ユーザ情報はUserAuthMiddleware
func GetUserFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userContextKey).(int)
	return userID, ok
}
