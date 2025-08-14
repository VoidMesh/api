package handlers

import (
	"context"
	"errors"

	"github.com/VoidMesh/api/api/internal/logging"
	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/VoidMesh/api/api/services/terrain"
	"github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// terrainServiceServer implements the TerrainService gRPC service
type terrainServiceServer struct {
	terrainV1.UnimplementedTerrainServiceServer
	terrainService TerrainService
	logger         LoggerInterface
}

// NewTerrainServer creates a new terrain server with dependency injection
func NewTerrainServer(
	terrainService TerrainService,
	logger LoggerInterface,
) terrainV1.TerrainServiceServer {
	logger.Debug("Creating new TerrainService server instance")
	return &terrainServiceServer{
		terrainService: terrainService,
		logger:         logger,
	}
}

// NewTerrainHandler creates a new terrain handler (backward compatibility)
// Deprecated: Use NewTerrainServer with dependency injection instead
func NewTerrainHandler(terrainService *terrain.Service) terrainV1.TerrainServiceServer {
	logger := logging.WithComponent("terrain-handler")
	logger.Debug("Creating terrain handler with legacy constructor")
	
	// Convert log.Logger to LoggerInterface
	loggerWrapper := &LoggerWrapper{Logger: logger}
	
	// Create service wrapper
	serviceWrapper := NewTerrainService(terrainService)
	
	return NewTerrainServer(serviceWrapper, loggerWrapper)
}

// GetTerrainTypes returns all available terrain types
func (s *terrainServiceServer) GetTerrainTypes(ctx context.Context, req *terrainV1.GetTerrainTypesRequest) (*terrainV1.GetTerrainTypesResponse, error) {
	logger := s.logger.With("operation", "GetTerrainTypes")
	logger.Debug("Received GetTerrainTypes request")

	terrainTypes, err := s.terrainService.GetTerrainTypes(ctx)
	if err != nil {
		logger.Error("Failed to get terrain types", "error", err)
		
		// Handle context-specific errors
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.Canceled, "request was cancelled")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		}
		
		// If it's already a gRPC status error, return it as-is
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		
		// For regular Go errors, preserve the original message but wrap with Internal code
		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.Info("Retrieved terrain types", "count", len(terrainTypes))
	return &terrainV1.GetTerrainTypesResponse{
		TerrainTypes: terrainTypes,
	}, nil
}

// LoggerWrapper adapts log.Logger to LoggerInterface
type LoggerWrapper struct {
	Logger *log.Logger
}

func (w *LoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	w.Logger.Debug(msg, keysAndValues...)
}

func (w *LoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	w.Logger.Info(msg, keysAndValues...)
}

func (w *LoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	w.Logger.Warn(msg, keysAndValues...)
}

func (w *LoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	w.Logger.Error(msg, keysAndValues...)
}

func (w *LoggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	return &LoggerWrapper{Logger: w.Logger.With(keysAndValues...)}
}
