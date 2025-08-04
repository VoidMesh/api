package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/VoidMesh/api/api/services/terrain"
	"github.com/charmbracelet/log"
)

// TerrainHandler implements the TerrainService gRPC service
type TerrainHandler struct {
	terrainV1.UnimplementedTerrainServiceServer
	terrainService *terrain.Service
	logger         *log.Logger
}

// NewTerrainHandler creates a new terrain handler
func NewTerrainHandler(terrainService *terrain.Service) *TerrainHandler {
	return &TerrainHandler{
		terrainService: terrainService,
		logger:         logging.WithComponent("terrain-handler"),
	}
}

// GetTerrainTypes returns all available terrain types
func (h *TerrainHandler) GetTerrainTypes(ctx context.Context, req *terrainV1.GetTerrainTypesRequest) (*terrainV1.GetTerrainTypesResponse, error) {
	logger := h.logger.With("operation", "GetTerrainTypes")
	logger.Debug("Received GetTerrainTypes request")

	terrainTypes, err := h.terrainService.GetTerrainTypes(ctx)
	if err != nil {
		logger.Error("Failed to get terrain types", "error", err)
		return nil, err
	}

	logger.Info("Retrieved terrain types", "count", len(terrainTypes))
	return &terrainV1.GetTerrainTypesResponse{
		TerrainTypes: terrainTypes,
	}, nil
}
