package character

import (
	"context"
	"fmt"
	"time"

	"github.com/VoidMesh/platform/api/db"
	characterV1 "github.com/VoidMesh/platform/api/proto/character/v1"
	chunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// movementCache stores last movement times for rate limiting
var movementCache = make(map[string]time.Time)

const (
	MovementCooldown = 50 * time.Millisecond // 50ms between moves for smoother gameplay
	MaxMoveDistance  = 1                     // Max 1 cell per move
)

// MoveCharacter handles character movement with anti-cheat validation
func (s *Service) MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error) {
	// Get character first
	charUUID, err := parseUUID(req.CharacterId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID: %v", err)
	}

	character, err := db.New(s.db).GetCharacterById(ctx, charUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "character not found: %v", err)
	}

	// Anti-cheat validation
	if !s.validateMovement(character, req.NewX, req.NewY) {
		return &characterV1.MoveCharacterResponse{
			Success:      false,
			ErrorMessage: "Invalid movement: too far or too fast",
		}, nil
	}

	// Check rate limiting
	characterID := req.CharacterId
	lastMove, exists := movementCache[characterID]
	if exists && time.Since(lastMove) < MovementCooldown {
		return &characterV1.MoveCharacterResponse{
			Success:      false,
			ErrorMessage: "Movement too fast, please wait",
		}, nil
	}

	// Validate destination terrain
	valid, err := s.isValidMovePosition(ctx, req.NewX, req.NewY)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate position: %v", err)
	}
	if !valid {
		return &characterV1.MoveCharacterResponse{
			Success:      false,
			ErrorMessage: "Cannot move to that position (water or stone)",
		}, nil
	}

	// Calculate new chunk coordinates
	newChunkX, newChunkY := s.worldToChunkCoords(req.NewX, req.NewY)

	// Update character position
	updatedCharacter, err := db.New(s.db).UpdateCharacterPosition(ctx, db.UpdateCharacterPositionParams{
		ID:     charUUID,
		X:      req.NewX,
		Y:      req.NewY,
		ChunkX: newChunkX,
		ChunkY: newChunkY,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update character position: %v", err)
	}

	// Update movement cache
	movementCache[characterID] = time.Now()

	return &characterV1.MoveCharacterResponse{
		Character: s.dbCharacterToProto(updatedCharacter),
		Success:   true,
	}, nil
}

// validateMovement checks if the movement is valid (distance and speed)
func (s *Service) validateMovement(character db.Character, newX, newY int32) bool {
	// Calculate distance
	deltaX := abs32(newX - character.X)
	deltaY := abs32(newY - character.Y)

	// Check maximum distance (Manhattan distance)
	distance := deltaX + deltaY
	if distance > MaxMoveDistance {
		return false
	}

	// Only allow orthogonal movement (no diagonal)
	if deltaX > 0 && deltaY > 0 {
		return false
	}

	return true
}

// isValidMovePosition checks if a position is valid for movement
func (s *Service) isValidMovePosition(ctx context.Context, x, y int32) (bool, error) {
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

	// Validate coordinates are within chunk bounds
	if localX < 0 || localX >= s.chunkSize || localY < 0 || localY >= s.chunkSize {
		return false, fmt.Errorf("coordinates out of chunk bounds")
	}

	// Get the terrain cell (row-major order)
	index := localY*s.chunkSize + localX
	if index < 0 || index >= int32(len(chunkData.Cells)) {
		return false, fmt.Errorf("invalid cell index")
	}

	cell := chunkData.Cells[index]

	// Check if terrain is walkable
	switch cell.TerrainType {
	case chunkV1.TerrainType_TERRAIN_TYPE_WATER, chunkV1.TerrainType_TERRAIN_TYPE_STONE:
		return false, nil // Not walkable
	case chunkV1.TerrainType_TERRAIN_TYPE_GRASS,
		chunkV1.TerrainType_TERRAIN_TYPE_SAND,
		chunkV1.TerrainType_TERRAIN_TYPE_DIRT:
		return true, nil // Walkable
	default:
		return false, nil // Unknown terrain type, assume not walkable
	}
}
