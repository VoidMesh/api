package handlers

import (
	"context"

	worldV1 "github.com/VoidMesh/platform/api/proto/world/v1"
	"github.com/VoidMesh/platform/api/services/chunk"
	"github.com/VoidMesh/platform/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
)

type worldServiceServer struct {
	worldV1.UnimplementedWorldServiceServer
	service *world.Service
}

func NewWorldServer(db *pgxpool.Pool) worldV1.WorldServiceServer {
	// Get world seed from database
	worldSeed, err := getWorldSeed(db)
	if err != nil {
		worldSeed = 12345 // Default seed
	}

	// Create chunk service
	chunkService := chunk.NewService(db, worldSeed)

	// Create world service
	worldService := world.NewService(db, chunkService, worldSeed)

	return &worldServiceServer{
		service: worldService,
	}
}

// GetWorldInfo gets world information
func (s *worldServiceServer) GetWorldInfo(ctx context.Context, req *worldV1.GetWorldInfoRequest) (*worldV1.GetWorldInfoResponse, error) {
	return s.service.GetWorldInfo(ctx, req)
}
