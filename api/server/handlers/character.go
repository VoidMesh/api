package handlers

import (
	"context"
	"time"

	"github.com/VoidMesh/api/api/internal/logging"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/jackc/pgx/v5/pgxpool"
)

type characterServiceServer struct {
	characterV1.UnimplementedCharacterServiceServer
	service *character.Service
}

func NewCharacterServer(db *pgxpool.Pool) characterV1.CharacterServiceServer {
	logger := logging.GetLogger()
	logger.Debug("Creating new CharacterService server instance")

	// Get world seed from database
	logger.Debug("Loading world seed from database")
	worldSeed, err := getWorldSeed(db)
	if err != nil {
		logger.Warn("Failed to load world seed from database, using default", "error", err, "default_seed", 12345)
		worldSeed = 12345 // Default seed
	} else {
		logger.Debug("World seed loaded successfully", "seed", worldSeed)
	}

	// Create chunk service with noise generator
	logger.Debug("Initializing chunk service", "world_seed", worldSeed)
	noiseGen := noise.NewGenerator(worldSeed)
	chunkService := chunk.NewService(db, worldSeed, noiseGen)

	// Create character service
	logger.Debug("Initializing character service")
	characterService := character.NewService(db, chunkService)

	return &characterServiceServer{
		service: characterService,
	}
}

// CreateCharacter creates a new character
func (s *characterServiceServer) CreateCharacter(ctx context.Context, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error) {
	logger := logging.WithFields("operation", "CreateCharacter", "user_id", req.UserId, "character_name", req.Name, "spawn_x", req.SpawnX, "spawn_y", req.SpawnY)
	logger.Debug("Creating new character")

	start := time.Now()
	resp, err := s.service.CreateCharacter(ctx, req)
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
	resp, err := s.service.GetCharacter(ctx, req)
	duration := time.Since(start)

	if err != nil {
		logger.Warn("Character retrieval failed", "error", err, "duration", duration)
		return nil, err
	}

	logger.Debug("Character retrieved successfully", "x", resp.Character.X, "y", resp.Character.Y, "duration", duration)
	return resp, nil
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
	logger := logging.WithFields("operation", "MoveCharacter", "character_id", req.CharacterId, "new_x", req.NewX, "new_y", req.NewY)
	logger.Debug("Processing character movement request")

	start := time.Now()
	resp, err := s.service.MoveCharacter(ctx, req)
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
