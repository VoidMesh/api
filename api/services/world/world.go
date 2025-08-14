package world

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/VoidMesh/api/api/internal/logging"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VoidMesh/api/api/db"
)

// LoggerInterface abstracts logging operations for dependency injection.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) LoggerInterface
}

// DefaultLoggerWrapper wraps the internal logging package.
type DefaultLoggerWrapper struct{}

// NewDefaultLoggerWrapper creates a new default logger wrapper.
func NewDefaultLoggerWrapper() LoggerInterface {
	return &DefaultLoggerWrapper{}
}

func (l *DefaultLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Debug(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Info(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Warn(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Error(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	// For now, return self for simplicity
	return l
}

// Service provides operations on worlds
type Service struct {
	db     DatabaseInterface
	logger LoggerInterface
}

// NewService creates a new world service with dependency injection.
func NewService(db DatabaseInterface, logger LoggerInterface) *Service {
	componentLogger := logger.With("component", "world-service")
	componentLogger.Debug("Creating new world service")
	return &Service{
		db:     db,
		logger: componentLogger,
	}
}

// NewServiceWithPool creates a service with a database pool (convenience constructor for production use).
func NewServiceWithPool(pool *pgxpool.Pool, logger LoggerInterface) *Service {
	return NewService(NewDatabaseWrapper(pool), logger)
}

// GetDefaultWorld gets or creates the default world
func (s *Service) GetDefaultWorld(ctx context.Context) (db.World, error) {
	world, err := s.db.GetDefaultWorld(ctx)
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
	return s.db.GetWorldByID(ctx, id)
}

// ListWorlds gets all worlds
func (s *Service) ListWorlds(ctx context.Context) ([]db.World, error) {
	return s.db.ListWorlds(ctx)
}

// CreateWorld creates a new world with a random seed
func (s *Service) createWorld(ctx context.Context, name string) (db.World, error) {
	seed := rand.Int63()
	return s.db.CreateWorld(ctx, db.CreateWorldParams{
		Name: name,
		Seed: seed,
	})
}

// UpdateWorld updates a world's name
func (s *Service) UpdateWorld(ctx context.Context, id pgtype.UUID, name string) (db.World, error) {
	return s.db.UpdateWorld(ctx, db.UpdateWorldParams{
		ID:   id,
		Name: name,
	})
}

// DeleteWorld deletes a world
func (s *Service) DeleteWorld(ctx context.Context, id pgtype.UUID) error {
	return s.db.DeleteWorld(ctx, id)
}

// ChunkSize returns the chunk size (hardcoded to 32)
func (s *Service) ChunkSize() int32 {
	return 32
}
