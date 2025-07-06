package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func SetupMiddleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		// Request ID for tracing
		middleware.RequestID,

		// Logging middleware
		middleware.Logger,

		// Recovery middleware
		middleware.Recoverer,

		// CORS middleware for public API
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}),

		// Content type middleware
		middleware.SetHeader("Content-Type", "application/json"),

		// Timeout middleware
		middleware.Timeout(30 * time.Second),

		// Rate limiting would go here in production
		// middleware.ThrottleBacklog(100, 1000, 30*time.Second),
	}
}

func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	return middleware.ThrottleBacklog(requestsPerMinute, requestsPerMinute*2, time.Minute)
}
