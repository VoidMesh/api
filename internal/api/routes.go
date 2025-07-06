package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func SetupRoutes(handler *Handler) *chi.Mux {
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
		// Chunk routes
		r.Get("/chunks/{x}/{z}/nodes", handler.GetChunk)

		// Harvest routes
		r.Post("/harvest/start", handler.StartHarvest)
		r.Put("/harvest/sessions/{sessionId}", handler.HarvestResource)

		// Player routes
		r.Get("/players/{playerId}/sessions", handler.GetPlayerSessions)
	})

	return r
}
