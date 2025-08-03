package handlers

import (
	"context"
	"os"

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
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "terrain-handler",
	})

	return &TerrainHandler{
		terrainService: terrainService,
		logger:         logger,
	}
}

// GetTerrainTypes returns all available terrain types
func (h *TerrainHandler) GetTerrainTypes(ctx context.Context, req *terrainV1.GetTerrainTypesRequest) (*terrainV1.GetTerrainTypesResponse, error) {
	h.logger.Debug("Getting all terrain types")

	terrainTypes, err := h.terrainService.GetTerrainTypes(ctx)
	if err != nil {
		h.logger.Error("Failed to get terrain types", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved terrain types", "count", len(terrainTypes))
	return &terrainV1.GetTerrainTypesResponse{
		TerrainTypes: terrainTypes,
	}, nil
}
