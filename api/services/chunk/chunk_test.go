package chunk

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/db"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/internal/testutil"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockDatabaseInterface provides a mock database for testing
type MockDatabaseInterface struct {
	chunks          map[string]db.Chunk
	shouldReturnErr bool
	getCallCount    int
	createCallCount int
}

func NewMockDatabase() *MockDatabaseInterface {
	return &MockDatabaseInterface{
		chunks: make(map[string]db.Chunk),
	}
}

func (m *MockDatabaseInterface) GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error) {
	m.getCallCount++
	if m.shouldReturnErr {
		return db.Chunk{}, errors.New("database error")
	}

	key := fmt.Sprintf("%x_%d_%d", arg.WorldID.Bytes, arg.ChunkX, arg.ChunkY)
	chunk, exists := m.chunks[key]
	if !exists {
		return db.Chunk{}, errors.New("chunk not found")
	}
	return chunk, nil
}

func (m *MockDatabaseInterface) CreateChunk(ctx context.Context, arg db.CreateChunkParams) (db.Chunk, error) {
	m.createCallCount++
	if m.shouldReturnErr {
		return db.Chunk{}, errors.New("database error")
	}

	chunk := db.Chunk{
		WorldID:     arg.WorldID,
		ChunkX:      arg.ChunkX,
		ChunkY:      arg.ChunkY,
		ChunkData:   arg.ChunkData,
		GeneratedAt: pgtype.Timestamp{Valid: true, Time: time.Now()},
	}

	key := fmt.Sprintf("%x_%d_%d", arg.WorldID.Bytes, arg.ChunkX, arg.ChunkY)
	m.chunks[key] = chunk
	return chunk, nil
}

func (m *MockDatabaseInterface) ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error) {
	if m.shouldReturnErr {
		return false, errors.New("database error")
	}

	key := fmt.Sprintf("%x_%d_%d", arg.WorldID.Bytes, arg.ChunkX, arg.ChunkY)
	_, exists := m.chunks[key]
	return exists, nil
}

func (m *MockDatabaseInterface) SetShouldReturnError(shouldErr bool) {
	m.shouldReturnErr = shouldErr
}

func (m *MockDatabaseInterface) AddChunk(worldID pgtype.UUID, chunkX, chunkY int32, chunkData []byte) {
	key := fmt.Sprintf("%x_%d_%d", worldID.Bytes, chunkX, chunkY)
	m.chunks[key] = db.Chunk{
		WorldID:     worldID,
		ChunkX:      chunkX,
		ChunkY:      chunkY,
		ChunkData:   chunkData,
		GeneratedAt: pgtype.Timestamp{Valid: true, Time: time.Now()},
	}
}

func (m *MockDatabaseInterface) GetCallCount() int {
	return m.getCallCount
}

func (m *MockDatabaseInterface) GetCreateCallCount() int {
	return m.createCallCount
}

// MockNoiseGenerator provides controlled noise values for testing
type MockNoiseGenerator struct {
	noiseValues map[string]float64
	seed        int64
}

func NewMockNoiseGenerator(seed int64) *MockNoiseGenerator {
	return &MockNoiseGenerator{
		noiseValues: make(map[string]float64),
		seed:        seed,
	}
}

func (m *MockNoiseGenerator) GetTerrainNoise(x, y int, scale float64) float64 {
	key := fmt.Sprintf("%d_%d_%.1f", x, y, scale)
	if value, exists := m.noiseValues[key]; exists {
		return value
	}
	// Default to grass terrain range
	return 0.0
}

func (m *MockNoiseGenerator) GetSeed() int64 {
	return m.seed
}

func (m *MockNoiseGenerator) SetNoiseValue(x, y int, scale float64, value float64) {
	key := fmt.Sprintf("%d_%d_%.1f", x, y, scale)
	m.noiseValues[key] = value
}

// MockWorldService provides a controlled world for testing
type MockWorldService struct {
	defaultWorld db.World
	shouldErr    bool
}

