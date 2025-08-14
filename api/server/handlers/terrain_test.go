package handlers

import (
	"context"
	"errors"
	"io"
	"testing"

	mockhandlers "github.com/VoidMesh/api/api/internal/testmocks/handlers"
	"github.com/VoidMesh/api/api/internal/testutil"
	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestTerrainServiceServer_GetTerrainTypes demonstrates comprehensive testing patterns for terrain type retrieval
func TestTerrainServiceServer_GetTerrainTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name       string
		request    *terrainV1.GetTerrainTypesRequest
		setupMocks func(mockTerrain *mockhandlers.MockTerrainService)
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse)
	}{
		{
			name:    "successful terrain types retrieval - all 5 terrain types",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Expected terrain types from service
				expectedTerrainTypes := createStandardTerrainTypes()
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(expectedTerrainTypes, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.TerrainTypes)
				require.Len(t, resp.TerrainTypes, 5)

				// Validate all terrain types are present and have correct properties
				terrainByType := make(map[terrainV1.TerrainType]*terrainV1.TerrainTypeInfo)
				for _, terrain := range resp.TerrainTypes {
					terrainByType[terrain.Type] = terrain
				}

				// Validate GRASS terrain
				grass := terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_GRASS]
				require.NotNil(t, grass)
				assert.Equal(t, "Grass", grass.Name)
				assert.Equal(t, "Grassy plains with lush vegetation", grass.Description)
				assert.Equal(t, "#7CFC00", grass.Visual.BaseColor)
				assert.Equal(t, float32(1.0), grass.Properties.MovementSpeedMultiplier)
				assert.False(t, grass.Properties.IsWater)
				assert.True(t, grass.Properties.IsPassable)

				// Validate WATER terrain
				water := terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_WATER]
				require.NotNil(t, water)
				assert.Equal(t, "Water", water.Name)
				assert.Equal(t, "#1E90FF", water.Visual.BaseColor)
				assert.Equal(t, float32(0.5), water.Properties.MovementSpeedMultiplier)
				assert.True(t, water.Properties.IsWater)
				assert.True(t, water.Properties.IsPassable)

				// Validate STONE terrain
				stone := terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_STONE]
				require.NotNil(t, stone)
				assert.Equal(t, "Stone", stone.Name)
				assert.Equal(t, "#708090", stone.Visual.BaseColor)
				assert.Equal(t, float32(0.8), stone.Properties.MovementSpeedMultiplier)
				assert.False(t, stone.Properties.IsWater)

				// Validate SAND terrain
				sand := terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_SAND]
				require.NotNil(t, sand)
				assert.Equal(t, "Sand", sand.Name)
				assert.Equal(t, "#FFD700", sand.Visual.BaseColor)
				assert.Equal(t, float32(0.7), sand.Properties.MovementSpeedMultiplier)

				// Validate DIRT terrain
				dirt := terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_DIRT]
				require.NotNil(t, dirt)
				assert.Equal(t, "Dirt", dirt.Name)
				assert.Equal(t, "#8B4513", dirt.Visual.BaseColor)
				assert.Equal(t, float32(0.9), dirt.Properties.MovementSpeedMultiplier)
			},
		},
		{
			name:    "terrain types ordering validation",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Return terrain types in expected order
				expectedTerrainTypes := createStandardTerrainTypes()
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(expectedTerrainTypes, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.Len(t, resp.TerrainTypes, 5)

				// Verify terrain types are in the expected order: GRASS, WATER, STONE, SAND, DIRT
				expectedOrder := []terrainV1.TerrainType{
					terrainV1.TerrainType_TERRAIN_TYPE_GRASS,
					terrainV1.TerrainType_TERRAIN_TYPE_WATER,
					terrainV1.TerrainType_TERRAIN_TYPE_STONE,
					terrainV1.TerrainType_TERRAIN_TYPE_SAND,
					terrainV1.TerrainType_TERRAIN_TYPE_DIRT,
				}

				for i, expectedType := range expectedOrder {
					assert.Equal(t, expectedType, resp.TerrainTypes[i].Type,
						"Terrain type at index %d should be %v, got %v", i, expectedType, resp.TerrainTypes[i].Type)
				}
			},
		},
		{
			name:    "proto field validation - visual properties",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(createStandardTerrainTypes(), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.Len(t, resp.TerrainTypes, 5)

				// Validate all terrain types have complete visual properties
				for i, terrain := range resp.TerrainTypes {
					require.NotNil(t, terrain.Visual, "Terrain at index %d missing visual properties", i)
					assert.NotEmpty(t, terrain.Visual.BaseColor, "Terrain %s missing base color", terrain.Name)
					assert.NotEmpty(t, terrain.Visual.Texture, "Terrain %s missing texture", terrain.Name)
					assert.GreaterOrEqual(t, terrain.Visual.Roughness, float32(0.0), "Terrain %s roughness below 0.0", terrain.Name)
					assert.LessOrEqual(t, terrain.Visual.Roughness, float32(1.0), "Terrain %s roughness above 1.0", terrain.Name)

					// Validate color format (hex colors)
					assert.Regexp(t, `^#[0-9A-Fa-f]{6}$`, terrain.Visual.BaseColor,
						"Terrain %s has invalid color format: %s", terrain.Name, terrain.Visual.BaseColor)
				}
			},
		},
		{
			name:    "movement speed multiplier boundaries validation",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(createStandardTerrainTypes(), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)

				// Validate movement speed multipliers are within reasonable bounds
				for _, terrain := range resp.TerrainTypes {
					assert.Greater(t, terrain.Properties.MovementSpeedMultiplier, float32(0.0),
						"Terrain %s has non-positive movement speed", terrain.Name)
					assert.LessOrEqual(t, terrain.Properties.MovementSpeedMultiplier, float32(1.0),
						"Terrain %s has movement speed above 1.0", terrain.Name)
				}

				// Verify specific expected values
				terrainByType := make(map[terrainV1.TerrainType]*terrainV1.TerrainTypeInfo)
				for _, terrain := range resp.TerrainTypes {
					terrainByType[terrain.Type] = terrain
				}

				assert.Equal(t, float32(1.0), terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_GRASS].Properties.MovementSpeedMultiplier)
				assert.Equal(t, float32(0.5), terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_WATER].Properties.MovementSpeedMultiplier)
				assert.Equal(t, float32(0.8), terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_STONE].Properties.MovementSpeedMultiplier)
				assert.Equal(t, float32(0.7), terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_SAND].Properties.MovementSpeedMultiplier)
				assert.Equal(t, float32(0.9), terrainByType[terrainV1.TerrainType_TERRAIN_TYPE_DIRT].Properties.MovementSpeedMultiplier)
			},
		},
		{
			name:    "water terrain identification validation",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(createStandardTerrainTypes(), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)

				waterCount := 0
				nonWaterCount := 0

				for _, terrain := range resp.TerrainTypes {
					if terrain.Properties.IsWater {
						waterCount++
						// Only water terrain should be marked as water
						assert.Equal(t, terrainV1.TerrainType_TERRAIN_TYPE_WATER, terrain.Type,
							"Non-water terrain %s incorrectly marked as water", terrain.Name)
					} else {
						nonWaterCount++
					}
					
					// All terrain types should be passable in current implementation
					assert.True(t, terrain.Properties.IsPassable,
						"Terrain %s should be passable", terrain.Name)
				}

				// Verify exactly one water terrain type
				assert.Equal(t, 1, waterCount, "Expected exactly 1 water terrain type")
				assert.Equal(t, 4, nonWaterCount, "Expected exactly 4 non-water terrain types")
			},
		},
		// Error handling tests
		{
			name:    "service layer error propagation",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Simulate service error
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "database connection failed",
		},
		{
			name:    "context cancellation handling",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Simulate context cancellation
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(nil, context.Canceled)
			},
			wantErr:  true,
			wantCode: codes.Canceled,
		},
		{
			name:    "context deadline exceeded",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Simulate context timeout
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(nil, context.DeadlineExceeded)
			},
			wantErr:  true,
			wantCode: codes.DeadlineExceeded,
		},
		{
			name:    "service unavailable error",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Simulate service unavailable
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(nil, status.Error(codes.Unavailable, "terrain service temporarily unavailable"))
			},
			wantErr:  true,
			wantCode: codes.Unavailable,
			wantMsg:  "terrain service temporarily unavailable",
		},
		{
			name:    "malformed service response handling",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Return internal error for malformed response
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(nil, status.Error(codes.Internal, "invalid terrain type data"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "invalid terrain type data",
		},
		// Edge case tests
		{
			name:    "empty terrain types response handling",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Return empty terrain types array
				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return([]*terrainV1.TerrainTypeInfo{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.TerrainTypes)
				assert.Len(t, resp.TerrainTypes, 0, "Expected empty terrain types array")
			},
		},
		{
			name:    "unicode handling in terrain descriptions",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Return terrain types with Unicode characters
				unicodeTerrainTypes := []*terrainV1.TerrainTypeInfo{
					{
						Type:        terrainV1.TerrainType_TERRAIN_TYPE_GRASS,
						Name:        "Grass 草",
						Description: "Terrain with Unicode: 🌱🌿 Green fields with émojis and accènts",
						Visual: &terrainV1.TerrainVisual{
							BaseColor: "#7CFC00",
							Texture:   "grass_texture_🌱",
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
						Name:        "Water 水",
						Description: "Aquatic terrain: 🌊💧 Bodies of wäter with spëcial çhars",
						Visual: &terrainV1.TerrainVisual{
							BaseColor: "#1E90FF",
							Texture:   "water_texture_🌊",
							Roughness: 0.1,
						},
						Properties: &terrainV1.TerrainProperties{
							MovementSpeedMultiplier: 0.5,
							IsWater:                 true,
							IsPassable:              true,
						},
					},
				}

				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(unicodeTerrainTypes, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.Len(t, resp.TerrainTypes, 2)

				// Validate Unicode characters are preserved
				grassTerrain := resp.TerrainTypes[0]
				assert.Equal(t, "Grass 草", grassTerrain.Name)
				assert.Contains(t, grassTerrain.Description, "🌱🌿")
				assert.Contains(t, grassTerrain.Description, "émojis")
				assert.Contains(t, grassTerrain.Description, "accènts")
				assert.Contains(t, grassTerrain.Visual.Texture, "🌱")

				waterTerrain := resp.TerrainTypes[1]
				assert.Equal(t, "Water 水", waterTerrain.Name)
				assert.Contains(t, waterTerrain.Description, "🌊💧")
				assert.Contains(t, waterTerrain.Description, "wäter")
				assert.Contains(t, waterTerrain.Description, "spëcial")
				assert.Contains(t, waterTerrain.Description, "çhars")
			},
		},
		{
			name:    "large terrain collections performance test",
			request: &terrainV1.GetTerrainTypesRequest{},
			setupMocks: func(mockTerrain *mockhandlers.MockTerrainService) {
				// Generate large collection of terrain types
				largeTerrainTypes := make([]*terrainV1.TerrainTypeInfo, 100)
				for i := 0; i < 100; i++ {
					terrainType := terrainV1.TerrainType_TERRAIN_TYPE_GRASS
					if i%5 == 1 {
						terrainType = terrainV1.TerrainType_TERRAIN_TYPE_WATER
					} else if i%5 == 2 {
						terrainType = terrainV1.TerrainType_TERRAIN_TYPE_STONE
					} else if i%5 == 3 {
						terrainType = terrainV1.TerrainType_TERRAIN_TYPE_SAND
					} else if i%5 == 4 {
						terrainType = terrainV1.TerrainType_TERRAIN_TYPE_DIRT
					}

					largeTerrainTypes[i] = &terrainV1.TerrainTypeInfo{
						Type:        terrainType,
						Name:        "Terrain Type " + string(rune(i)),
						Description: "Generated terrain type for performance testing",
						Visual: &terrainV1.TerrainVisual{
							BaseColor: "#7CFC00",
							Texture:   "test_texture",
							Roughness: float32(i%10) / 10.0,
						},
						Properties: &terrainV1.TerrainProperties{
							MovementSpeedMultiplier: 0.5 + float32(i%5)/10.0,
							IsWater:                 terrainType == terrainV1.TerrainType_TERRAIN_TYPE_WATER,
							IsPassable:              true,
						},
					}
				}

				mockTerrain.EXPECT().
					GetTerrainTypes(gomock.Any()).
					Return(largeTerrainTypes, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *terrainV1.GetTerrainTypesResponse) {
				require.NotNil(t, resp)
				require.Len(t, resp.TerrainTypes, 100, "Expected 100 terrain types for performance test")

				// Validate all terrain types have required fields
				for i, terrain := range resp.TerrainTypes {
					assert.NotEmpty(t, terrain.Name, "Terrain at index %d missing name", i)
					assert.NotNil(t, terrain.Visual, "Terrain at index %d missing visual", i)
					assert.NotNil(t, terrain.Properties, "Terrain at index %d missing properties", i)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mocks
			mockTerrain := mockhandlers.NewMockTerrainService(ctrl)

			// Setup mock expectations
			if tt.setupMocks != nil {
				tt.setupMocks(mockTerrain)
			}

			// Create server instance with mocks (using real logger to avoid import cycle)
			server := &terrainServiceServer{
				terrainService: mockTerrain,
				logger:         &LoggerWrapper{Logger: log.New(io.Discard)}, // Use real logger with discard output
			}

			// Execute the method under test
			resp, err := server.GetTerrainTypes(context.Background(), tt.request)

			// Validate error expectations
			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp, "Response should be nil on error")
				return
			}

			// Validate success expectations
			testutil.AssertNoGRPCError(t, err)
			require.NotNil(t, resp, "Response should not be nil on success")

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

// TestTerrainServiceServer_GetTerrainTypes_Parallel demonstrates concurrent access patterns
func TestTerrainServiceServer_GetTerrainTypes_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockTerrain := mockhandlers.NewMockTerrainService(ctrl)

	mockTerrain.EXPECT().
		GetTerrainTypes(gomock.Any()).
		Return(createStandardTerrainTypes(), nil).
		AnyTimes()

	// Create server instance
	server := &terrainServiceServer{
		terrainService: mockTerrain,
		logger:         &LoggerWrapper{Logger: log.New(io.Discard)},
	}

	// Run concurrent requests
	const numGoroutines = 10
	results := make(chan *terrainV1.GetTerrainTypesResponse, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			resp, err := server.GetTerrainTypes(context.Background(), &terrainV1.GetTerrainTypesRequest{})
			if err != nil {
				errors <- err
			} else {
				results <- resp
			}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case resp := <-results:
			require.NotNil(t, resp)
			assert.Len(t, resp.TerrainTypes, 5)
		case err := <-errors:
			t.Fatalf("Unexpected error in concurrent test: %v", err)
		}
	}
}

// BenchmarkTerrainHandler_GetTerrainTypes benchmarks the terrain handler performance
func BenchmarkTerrainHandler_GetTerrainTypes(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockTerrain := mockhandlers.NewMockTerrainService(ctrl)

	mockTerrain.EXPECT().
		GetTerrainTypes(gomock.Any()).
		Return(createStandardTerrainTypes(), nil).
		AnyTimes()

	// Create server instance
	server := &terrainServiceServer{
		terrainService: mockTerrain,
		logger:         &LoggerWrapper{Logger: log.New(io.Discard)},
	}

	request := &terrainV1.GetTerrainTypesRequest{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := server.GetTerrainTypes(context.Background(), request)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkTerrainHandler_GetTerrainTypes_LargeCollection benchmarks performance with large terrain collections
func BenchmarkTerrainHandler_GetTerrainTypes_LargeCollection(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockTerrain := mockhandlers.NewMockTerrainService(ctrl)

	// Generate large terrain collection for benchmark
	largeTerrainTypes := make([]*terrainV1.TerrainTypeInfo, 1000)
	for i := 0; i < 1000; i++ {
		terrainType := terrainV1.TerrainType_TERRAIN_TYPE_GRASS
		if i%5 == 1 {
			terrainType = terrainV1.TerrainType_TERRAIN_TYPE_WATER
		} else if i%5 == 2 {
			terrainType = terrainV1.TerrainType_TERRAIN_TYPE_STONE
		} else if i%5 == 3 {
			terrainType = terrainV1.TerrainType_TERRAIN_TYPE_SAND
		} else if i%5 == 4 {
			terrainType = terrainV1.TerrainType_TERRAIN_TYPE_DIRT
		}

		largeTerrainTypes[i] = &terrainV1.TerrainTypeInfo{
			Type:        terrainType,
			Name:        "Benchmark Terrain Type " + string(rune(i)),
			Description: "Generated terrain type for benchmark testing",
			Visual: &terrainV1.TerrainVisual{
				BaseColor: "#7CFC00",
				Texture:   "benchmark_texture",
				Roughness: float32(i%10) / 10.0,
			},
			Properties: &terrainV1.TerrainProperties{
				MovementSpeedMultiplier: 0.5 + float32(i%5)/10.0,
				IsWater:                 terrainType == terrainV1.TerrainType_TERRAIN_TYPE_WATER,
				IsPassable:              true,
			},
		}
	}

	mockTerrain.EXPECT().
		GetTerrainTypes(gomock.Any()).
		Return(largeTerrainTypes, nil).
		AnyTimes()

	// Create server instance
	server := &terrainServiceServer{
		terrainService: mockTerrain,
		logger:         &LoggerWrapper{Logger: log.New(io.Discard)},
	}

	request := &terrainV1.GetTerrainTypesRequest{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := server.GetTerrainTypes(context.Background(), request)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkTerrainHandler_GetTerrainTypes_Concurrent benchmarks concurrent request performance  
func BenchmarkTerrainHandler_GetTerrainTypes_Concurrent(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockTerrain := mockhandlers.NewMockTerrainService(ctrl)

	mockTerrain.EXPECT().
		GetTerrainTypes(gomock.Any()).
		Return(createStandardTerrainTypes(), nil).
		AnyTimes()

	// Create server instance
	server := &terrainServiceServer{
		terrainService: mockTerrain,
		logger:         &LoggerWrapper{Logger: log.New(io.Discard)},
	}

	request := &terrainV1.GetTerrainTypesRequest{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := server.GetTerrainTypes(context.Background(), request)
			if err != nil {
				b.Fatalf("Concurrent benchmark failed: %v", err)
			}
		}
	})
}

// createStandardTerrainTypes creates the standard set of 5 terrain types for testing
func createStandardTerrainTypes() []*terrainV1.TerrainTypeInfo {
	return []*terrainV1.TerrainTypeInfo{
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
}