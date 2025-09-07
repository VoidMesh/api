package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	characterActionsV1 "github.com/VoidMesh/api/api/proto/character_actions/v1"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	"github.com/VoidMesh/api/api/server/middleware"
	"github.com/VoidMesh/api/api/services/character_actions"
	"github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type characterActionsServiceServer struct {
	characterActionsV1.UnimplementedCharacterActionsServiceServer
	characterActionsService CharacterActionsService
	logger                  *log.Logger
}

// CharacterActionsService defines the interface for character actions service
type CharacterActionsService interface {
	HarvestResource(ctx context.Context, userID, characterID string, resourceNodeID int32) ([]*characterActionsV1.HarvestResult, *inventoryV1.InventoryItem, error)
}

// CharacterActionsServiceAdapter adapts the character actions service to the handler interface
type CharacterActionsServiceAdapter struct {
	service *character_actions.Service
}

func NewCharacterActionsServiceAdapter(service *character_actions.Service) *CharacterActionsServiceAdapter {
	return &CharacterActionsServiceAdapter{service: service}
}

func (a *CharacterActionsServiceAdapter) HarvestResource(ctx context.Context, userID, characterID string, resourceNodeID int32) ([]*characterActionsV1.HarvestResult, *inventoryV1.InventoryItem, error) {
	results, updatedItem, err := a.service.HarvestResource(ctx, userID, characterID, resourceNodeID)
	if err != nil {
		return nil, nil, err
	}

	return results, updatedItem, nil
}

func NewCharacterActionsServer(
	characterActionsService CharacterActionsService,
) characterActionsV1.CharacterActionsServiceServer {
	logger := logging.WithComponent("character-actions-handler")
	logger.Debug("Creating new CharacterActionsService server instance")
	return &characterActionsServiceServer{
		characterActionsService: characterActionsService,
		logger:                  logger,
	}
}

// NewCharacterActionsServiceWithPool creates a character actions service with all dependencies
func NewCharacterActionsServiceWithPool(
	characterService *character_actions.Service,
) *CharacterActionsServiceAdapter {
	return NewCharacterActionsServiceAdapter(characterService)
}

// HarvestResource handles resource harvesting requests
func (s *characterActionsServiceServer) HarvestResource(ctx context.Context, req *characterActionsV1.HarvestResourceRequest) (*characterActionsV1.HarvestResourceResponse, error) {
	// Extract user ID from JWT context for authorization
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		s.logger.Warn("Failed to get user ID from context")
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	s.logger.Debug("Processing harvest resource request",
		"user_id", userID,
		"character_id", req.CharacterId,
		"resource_node_id", req.ResourceNodeId)

	// Validate request
	if req.CharacterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "character_id is required")
	}

	if req.ResourceNodeId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "resource_node_id must be positive")
	}

	// Call the character actions service
	harvestResults, updatedItem, err := s.characterActionsService.HarvestResource(ctx, userID, req.CharacterId, req.ResourceNodeId)
	if err != nil {
		s.logger.Error("Failed to harvest resource",
			"user_id", userID,
			"character_id", req.CharacterId,
			"resource_node_id", req.ResourceNodeId,
			"error", err)
		return nil, err // Let the service layer handle error codes
	}

	s.logger.Debug("Successfully harvested resource",
		"user_id", userID,
		"character_id", req.CharacterId,
		"resource_node_id", req.ResourceNodeId,
		"results_count", len(harvestResults))

	return &characterActionsV1.HarvestResourceResponse{
		Success:     true,
		Results:     harvestResults,
		UpdatedItem: updatedItem,
	}, nil
}