func NewMockWorldService() *MockWorldService {
	defaultWorldID := pgtype.UUID{}
	defaultWorldID.Scan("550e8400-e29b-41d4-a716-446655440000")

	return &MockWorldService{
		defaultWorld: db.World{
			ID:   defaultWorldID,
			Name: "TestWorld",
			Seed: 12345,
			CreatedAt: pgtype.Timestamp{
				Valid: true,
				Time:  time.Now(),
			},
		},
	}
}

func (m *MockWorldService) GetDefaultWorld(ctx context.Context) (db.World, error) {
	if m.shouldErr {
		return db.World{}, errors.New("world service error")
	}
	return m.defaultWorld, nil
}

func (m *MockWorldService) GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error) {
	if m.shouldErr {
		return db.World{}, errors.New("world service error")
	}
	return m.defaultWorld, nil
}

func (m *MockWorldService) ChunkSize() int32 {
	return ChunkSize
}

func (m *MockWorldService) SetShouldError(shouldErr bool) {
	m.shouldErr = shouldErr
}

func (m *MockWorldService) SetSeed(seed int64) {
	m.defaultWorld.Seed = seed
}

// MockResourceNodeIntegration provides controlled resource node behavior
type MockResourceNodeIntegration struct {
	shouldErr         bool
	resourceNodes     []*resourceNodeV1.ResourceNode
	generateCallCount int
	attachCallCount   int
}

func NewMockResourceNodeIntegration() *MockResourceNodeIntegration {
	return &MockResourceNodeIntegration{
		resourceNodes: make([]*resourceNodeV1.ResourceNode, 0),
	}
}

func (m *MockResourceNodeIntegration) GenerateAndAttachResourceNodes(ctx context.Context, chunk *chunkV1.ChunkData) error {
	m.generateCallCount++
	if m.shouldErr {
		return errors.New("resource generation error")
	}
	
	// Attach predefined resource nodes to chunk
	chunk.ResourceNodes = m.resourceNodes
	return nil
}

func (m *MockResourceNodeIntegration) AttachResourceNodesToChunk(ctx context.Context, chunk *chunkV1.ChunkData) error {
	m.attachCallCount++
	if m.shouldErr {
		return errors.New("resource attachment error")
	}
	
	// Attach predefined resource nodes to chunk
	chunk.ResourceNodes = m.resourceNodes
	return nil
}

func (m *MockResourceNodeIntegration) SetShouldError(shouldErr bool) {
	m.shouldErr = shouldErr
}

func (m *MockResourceNodeIntegration) SetResourceNodes(nodes []*resourceNodeV1.ResourceNode) {
	m.resourceNodes = nodes
}

func (m *MockResourceNodeIntegration) GetGenerateCallCount() int {
	return m.generateCallCount
}

func (m *MockResourceNodeIntegration) GetAttachCallCount() int {
	return m.attachCallCount
}

// MockLogger provides a simple logger implementation for testing
type MockLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]LogEntry, 0),
	}
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.addLog("DEBUG", msg, keysAndValues...)
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.addLog("INFO", msg, keysAndValues...)
}

func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.addLog("WARN", msg, keysAndValues...)
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.addLog("ERROR", msg, keysAndValues...)
}

func (m *MockLogger) With(keysAndValues ...interface{}) LoggerInterface {
	// For simplicity, return the same instance
	return m
}

func (m *MockLogger) addLog(level, msg string, keysAndValues ...interface{}) {
	fields := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			fields[key] = keysAndValues[i+1]
		}
	}

	m.logs = append(m.logs, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fields,
	})
}

func (m *MockLogger) GetLogs() []LogEntry {
	return m.logs
}

func (m *MockLogger) GetLogCount(level string) int {
	count := 0
	for _, log := range m.logs {
		if log.Level == level {
			count++
		}
	}
	return count
}

