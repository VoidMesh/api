package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chunkServiceWrapper implements the ChunkService interface using the real chunk service
type chunkServiceWrapper struct {
	service *chunk.Service
}

// NewChunkService creates a new ChunkService implementation
func NewChunkService(service *chunk.Service) ChunkService {
	return &chunkServiceWrapper{
		service: service,
	}
}

// NewChunkServiceWithPool creates a chunk service with all dependencies wired up
// This function creates the necessary services and dependencies for production use
func NewChunkServiceWithPool(dbPool *pgxpool.Pool) (ChunkService, error) {
	// Create world service (needed for chunk service)
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)

	// Get default world or create if it doesn't exist
	defaultWorld, err := worldService.GetDefaultWorld(context.Background())
	if err != nil {
		return nil, err
	}

	// Create noise generator for chunk generation
	noiseGen := noise.NewGenerator(defaultWorld.Seed)

	// Create chunk service with all dependencies
	chunkService := chunk.NewServiceWithPool(dbPool, worldService, noiseGen.(*noise.Generator))

	return NewChunkService(chunkService), nil
}

// NewChunkServerWithPool creates a complete chunk handler with all dependencies for production use
func NewChunkServerWithPool(dbPool *pgxpool.Pool) (chunkV1.ChunkServiceServer, error) {
	// Create chunk service
	chunkService, err := NewChunkServiceWithPool(dbPool)
	if err != nil {
		return nil, err
	}

	// Create world service wrapper
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)
	worldServiceWrapper := NewWorldService(worldService)

	// Create logger wrapper
	logger := logging.WithComponent("chunk-handler")
	loggerWrapper := NewLoggerWrapper(logger)

	// Create the handler with dependency injection
	return NewChunkServer(chunkService, worldServiceWrapper, loggerWrapper), nil
}

// loggerWrapper adapts charmbracelet/log.Logger to LoggerInterface
type loggerWrapper struct {
	logger *log.Logger
}

// NewLoggerWrapper creates a LoggerInterface implementation from charmbracelet/log.Logger
func NewLoggerWrapper(logger *log.Logger) LoggerInterface {
	return &loggerWrapper{logger: logger}
}

// Debug logs a debug level message with optional key-value pairs
func (l *loggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

// Info logs an info level message with optional key-value pairs
func (l *loggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

// Warn logs a warning level message with optional key-value pairs
func (l *loggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, keysAndValues...)
}

// Error logs an error level message with optional key-value pairs
func (l *loggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
}

// With returns a new logger with the given key-value pairs added to the context
func (l *loggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	newLogger := l.logger.With(keysAndValues...)
	return &loggerWrapper{logger: newLogger}
}


// GetOrCreateChunk retrieves an existing chunk or creates a new one
func (w *chunkServiceWrapper) GetOrCreateChunk(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	return w.service.GetOrCreateChunk(ctx, chunkX, chunkY)
}

// GetChunksInRange retrieves multiple chunks in a rectangular area
func (w *chunkServiceWrapper) GetChunksInRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*chunkV1.ChunkData, error) {
	return w.service.GetChunksInRange(ctx, minX, maxX, minY, maxY)
}

// GetChunksInRadius retrieves chunks in a circular area
func (w *chunkServiceWrapper) GetChunksInRadius(ctx context.Context, centerX, centerY, radius int32) ([]*chunkV1.ChunkData, error) {
	return w.service.GetChunksInRadius(ctx, centerX, centerY, radius)
}