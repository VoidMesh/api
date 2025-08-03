package terrain

import (
	"context"
	"os"

	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/charmbracelet/log"
)

// Service provides terrain information and operations
type Service struct {
	logger *log.Logger
}

// NewService creates a new terrain service
func NewService() *Service {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "terrain-service",
	})

	return &Service{
		logger: logger,
	}
}

// GetTerrainTypes returns all available terrain types with their properties
func (s *Service) GetTerrainTypes(ctx context.Context) ([]*terrainV1.TerrainTypeInfo, error) {
	s.logger.Debug("Getting all terrain types")

	// Create static terrain type definitions
	terrainTypes := []*terrainV1.TerrainTypeInfo{
		{
			Type:        terrainV1.TerrainType_TERRAIN_TYPE_GRASS,
			Name:        "Grass",
			Description: "Grassy plains with lush vegetation",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#7CFC00",
				Texture:   "grass_texture",
				Roughness: 0.4,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 1.0,
				IsWater:                 false,
				IsPassable:              true,
			},
		},
		{
			Type:        terrainV1.TerrainType_TERRAIN_TYPE_WATER,
			Name:        "Water",
			Description: "Bodies of water, including lakes, rivers, and oceans",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#1E90FF",
				Texture:   "water_texture",
				Roughness: 0.1,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 0.5,
				IsWater:                 true,
				IsPassable:              true,
			},
		},
		{
			Type:        terrainV1.TerrainType_TERRAIN_TYPE_STONE,
			Name:        "Stone",
			Description: "Rocky terrain with little vegetation",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#708090",
				Texture:   "stone_texture",
				Roughness: 0.8,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 0.8,
				IsWater:                 false,
				IsPassable:              true,
			},
		},
		{
			Type:        terrainV1.TerrainType_TERRAIN_TYPE_SAND,
			Name:        "Sand",
			Description: "Sandy desert or beach areas",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#FFD700",
				Texture:   "sand_texture",
				Roughness: 0.6,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 0.7,
				IsWater:                 false,
				IsPassable:              true,
			},
		},
		{
			Type:        terrainV1.TerrainType_TERRAIN_TYPE_DIRT,
			Name:        "Dirt",
			Description: "Bare earth or dirt paths",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#8B4513",
				Texture:   "dirt_texture",
				Roughness: 0.5,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 0.9,
				IsWater:                 false,
				IsPassable:              true,
			},
		},
	}

	return terrainTypes, nil
}