func TestNewService(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		setupMocks   func() (DatabaseInterface, NoiseGeneratorInterface, WorldServiceInterface, ResourceNodeIntegrationInterface, LoggerInterface)
		expectFields func(t *testing.T, service *Service)
	}{
		{
			name: "successful service creation",
			setupMocks: func() (DatabaseInterface, NoiseGeneratorInterface, WorldServiceInterface, ResourceNodeIntegrationInterface, LoggerInterface) {
				db := NewMockDatabase()
				noise := NewMockNoiseGenerator(12345)
				world := NewMockWorldService()
				resources := NewMockResourceNodeIntegration()
				logger := NewMockLogger()
				return db, noise, world, resources, logger
			},
			expectFields: func(t *testing.T, service *Service) {
				assert.NotNil(t, service.db)
				assert.NotNil(t, service.noiseGen)
				assert.NotNil(t, service.worldService)
				assert.NotNil(t, service.resourceNodeIntegration)
				assert.NotNil(t, service.logger)
				assert.Equal(t, int32(ChunkSize), service.chunkSize)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, noise, world, resources, logger := tt.setupMocks()
			service := NewService(db, noise, world, resources, logger)
			
			require.NotNil(t, service)
			tt.expectFields(t, service)
		})
	}
}

func TestService_GenerateChunk(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		chunkX, chunkY int32
		setupNoise     func(noise *MockNoiseGenerator)
		setupWorld     func(world *MockWorldService)
		expectError    bool
		validateChunk  func(t *testing.T, chunk *chunkV1.ChunkData)
	}{
		{
			name:   "successful chunk generation with default terrain",
			chunkX: 0,
			chunkY: 0,
			setupNoise: func(noise *MockNoiseGenerator) {
				// Default noise values result in grass terrain (0.0 range)
			},
			setupWorld: func(world *MockWorldService) {
				world.SetSeed(12345)
			},
			expectError: false,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(0), chunk.ChunkX)
				assert.Equal(t, int32(0), chunk.ChunkY)
				assert.Equal(t, int64(12345), chunk.Seed)
				assert.NotNil(t, chunk.GeneratedAt)
				assert.Len(t, chunk.Cells, ChunkSize*ChunkSize)
				
				// All cells should be grass with default noise values
				for _, cell := range chunk.Cells {
					assert.Equal(t, chunkV1.TerrainType_TERRAIN_TYPE_GRASS, cell.TerrainType)
				}
			},
		},
		{
			name:   "chunk generation with water terrain",
			chunkX: 1,
			chunkY: 1,
			setupNoise: func(noise *MockNoiseGenerator) {
				// Set noise values to generate water terrain (< -0.3)
				for y := int32(0); y < ChunkSize; y++ {
					for x := int32(0); x < ChunkSize; x++ {
						worldX := 1*ChunkSize + x
						worldY := 1*ChunkSize + y
						noise.SetNoiseValue(int(worldX), int(worldY), 100.0, -0.5)
						noise.SetNoiseValue(int(worldX), int(worldY), 20.0, -0.1)
					}
				}
			},
			setupWorld: func(world *MockWorldService) {
				world.SetSeed(54321)
			},
			expectError: false,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(1), chunk.ChunkX)
				assert.Equal(t, int32(1), chunk.ChunkY)
				assert.Equal(t, int64(54321), chunk.Seed)
				
				// All cells should be water with configured noise values
				for _, cell := range chunk.Cells {
					assert.Equal(t, chunkV1.TerrainType_TERRAIN_TYPE_WATER, cell.TerrainType)
				}
			},
		},
		{
			name:   "chunk generation with mixed terrain types",
			chunkX: 2,
			chunkY: 2,
			setupNoise: func(noise *MockNoiseGenerator) {
				// Create varied terrain by setting different noise values
				for y := int32(0); y < ChunkSize; y++ {
					for x := int32(0); x < ChunkSize; x++ {
						worldX := 2*ChunkSize + x
						worldY := 2*ChunkSize + y
						
						if x < ChunkSize/4 {
							// Water area
							noise.SetNoiseValue(int(worldX), int(worldY), 100.0, -0.5)
							noise.SetNoiseValue(int(worldX), int(worldY), 20.0, -0.1)
						} else if x < ChunkSize/2 {
							// Sand area (combined noise < -0.1)
							noise.SetNoiseValue(int(worldX), int(worldY), 100.0, -0.15)
							noise.SetNoiseValue(int(worldX), int(worldY), 20.0, 0.0)
						} else if x < 3*ChunkSize/4 {
							// Grass area (combined noise < 0.2)
							noise.SetNoiseValue(int(worldX), int(worldY), 100.0, 0.1)
							noise.SetNoiseValue(int(worldX), int(worldY), 20.0, 0.0)
						} else {
							// Stone area (combined noise >= 0.5)
							noise.SetNoiseValue(int(worldX), int(worldY), 100.0, 0.7)
							noise.SetNoiseValue(int(worldX), int(worldY), 20.0, 0.1)
						}
					}
				}
			},
			setupWorld: func(world *MockWorldService) {
				world.SetSeed(99999)
			},
			expectError: false,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(2), chunk.ChunkX)
				assert.Equal(t, int32(2), chunk.ChunkY)
				
				// Count terrain types to verify variety
				terrainCounts := make(map[chunkV1.TerrainType]int)
				for _, cell := range chunk.Cells {
					terrainCounts[cell.TerrainType]++
				}
				
				// Should have multiple terrain types
				assert.True(t, len(terrainCounts) > 1, "Should have multiple terrain types")
				assert.Contains(t, terrainCounts, chunkV1.TerrainType_TERRAIN_TYPE_WATER)
				assert.Contains(t, terrainCounts, chunkV1.TerrainType_TERRAIN_TYPE_SAND)
				assert.Contains(t, terrainCounts, chunkV1.TerrainType_TERRAIN_TYPE_GRASS)
				assert.Contains(t, terrainCounts, chunkV1.TerrainType_TERRAIN_TYPE_STONE)
			},
		},
		{
			name:   "world service error",
			chunkX: 0,
			chunkY: 0,
			setupNoise: func(noise *MockNoiseGenerator) {
				// Normal noise setup
			},
			setupWorld: func(world *MockWorldService) {
				world.SetShouldError(true)
			},
			expectError: true,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				// Should not be called due to error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDatabase()
			noise := NewMockNoiseGenerator(12345)
			world := NewMockWorldService()
			resources := NewMockResourceNodeIntegration()
			logger := NewMockLogger()

			tt.setupNoise(noise)
			tt.setupWorld(world)

			service := NewService(db, noise, world, resources, logger)
			ctx := testutil.CreateTestContext()

			chunk, err := service.GenerateChunk(ctx, tt.chunkX, tt.chunkY)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, chunk)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, chunk)
				tt.validateChunk(t, chunk)
			}
		})
	}
}

