package terrain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VoidMesh/api/api/internal/testutil"
	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
)

// MockLogger implements LoggerInterface for testing
type MockLogger struct {
	calls []LogCall
}

type LogCall struct {
	Level  string
	Msg    string
	KeyVal []interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		calls: make([]LogCall, 0),
	}
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.calls = append(m.calls, LogCall{Level: "debug", Msg: msg, KeyVal: keysAndValues})
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.calls = append(m.calls, LogCall{Level: "info", Msg: msg, KeyVal: keysAndValues})
}

func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.calls = append(m.calls, LogCall{Level: "warn", Msg: msg, KeyVal: keysAndValues})
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.calls = append(m.calls, LogCall{Level: "error", Msg: msg, KeyVal: keysAndValues})
}

func (m *MockLogger) With(keysAndValues ...interface{}) LoggerInterface {
	// Return self for simplicity in tests
	return m
}

func (m *MockLogger) GetCalls() []LogCall {
	return m.calls
}

func (m *MockLogger) GetCallsOfLevel(level string) []LogCall {
	var filtered []LogCall
	for _, call := range m.calls {
		if call.Level == level {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

func TestNewService(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		logger         LoggerInterface
		expectLogger   func(t *testing.T, service *Service)
		expectLogCalls func(t *testing.T, mockLogger *MockLogger)
	}{
		{
			name:   "successful service creation with mock logger",
			logger: NewMockLogger(),
			expectLogger: func(t *testing.T, service *Service) {
				assert.NotNil(t, service.logger)
			},
			expectLogCalls: func(t *testing.T, mockLogger *MockLogger) {
				debugCalls := mockLogger.GetCallsOfLevel("debug")
				assert.Len(t, debugCalls, 1)
				assert.Equal(t, "Creating new terrain service", debugCalls[0].Msg)
			},
		},
		{
			name:   "service creation with default logger wrapper",
			logger: NewDefaultLoggerWrapper(),
			expectLogger: func(t *testing.T, service *Service) {
				assert.NotNil(t, service.logger)
			},
			expectLogCalls: func(t *testing.T, mockLogger *MockLogger) {
				// Default logger wrapper doesn't accumulate calls in our mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.logger)

			require.NotNil(t, service)
			tt.expectLogger(t, service)

			if mockLogger, ok := tt.logger.(*MockLogger); ok {
				tt.expectLogCalls(t, mockLogger)
			}
		})
	}
}

func TestNewServiceWithDefaultLogger(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	service := NewServiceWithDefaultLogger()

	require.NotNil(t, service)
	assert.NotNil(t, service.logger)

	// Ensure the service has a default logger wrapper
	_, ok := service.logger.(*DefaultLoggerWrapper)
	assert.True(t, ok, "Expected service to use DefaultLoggerWrapper")
}

func TestDefaultLoggerWrapper(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name       string
		testMethod func(t *testing.T, wrapper LoggerInterface)
	}{
		{
			name: "debug method",
			testMethod: func(t *testing.T, wrapper LoggerInterface) {
				// Should not panic
				wrapper.Debug("test debug message", "key", "value")
			},
		},
		{
			name: "info method",
			testMethod: func(t *testing.T, wrapper LoggerInterface) {
				// Should not panic
				wrapper.Info("test info message", "key", "value")
			},
		},
		{
			name: "warn method",
			testMethod: func(t *testing.T, wrapper LoggerInterface) {
				// Should not panic
				wrapper.Warn("test warn message", "key", "value")
			},
		},
		{
			name: "error method",
			testMethod: func(t *testing.T, wrapper LoggerInterface) {
				// Should not panic
				wrapper.Error("test error message", "key", "value")
			},
		},
		{
			name: "with method",
			testMethod: func(t *testing.T, wrapper LoggerInterface) {
				newWrapper := wrapper.With("component", "test")
				assert.NotNil(t, newWrapper)
				// Should return self for simplicity
				assert.Equal(t, wrapper, newWrapper)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := NewDefaultLoggerWrapper()
			require.NotNil(t, wrapper)

			// Test should not panic
			assert.NotPanics(t, func() {
				tt.testMethod(t, wrapper)
			})
		})
	}
}

func TestService_GetTerrainTypes(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name                 string
		setupLogger          func() LoggerInterface
		expectError          bool
		expectedTerrainCount int
		validateTerrainTypes func(t *testing.T, terrainTypes []*terrainV1.TerrainTypeInfo)
		validateLogCalls     func(t *testing.T, mockLogger *MockLogger)
	}{
		{
			name: "successful terrain types retrieval",
			setupLogger: func() LoggerInterface {
				return NewMockLogger()
			},
			expectError:          false,
			expectedTerrainCount: 5, // GRASS, WATER, STONE, SAND, DIRT
			validateTerrainTypes: func(t *testing.T, terrainTypes []*terrainV1.TerrainTypeInfo) {
				// Verify all expected terrain types are present
				expectedTypes := map[terrainV1.TerrainType]string{
					terrainV1.TerrainType_TERRAIN_TYPE_GRASS: "Grass",
					terrainV1.TerrainType_TERRAIN_TYPE_WATER: "Water",
					terrainV1.TerrainType_TERRAIN_TYPE_STONE: "Stone",
					terrainV1.TerrainType_TERRAIN_TYPE_SAND:  "Sand",
					terrainV1.TerrainType_TERRAIN_TYPE_DIRT:  "Dirt",
				}

				foundTypes := make(map[terrainV1.TerrainType]bool)
				for _, terrain := range terrainTypes {
					assert.NotNil(t, terrain)
					assert.NotNil(t, terrain.Visual)
					assert.NotNil(t, terrain.Properties)

					expectedName, exists := expectedTypes[terrain.Type]
					assert.True(t, exists, "Unexpected terrain type: %v", terrain.Type)
					assert.Equal(t, expectedName, terrain.Name)
					assert.NotEmpty(t, terrain.Description)
					assert.NotEmpty(t, terrain.Visual.BaseColor)
					assert.NotEmpty(t, terrain.Visual.Texture)
					assert.GreaterOrEqual(t, terrain.Visual.Roughness, float32(0.0))
					assert.LessOrEqual(t, terrain.Visual.Roughness, float32(1.0))

					foundTypes[terrain.Type] = true
				}

				// Ensure all expected types were found
				assert.Len(t, foundTypes, len(expectedTypes))
			},
			validateLogCalls: func(t *testing.T, mockLogger *MockLogger) {
				debugCalls := mockLogger.GetCallsOfLevel("debug")
				assert.Len(t, debugCalls, 2) // Service creation + GetTerrainTypes call
				assert.Equal(t, "Getting all terrain types", debugCalls[1].Msg)
			},
		},
		{
			name: "terrain types with default logger",
			setupLogger: func() LoggerInterface {
				return NewDefaultLoggerWrapper()
			},
			expectError:          false,
			expectedTerrainCount: 5,
			validateTerrainTypes: func(t *testing.T, terrainTypes []*terrainV1.TerrainTypeInfo) {
				// Basic validation for default logger case
				assert.Len(t, terrainTypes, 5)
				for _, terrain := range terrainTypes {
					assert.NotNil(t, terrain)
					assert.NotEqual(t, terrainV1.TerrainType_TERRAIN_TYPE_UNSPECIFIED, terrain.Type)
				}
			},
			validateLogCalls: func(t *testing.T, mockLogger *MockLogger) {
				// No validation for default logger
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := tt.setupLogger()
			service := NewService(logger)
			ctx := testutil.CreateTestContext()

			terrainTypes, err := service.GetTerrainTypes(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, terrainTypes)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, terrainTypes)
				assert.Len(t, terrainTypes, tt.expectedTerrainCount)

				tt.validateTerrainTypes(t, terrainTypes)
			}

			if mockLogger, ok := logger.(*MockLogger); ok {
				tt.validateLogCalls(t, mockLogger)
			}
		})
	}
}

