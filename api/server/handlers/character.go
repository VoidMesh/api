package handlers

import (
	"context"

	characterV1 "github.com/VoidMesh/platform/api/proto/character/v1"
	"github.com/VoidMesh/platform/api/services/character"
	"github.com/VoidMesh/platform/api/services/chunk"
	"github.com/jackc/pgx/v5/pgxpool"
)

type characterServiceServer struct {
	characterV1.UnimplementedCharacterServiceServer
	service *character.Service
}

func NewCharacterServer(db *pgxpool.Pool) characterV1.CharacterServiceServer {
	// Get world seed from database
	worldSeed, err := getWorldSeed(db)
	if err != nil {
		worldSeed = 12345 // Default seed
	}

	// Create chunk service
	chunkService := chunk.NewService(db, worldSeed)

	// Create world service
	worldService := character.NewService(db, chunkService)

	return &characterServiceServer{
		service: worldService,
	}
}

// CreateCharacter creates a new character
func (s *characterServiceServer) CreateCharacter(ctx context.Context, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error) {
	return s.service.CreateCharacter(ctx, req)
}

// GetCharacter gets a character by ID
func (s *characterServiceServer) GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error) {
	return s.service.GetCharacter(ctx, req)
}

// GetCharactersByUser gets all characters for a user
func (s *characterServiceServer) GetCharactersByUser(ctx context.Context, req *characterV1.GetCharactersByUserRequest) (*characterV1.GetCharactersByUserResponse, error) {
	return s.service.GetCharactersByUser(ctx, req)
}

// DeleteCharacter deletes a character
func (s *characterServiceServer) DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error) {
	return s.service.DeleteCharacter(ctx, req)
}

// MoveCharacter moves a character
func (s *characterServiceServer) MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error) {
	return s.service.MoveCharacter(ctx, req)
}
