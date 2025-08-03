package world

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VoidMesh/api/api/db"
)

// Service provides operations on worlds
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	logger  *log.Logger
}

// NewService creates a new world service
func NewService(pool *pgxpool.Pool, logger *log.Logger) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
		logger:  logger.With("component", "world-service"),
	}
}

// GetDefaultWorld gets or creates the default world
func (s *Service) GetDefaultWorld(ctx context.Context) (db.World, error) {
	world, err := s.queries.GetDefaultWorld(ctx)
	if err != nil {
		// Create a default world if none exists
		s.logger.Info("No default world found, creating a new one")
		world, err = s.createWorld(ctx, "VoidMesh World")
		if err != nil {
			return db.World{}, fmt.Errorf("failed to create default world: %w", err)
		}
	}
	return world, nil
}

// GetWorldByID gets a world by ID
func (s *Service) GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error) {
	return s.queries.GetWorldByID(ctx, id)
}

// ListWorlds gets all worlds
func (s *Service) ListWorlds(ctx context.Context) ([]db.World, error) {
	return s.queries.ListWorlds(ctx)
}

// CreateWorld creates a new world with a random seed
func (s *Service) createWorld(ctx context.Context, name string) (db.World, error) {
	seed := rand.Int63()
	return s.queries.CreateWorld(ctx, db.CreateWorldParams{
		Name: name,
		Seed: seed,
	})
}

// UpdateWorld updates a world's name
func (s *Service) UpdateWorld(ctx context.Context, id pgtype.UUID, name string) (db.World, error) {
	return s.queries.UpdateWorld(ctx, db.UpdateWorldParams{
		ID:   id,
		Name: name,
	})
}

// DeleteWorld deletes a world
func (s *Service) DeleteWorld(ctx context.Context, id pgtype.UUID) error {
	return s.queries.DeleteWorld(ctx, id)
}

// ChunkSize returns the chunk size (hardcoded to 32)
func (s *Service) ChunkSize() int32 {
	return 32
}
