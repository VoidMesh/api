package chunk

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseInterface abstracts database operations for the chunk service.
type DatabaseInterface interface {
	GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error)
	CreateChunk(ctx context.Context, arg db.CreateChunkParams) (db.Chunk, error)
	ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error)
}

// DatabaseWrapper implements DatabaseInterface using the actual database connection.
type DatabaseWrapper struct {
	queries *db.Queries
}

// NewDatabaseWrapper creates a new database wrapper with the given connection pool.
func NewDatabaseWrapper(pool *pgxpool.Pool) DatabaseInterface {
	return &DatabaseWrapper{
		queries: db.New(pool),
	}
}

func (d *DatabaseWrapper) GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error) {
	return d.queries.GetChunk(ctx, arg)
}

func (d *DatabaseWrapper) CreateChunk(ctx context.Context, arg db.CreateChunkParams) (db.Chunk, error) {
	return d.queries.CreateChunk(ctx, arg)
}

func (d *DatabaseWrapper) ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error) {
	return d.queries.ChunkExists(ctx, arg)
}

// NoiseGeneratorInterface defines the interface for noise generation operations.
type NoiseGeneratorInterface interface {
	GetTerrainNoise(x, y int, scale float64) float64
	GetSeed() int64
}

// WorldServiceInterface defines the interface for world service operations.
type WorldServiceInterface interface {
	GetDefaultWorld(ctx context.Context) (db.World, error)
	GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error)
	ChunkSize() int32
}

// LoggerInterface abstracts logging operations for dependency injection.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) LoggerInterface
}

// ResourceNodeIntegrationInterface defines the interface for resource node integration.
type ResourceNodeIntegrationInterface interface {
	AttachResourceNodesToChunk(ctx context.Context, chunk *chunkV1.ChunkData) error
	GenerateAndAttachResourceNodes(ctx context.Context, chunk *chunkV1.ChunkData) error
}

// Adapter types to implement interfaces for existing services

// NoiseGeneratorAdapter adapts a noise.GeneratorInterface to our interface.
type NoiseGeneratorAdapter struct {
	generator noise.GeneratorInterface
}

func NewNoiseGeneratorAdapter(generator noise.GeneratorInterface) NoiseGeneratorInterface {
	return &NoiseGeneratorAdapter{generator: generator}
}

func (n *NoiseGeneratorAdapter) GetTerrainNoise(x, y int, scale float64) float64 {
	return n.generator.GetTerrainNoise(x, y, scale)
}

func (n *NoiseGeneratorAdapter) GetSeed() int64 {
	return n.generator.GetSeed()
}

// WorldServiceAdapter adapts a world.Service to our interface.
type WorldServiceAdapter struct {
	service *world.Service
}

func NewWorldServiceAdapter(service *world.Service) WorldServiceInterface {
	return &WorldServiceAdapter{service: service}
}

func (w *WorldServiceAdapter) GetDefaultWorld(ctx context.Context) (db.World, error) {
	return w.service.GetDefaultWorld(ctx)
}

func (w *WorldServiceAdapter) GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error) {
	return w.service.GetWorldByID(ctx, id)
}

func (w *WorldServiceAdapter) ChunkSize() int32 {
	return w.service.ChunkSize()
}

// DefaultLoggerWrapper wraps the internal logging package.
type DefaultLoggerWrapper struct{}

func NewDefaultLoggerWrapper() LoggerInterface {
	return &DefaultLoggerWrapper{}
}

func (l *DefaultLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	return l
}