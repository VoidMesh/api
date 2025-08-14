package handlers

import (
	"context"
	"time"

	"github.com/VoidMesh/api/api/internal/logging"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/VoidMesh/api/api/server/middleware"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type characterServiceServer struct {
	characterV1.UnimplementedCharacterServiceServer
	characterService CharacterService
	logger           *log.Logger
}

func NewCharacterServer(
	characterService CharacterService,
) characterV1.CharacterServiceServer {
	logger := logging.WithComponent("character-handler")
	logger.Debug("Creating new CharacterService server instance")
	return &characterServiceServer{
		characterService: characterService,
		logger:           logger,
	}
}

// NewCharacterServerWithPool creates a character server with all dependencies wired up
// This function maintains backward compatibility while providing dependency injection
func NewCharacterServerWithPool(dbPool *pgxpool.Pool) (characterV1.CharacterServiceServer, error) {
	logger := logging.WithComponent("character-handler")
	logger.Debug("Creating CharacterService server with dependency injection")

	// Create character service with all dependencies
	characterService, err := NewCharacterServiceWithPool(dbPool)
	if err != nil {
		logger.Error("Failed to create character service", "error", err)
		return nil, err
	}

	return NewCharacterServer(characterService), nil
}

// CreateCharacter creates a new character
func (s *characterServiceServer) CreateCharacter(ctx context.Context, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error) {
	// Extract user ID from context metadata (set by JWT middleware)
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	logger := logging.WithFields("operation", "CreateCharacter", "user_id", userID, "character_name", req.Name, "spawn_x", req.SpawnX, "spawn_y", req.SpawnY)
	logger.Debug("Creating new character")

	start := time.Now()
	resp, err := s.characterService.CreateCharacter(ctx, userID, req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Character creation failed", "error", err, "duration", duration)
		return nil, err
	}

	logger.Info("Character created successfully", "character_id", resp.Character.Id, "duration", duration)
	return resp, nil
}

// GetCharacter gets a character by ID
func (s *characterServiceServer) GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error) {
	logger := logging.WithFields("operation", "GetCharacter", "character_id", req.CharacterId)
	logger.Debug("Retrieving character")

	start := time.Now()
	resp, err := s.characterService.GetCharacter(ctx, req)
	duration := time.Since(start)

	if err != nil {
		logger.Warn("Character retrieval failed", "error", err, "duration", duration)
		return nil, err
	}

	logger.Debug("Character retrieved successfully", "x", resp.Character.X, "y", resp.Character.Y, "duration", duration)
	return resp, nil
}

// GetMyCharacters gets all characters for the authenticated user
func (s *characterServiceServer) GetMyCharacters(ctx context.Context, req *characterV1.GetMyCharactersRequest) (*characterV1.GetMyCharactersResponse, error) {
	// Extract user ID from context metadata (set by JWT middleware)
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	logger := s.logger.With("operation", "GetMyCharacters", "user_id", userID)
	logger.Debug("Received GetMyCharacters request")

	start := time.Now()
	resp, err := s.characterService.GetUserCharacters(ctx, userID)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Failed to get characters for user", "error", err, "duration", duration)
		return nil, err
	}

	logger.Info("Successfully retrieved characters for user", "count", len(resp.Characters), "duration", duration)
	return resp, nil
}

// DeleteCharacter deletes a character
func (s *characterServiceServer) DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error) {
	logger := s.logger.With("operation", "DeleteCharacter", "character_id", req.CharacterId)
	logger.Debug("Received DeleteCharacter request")

	start := time.Now()
	resp, err := s.characterService.DeleteCharacter(ctx, req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Failed to delete character", "error", err, "duration", duration)
		return nil, err
	}

	logger.Info("Character deleted successfully", "character_id", req.CharacterId, "duration", duration)
	return resp, nil
}

// MoveCharacter moves a character
func (s *characterServiceServer) MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error) {
	logger := logging.WithFields("operation", "MoveCharacter", "character_id", req.CharacterId, "new_x", req.NewX, "new_y", req.NewY)
	logger.Debug("Processing character movement request")

	start := time.Now()
	resp, err := s.characterService.MoveCharacter(ctx, req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Character movement failed", "error", err, "duration", duration)
		return nil, err
	}

	if resp.Success {
		logger.Info("Character moved successfully", "final_x", resp.Character.X, "final_y", resp.Character.Y, "duration", duration)
	} else {
		logger.Warn("Character movement rejected", "reason", resp.ErrorMessage, "duration", duration)
	}
	return resp, nil
}
