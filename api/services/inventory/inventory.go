package inventory

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/uuid"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service provides inventory management operations.
type Service struct {
	db                  DatabaseInterface
	characterService    CharacterServiceInterface
	resourceNodeService ResourceNodeServiceInterface
	logger              LoggerInterface
}

// NewService creates a new inventory service with dependency injection.
func NewService(
	db DatabaseInterface,
	characterService CharacterServiceInterface,
	resourceNodeService ResourceNodeServiceInterface,
	logger LoggerInterface,
) *Service {
	componentLogger := logger.With("component", "inventory-service")
	componentLogger.Debug("Creating new inventory service")
	return &Service{
		db:                  db,
		characterService:    characterService,
		resourceNodeService: resourceNodeService,
		logger:              componentLogger,
	}
}

// NewServiceWithPool creates a service with concrete implementations (convenience constructor for production use).
func NewServiceWithPool(
	pool *pgxpool.Pool,
	characterService *character.Service,
	resourceNodeService *resource_node.NodeService,
) *Service {
	logger := NewDefaultLoggerWrapper()
	return NewService(
		NewDatabaseWrapper(pool),
		NewCharacterServiceAdapter(characterService),
		NewResourceNodeServiceAdapter(resourceNodeService),
		logger,
	)
}

// Helper function to convert DB inventory item to proto
func (s *Service) dbInventoryItemToProto(ctx context.Context, item db.CharacterInventory, includeResourceType bool) (*inventoryV1.InventoryItem, error) {
	protoItem := &inventoryV1.InventoryItem{
		Id:                 item.ID,
		CharacterId:        uuid.PgtypeToNormalizedString(item.CharacterID),
		ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId(item.ResourceNodeTypeID),
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
	if !uuid.ValidateFormat(characterID) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	characterPgUUID, err := uuid.StringToPgtype(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	dbItems, err := s.db.GetCharacterInventory(ctx, characterPgUUID)
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
	if !uuid.ValidateFormat(characterID) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	characterPgUUID, err := uuid.StringToPgtype(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	// Check if item already exists
	exists, err := s.db.InventoryItemExists(ctx, db.InventoryItemExistsParams{
		CharacterID:        characterPgUUID,
		ResourceNodeTypeID: int32(resourceNodeTypeID),
	})
	if err != nil {
		s.logger.Error("Failed to check inventory item existence", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to check inventory")
	}

	var dbItem db.CharacterInventory
	if exists {
		// Update existing item
		dbItem, err = s.db.AddInventoryItemQuantity(ctx, db.AddInventoryItemQuantityParams{
			CharacterID:        characterPgUUID,
			ResourceNodeTypeID: int32(resourceNodeTypeID),
			Quantity:           quantity,
		})
	} else {
		// Create new item
		dbItem, err = s.db.CreateInventoryItem(ctx, db.CreateInventoryItemParams{
			CharacterID:        characterPgUUID,
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
	if !uuid.ValidateFormat(characterID) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	characterPgUUID, err := uuid.StringToPgtype(characterID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid character ID format")
	}

	// Try to remove quantity
	dbItem, err := s.db.RemoveInventoryItemQuantity(ctx, db.RemoveInventoryItemQuantityParams{
		CharacterID:        characterPgUUID,
		ResourceNodeTypeID: int32(resourceNodeTypeID),
		Quantity:           quantity,
	})
	if err != nil {
		s.logger.Error("Failed to remove inventory item quantity", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to remove inventory item or insufficient quantity")
	}

	// If quantity is now 0 or less, delete the item
	if dbItem.Quantity <= 0 {
		err = s.db.DeleteInventoryItem(ctx, db.DeleteInventoryItemParams{
			CharacterID:        characterPgUUID,
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

