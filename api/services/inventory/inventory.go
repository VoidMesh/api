package inventory

import (
	"context"
	"encoding/hex"
	"math/rand"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/logging"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	db                  *pgxpool.Pool
	characterService    *character.Service
	resourceNodeService *resource_node.NodeService
	logger              *log.Logger
}

func NewService(db *pgxpool.Pool, characterService *character.Service, resourceNodeService *resource_node.NodeService) *Service {
	logger := logging.GetLogger()
	logger.Debug("Creating new inventory service")
	return &Service{
		db:                  db,
		characterService:    characterService,
		resourceNodeService: resourceNodeService,
		logger:              logger,
	}
}

// Helper function to convert DB inventory item to proto
func (s *Service) dbInventoryItemToProto(ctx context.Context, item db.CharacterInventory, includeResourceType bool) (*inventoryV1.InventoryItem, error) {
	protoItem := &inventoryV1.InventoryItem{
		Id:                  item.ID,
		CharacterId:         hex.EncodeToString(item.CharacterID.Bytes[:]),
		ResourceNodeTypeId:  resourceNodeV1.ResourceNodeTypeId(item.ResourceNodeTypeID),
		Quantity:           item.Quantity,
	}

	if item.CreatedAt.Valid {
		protoItem.CreatedAt = timestamppb.New(item.CreatedAt.Time)
	}
	if item.UpdatedAt.Valid {
		protoItem.UpdatedAt = timestamppb.New(item.UpdatedAt.Time)
	}

	// Optionally populate resource node type information
	if includeResourceType {
		resourceTypes, err := s.resourceNodeService.GetResourceNodeTypes(ctx)
		if err != nil {
			s.logger.Warn("Failed to get resource node types", "error", err)
		} else {
			for _, resourceType := range resourceTypes {
				if resourceType.Id == int32(item.ResourceNodeTypeID) {
					protoItem.ResourceNodeType = resourceType
					break
				}
			}
		}
	}

	return protoItem, nil
}

