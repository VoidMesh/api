package handlers

import (
	"context"
	"strconv"

	"github.com/VoidMesh/platform/api/db"
	chunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	"github.com/VoidMesh/platform/api/services/chunk"
	"github.com/jackc/pgx/v5/pgxpool"
)

type chunkServiceServer struct {
	chunkV1.UnimplementedChunkServiceServer
	service *chunk.Service
}

func NewChunkServer(db *pgxpool.Pool) chunkV1.ChunkServiceServer {
	// Get world seed from database
	worldSeed, err := getWorldSeedForChunk(db)
	if err != nil {
		worldSeed = 12345 // Default seed
	}

	// Create chunk service
	chunkService := chunk.NewService(db, worldSeed)

	return &chunkServiceServer{
		service: chunkService,
	}
}

// getWorldSeedForChunk retrieves the world seed from database
func getWorldSeedForChunk(pool *pgxpool.Pool) (int64, error) {
	ctx := context.Background()
	setting, err := db.New(pool).GetWorldSetting(ctx, "seed")
	if err != nil {
		return 0, err
	}
	
	seed, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return 0, err
	}
	
	return seed, nil
}

// GetChunk retrieves a single chunk
func (s *chunkServiceServer) GetChunk(ctx context.Context, req *chunkV1.GetChunkRequest) (*chunkV1.GetChunkResponse, error) {
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
	chunks, err := s.service.GetChunksInRadius(ctx, req.CenterChunkX, req.CenterChunkY, req.Radius)
	if err != nil {
		return nil, err
	}

	return &chunkV1.GetChunksInRadiusResponse{
		Chunks: chunks,
	}, nil
}