func TestService_GetTerrainTypes_TerrainProperties(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockLogger := NewMockLogger()
	service := NewService(mockLogger)
	ctx := testutil.CreateTestContext()

	terrainTypes, err := service.GetTerrainTypes(ctx)
	require.NoError(t, err)
	require.NotNil(t, terrainTypes)

	// Test specific terrain properties
	tests := []struct {
		terrainType                terrainV1.TerrainType
		expectedName               string
		expectedIsWater            bool
		expectedIsPassable         bool
		expectedMovementMultiplier float32
		expectedColor              string
		expectedTexture            string
		expectedRoughnessRange     [2]float32 // min, max
	}{
		{
			terrainType:                terrainV1.TerrainType_TERRAIN_TYPE_GRASS,
			expectedName:               "Grass",
			expectedIsWater:            false,
			expectedIsPassable:         true,
			expectedMovementMultiplier: 1.0,
			expectedColor:              "#7CFC00",
			expectedTexture:            "grass_texture",
			expectedRoughnessRange:     [2]float32{0.3, 0.5},
		},
		{
			terrainType:                terrainV1.TerrainType_TERRAIN_TYPE_WATER,
			expectedName:               "Water",
			expectedIsWater:            true,
			expectedIsPassable:         true,
			expectedMovementMultiplier: 0.5,
			expectedColor:              "#1E90FF",
			expectedTexture:            "water_texture",
			expectedRoughnessRange:     [2]float32{0.0, 0.2},
		},
		{
			terrainType:                terrainV1.TerrainType_TERRAIN_TYPE_STONE,
			expectedName:               "Stone",
			expectedIsWater:            false,
			expectedIsPassable:         true,
			expectedMovementMultiplier: 0.8,
			expectedColor:              "#708090",
			expectedTexture:            "stone_texture",
			expectedRoughnessRange:     [2]float32{0.7, 0.9},
		},
		{
			terrainType:                terrainV1.TerrainType_TERRAIN_TYPE_SAND,
			expectedName:               "Sand",
			expectedIsWater:            false,
			expectedIsPassable:         true,
			expectedMovementMultiplier: 0.7,
			expectedColor:              "#FFD700",
			expectedTexture:            "sand_texture",
			expectedRoughnessRange:     [2]float32{0.5, 0.7},
		},
		{
			terrainType:                terrainV1.TerrainType_TERRAIN_TYPE_DIRT,
			expectedName:               "Dirt",
			expectedIsWater:            false,
			expectedIsPassable:         true,
			expectedMovementMultiplier: 0.9,
			expectedColor:              "#8B4513",
			expectedTexture:            "dirt_texture",
			expectedRoughnessRange:     [2]float32{0.4, 0.6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName+"_properties", func(t *testing.T) {
			var foundTerrain *terrainV1.TerrainTypeInfo
			for _, terrain := range terrainTypes {
				if terrain.Type == tt.terrainType {
					foundTerrain = terrain
					break
				}
			}

			require.NotNil(t, foundTerrain, "Terrain type %v not found", tt.terrainType)

			// Test basic properties
			assert.Equal(t, tt.expectedName, foundTerrain.Name)
			assert.NotEmpty(t, foundTerrain.Description)

			// Test terrain properties
			require.NotNil(t, foundTerrain.Properties)
			assert.Equal(t, tt.expectedIsWater, foundTerrain.Properties.IsWater)
			assert.Equal(t, tt.expectedIsPassable, foundTerrain.Properties.IsPassable)
			assert.Equal(t, tt.expectedMovementMultiplier, foundTerrain.Properties.MovementSpeedMultiplier)

			// Test visual properties
			require.NotNil(t, foundTerrain.Visual)
			assert.Equal(t, tt.expectedColor, foundTerrain.Visual.BaseColor)
			assert.Equal(t, tt.expectedTexture, foundTerrain.Visual.Texture)
			assert.GreaterOrEqual(t, foundTerrain.Visual.Roughness, tt.expectedRoughnessRange[0])
			assert.LessOrEqual(t, foundTerrain.Visual.Roughness, tt.expectedRoughnessRange[1])
		})
	}
}

func TestService_GetTerrainTypes_MovementSpeedValidation(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockLogger := NewMockLogger()
	service := NewService(mockLogger)
	ctx := testutil.CreateTestContext()

	terrainTypes, err := service.GetTerrainTypes(ctx)
	require.NoError(t, err)
	require.NotNil(t, terrainTypes)

	// Test movement speed multiplier constraints
	for _, terrain := range terrainTypes {
		t.Run(terrain.Name+"_movement_speed", func(t *testing.T) {
			require.NotNil(t, terrain.Properties)

			// Movement speed should be between 0.0 and 1.0 (inclusive)
			assert.GreaterOrEqual(t, terrain.Properties.MovementSpeedMultiplier, float32(0.0),
				"Movement speed multiplier should be >= 0.0 for %s", terrain.Name)
			assert.LessOrEqual(t, terrain.Properties.MovementSpeedMultiplier, float32(1.0),
				"Movement speed multiplier should be <= 1.0 for %s", terrain.Name)
		})
	}
}

func TestService_GetTerrainTypes_VisualPropertiesValidation(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockLogger := NewMockLogger()
	service := NewService(mockLogger)
	ctx := testutil.CreateTestContext()

	terrainTypes, err := service.GetTerrainTypes(ctx)
	require.NoError(t, err)
	require.NotNil(t, terrainTypes)

	// Test visual properties constraints
	for _, terrain := range terrainTypes {
		t.Run(terrain.Name+"_visual_properties", func(t *testing.T) {
			require.NotNil(t, terrain.Visual)

			// Base color should be a valid hex color
			assert.Regexp(t, "^#[0-9A-Fa-f]{6}$", terrain.Visual.BaseColor,
				"Base color should be a valid hex color for %s", terrain.Name)

			// Texture should not be empty
			assert.NotEmpty(t, terrain.Visual.Texture,
				"Texture should not be empty for %s", terrain.Name)

			// Roughness should be between 0.0 and 1.0
			assert.GreaterOrEqual(t, terrain.Visual.Roughness, float32(0.0),
				"Roughness should be >= 0.0 for %s", terrain.Name)
			assert.LessOrEqual(t, terrain.Visual.Roughness, float32(1.0),
				"Roughness should be <= 1.0 for %s", terrain.Name)
		})
	}
}

func TestService_GetTerrainTypes_PassabilityValidation(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockLogger := NewMockLogger()
	service := NewService(mockLogger)
	ctx := testutil.CreateTestContext()

	terrainTypes, err := service.GetTerrainTypes(ctx)
	require.NoError(t, err)
	require.NotNil(t, terrainTypes)

	// Test passability logic
	for _, terrain := range terrainTypes {
		t.Run(terrain.Name+"_passability", func(t *testing.T) {
			require.NotNil(t, terrain.Properties)

			// All terrain types in this implementation should be passable
			assert.True(t, terrain.Properties.IsPassable,
				"All terrain types should be passable in current implementation for %s", terrain.Name)

			// Water terrain should have IsWater = true
			if terrain.Type == terrainV1.TerrainType_TERRAIN_TYPE_WATER {
				assert.True(t, terrain.Properties.IsWater,
					"Water terrain should have IsWater = true")
			} else {
				assert.False(t, terrain.Properties.IsWater,
					"Non-water terrain should have IsWater = false for %s", terrain.Name)
			}
		})
	}
}

func TestService_GetTerrainTypes_ConsistencyValidation(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockLogger := NewMockLogger()
	service := NewService(mockLogger)
	ctx := testutil.CreateTestContext()

	// Call the method multiple times to ensure consistency
	terrainTypes1, err1 := service.GetTerrainTypes(ctx)
	require.NoError(t, err1)

	terrainTypes2, err2 := service.GetTerrainTypes(ctx)
	require.NoError(t, err2)

	// Results should be consistent across calls
	assert.Equal(t, len(terrainTypes1), len(terrainTypes2))

	// Create maps for easier comparison
	terrainMap1 := make(map[terrainV1.TerrainType]*terrainV1.TerrainTypeInfo)
	terrainMap2 := make(map[terrainV1.TerrainType]*terrainV1.TerrainTypeInfo)

	for _, terrain := range terrainTypes1 {
		terrainMap1[terrain.Type] = terrain
	}

	for _, terrain := range terrainTypes2 {
		terrainMap2[terrain.Type] = terrain
	}

	// Compare each terrain type
	for terrainType, terrain1 := range terrainMap1 {
		terrain2, exists := terrainMap2[terrainType]
		require.True(t, exists, "Terrain type %v should exist in both calls", terrainType)

		assert.Equal(t, terrain1.Name, terrain2.Name)
		assert.Equal(t, terrain1.Description, terrain2.Description)
		assert.Equal(t, terrain1.Visual.BaseColor, terrain2.Visual.BaseColor)
		assert.Equal(t, terrain1.Visual.Texture, terrain2.Visual.Texture)
		assert.Equal(t, terrain1.Visual.Roughness, terrain2.Visual.Roughness)
		assert.Equal(t, terrain1.Properties.MovementSpeedMultiplier, terrain2.Properties.MovementSpeedMultiplier)
		assert.Equal(t, terrain1.Properties.IsWater, terrain2.Properties.IsWater)
		assert.Equal(t, terrain1.Properties.IsPassable, terrain2.Properties.IsPassable)
	}

	// Verify multiple debug log calls
	debugCalls := mockLogger.GetCallsOfLevel("debug")
	assert.Len(t, debugCalls, 3) // Service creation + 2 GetTerrainTypes calls
}
