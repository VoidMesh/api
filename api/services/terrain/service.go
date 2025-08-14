package terrain

import (
	"context"

	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/VoidMesh/api/api/internal/logging"
)

// LoggerInterface abstracts logging operations for dependency injection.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) LoggerInterface
}

// DefaultLoggerWrapper wraps the internal logging package.
type DefaultLoggerWrapper struct{}

// NewDefaultLoggerWrapper creates a new default logger wrapper.
func NewDefaultLoggerWrapper() LoggerInterface {
	return &DefaultLoggerWrapper{}
}

func (l *DefaultLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Debug(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Info(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Warn(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	logger := logging.GetLogger()
	logger.Error(msg, keysAndValues...)
}

func (l *DefaultLoggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	// For now, return self for simplicity
	return l
}

// Service provides terrain information and operations
type Service struct {
	logger LoggerInterface
}

// NewService creates a new terrain service with dependency injection.
func NewService(logger LoggerInterface) *Service {
	componentLogger := logger.With("component", "terrain-service")
	componentLogger.Debug("Creating new terrain service")
	return &Service{
		logger: componentLogger,
	}
}

// NewServiceWithDefaultLogger creates a service with the default logger (convenience constructor for production use).
func NewServiceWithDefaultLogger() *Service {
	return NewService(NewDefaultLoggerWrapper())
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