func TestService_GetOrCreateChunk(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name              string
		chunkX, chunkY    int32
		setupDatabase     func(db *MockDatabaseInterface, world *MockWorldService)
		setupResources    func(resources *MockResourceNodeIntegration)
		setupWorld        func(world *MockWorldService)
		expectError       bool
		expectGeneration  bool
		expectAttachment  bool
		validateChunk     func(t *testing.T, chunk *chunkV1.ChunkData)
	}{
		{
			name:   "retrieve existing chunk from database",
			chunkX: 0,
			chunkY: 0,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// Pre-create chunk data
				chunkData := &chunkV1.ChunkData{
					ChunkX: 0,
					ChunkY: 0,
					Cells:  make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize),
					Seed:   12345,
					GeneratedAt: timestamppb.New(time.Now()),
				}
				for i := range chunkData.Cells {
					chunkData.Cells[i] = &chunkV1.TerrainCell{
						TerrainType: chunkV1.TerrainType_TERRAIN_TYPE_GRASS,
					}
				}

				serialized, _ := proto.Marshal(chunkData)
				db.AddChunk(world.defaultWorld.ID, 0, 0, serialized)
			},
			setupResources: func(resources *MockResourceNodeIntegration) {
				// Should attach existing resources
				testResources := []*resourceNodeV1.ResourceNode{
					{
						Id:      1,
						ChunkX:  0,
						ChunkY:  0,
						X:    10,
						Y:    15,
						Size:    2,
						ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_METAL_ORE,
						ResourceNodeType: &resourceNodeV1.ResourceNodeType{
							Id:   1,
							Name: "Iron",
						},
					},
				}
				resources.SetResourceNodes(testResources)
			},
			setupWorld: func(world *MockWorldService) {
				// Default setup
			},
			expectError:      false,
			expectGeneration: false,
			expectAttachment: true,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(0), chunk.ChunkX)
				assert.Equal(t, int32(0), chunk.ChunkY)
				assert.Len(t, chunk.ResourceNodes, 1)
				assert.Equal(t, "Iron", chunk.ResourceNodes[0].ResourceNodeType.Name)
			},
		},
		{
			name:   "generate new chunk when not in database",
			chunkX: 1,
			chunkY: 1,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// No pre-existing chunk
			},
			setupResources: func(resources *MockResourceNodeIntegration) {
				// Should generate new resources
				testResources := []*resourceNodeV1.ResourceNode{
					{
						Id:      2,
						ChunkX:  1,
						ChunkY:  1,
						X:    5,
						Y:    25,
						Size:    3,
						ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_STONE_VEIN,
						ResourceNodeType: &resourceNodeV1.ResourceNodeType{
							Id:   2,
							Name: "Coal",
						},
					},
				}
				resources.SetResourceNodes(testResources)
			},
			setupWorld: func(world *MockWorldService) {
				world.SetSeed(54321)
			},
			expectError:      false,
			expectGeneration: true,
			expectAttachment: false,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(1), chunk.ChunkX)
				assert.Equal(t, int32(1), chunk.ChunkY)
				assert.Equal(t, int64(54321), chunk.Seed)
				assert.Len(t, chunk.ResourceNodes, 1)
				assert.Equal(t, "Coal", chunk.ResourceNodes[0].ResourceNodeType.Name)
			},
		},
		{
			name:   "database error during chunk retrieval",
			chunkX: 2,
			chunkY: 2,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				db.SetShouldReturnError(true)
			},
			setupResources: func(resources *MockResourceNodeIntegration) {
				// Setup but won't be called due to DB error
			},
			setupWorld: func(world *MockWorldService) {
				// Default setup
			},
			expectError:      true,
			expectGeneration: false,
			expectAttachment: false,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				// Should not be called due to error
			},
		},
		{
			name:   "resource attachment failure on existing chunk",
			chunkX: 3,
			chunkY: 3,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// Pre-create chunk
				chunkData := &chunkV1.ChunkData{
					ChunkX: 3,
					ChunkY: 3,
					Cells:  make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize),
					Seed:   12345,
					GeneratedAt: timestamppb.New(time.Now()),
				}
				for i := range chunkData.Cells {
					chunkData.Cells[i] = &chunkV1.TerrainCell{
						TerrainType: chunkV1.TerrainType_TERRAIN_TYPE_GRASS,
					}
				}

				serialized, _ := proto.Marshal(chunkData)
				db.AddChunk(world.defaultWorld.ID, 3, 3, serialized)
			},
			setupResources: func(resources *MockResourceNodeIntegration) {
				resources.SetShouldError(true)
			},
			setupWorld: func(world *MockWorldService) {
				// Default setup
			},
			expectError:      false, // Resource attachment failure doesn't fail chunk retrieval
			expectGeneration: false,
			expectAttachment: true,
			validateChunk: func(t *testing.T, chunk *chunkV1.ChunkData) {
				assert.Equal(t, int32(3), chunk.ChunkX)
				assert.Equal(t, int32(3), chunk.ChunkY)
				// Resource attachment failed, so no resources should be attached
				assert.Len(t, chunk.ResourceNodes, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDatabase()
			noise := NewMockNoiseGenerator(12345)
			world := NewMockWorldService()
			resources := NewMockResourceNodeIntegration()
			logger := NewMockLogger()

			tt.setupDatabase(db, world)
			tt.setupResources(resources)
			tt.setupWorld(world)

			service := NewService(db, noise, world, resources, logger)
			ctx := testutil.CreateTestContext()

			chunk, err := service.GetOrCreateChunk(ctx, tt.chunkX, tt.chunkY)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, chunk)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, chunk)
				tt.validateChunk(t, chunk)

				if tt.expectGeneration {
					assert.Equal(t, 1, resources.GetGenerateCallCount())
					assert.Equal(t, 1, db.GetCreateCallCount())
				} else {
					assert.Equal(t, 0, resources.GetGenerateCallCount())
				}

				if tt.expectAttachment {
					assert.Equal(t, 1, resources.GetAttachCallCount())
				}
			}
		})
	}
}

