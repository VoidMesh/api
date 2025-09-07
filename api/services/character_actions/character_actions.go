package character_actions

import (
	"context"
	"encoding/hex"
	"math"
	"math/rand"
	"time"

	"github.com/VoidMesh/api/api/db"
	characterActionsV1 "github.com/VoidMesh/api/api/proto/character_actions/v1"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service provides character action operations.
type Service struct {
	db                  DatabaseInterface
	inventoryService    InventoryServiceInterface
	characterService    CharacterServiceInterface
	resourceNodeService ResourceNodeServiceInterface
	logger              LoggerInterface
	rng                 *rand.Rand
}

// NewService creates a new character actions service with dependency injection.
func NewService(
	db DatabaseInterface,
	inventoryService InventoryServiceInterface,
	characterService CharacterServiceInterface,
	resourceNodeService ResourceNodeServiceInterface,
	logger LoggerInterface,
) *Service {
	componentLogger := logger.With("component", "character-actions-service")
	componentLogger.Debug("Creating new character actions service")
	return &Service{
		db:                  db,
		inventoryService:    inventoryService,
		characterService:    characterService,
		resourceNodeService: resourceNodeService,
		logger:              componentLogger,
		rng:                 rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// HarvestResource processes harvesting from a resource node
func (s *Service) HarvestResource(ctx context.Context, userID, characterID string, resourceNodeID int32) ([]*characterActionsV1.HarvestResult, *inventoryV1.InventoryItem, error) {
	s.logger.Debug("Harvesting resource node", "user_id", userID, "character_id", characterID, "resource_node_id", resourceNodeID)

	// Validate character ID format
	characterUUID, err := hex.DecodeString(characterID)
	if err != nil {
		s.logger.Warn("Invalid character ID format", "character_id", characterID, "error", err)
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}
	if len(characterUUID) != 16 {
		s.logger.Warn("Character ID must be 32 hex characters", "character_id", characterID, "length", len(characterID))
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid character ID length")
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
			"resource_node_pos", map[string]int32{"chunk_x": resourceNode.ChunkX, "chunk_y": resourceNode.ChunkY, "x": resourceNode.PosX, "y": resourceNode.PosY})
		return nil, nil, status.Errorf(codes.FailedPrecondition, "character is too far from resource node")
	}

	// Get resource node type information
	resourceTypes, err := s.resourceNodeService.GetResourceNodeTypes(ctx)
	if err != nil {
		s.logger.Error("Failed to get resource node types", "error", err)
		return nil, nil, status.Errorf(codes.Internal, "failed to get resource information")
	}

	var resourceType *resourceNodeV1.ResourceNodeType
	for _, rt := range resourceTypes {
		if rt.Id == resourceNode.ResourceNodeTypeID {
			resourceType = rt
			break
		}
	}

	if resourceType == nil {
		s.logger.Error("Resource node type not found", "resource_node_type_id", resourceNode.ResourceNodeTypeID)
		return nil, nil, status.Errorf(codes.Internal, "resource node type not found")
	}

	// Calculate harvest results
	var harvestResults []*characterActionsV1.HarvestResult

	// Primary yield
	yieldRange := resourceType.Properties.YieldMax - resourceType.Properties.YieldMin + 1
	primaryYield := resourceType.Properties.YieldMin + s.rng.Int31n(yieldRange)

	harvestResults = append(harvestResults, &characterActionsV1.HarvestResult{
		ItemName:        resourceType.Name,
		Quantity:        primaryYield,
		IsSecondaryDrop: false,
	})

	// Add primary resources to inventory
	updatedItem, err := s.inventoryService.AddInventoryItem(ctx, characterID, resourceNodeV1.ResourceNodeTypeId(resourceType.Id), primaryYield)
	if err != nil {
		s.logger.Error("Failed to add primary harvest to inventory", "error", err)
		return nil, nil, status.Errorf(codes.Internal, "failed to add harvested resources")
	}

	// Process secondary drops
	for _, secondaryDrop := range resourceType.Properties.SecondaryDrops {
		if s.rng.Float32() < secondaryDrop.Chance {
			dropRange := secondaryDrop.MaxAmount - secondaryDrop.MinAmount + 1
			dropAmount := secondaryDrop.MinAmount + s.rng.Int31n(dropRange)

			harvestResults = append(harvestResults, &characterActionsV1.HarvestResult{
				ItemName:        secondaryDrop.Name,
				Quantity:        dropAmount,
				IsSecondaryDrop: true,
			})

			// TODO: Map secondary drop names to resource node type IDs
			// For now, we're not adding secondary drops to inventory
			// This would require a mapping system or separate item types
		}
	}

	s.logger.Debug("Completed resource harvest",
		"character_id", characterID,
		"resource_node_id", resourceNodeID,
		"primary_yield", primaryYield,
		"total_results", len(harvestResults))

	return harvestResults, updatedItem, nil
}

// isCharacterInRange checks if character is within harvesting range of the resource node
func (s *Service) isCharacterInRange(character *db.Character, resourceNode *db.ResourceNode) bool {
	// Check if in same chunk first
	if character.ChunkX != resourceNode.ChunkX || character.ChunkY != resourceNode.ChunkY {
		// For now, only allow harvesting within the same chunk
		// In the future, we could allow harvesting across chunk boundaries
		return false
	}

	// Calculate distance within chunk
	dx := float64(character.X - resourceNode.PosX)
	dy := float64(character.Y - resourceNode.PosY)
	distance := math.Sqrt(dx*dx + dy*dy)

	// Allow harvesting within 3 units (adjust as needed for game balance)
	const maxHarvestDistance = 3.0
	return distance <= maxHarvestDistance
}

// validateCharacterOwnership checks if the character belongs to the specified user
func (s *Service) validateCharacterOwnership(character *db.Character, userID string) error {
	characterUserID := character.UserID.String()
	if characterUserID != userID {
		s.logger.Warn("Character ownership validation failed", 
			"character_id", character.ID.String(), 
			"character_user_id", characterUserID, 
			"requesting_user_id", userID)
		return status.Errorf(codes.PermissionDenied, "character not owned by user")
	}
	return nil
}