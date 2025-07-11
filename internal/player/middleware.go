package player

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
)

// ContextKey represents the type used for context keys
type ContextKey string

// Context keys for player data
const (
	PlayerContextKey ContextKey = "player"
)

// AuthMiddleware creates a middleware that requires authentication
func AuthMiddleware(playerManager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeErrorResponse(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				writeErrorResponse(w, "Invalid authorization format. Use: Bearer <token>", http.StatusUnauthorized)
				return
			}

			sessionToken := tokenParts[1]
			if sessionToken == "" {
				writeErrorResponse(w, "Session token required", http.StatusUnauthorized)
				return
			}

			// Authenticate session
			player, err := playerManager.AuthenticateSession(r.Context(), sessionToken)
			if err != nil {
				log.Debug("Authentication failed", "error", err, "ip", r.RemoteAddr)
				writeErrorResponse(w, "Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Add player to request context
			ctx := context.WithValue(r.Context(), PlayerContextKey, player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware creates a middleware that optionally authenticates
// If authentication is provided and valid, the player is added to context
// If no authentication is provided, the request continues without player context
func OptionalAuthMiddleware(playerManager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No authentication provided, continue without player
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				// Invalid format, continue without player
				next.ServeHTTP(w, r)
				return
			}

			sessionToken := tokenParts[1]
			if sessionToken == "" {
				// Empty token, continue without player
				next.ServeHTTP(w, r)
				return
			}

			// Try to authenticate session
			player, err := playerManager.AuthenticateSession(r.Context(), sessionToken)
			if err != nil {
				// Authentication failed, continue without player
				log.Debug("Optional authentication failed", "error", err, "ip", r.RemoteAddr)
				next.ServeHTTP(w, r)
				return
			}

			// Add player to request context
			ctx := context.WithValue(r.Context(), PlayerContextKey, player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetPlayerFromContext retrieves the authenticated player from the request context
func GetPlayerFromContext(ctx context.Context) (*Player, bool) {
	player, ok := ctx.Value(PlayerContextKey).(*Player)
	return player, ok
}

// RequirePlayer is a helper function to get the player from context or return an error
func RequirePlayer(ctx context.Context) (*Player, error) {
	player, ok := GetPlayerFromContext(ctx)
	if !ok {
		return nil, &AuthError{Message: "Player not found in context"}
	}
	return player, nil
}

// AuthError represents an authentication error
type AuthError struct {
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// writeErrorResponse writes a JSON error response
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   message,
		Code:    statusCode,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

// CORS middleware to handle cross-origin requests
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get player info if available
			playerInfo := "anonymous"
			if player, ok := GetPlayerFromContext(r.Context()); ok {
				playerInfo = player.Username
			}

			log.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"player", playerInfo,
			)

			next.ServeHTTP(w, r)

			log.Debug("HTTP Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"player", playerInfo,
			)
		})
	}
}