func TestService_GetTerrainType(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		x, y           int32
		setupNoise     func(noise *MockNoiseGenerator)
		expectedTerrain chunkV1.TerrainType
	}{
		{
			name: "water terrain from low noise values",
			x:    10,
			y:    10,
			setupNoise: func(noise *MockNoiseGenerator) {
				noise.SetNoiseValue(10, 10, 100.0, -0.5) // Low elevation
				noise.SetNoiseValue(10, 10, 20.0, -0.1)  // Low detail
			},
			expectedTerrain: chunkV1.TerrainType_TERRAIN_TYPE_WATER,
		},
		{
			name: "sand terrain from slightly higher noise",
			x:    20,
			y:    20,
			setupNoise: func(noise *MockNoiseGenerator) {
				noise.SetNoiseValue(20, 20, 100.0, -0.2) // Medium-low elevation
				noise.SetNoiseValue(20, 20, 20.0, 0.0)   // Neutral detail
			},
			expectedTerrain: chunkV1.TerrainType_TERRAIN_TYPE_SAND,
		},
		{
			name: "grass terrain from moderate noise",
			x:    30,
			y:    30,
			setupNoise: func(noise *MockNoiseGenerator) {
				noise.SetNoiseValue(30, 30, 100.0, 0.1) // Medium elevation
				noise.SetNoiseValue(30, 30, 20.0, 0.0)  // Neutral detail
			},
			expectedTerrain: chunkV1.TerrainType_TERRAIN_TYPE_GRASS,
		},
		{
			name: "dirt terrain from higher noise",
			x:    40,
			y:    40,
			setupNoise: func(noise *MockNoiseGenerator) {
				noise.SetNoiseValue(40, 40, 100.0, 0.3) // Higher elevation
				noise.SetNoiseValue(40, 40, 20.0, 0.1)  // Slight detail
			},
			expectedTerrain: chunkV1.TerrainType_TERRAIN_TYPE_DIRT,
		},
		{
			name: "stone terrain from high noise values",
			x:    50,
			y:    50,
			setupNoise: func(noise *MockNoiseGenerator) {
				noise.SetNoiseValue(50, 50, 100.0, 0.7) // High elevation
				noise.SetNoiseValue(50, 50, 20.0, 0.1)  // Slight detail
			},
			expectedTerrain: chunkV1.TerrainType_TERRAIN_TYPE_STONE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDatabase()
			noise := NewMockNoiseGenerator(12345)
			world := NewMockWorldService()
			resources := NewMockResourceNodeIntegration()
			logger := NewMockLogger()

			tt.setupNoise(noise)

			service := NewService(db, noise, world, resources, logger)

			terrainType := service.getTerrainType(tt.x, tt.y)
			assert.Equal(t, tt.expectedTerrain, terrainType)
		})
	}
}

