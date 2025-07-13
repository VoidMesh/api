package handlers

import (
	"context"
	"strconv"

	"github.com/VoidMesh/platform/api/db"
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

// getWorldSeed retrieves the world seed from database
func getWorldSeed(pool *pgxpool.Pool) (int64, error) {
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

// CreateCharacter creates a new character
func (s *worldServiceServer) CreateCharacter(ctx context.Context, req *worldV1.CreateCharacterRequest) (*worldV1.CreateCharacterResponse, error) {
	return s.service.CreateCharacter(ctx, req)
}

// GetCharacter gets a character by ID
func (s *worldServiceServer) GetCharacter(ctx context.Context, req *worldV1.GetCharacterRequest) (*worldV1.GetCharacterResponse, error) {
	return s.service.GetCharacter(ctx, req)
}

// GetCharactersByUser gets all characters for a user
func (s *worldServiceServer) GetCharactersByUser(ctx context.Context, req *worldV1.GetCharactersByUserRequest) (*worldV1.GetCharactersByUserResponse, error) {
	return s.service.GetCharactersByUser(ctx, req)
}

// DeleteCharacter deletes a character
func (s *worldServiceServer) DeleteCharacter(ctx context.Context, req *worldV1.DeleteCharacterRequest) (*worldV1.DeleteCharacterResponse, error) {
	return s.service.DeleteCharacter(ctx, req)
}

// MoveCharacter moves a character
func (s *worldServiceServer) MoveCharacter(ctx context.Context, req *worldV1.MoveCharacterRequest) (*worldV1.MoveCharacterResponse, error) {
	return s.service.MoveCharacter(ctx, req)
}

// GetWorldInfo gets world information
func (s *worldServiceServer) GetWorldInfo(ctx context.Context, req *worldV1.GetWorldInfoRequest) (*worldV1.GetWorldInfoResponse, error) {
	return s.service.GetWorldInfo(ctx, req)
}
