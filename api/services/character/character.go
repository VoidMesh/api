package character

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/logging"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	db           *pgxpool.Pool
	chunkService *chunk.Service
	chunkSize    int32
}

func NewService(db *pgxpool.Pool, chunkService *chunk.Service) *Service {
	logger := logging.GetLogger()
	logger.Debug("Creating new character service", "chunk_size", chunk.ChunkSize)
	return &Service{
		db:           db,
		chunkService: chunkService,
		chunkSize:    chunk.ChunkSize,
	}
}

// Helper function to convert DB character to proto character
func (s *Service) dbCharacterToProto(char db.Character) *characterV1.Character {
	protoChar := &characterV1.Character{
		Id:     hex.EncodeToString(char.ID.Bytes[:]),
		UserId: hex.EncodeToString(char.UserID.Bytes[:]),
		Name:   char.Name,
		X:      char.X,
		Y:      char.Y,
		ChunkX: char.ChunkX,
		ChunkY: char.ChunkY,
	}

	if char.CreatedAt.Valid {
		protoChar.CreatedAt = timestamppb.New(char.CreatedAt.Time)
	}

	return protoChar
}

// Helper function to parse UUID string to pgtype.UUID
func parseUUID(uuidStr string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(uuidStr)
	return uuid, err
}

// worldToChunkCoords converts world coordinates to chunk coordinates
func (s *Service) worldToChunkCoords(x, y int32) (chunkX, chunkY int32) {
	chunkX = x / s.chunkSize
	chunkY = y / s.chunkSize

	// Handle negative coordinates properly
	if x < 0 && x%s.chunkSize != 0 {
		chunkX--
	}
	if y < 0 && y%s.chunkSize != 0 {
		chunkY--
	}

	return chunkX, chunkY
}

// isValidSpawnPosition checks if the given position is a valid spawn location
func (s *Service) isValidSpawnPosition(ctx context.Context, x, y int32) (bool, error) {
	chunkX, chunkY := s.worldToChunkCoords(x, y)

	// Get the chunk
	chunkData, err := s.chunkService.GetOrCreateChunk(ctx, chunkX, chunkY)
	if err != nil {
		return false, err
	}

	// Calculate local coordinates within the chunk
	localX := x - chunkX*s.chunkSize
	localY := y - chunkY*s.chunkSize

	// Handle negative coordinates
	if localX < 0 {
		localX += s.chunkSize
	}
	if localY < 0 {
		localY += s.chunkSize
	}

	// Get the terrain cell (row-major order)
	index := localY*s.chunkSize + localX
	if index < 0 || index >= int32(len(chunkData.Cells)) {
		return false, nil
	}

	cell := chunkData.Cells[index]

	// Check if terrain is walkable (not water or stone)
	switch cell.TerrainType {
	case chunkV1.TerrainType_TERRAIN_TYPE_WATER, chunkV1.TerrainType_TERRAIN_TYPE_STONE:
		return false, nil
	default:
		return true, nil
	}
}

// CreateCharacter creates a new character for a user
func (s *Service) CreateCharacter(ctx context.Context, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error) {
	logger := logging.WithFields("user_id", req.UserId, "character_name", req.Name, "spawn_x", req.SpawnX, "spawn_y", req.SpawnY)
	logger.Debug("Starting character creation process")

	start := time.Now()

	userUUID, err := parseUUID(req.UserId)
	if err != nil {
		logger.Error("Invalid user ID format", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}
	logger.Debug("User ID parsed successfully")

	// Set spawn position (default to 0,0 if not specified)
	spawnX := req.SpawnX
	spawnY := req.SpawnY

	// Validate spawn position
	valid, err := s.isValidSpawnPosition(ctx, spawnX, spawnY)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate spawn position: %v", err)
	}
	if !valid {
		// Try to find a nearby valid spawn position
		spawnX, spawnY = s.findNearbySpawnPosition(ctx, spawnX, spawnY)
	}

	chunkX, chunkY := s.worldToChunkCoords(spawnX, spawnY)

	logger.Debug("Creating character record in database")
	character, err := db.New(s.db).CreateCharacter(ctx, db.CreateCharacterParams{
		UserID: userUUID,
		Name:   req.Name,
		X:      spawnX,
		Y:      spawnY,
		ChunkX: chunkX,
		ChunkY: chunkY,
	})
	if err != nil {
		if err.Error() == "duplicate key value violates unique constraint" {
			logger.Warn("Character creation failed: name already exists", "name", req.Name)
			return nil, status.Errorf(codes.AlreadyExists, "character with name '%s' already exists for this user", req.Name)
		}
		logger.Error("Failed to create character in database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create character: %v", err)
	}

	duration := time.Since(start)
	characterID := hex.EncodeToString(character.ID.Bytes[:])
	logger.Info("Character created successfully", "character_id", characterID, "final_x", character.X, "final_y", character.Y, "duration", duration)

	return &characterV1.CreateCharacterResponse{
		Character: s.dbCharacterToProto(character),
	}, nil
}

// findNearbySpawnPosition finds a nearby valid spawn position
func (s *Service) findNearbySpawnPosition(ctx context.Context, x, y int32) (int32, int32) {
	// Try positions in a spiral pattern around the original position
	for radius := int32(1); radius <= 10; radius++ {
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				// Only check positions on the edge of the current radius
				if abs32(dx) == radius || abs32(dy) == radius {
					testX := x + dx
					testY := y + dy
					if valid, _ := s.isValidSpawnPosition(ctx, testX, testY); valid {
						return testX, testY
					}
				}
			}
		}
	}
	// If no valid position found, return original position
	return x, y
}

func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// GetCharacter retrieves a character by ID
func (s *Service) GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error) {
	charUUID, err := parseUUID(req.CharacterId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID: %v", err)
	}

	character, err := db.New(s.db).GetCharacterById(ctx, charUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "character not found: %v", err)
	}

	return &characterV1.GetCharacterResponse{
		Character: s.dbCharacterToProto(character),
	}, nil
}

// GetCharactersByUser retrieves all characters for a user
func (s *Service) GetCharactersByUser(ctx context.Context, req *characterV1.GetCharactersByUserRequest) (*characterV1.GetCharactersByUserResponse, error) {
	userUUID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	characters, err := db.New(s.db).GetCharactersByUser(ctx, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get characters: %v", err)
	}

	var protoCharacters []*characterV1.Character
	for _, char := range characters {
		protoCharacters = append(protoCharacters, s.dbCharacterToProto(char))
	}

	return &characterV1.GetCharactersByUserResponse{
		Characters: protoCharacters,
	}, nil
}

// DeleteCharacter deletes a character
func (s *Service) DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error) {
	charUUID, err := parseUUID(req.CharacterId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID: %v", err)
	}

	err = db.New(s.db).DeleteCharacter(ctx, charUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete character: %v", err)
	}

	return &characterV1.DeleteCharacterResponse{
		Success: true,
	}, nil
}