func TestService_GetChunksInRange(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name                      string
		minX, maxX, minY, maxY    int32
		setupDatabase             func(db *MockDatabaseInterface, world *MockWorldService)
		expectError               bool
		expectedChunkCount        int
		validateCoordinates       func(t *testing.T, chunks []*chunkV1.ChunkData)
	}{
		{
			name: "get 2x2 range of chunks",
			minX: 0, maxX: 1, minY: 0, maxY: 1,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// No pre-existing chunks, all will be generated
			},
			expectError:        false,
			expectedChunkCount: 4,
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				coords := make(map[string]bool)
				for _, chunk := range chunks {
					key := fmt.Sprintf("%d,%d", chunk.ChunkX, chunk.ChunkY)
					coords[key] = true
				}
				assert.True(t, coords["0,0"])
				assert.True(t, coords["0,1"])
				assert.True(t, coords["1,0"])
				assert.True(t, coords["1,1"])
			},
		},
		{
			name: "get single chunk range",
			minX: 5, maxX: 5, minY: 5, maxY: 5,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// No setup needed
			},
			expectError:        false,
			expectedChunkCount: 1,
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				assert.Equal(t, int32(5), chunks[0].ChunkX)
				assert.Equal(t, int32(5), chunks[0].ChunkY)
			},
		},
		{
			name: "empty range",
			minX: 0, maxX: -1, minY: 0, maxY: 0,
			setupDatabase: func(db *MockDatabaseInterface, world *MockWorldService) {
				// No setup needed
			},
			expectError:        false,
			expectedChunkCount: 0,
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				// No validation needed for empty result
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDatabase()
			noise := NewMockNoiseGenerator(12345)
			world := NewMockWorldService()
			resources := NewMockResourceNodeIntegration()
			logger := NewMockLogger()

			tt.setupDatabase(db, world)

			service := NewService(db, noise, world, resources, logger)
			ctx := testutil.CreateTestContext()

			chunks, err := service.GetChunksInRange(ctx, tt.minX, tt.maxX, tt.minY, tt.maxY)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, chunks)
			} else {
				assert.NoError(t, err)
				assert.Len(t, chunks, tt.expectedChunkCount)
				if tt.expectedChunkCount > 0 {
					tt.validateCoordinates(t, chunks)
				}
			}
		})
	}
}

