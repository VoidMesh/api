package handlers

import (
	"context"

	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
)

// characterServiceWrapper implements the CharacterService interface using the real character service
type characterServiceWrapper struct {
	service *character.Service
}

// NewCharacterService creates a new CharacterService implementation
func NewCharacterService(service *character.Service) CharacterService {
	return &characterServiceWrapper{
		service: service,
	}
}

// NewCharacterServiceWithPool creates a character service with all dependencies wired up
// This function creates the necessary services and dependencies
func NewCharacterServiceWithPool(dbPool *pgxpool.Pool) (CharacterService, error) {
	// Create world service
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)

	// Get default world or create if it doesn't exist
	defaultWorld, err := worldService.GetDefaultWorld(context.Background())
	if err != nil {
		return nil, err
	}

	// Create chunk service with noise generator
	noiseGen := noise.NewGenerator(defaultWorld.Seed)
	chunkService := chunk.NewServiceWithPool(dbPool, worldService, noiseGen.(*noise.Generator))

	// Create character service
	characterService := character.NewServiceWithPool(dbPool, chunkService)

	return NewCharacterService(characterService), nil
}

// CreateCharacter creates a new character for a user
func (w *characterServiceWrapper) CreateCharacter(ctx context.Context, userID string, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error) {
	return w.service.CreateCharacter(ctx, userID, req)
}

// GetCharacter retrieves a character by ID
func (w *characterServiceWrapper) GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error) {
	return w.service.GetCharacter(ctx, req)
}

// GetUserCharacters retrieves all characters for a user
func (w *characterServiceWrapper) GetUserCharacters(ctx context.Context, userID string) (*characterV1.GetMyCharactersResponse, error) {
	return w.service.GetUserCharacters(ctx, userID)
}

// DeleteCharacter deletes a character
func (w *characterServiceWrapper) DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error) {
	return w.service.DeleteCharacter(ctx, req)
}

// MoveCharacter moves a character
func (w *characterServiceWrapper) MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error) {
	return w.service.MoveCharacter(ctx, req)
}