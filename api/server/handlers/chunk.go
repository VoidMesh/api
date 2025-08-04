package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chunkServiceServer struct {
	chunkV1.UnimplementedChunkServiceServer
	service      *chunk.Service
	worldService *world.Service
	logger       *log.Logger
}

func NewChunkServer(db *pgxpool.Pool, worldService *world.Service, noiseGen *noise.Generator) chunkV1.ChunkServiceServer {
	logger := logging.WithComponent("chunk-handler")
	logger.Debug("Creating new ChunkService server instance")

	// Create chunk service with shared noise generator
	chunkService := chunk.NewService(db, worldService, noiseGen)

	return &chunkServiceServer{
		service:      chunkService,
		worldService: worldService,
		logger:       logger,
	}
}

// GetChunk retrieves a single chunk
func (s *chunkServiceServer) GetChunk(ctx context.Context, req *chunkV1.GetChunkRequest) (*chunkV1.GetChunkResponse, error) {
	logger := s.logger.With("operation", "GetChunk", "chunk_x", req.ChunkX, "chunk_y", req.ChunkY)
	logger.Debug("Received GetChunk request")

	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		// If world_id is not provided, use default world
		logger.Debug("World ID not provided, using default world")
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			logger.Error("Failed to get default world", "error", err)
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			logger.Warn("Invalid world ID format", "world_id", req.WorldId, "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching or creating chunk")
	chunk, err := s.service.GetOrCreateChunk(ctx, req.ChunkX, req.ChunkY)
	if err != nil {
		logger.Error("Failed to get or create chunk", "error", err)
		return nil, err
	}

	logger.Info("Successfully retrieved chunk")
	return &chunkV1.GetChunkResponse{
		Chunk: chunk,
	}, nil
}

// GetChunks retrieves multiple chunks in a rectangular area
func (s *chunkServiceServer) GetChunks(ctx context.Context, req *chunkV1.GetChunksRequest) (*chunkV1.GetChunksResponse, error) {
	logger := s.logger.With("operation", "GetChunks", "min_x", req.MinChunkX, "max_x", req.MaxChunkX, "min_y", req.MinChunkY, "max_y", req.MaxChunkY)
	logger.Debug("Received GetChunks request")

	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		logger.Debug("World ID not provided, using default world")
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			logger.Error("Failed to get default world", "error", err)
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			logger.Warn("Invalid world ID format", "world_id", req.WorldId, "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching chunks in range")
	chunks, err := s.service.GetChunksInRange(ctx, req.MinChunkX, req.MaxChunkX, req.MinChunkY, req.MaxChunkY)
	if err != nil {
		logger.Error("Failed to get chunks in range", "error", err)
		return nil, err
	}

	logger.Info("Successfully retrieved chunks in range", "count", len(chunks))
	return &chunkV1.GetChunksResponse{
		Chunks: chunks,
	}, nil
}

// GetChunksInRadius retrieves chunks in a circular area
func (s *chunkServiceServer) GetChunksInRadius(ctx context.Context, req *chunkV1.GetChunksInRadiusRequest) (*chunkV1.GetChunksInRadiusResponse, error) {
	logger := s.logger.With("operation", "GetChunksInRadius", "center_x", req.CenterChunkX, "center_y", req.CenterChunkY, "radius", req.Radius)
	logger.Debug("Received GetChunksInRadius request")

	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		logger.Debug("World ID not provided, using default world")
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			logger.Error("Failed to get default world", "error", err)
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			logger.Warn("Invalid world ID format", "world_id", req.WorldId, "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching chunks in radius")
	chunks, err := s.service.GetChunksInRadius(ctx, req.CenterChunkX, req.CenterChunkY, req.Radius)
	if err != nil {
		logger.Error("Failed to get chunks in radius", "error", err)
		return nil, err
	}

	logger.Info("Successfully retrieved chunks in radius", "count", len(chunks))
	return &chunkV1.GetChunksInRadiusResponse{
		Chunks: chunks,
	}, nil
}