func TestService_GetChunksInRadius(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name                     string
		centerX, centerY, radius int32
		expectError              bool
		expectedChunkCount       int
		validateCoordinates      func(t *testing.T, chunks []*chunkV1.ChunkData)
	}{
		{
			name: "radius 1 around center (0,0)",
			centerX: 0, centerY: 0, radius: 1,
			expectError:        false,
			expectedChunkCount: 5, // center + 4 adjacent chunks
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				coords := make(map[string]bool)
				for _, chunk := range chunks {
					key := fmt.Sprintf("%d,%d", chunk.ChunkX, chunk.ChunkY)
					coords[key] = true
				}
				// Center and adjacent chunks (Manhattan distance <= 1)
				assert.True(t, coords["0,0"])   // center
				assert.True(t, coords["-1,0"])  // left
				assert.True(t, coords["1,0"])   // right
				assert.True(t, coords["0,-1"])  // up
				assert.True(t, coords["0,1"])   // down
			},
		},
		{
			name: "radius 0 (only center chunk)",
			centerX: 5, centerY: 5, radius: 0,
			expectError:        false,
			expectedChunkCount: 1,
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				assert.Equal(t, int32(5), chunks[0].ChunkX)
				assert.Equal(t, int32(5), chunks[0].ChunkY)
			},
		},
		{
			name: "radius 2 around center (0,0)",
			centerX: 0, centerY: 0, radius: 2,
			expectError:        false,
			expectedChunkCount: 13, // center + all chunks within Manhattan distance 2
			validateCoordinates: func(t *testing.T, chunks []*chunkV1.ChunkData) {
				coords := make(map[string]bool)
				for _, chunk := range chunks {
					key := fmt.Sprintf("%d,%d", chunk.ChunkX, chunk.ChunkY)
					coords[key] = true
					
					// Verify Manhattan distance is <= 2
					distance := abs(chunk.ChunkX-0) + abs(chunk.ChunkY-0)
					assert.LessOrEqual(t, distance, int32(2))
				}
				
				// Should include center and some specific chunks
				assert.True(t, coords["0,0"])   // center
				assert.True(t, coords["2,0"])   // radius 2 on x-axis
				assert.True(t, coords["-2,0"])  // radius 2 on negative x-axis
				assert.True(t, coords["0,2"])   // radius 2 on y-axis
				assert.True(t, coords["0,-2"])  // radius 2 on negative y-axis
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDatabase()
			noise := NewMockNoiseGenerator(12345)
			world := NewMockWorldService()
			resources := NewMockResourceNodeIntegration()
			logger := NewMockLogger()

			service := NewService(db, noise, world, resources, logger)
			ctx := testutil.CreateTestContext()

			chunks, err := service.GetChunksInRadius(ctx, tt.centerX, tt.centerY, tt.radius)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, chunks)
			} else {
				assert.NoError(t, err)
				assert.Len(t, chunks, tt.expectedChunkCount)
				if tt.expectedChunkCount > 0 {
					tt.validateCoordinates(t, chunks)
				}
			}
		})
	}
}

