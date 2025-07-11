package api

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func SetupMiddleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		// Request ID for tracing
		middleware.RequestID,

		// Debug logging middleware
		DebugLoggingMiddleware,

		// Default logging middleware
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

// DebugLoggingMiddleware provides detailed debug logging for requests
func DebugLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := middleware.GetReqID(r.Context())

		log.Debug("Request started",
			"method", r.Method,
			"url", r.URL.String(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"request_id", reqID,
			"content_length", r.ContentLength,
		)

		// Log headers in debug mode
		for name, values := range r.Header {
			for _, value := range values {
				log.Debug("Request header", "name", name, "value", value, "request_id", reqID)
			}
		}

		// Wrap the response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		log.Debug("Request completed",
			"method", r.Method,
			"url", r.URL.String(),
			"status", ww.Status(),
			"bytes_written", ww.BytesWritten(),
			"duration", duration,
			"request_id", reqID,
		)
	})
}
