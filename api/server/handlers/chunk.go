package handlers

import (
	"context"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chunkServiceServer struct {
	chunkV1.UnimplementedChunkServiceServer
	chunkService ChunkService
	worldService WorldService
	logger       LoggerInterface
}

// NewChunkServer creates a chunk server with interface dependencies (primary constructor for testing)
func NewChunkServer(
	chunkService ChunkService,
	worldService WorldService,
	logger LoggerInterface,
) chunkV1.ChunkServiceServer {
	logger.Debug("Creating new ChunkService server instance")
	return &chunkServiceServer{
		chunkService: chunkService,
		worldService: worldService,
		logger:       logger,
	}
}


// resolveWorldID resolves the world ID from the request, using the default world if not provided
func (s *chunkServiceServer) resolveWorldID(ctx context.Context, worldIDBytes []byte, logger LoggerInterface) (pgtype.UUID, error) {
	var worldID pgtype.UUID
	
	if len(worldIDBytes) == 0 {
		logger.Debug("World ID not provided, using default world")
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			logger.Error("Failed to get default world", "error", err)
			return worldID, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(worldIDBytes)
		if err != nil {
			logger.Warn("Invalid world ID format", "world_id", worldIDBytes, "error", err)
			return worldID, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}
	
	return worldID, nil
}

// GetChunk retrieves a single chunk
func (s *chunkServiceServer) GetChunk(ctx context.Context, req *chunkV1.GetChunkRequest) (*chunkV1.GetChunkResponse, error) {
	logger := s.logger.With("operation", "GetChunk", "chunk_x", req.ChunkX, "chunk_y", req.ChunkY)
	logger.Debug("Received GetChunk request")

	// Resolve world ID using helper method
	worldID, err := s.resolveWorldID(ctx, req.WorldId, logger)
	if err != nil {
		return nil, err
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching or creating chunk")
	chunk, err := s.chunkService.GetOrCreateChunk(ctx, req.ChunkX, req.ChunkY)
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

	// Resolve world ID using helper method
	worldID, err := s.resolveWorldID(ctx, req.WorldId, logger)
	if err != nil {
		return nil, err
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching chunks in range")
	chunks, err := s.chunkService.GetChunksInRange(ctx, req.MinChunkX, req.MaxChunkX, req.MinChunkY, req.MaxChunkY)
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

	// Resolve world ID using helper method
	worldID, err := s.resolveWorldID(ctx, req.WorldId, logger)
	if err != nil {
		return nil, err
	}
	logger = logger.With("world_id", worldID.Bytes)

	logger.Debug("Fetching chunks in radius")
	chunks, err := s.chunkService.GetChunksInRadius(ctx, req.CenterChunkX, req.CenterChunkY, req.Radius)
	if err != nil {
		logger.Error("Failed to get chunks in radius", "error", err)
		return nil, err
	}

	logger.Info("Successfully retrieved chunks in radius", "count", len(chunks))
	return &chunkV1.GetChunksInRadiusResponse{
		Chunks: chunks,
	}, nil
}