func TestService_ChunkWorkerConcurrency(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Test that the chunk worker system handles concurrent chunk generation properly
	db := NewMockDatabase()
	noise := NewMockNoiseGenerator(12345)
	world := NewMockWorldService()
	resources := NewMockResourceNodeIntegration()
	logger := NewMockLogger()

	service := NewService(db, noise, world, resources, logger)
	ctx := testutil.CreateTestContext()

	// Request a large range to trigger parallel processing
	chunks, err := service.GetChunksInRange(ctx, 0, 3, 0, 3) // 4x4 = 16 chunks

	assert.NoError(t, err)
	assert.Len(t, chunks, 16)

	// Verify all chunks were generated correctly
	coords := make(map[string]bool)
	for _, chunk := range chunks {
		key := fmt.Sprintf("%d,%d", chunk.ChunkX, chunk.ChunkY)
		coords[key] = true
		
		// Each chunk should have the correct structure
		assert.Len(t, chunk.Cells, ChunkSize*ChunkSize)
		assert.NotNil(t, chunk.GeneratedAt)
	}

	// Verify we got all expected coordinates
	for x := int32(0); x <= 3; x++ {
		for y := int32(0); y <= 3; y++ {
			key := fmt.Sprintf("%d,%d", x, y)
			assert.True(t, coords[key], "Missing chunk at %s", key)
		}
	}
}

func TestService_ErrorHandling(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name        string
		setupMocks  func() (*MockDatabaseInterface, *MockNoiseGenerator, *MockWorldService, *MockResourceNodeIntegration, *MockLogger)
		operation   func(service *Service, ctx context.Context) error
		expectError bool
		errorCheck  func(t *testing.T, err error)
	}{
		{
			name: "world service error in GenerateChunk",
			setupMocks: func() (*MockDatabaseInterface, *MockNoiseGenerator, *MockWorldService, *MockResourceNodeIntegration, *MockLogger) {
				db := NewMockDatabase()
				noise := NewMockNoiseGenerator(12345)
				world := NewMockWorldService()
				world.SetShouldError(true)
				resources := NewMockResourceNodeIntegration()
				logger := NewMockLogger()
				return db, noise, world, resources, logger
			},
			operation: func(service *Service, ctx context.Context) error {
				_, err := service.GenerateChunk(ctx, 0, 0)
				return err
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to get default world")
			},
		},
		{
			name: "database error in GetOrCreateChunk save",
			setupMocks: func() (*MockDatabaseInterface, *MockNoiseGenerator, *MockWorldService, *MockResourceNodeIntegration, *MockLogger) {
				db := NewMockDatabase()
				noise := NewMockNoiseGenerator(12345)
				world := NewMockWorldService()
				resources := NewMockResourceNodeIntegration()
				logger := NewMockLogger()
				
				// Set DB to error on create operations (save)
				db.SetShouldReturnError(true)
				return db, noise, world, resources, logger
			},
			operation: func(service *Service, ctx context.Context) error {
				_, err := service.GetOrCreateChunk(ctx, 0, 0)
				return err
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to save chunk")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, noise, world, resources, logger := tt.setupMocks()
			service := NewService(db, noise, world, resources, logger)
			ctx := testutil.CreateTestContext()

			err := tt.operation(service, ctx)

			if tt.expectError {
				assert.Error(t, err)
				tt.errorCheck(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}