// GetCharacterInventory retrieves all inventory items for a character
func (s *Service) GetCharacterInventory(ctx context.Context, characterID string) ([]*inventoryV1.InventoryItem, error) {
	s.logger.Debug("Getting character inventory", "character_id", characterID)

	// Parse character ID
	characterUUID, err := hex.DecodeString(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	var characterUUIDBytes [16]byte
	copy(characterUUIDBytes[:], characterUUID)

	queries := db.New(s.db)
	dbItems, err := queries.GetCharacterInventory(ctx, pgtype.UUID{Bytes: characterUUIDBytes, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get character inventory", "character_id", characterID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve inventory")
	}

	var items []*inventoryV1.InventoryItem
	for _, dbItem := range dbItems {
		protoItem, err := s.dbInventoryItemToProto(ctx, dbItem, true)
		if err != nil {
			s.logger.Warn("Failed to convert inventory item to proto", "item_id", dbItem.ID, "error", err)
			continue
		}
		items = append(items, protoItem)
	}

	s.logger.Debug("Retrieved character inventory", "character_id", characterID, "item_count", len(items))
	return items, nil
}

// AddInventoryItem adds or updates an inventory item
func (s *Service) AddInventoryItem(ctx context.Context, characterID string, resourceNodeTypeID resourceNodeV1.ResourceNodeTypeId, quantity int32) (*inventoryV1.InventoryItem, error) {
	s.logger.Debug("Adding inventory item", "character_id", characterID, "resource_type", resourceNodeTypeID, "quantity", quantity)

	if quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "quantity must be positive")
	}

	// Parse character ID
	characterUUID, err := hex.DecodeString(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	var characterUUIDBytes [16]byte
	copy(characterUUIDBytes[:], characterUUID)

	queries := db.New(s.db)

	// Check if item already exists
	exists, err := queries.InventoryItemExists(ctx, db.InventoryItemExistsParams{
		CharacterID:         pgtype.UUID{Bytes: characterUUIDBytes, Valid: true},
		ResourceNodeTypeID: int32(resourceNodeTypeID),
	})
	if err != nil {
		s.logger.Error("Failed to check inventory item existence", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to check inventory")
	}

	var dbItem db.CharacterInventory
	if exists {
		// Update existing item
		dbItem, err = queries.AddInventoryItemQuantity(ctx, db.AddInventoryItemQuantityParams{
			CharacterID:         pgtype.UUID{Bytes: characterUUIDBytes, Valid: true},
			ResourceNodeTypeID: int32(resourceNodeTypeID),
			Quantity:           quantity,
		})
	} else {
		// Create new item
		dbItem, err = queries.CreateInventoryItem(ctx, db.CreateInventoryItemParams{
			CharacterID:         pgtype.UUID{Bytes: characterUUIDBytes, Valid: true},
			ResourceNodeTypeID: int32(resourceNodeTypeID),
			Quantity:           quantity,
		})
	}

	if err != nil {
		s.logger.Error("Failed to add inventory item", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to add inventory item")
	}

	protoItem, err := s.dbInventoryItemToProto(ctx, dbItem, true)
	if err != nil {
		s.logger.Error("Failed to convert inventory item to proto", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to process inventory item")
	}

	s.logger.Debug("Added inventory item", "character_id", characterID, "resource_type", resourceNodeTypeID, "new_quantity", dbItem.Quantity)
	return protoItem, nil
}

// RemoveInventoryItem removes quantity from an inventory item
func (s *Service) RemoveInventoryItem(ctx context.Context, characterID string, resourceNodeTypeID resourceNodeV1.ResourceNodeTypeId, quantity int32) (*inventoryV1.InventoryItem, error) {
	s.logger.Debug("Removing inventory item", "character_id", characterID, "resource_type", resourceNodeTypeID, "quantity", quantity)

	if quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "quantity must be positive")
	}

	// Parse character ID
	characterUUID, err := hex.DecodeString(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	var characterUUIDBytes [16]byte
	copy(characterUUIDBytes[:], characterUUID)

	queries := db.New(s.db)

	// Try to remove quantity
	dbItem, err := queries.RemoveInventoryItemQuantity(ctx, db.RemoveInventoryItemQuantityParams{
		CharacterID:         pgtype.UUID{Bytes: characterUUIDBytes, Valid: true},
		ResourceNodeTypeID: int32(resourceNodeTypeID),
		Quantity:           quantity,
	})

	if err != nil {
		s.logger.Error("Failed to remove inventory item quantity", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to remove inventory item or insufficient quantity")
	}

	// If quantity is now 0 or less, delete the item
	if dbItem.Quantity <= 0 {
		err = queries.DeleteInventoryItem(ctx, db.DeleteInventoryItemParams{
			CharacterID:         pgtype.UUID{Bytes: characterUUIDBytes, Valid: true},
			ResourceNodeTypeID: int32(resourceNodeTypeID),
		})
		if err != nil {
			s.logger.Warn("Failed to delete empty inventory item", "error", err)
		}
		return nil, nil // Item was completely removed
	}

	protoItem, err := s.dbInventoryItemToProto(ctx, dbItem, true)
	if err != nil {
		s.logger.Error("Failed to convert inventory item to proto", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to process inventory item")
	}

	s.logger.Debug("Removed inventory item quantity", "character_id", characterID, "resource_type", resourceNodeTypeID, "remaining_quantity", dbItem.Quantity)
	return protoItem, nil
}

// HarvestResourceNode processes harvesting from a resource node
func (s *Service) HarvestResourceNode(ctx context.Context, characterID string, resourceNodeID int32) ([]*inventoryV1.HarvestResult, *inventoryV1.InventoryItem, error) {
	s.logger.Debug("Harvesting resource node", "character_id", characterID, "resource_node_id", resourceNodeID)

	// TODO: Add validation:
	// 1. Check if character exists and is owned by the requesting user
	// 2. Check if resource node exists and is within interaction range
	// 3. Check harvest cooldowns/timers
	// 4. Validate character position vs resource node position

	queries := db.New(s.db)
	
	// Get resource node information
	resourceNode, err := queries.GetResourceNode(ctx, resourceNodeID)
	if err != nil {
		s.logger.Error("Failed to get resource node", "resource_node_id", resourceNodeID, "error", err)
		return nil, nil, status.Errorf(codes.NotFound, "resource node not found")
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
		return nil, nil, status.Errorf(codes.Internal, "resource node type not found")
	}

	// Calculate harvest results
	var harvestResults []*inventoryV1.HarvestResult
	
	// Primary yield
	yieldRange := resourceType.Properties.YieldMax - resourceType.Properties.YieldMin + 1
	primaryYield := resourceType.Properties.YieldMin + rand.Int31n(yieldRange)
	
	harvestResults = append(harvestResults, &inventoryV1.HarvestResult{
		ItemName:        resourceType.Name,
		Quantity:        primaryYield,
		IsSecondaryDrop: false,
	})

	// Add primary resources to inventory
	updatedItem, err := s.AddInventoryItem(ctx, characterID, resourceNodeV1.ResourceNodeTypeId(resourceType.Id), primaryYield)
	if err != nil {
		s.logger.Error("Failed to add primary harvest to inventory", "error", err)
		return nil, nil, status.Errorf(codes.Internal, "failed to add harvested resources")
	}

	// Process secondary drops
	for _, secondaryDrop := range resourceType.Properties.SecondaryDrops {
		if rand.Float32() < secondaryDrop.Chance {
			dropRange := secondaryDrop.MaxAmount - secondaryDrop.MinAmount + 1
			dropAmount := secondaryDrop.MinAmount + rand.Int31n(dropRange)
			
			harvestResults = append(harvestResults, &inventoryV1.HarvestResult{
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