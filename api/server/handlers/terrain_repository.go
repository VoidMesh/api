package handlers

import (
	"context"

	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/VoidMesh/api/api/services/terrain"
)

// terrainServiceWrapper implements the TerrainService interface using the real terrain service
type terrainServiceWrapper struct {
	service *terrain.Service
}

// NewTerrainService creates a new TerrainService implementation
func NewTerrainService(service *terrain.Service) TerrainService {
	return &terrainServiceWrapper{
		service: service,
	}
}

// NewTerrainServiceWithDefaultLogger creates a terrain service with all dependencies wired up
// This function creates the necessary services and dependencies
func NewTerrainServiceWithDefaultLogger() TerrainService {
	terrainService := terrain.NewServiceWithDefaultLogger()
	return NewTerrainService(terrainService)
}

// GetTerrainTypes returns all available terrain types with their properties
func (w *terrainServiceWrapper) GetTerrainTypes(ctx context.Context) ([]*terrainV1.TerrainTypeInfo, error) {
	return w.service.GetTerrainTypes(ctx)
}