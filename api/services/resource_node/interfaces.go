package resource_node

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseInterface abstracts database operations for the resource node service.
type DatabaseInterface interface {
	CreateResourceNode(ctx context.Context, arg db.CreateResourceNodeParams) (db.ResourceNode, error)
	DeleteResourceNodesInChunk(ctx context.Context, arg db.DeleteResourceNodesInChunkParams) error
	GetResourceNodesInChunk(ctx context.Context, arg db.GetResourceNodesInChunkParams) ([]db.ResourceNode, error)
	GetResourceNodesInChunks(ctx context.Context, arg db.GetResourceNodesInChunksParams) ([]db.ResourceNode, error)
	GetResourceNodesInChunkRange(ctx context.Context, arg db.GetResourceNodesInChunkRangeParams) ([]db.ResourceNode, error)
	ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error)
	GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error)
	GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error)
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

func (d *DatabaseWrapper) CreateResourceNode(ctx context.Context, arg db.CreateResourceNodeParams) (db.ResourceNode, error) {
	return d.queries.CreateResourceNode(ctx, arg)
}

func (d *DatabaseWrapper) DeleteResourceNodesInChunk(ctx context.Context, arg db.DeleteResourceNodesInChunkParams) error {
	return d.queries.DeleteResourceNodesInChunk(ctx, arg)
}

func (d *DatabaseWrapper) GetResourceNodesInChunk(ctx context.Context, arg db.GetResourceNodesInChunkParams) ([]db.ResourceNode, error) {
	return d.queries.GetResourceNodesInChunk(ctx, arg)
}

func (d *DatabaseWrapper) GetResourceNodesInChunks(ctx context.Context, arg db.GetResourceNodesInChunksParams) ([]db.ResourceNode, error) {
	return d.queries.GetResourceNodesInChunks(ctx, arg)
}

func (d *DatabaseWrapper) GetResourceNodesInChunkRange(ctx context.Context, arg db.GetResourceNodesInChunkRangeParams) ([]db.ResourceNode, error) {
	return d.queries.GetResourceNodesInChunkRange(ctx, arg)
}

func (d *DatabaseWrapper) ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error) {
	return d.queries.ChunkExists(ctx, arg)
}

func (d *DatabaseWrapper) GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error) {
	return d.queries.GetChunk(ctx, arg)
}

func (d *DatabaseWrapper) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	return d.queries.GetResourceNode(ctx, id)
}

// NoiseGeneratorInterface defines the interface for noise generation operations.
type NoiseGeneratorInterface interface {
	GetTerrainNoise(x, y int, scale float64) float64
	GetSeed() int64
}

// WorldServiceInterface defines the interface for world service operations.
type WorldServiceInterface interface {
	GetDefaultWorld(ctx context.Context) (db.World, error)
}

// LoggerInterface abstracts logging operations for dependency injection.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) LoggerInterface
}

// RandomGeneratorInterface defines the interface for random number generation.
type RandomGeneratorInterface interface {
	Intn(n int) int
	Int31n(n int32) int32
	Float32() float32
	Shuffle(n int, swap func(i, j int))
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