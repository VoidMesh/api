package character_actions

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/uuid"
	characterActionsV1 "github.com/VoidMesh/api/api/proto/character_actions/v1"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service provides character action operations.
type Service struct {
	db               DatabaseInterface
	inventoryService InventoryServiceInterface
	characterService CharacterServiceInterface
	logger           LoggerInterface
	rng              *rand.Rand
}

// NewService creates a new character actions service with dependency injection.
func NewService(
	db DatabaseInterface,
	inventoryService InventoryServiceInterface,
	characterService CharacterServiceInterface,
	logger LoggerInterface,
) *Service {
	componentLogger := logger.With("component", "character-actions-service")
	componentLogger.Debug("Creating new character actions service")
	return &Service{
		db:               db,
		inventoryService: inventoryService,
		characterService: characterService,
		logger:           componentLogger,
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// HarvestResource processes harvesting from a resource node
func (s *Service) HarvestResource(ctx context.Context, userID, characterID string, resourceNodeID int32) ([]*characterActionsV1.HarvestResult, *inventoryV1.InventoryItem, error) {
	s.logger.Debug("Harvesting resource node", "user_id", userID, "character_id", characterID, "resource_node_id", resourceNodeID)

	// Validate character ID format
	if !uuid.ValidateFormat(characterID) {
		s.logger.Warn("Invalid character ID format", "character_id", characterID)
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	// Get character information
	character, err := s.characterService.GetCharacterByID(ctx, characterID)
	if err != nil {
		s.logger.Error("Failed to get character", "character_id", characterID, "error", err)
		return nil, nil, status.Errorf(codes.NotFound, "character not found")
	}

	// Validate character ownership
	if err := s.validateCharacterOwnership(character, userID); err != nil {
		return nil, nil, err
	}

	// Get resource node information
	resourceNode, err := s.db.GetResourceNode(ctx, resourceNodeID)
	if err != nil {
		s.logger.Error("Failed to get resource node", "resource_node_id", resourceNodeID, "error", err)
		return nil, nil, status.Errorf(codes.NotFound, "resource node not found")
	}

	// Validate character is in range of resource node (basic distance check)
	if !s.isCharacterInRange(character, &resourceNode) {
		s.logger.Warn("Character out of range",
			"character_id", characterID,
			"character_pos", map[string]int32{"chunk_x": character.ChunkX, "chunk_y": character.ChunkY, "x": character.X, "y": character.Y},
			"resource_node_pos", map[string]int32{"chunk_x": resourceNode.ChunkX, "chunk_y": resourceNode.ChunkY, "x": resourceNode.X, "y": resourceNode.Y})
		return nil, nil, status.Errorf(codes.FailedPrecondition, "character is too far from resource node")
	}


	// Get all possible drops for this resource node type from database
	drops, err := s.db.GetResourceNodeDrops(ctx, resourceNode.ResourceNodeTypeID)
	if err != nil {
		s.logger.Error("Failed to get resource node drops", "resource_node_type_id", resourceNode.ResourceNodeTypeID, "error", err)
		return nil, nil, status.Errorf(codes.Internal, "failed to get drop information")
	}

	// Process all drops for this resource node
	var harvestResults []*characterActionsV1.HarvestResult
	var lastUpdatedItem *inventoryV1.InventoryItem

	for _, drop := range drops {
		// Convert pgtype.Numeric to float64
		chanceFloat, err := drop.Chance.Float64Value()
		if err != nil {
			s.logger.Warn("Failed to convert chance to float", "drop_id", drop.ID, "item_name", drop.ItemName, "error", err)
			continue
		}

		// Roll for this drop based on chance
		if s.rng.Float64() < chanceFloat.Float64 {
			// Calculate quantity within range
			quantityRange := drop.MaxQuantity - drop.MinQuantity + 1
			quantity := drop.MinQuantity + s.rng.Int31n(quantityRange)

			// Add to harvest results
			harvestResults = append(harvestResults, &characterActionsV1.HarvestResult{
				ItemName: drop.ItemName,
				Quantity: quantity,
				IsSecondaryDrop: chanceFloat.Float64 < 1.0, // Items with 100% chance are "primary"
			})

			// Add to inventory using the item_id from database
			updatedItem, err := s.inventoryService.AddInventoryItem(ctx, characterID, drop.ItemID, quantity)
			if err != nil {
				s.logger.Error("Failed to add harvested item to inventory", 
					"item_id", drop.ItemID, 
					"item_name", drop.ItemName, 
					"quantity", quantity, 
					"error", err)
				return nil, nil, status.Errorf(codes.Internal, "failed to add harvested item: %s", drop.ItemName)
			}
			lastUpdatedItem = updatedItem
		}
	}

	s.logger.Debug("Completed resource harvest",
		"character_id", characterID,
		"resource_node_id", resourceNodeID,
		"total_drops", len(harvestResults),
		"total_items_types", len(drops))

	return harvestResults, lastUpdatedItem, nil
}

// isCharacterInRange checks if character is within harvesting range of the resource node
func (s *Service) isCharacterInRange(character *db.Character, resourceNode *db.ResourceNode) bool {
	// Calculate distance using global coordinates (much simpler!)
	dx := float64(character.X - resourceNode.X)
	dy := float64(character.Y - resourceNode.Y)
	distance := math.Sqrt(dx*dx + dy*dy)

	// Allow harvesting within 3 units (adjust as needed for game balance)
	const maxHarvestDistance = 3.0
	return distance <= maxHarvestDistance
}

// validateCharacterOwnership checks if the character belongs to the specified user
func (s *Service) validateCharacterOwnership(character *db.Character, userID string) error {
	characterUserID := uuid.PgtypeToString(character.UserID)
	if !uuid.Compare(characterUserID, userID) {
		s.logger.Warn("Character ownership validation failed", 
			"character_id", character.ID.String(), 
			"character_user_id", characterUserID, 
			"requesting_user_id", userID)
		return status.Errorf(codes.PermissionDenied, "character not owned by user")
	}
	return nil
}