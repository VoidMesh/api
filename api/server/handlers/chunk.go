package handlers

import (
	"context"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chunkServiceServer struct {
	chunkV1.UnimplementedChunkServiceServer
	service      *chunk.Service
	worldService *world.Service
}

func NewChunkServer(db *pgxpool.Pool, worldService *world.Service, noiseGen *noise.Generator) chunkV1.ChunkServiceServer {
	// Create chunk service with shared noise generator
	chunkService := chunk.NewService(db, worldService, noiseGen)

	return &chunkServiceServer{
		service:      chunkService,
		worldService: worldService,
	}
}

// GetChunk retrieves a single chunk
func (s *chunkServiceServer) GetChunk(ctx context.Context, req *chunkV1.GetChunkRequest) (*chunkV1.GetChunkResponse, error) {
	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		// If world_id is not provided, use default world
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}

	chunk, err := s.service.GetOrCreateChunk(ctx, req.ChunkX, req.ChunkY)
	if err != nil {
		return nil, err
	}

	return &chunkV1.GetChunkResponse{
		Chunk: chunk,
	}, nil
}

// GetChunks retrieves multiple chunks in a rectangular area
func (s *chunkServiceServer) GetChunks(ctx context.Context, req *chunkV1.GetChunksRequest) (*chunkV1.GetChunksResponse, error) {
	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		// If world_id is not provided, use default world
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}

	chunks, err := s.service.GetChunksInRange(ctx, req.MinChunkX, req.MaxChunkX, req.MinChunkY, req.MaxChunkY)
	if err != nil {
		return nil, err
	}

	return &chunkV1.GetChunksResponse{
		Chunks: chunks,
	}, nil
}

// GetChunksInRadius retrieves chunks in a circular area
func (s *chunkServiceServer) GetChunksInRadius(ctx context.Context, req *chunkV1.GetChunksInRadiusRequest) (*chunkV1.GetChunksInRadiusResponse, error) {
	// Get default world ID if not provided
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		// If world_id is not provided, use default world
		defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}

	chunks, err := s.service.GetChunksInRadius(ctx, req.CenterChunkX, req.CenterChunkY, req.Radius)
	if err != nil {
		return nil, err
	}

	return &chunkV1.GetChunksInRadiusResponse{
		Chunks: chunks,
	}, nil
}
