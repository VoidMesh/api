package api

import (
	"github.com/VoidMesh/api/internal/player"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func SetupRoutes(handler *Handler, playerHandlers *player.PlayerHandlers) *chi.Mux {
	r := chi.NewRouter()

	// Setup middleware
	for _, middleware := range SetupMiddleware() {
		r.Use(middleware)
	}

	// JSON content type
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Health check endpoint
	r.Get("/health", handler.HealthCheck)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public chunk routes (no authentication required)
		r.Get("/chunks/{x}/{z}/nodes", handler.GetChunk)

		// Player authentication routes (no auth required)
		playerHandlers.RegisterRoutes(r)

		// Protected harvest routes (authentication required)
		r.With(player.AuthMiddleware(handler.playerManager)).Group(func(r chi.Router) {
			r.Post("/harvest/start", handler.StartHarvest)
			r.Put("/harvest/sessions/{sessionId}", handler.HarvestResource)
			r.Get("/my/sessions", handler.GetPlayerSessions) // Changed from player-specific to authenticated user
		})
	})

	return r
}
