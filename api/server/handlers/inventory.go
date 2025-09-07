package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	"github.com/VoidMesh/api/api/server/middleware"
	"github.com/VoidMesh/api/api/services/inventory"
	"github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type inventoryServiceServer struct {
	inventoryV1.UnimplementedInventoryServiceServer
	inventoryService *inventory.Service
	logger           *log.Logger
}

func NewInventoryHandler(inventoryService *inventory.Service) inventoryV1.InventoryServiceServer {
	logger := logging.WithComponent("inventory-handler")
	logger.Debug("Creating new InventoryService server instance")
	return &inventoryServiceServer{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// GetCharacterInventory retrieves all inventory items for a character
func (s *inventoryServiceServer) GetCharacterInventory(ctx context.Context, req *inventoryV1.GetCharacterInventoryRequest) (*inventoryV1.GetCharacterInventoryResponse, error) {
	// Extract user ID from JWT context for authorization
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		s.logger.Warn("Failed to get user ID from context")
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	s.logger.Debug("Processing get character inventory request",
		"user_id", userID,
		"character_id", req.CharacterId)

	// Validate request
	if req.CharacterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "character_id is required")
	}

	// Call the inventory service
	items, err := s.inventoryService.GetCharacterInventory(ctx, req.CharacterId)
	if err != nil {
		s.logger.Error("Failed to get character inventory",
			"user_id", userID,
			"character_id", req.CharacterId,
			"error", err)
		return nil, err // Let the service layer handle error codes
	}

	s.logger.Debug("Successfully retrieved character inventory",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_count", len(items))

	return &inventoryV1.GetCharacterInventoryResponse{
		Items:      items,
		TotalItems: int32(len(items)),
	}, nil
}

// AddInventoryItem adds an item to character's inventory
func (s *inventoryServiceServer) AddInventoryItem(ctx context.Context, req *inventoryV1.AddInventoryItemRequest) (*inventoryV1.AddInventoryItemResponse, error) {
	// Extract user ID from JWT context for authorization
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		s.logger.Warn("Failed to get user ID from context")
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	s.logger.Debug("Processing add inventory item request",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_id", req.ItemId,
		"quantity", req.Quantity)

	// Validate request
	if req.CharacterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "character_id is required")
	}
	if req.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "quantity must be positive")
	}

	// Call the inventory service
	item, err := s.inventoryService.AddInventoryItem(ctx, req.CharacterId, req.ItemId, req.Quantity)
	if err != nil {
		s.logger.Error("Failed to add inventory item",
			"user_id", userID,
			"character_id", req.CharacterId,
			"item_id", req.ItemId,
			"quantity", req.Quantity,
			"error", err)
		return nil, err // Let the service layer handle error codes
	}

	s.logger.Debug("Successfully added inventory item",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_id", req.ItemId,
		"quantity", req.Quantity)

	return &inventoryV1.AddInventoryItemResponse{
		Item:    item,
		Success: true,
	}, nil
}

// RemoveInventoryItem removes quantity from an inventory item
func (s *inventoryServiceServer) RemoveInventoryItem(ctx context.Context, req *inventoryV1.RemoveInventoryItemRequest) (*inventoryV1.RemoveInventoryItemResponse, error) {
	// Extract user ID from JWT context for authorization
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		s.logger.Warn("Failed to get user ID from context")
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	s.logger.Debug("Processing remove inventory item request",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_id", req.ItemId,
		"quantity", req.Quantity)

	// Validate request
	if req.CharacterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "character_id is required")
	}
	if req.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "quantity must be positive")
	}

	// Call the inventory service
	item, err := s.inventoryService.RemoveInventoryItem(ctx, req.CharacterId, req.ItemId, req.Quantity)
	if err != nil {
		s.logger.Error("Failed to remove inventory item",
			"user_id", userID,
			"character_id", req.CharacterId,
			"item_id", req.ItemId,
			"quantity", req.Quantity,
			"error", err)
		return nil, err // Let the service layer handle error codes
	}

	s.logger.Debug("Successfully removed inventory item",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_id", req.ItemId,
		"quantity", req.Quantity)

	return &inventoryV1.RemoveInventoryItemResponse{
		Item:    item, // May be nil if item was completely removed
		Success: true,
	}, nil
}

// UpdateItemQuantity updates the quantity of an inventory item
func (s *inventoryServiceServer) UpdateItemQuantity(ctx context.Context, req *inventoryV1.UpdateItemQuantityRequest) (*inventoryV1.UpdateItemQuantityResponse, error) {
	// Extract user ID from JWT context for authorization
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		s.logger.Warn("Failed to get user ID from context")
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	s.logger.Debug("Processing update item quantity request",
		"user_id", userID,
		"character_id", req.CharacterId,
		"item_id", req.ItemId,
		"new_quantity", req.NewQuantity)

	// Validate request
	if req.CharacterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "character_id is required")
	}
	if req.NewQuantity < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "quantity cannot be negative")
	}

	// For now, we don't have UpdateItemQuantity in the service, so we'll implement it using Add/Remove logic
	// TODO: Add UpdateItemQuantity method to inventory service if needed
	return nil, status.Errorf(codes.Unimplemented, "update item quantity not yet implemented")
}