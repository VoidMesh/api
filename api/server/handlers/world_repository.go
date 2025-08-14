package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// worldServiceWrapper implements the WorldService interface using the real world service
type worldServiceWrapper struct {
	service *world.Service
}

// NewWorldService creates a new WorldService implementation
func NewWorldService(service *world.Service) WorldService {
	return &worldServiceWrapper{
		service: service,
	}
}

// NewWorldServiceFromConcreteService creates a WorldService wrapper from a concrete world service
// This maintains backward compatibility with the old NewWorldHandler function
func NewWorldServiceFromConcreteService(service *world.Service) WorldService {
	return NewWorldService(service)
}

// NewWorldServiceWithPool creates a world service with all dependencies wired up
// This function creates the necessary services and dependencies
func NewWorldServiceWithPool(dbPool *pgxpool.Pool) (WorldService, error) {
	// Create world service
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)

	return NewWorldService(worldService), nil
}

// GetWorldByID retrieves a world by ID
func (w *worldServiceWrapper) GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error) {
	return w.service.GetWorldByID(ctx, id)
}

// GetDefaultWorld gets or creates the default world
func (w *worldServiceWrapper) GetDefaultWorld(ctx context.Context) (db.World, error) {
	return w.service.GetDefaultWorld(ctx)
}

// ListWorlds retrieves all worlds
func (w *worldServiceWrapper) ListWorlds(ctx context.Context) ([]db.World, error) {
	return w.service.ListWorlds(ctx)
}

// UpdateWorld updates a world's name
func (w *worldServiceWrapper) UpdateWorld(ctx context.Context, id pgtype.UUID, name string) (db.World, error) {
	return w.service.UpdateWorld(ctx, id, name)
}

// DeleteWorld deletes a world
func (w *worldServiceWrapper) DeleteWorld(ctx context.Context, id pgtype.UUID) error {
	return w.service.DeleteWorld(ctx, id)
}