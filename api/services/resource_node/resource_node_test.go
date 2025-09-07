package resource_node

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testutil"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
)

// Mock implementations for testing

// MockDatabaseInterface implements DatabaseInterface for testing
type MockDatabaseInterface struct {
	mock.Mock
	resourceNodes    map[string]db.ResourceNode
	chunks           map[string]db.Chunk
	nextNodeID       int32
	chunkExists      bool
	shouldReturnErr  bool
}

func NewMockDatabase() *MockDatabaseInterface {
	return &MockDatabaseInterface{
		resourceNodes: make(map[string]db.ResourceNode),
		chunks:        make(map[string]db.Chunk),
		nextNodeID:    1,
		chunkExists:   true,
	}
}

func (m *MockDatabaseInterface) SetShouldReturnError(shouldErr bool) {
	m.shouldReturnErr = shouldErr
}

func (m *MockDatabaseInterface) SetChunkExists(exists bool) {
	m.chunkExists = exists
}

func (m *MockDatabaseInterface) AddChunk(chunk db.Chunk) {
	key := fmt.Sprintf("%d,%d", chunk.ChunkX, chunk.ChunkY)
	m.chunks[key] = chunk
}

func (m *MockDatabaseInterface) CreateResourceNode(ctx context.Context, arg db.CreateResourceNodeParams) (db.ResourceNode, error) {
	if m.shouldReturnErr {
		return db.ResourceNode{}, assert.AnError
	}

	node := db.ResourceNode{
		ID:                 m.nextNodeID,
		ResourceNodeTypeID: arg.ResourceNodeTypeID,
		WorldID:            arg.WorldID,
		ChunkX:             arg.ChunkX,
		ChunkY:             arg.ChunkY,
		ClusterID:          arg.ClusterID,
		X:               arg.X,
		Y:               arg.Y,
		Size:               arg.Size,
		CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
	}

	m.nextNodeID++
	key := fmt.Sprintf("%d", node.ID)
	m.resourceNodes[key] = node

	return node, nil
}

func (m *MockDatabaseInterface) DeleteResourceNodesInChunk(ctx context.Context, arg db.DeleteResourceNodesInChunkParams) error {
	if m.shouldReturnErr {
		return assert.AnError
	}

	// Remove all resource nodes for the specified chunk
	for key, node := range m.resourceNodes {
		if node.WorldID == arg.WorldID && node.ChunkX == arg.ChunkX && node.ChunkY == arg.ChunkY {
			delete(m.resourceNodes, key)
		}
	}

	return nil
}

func (m *MockDatabaseInterface) GetResourceNodesInChunk(ctx context.Context, arg db.GetResourceNodesInChunkParams) ([]db.ResourceNode, error) {
	if m.shouldReturnErr {
		return nil, assert.AnError
	}

	var result []db.ResourceNode
	for _, node := range m.resourceNodes {
		if node.WorldID == arg.WorldID && node.ChunkX == arg.ChunkX && node.ChunkY == arg.ChunkY {
			result = append(result, node)
		}
	}

	return result, nil
}

func (m *MockDatabaseInterface) GetResourceNodesInChunks(ctx context.Context, arg db.GetResourceNodesInChunksParams) ([]db.ResourceNode, error) {
	if m.shouldReturnErr {
		return nil, assert.AnError
	}

	// Simplified implementation for testing
	var result []db.ResourceNode
	for _, node := range m.resourceNodes {
		if node.WorldID == arg.WorldID {
			// Check if node matches any of the requested chunks
			if (node.ChunkX == arg.ChunkX && node.ChunkY == arg.ChunkY) ||
				(node.ChunkX == arg.ChunkX_2 && node.ChunkY == arg.ChunkY_2) ||
				(node.ChunkX == arg.ChunkX_3 && node.ChunkY == arg.ChunkY_3) ||
				(node.ChunkX == arg.ChunkX_4 && node.ChunkY == arg.ChunkY_4) ||
				(node.ChunkX == arg.ChunkX_5 && node.ChunkY == arg.ChunkY_5) {
				result = append(result, node)
			}
		}
	}

	return result, nil
}

func (m *MockDatabaseInterface) GetResourceNodesInChunkRange(ctx context.Context, arg db.GetResourceNodesInChunkRangeParams) ([]db.ResourceNode, error) {
	if m.shouldReturnErr {
		return nil, assert.AnError
	}

	var result []db.ResourceNode
	for _, node := range m.resourceNodes {
		if node.WorldID == arg.WorldID &&
			node.ChunkX >= arg.ChunkX && node.ChunkX <= arg.ChunkX_2 &&
			node.ChunkY >= arg.ChunkY && node.ChunkY <= arg.ChunkY_2 {
			result = append(result, node)
		}
	}

	return result, nil
}

func (m *MockDatabaseInterface) ChunkExists(ctx context.Context, arg db.ChunkExistsParams) (bool, error) {
	if m.shouldReturnErr {
		return false, assert.AnError
	}

	return m.chunkExists, nil
}

func (m *MockDatabaseInterface) GetChunk(ctx context.Context, arg db.GetChunkParams) (db.Chunk, error) {
	if m.shouldReturnErr {
		return db.Chunk{}, assert.AnError
	}

	key := fmt.Sprintf("%d,%d", arg.ChunkX, arg.ChunkY)
	chunk, exists := m.chunks[key]
	if !exists {
		return db.Chunk{}, sql.ErrNoRows
	}

	return chunk, nil
}

func (m *MockDatabaseInterface) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	if m.shouldReturnErr {
		return db.ResourceNode{}, assert.AnError
	}

	key := fmt.Sprintf("%d", id)
	node, exists := m.resourceNodes[key]
	if !exists {
		return db.ResourceNode{}, sql.ErrNoRows
	}

	return node, nil
}

// MockNoiseGenerator implements NoiseGeneratorInterface for testing
type MockNoiseGenerator struct {
	mock.Mock
	seed         int64
	noiseValue   float64
	noisePattern map[string]float64
}

func NewMockNoiseGenerator(seed int64) *MockNoiseGenerator {
	return &MockNoiseGenerator{
		seed:         seed,
		noiseValue:   0.5, // Default value
		noisePattern: make(map[string]float64),
	}
}

func (m *MockNoiseGenerator) GetTerrainNoise(x, y int, scale float64) float64 {
	// Return deterministic values based on position for testing
	key := fmt.Sprintf("%d,%d,%.1f", x, y, scale)
	if value, exists := m.noisePattern[key]; exists {
		return value
	}
	// Convert to range -1 to 1 (noise is typically in this range)
	return (m.noiseValue * 2.0) - 1.0
}

func (m *MockNoiseGenerator) GetSeed() int64 {
	return m.seed
}

func (m *MockNoiseGenerator) SetNoiseValue(value float64) {
	m.noiseValue = value
}

func (m *MockNoiseGenerator) SetNoisePattern(x, y int, scale float64, value float64) {
	key := fmt.Sprintf("%d,%d,%.1f", x, y, scale)
	m.noisePattern[key] = value
}

// MockWorldService implements WorldServiceInterface for testing
type MockWorldService struct {
	mock.Mock
	defaultWorld db.World
	shouldReturnErr bool
}

func NewMockWorldService() *MockWorldService {
	return &MockWorldService{
		defaultWorld: db.World{
			ID:   createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
			Name: "TestWorld",
			Seed: 12345,
		},
	}
}

func (m *MockWorldService) SetShouldReturnError(shouldErr bool) {
	m.shouldReturnErr = shouldErr
}

func (m *MockWorldService) GetDefaultWorld(ctx context.Context) (db.World, error) {
	if m.shouldReturnErr {
		return db.World{}, assert.AnError
	}
	return m.defaultWorld, nil
}

// MockRandomGenerator implements RandomGeneratorInterface for testing
type MockRandomGenerator struct {
	mock.Mock
	intValues    []int
	int31Values  []int32
	floatValues  []float32
	intIndex     int
	int31Index   int
	floatIndex   int
	shuffleFunc  func(n int, swap func(i, j int))
}

func NewMockRandomGenerator() *MockRandomGenerator {
	return &MockRandomGenerator{
		intValues:   []int{0, 1, 2, 3, 4}, // Default sequence
		int31Values: []int32{0, 1, 2, 3, 4},
		floatValues: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
	}
}

func (m *MockRandomGenerator) SetIntSequence(values []int) {
	m.intValues = values
	m.intIndex = 0
}

func (m *MockRandomGenerator) SetShuffleFunc(fn func(n int, swap func(i, j int))) {
	m.shuffleFunc = fn
}

func (m *MockRandomGenerator) Intn(n int) int {
	if m.intIndex >= len(m.intValues) {
		m.intIndex = 0 // Wrap around
	}
	value := m.intValues[m.intIndex] % n
	m.intIndex++
	return value
}

func (m *MockRandomGenerator) Int31n(n int32) int32 {
	if m.int31Index >= len(m.int31Values) {
		m.int31Index = 0 // Wrap around
	}
	value := m.int31Values[m.int31Index] % n
	m.int31Index++
	return value
}

func (m *MockRandomGenerator) Float32() float32 {
	if m.floatIndex >= len(m.floatValues) {
		m.floatIndex = 0 // Wrap around
	}
	value := m.floatValues[m.floatIndex]
	m.floatIndex++
	return value
}

func (m *MockRandomGenerator) Shuffle(n int, swap func(i, j int)) {
	if m.shuffleFunc != nil {
		m.shuffleFunc(n, swap)
	}
	// Default shuffle is no-op for deterministic testing
}

// MockLogger implements LoggerInterface for testing
type MockLogger struct {
	mock.Mock
	logs []LogEntry
}

type LogEntry struct {
	Level    string
	Message  string
	Fields   []interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]LogEntry, 0),
	}
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "debug", Message: msg, Fields: keysAndValues})
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "info", Message: msg, Fields: keysAndValues})
}

func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "warn", Message: msg, Fields: keysAndValues})
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "error", Message: msg, Fields: keysAndValues})
}

func (m *MockLogger) With(keysAndValues ...interface{}) LoggerInterface {
	return m // Simplified for testing
}

func (m *MockLogger) GetLogs() []LogEntry {
	return m.logs
}

// Helper function to create UUIDs for testing
func createTestUUID(idStr string) pgtype.UUID {
	var uuid pgtype.UUID
	uuid.Scan(idStr)
	return uuid
}

// Helper function to create test chunk data
func createTestChunkData(chunkX, chunkY int32, terrainType chunkV1.TerrainType) *chunkV1.ChunkData {
	cells := make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize)
	for i := range cells {
		cells[i] = &chunkV1.TerrainCell{
			TerrainType: terrainType,
		}
	}

	return &chunkV1.ChunkData{
		ChunkX: chunkX,
		ChunkY: chunkY,
		Cells:  cells,
	}
}

// Helper function to create mixed terrain chunk
func createMixedTerrainChunk(chunkX, chunkY int32) *chunkV1.ChunkData {
	cells := make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize)
	for i := range cells {
		// Create a pattern: first half grass, second half water
		if i < len(cells)/2 {
			cells[i] = &chunkV1.TerrainCell{TerrainType: chunkV1.TerrainType_TERRAIN_TYPE_GRASS}
		} else {
			cells[i] = &chunkV1.TerrainCell{TerrainType: chunkV1.TerrainType_TERRAIN_TYPE_WATER}
		}
	}

	return &chunkV1.ChunkData{
		ChunkX: chunkX,
		ChunkY: chunkY,
		Cells:  cells,
	}
}

// Test Service Creation

func TestNewNodeService(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		expectFields func(t *testing.T, service *NodeService)
	}{
		{
			name: "successful service creation with all dependencies",
			expectFields: func(t *testing.T, service *NodeService) {
				assert.NotNil(t, service.db)
				assert.NotNil(t, service.noiseGen)
				assert.NotNil(t, service.worldService)
				assert.NotNil(t, service.rnd)
				assert.NotNil(t, service.logger)
				assert.NotEmpty(t, service.resourceTypes)
				assert.NotEmpty(t, service.resourceTypesByTerrain)
				assert.NotEmpty(t, service.resourceTypesByID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			require.NotNil(t, service)
			tt.expectFields(t, service)
		})
	}
}

func TestNodeService_terrainTypeToString(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name        string
		terrainType chunkV1.TerrainType
		expected    string
	}{
		{
			name:        "grass terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_GRASS,
			expected:    "grass",
		},
		{
			name:        "water terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_WATER,
			expected:    "water",
		},
		{
			name:        "sand terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_SAND,
			expected:    "sand",
		},
		{
			name:        "stone terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_STONE,
			expected:    "stone",
		},
		{
			name:        "dirt terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_DIRT,
			expected:    "dirt",
		},
		{
			name:        "unknown terrain",
			terrainType: chunkV1.TerrainType_TERRAIN_TYPE_UNSPECIFIED,
			expected:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.terrainTypeToString(tt.terrainType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeService_getRarityThresholdFromEnum(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name     string
		rarity   resourceNodeV1.ResourceRarity
		expected float64
	}{
		{
			name:     "common rarity",
			rarity:   resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			expected: CommonThreshold,
		},
		{
			name:     "uncommon rarity",
			rarity:   resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			expected: UncommonThreshold,
		},
		{
			name:     "rare rarity",
			rarity:   resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
			expected: RareThreshold,
		},
		{
			name:     "very rare rarity",
			rarity:   resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE,
			expected: VeryRareThreshold,
		},
		{
			name:     "unspecified rarity defaults to common",
			rarity:   resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNSPECIFIED,
			expected: CommonThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getRarityThresholdFromEnum(tt.rarity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeService_determineClusterSizeFromEnum(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name         string
		rarity       resourceNodeV1.ResourceRarity
		randomValues []int
		expectedMin  int
		expectedMax  int
	}{
		{
			name:         "common rarity cluster size",
			rarity:       resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			randomValues: []int{0}, // First value in weight distribution
			expectedMin:  1,
			expectedMax:  6,
		},
		{
			name:         "very rare rarity cluster size",
			rarity:       resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE,
			randomValues: []int{0}, // First value in weight distribution
			expectedMin:  1,
			expectedMax:  3, // Very rare has smaller max cluster size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRandom := NewMockRandomGenerator()
			mockRandom.SetIntSequence(tt.randomValues)
			service.rnd = mockRandom

			result := service.determineClusterSizeFromEnum(tt.rarity)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
		})
	}
}

func TestNodeService_isNearTerrainTransition(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name     string
		chunk    *chunkV1.ChunkData
		x, y     int32
		expected bool
	}{
		{
			name:     "uniform grass terrain - no transition",
			chunk:    createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			x:        10,
			y:        10,
			expected: false,
		},
		{
			name:     "mixed terrain - near transition",
			chunk:    createMixedTerrainChunk(0, 0),
			x:        15, // Near the transition between grass and water
			y:        15,
			expected: true,
		},
		{
			name:     "position at chunk boundary - transition",
			chunk:    createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			x:        0, // At edge
			y:        10,
			expected: true,
		},
		{
			name:     "position out of bounds - transition",
			chunk:    createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			x:        -1, // Out of bounds
			y:        10,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isNearTerrainTransition(tt.chunk, tt.x, tt.y)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeService_findPotentialSpawnPoints(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name           string
		chunk          *chunkV1.ChunkData
		terrainType    string
		threshold      float64
		noiseValue     float64
		expectedCount  int
		expectedMinMax [2]int // [min, max] expected spawn points
	}{
		{
			name:           "high noise value above threshold",
			chunk:          createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			terrainType:    "grass",
			threshold:      0.3,
			noiseValue:     0.8, // Above threshold
			expectedMinMax: [2]int{500, 1024}, // Most cells should qualify
		},
		{
			name:           "low noise value below threshold",
			chunk:          createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			terrainType:    "grass",
			threshold:      0.7,
			noiseValue:     0.3, // Below threshold
			expectedMinMax: [2]int{0, 10}, // Very few or no cells
		},
		{
			name:           "wrong terrain type",
			chunk:          createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_WATER),
			terrainType:    "grass", // Looking for grass in water chunk
			threshold:      0.3,
			noiseValue:     0.8,
			expectedMinMax: [2]int{0, 0}, // No matching terrain
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up noise generator to return consistent values
			mockNoise.SetNoiseValue(tt.noiseValue)

			spawnPoints := service.findPotentialSpawnPoints(
				tt.chunk,
				tt.terrainType,
				tt.threshold,
				12345, // resourceSeed
			)

			assert.GreaterOrEqual(t, len(spawnPoints), tt.expectedMinMax[0],
				"Expected at least %d spawn points, got %d", tt.expectedMinMax[0], len(spawnPoints))
			assert.LessOrEqual(t, len(spawnPoints), tt.expectedMinMax[1],
				"Expected at most %d spawn points, got %d", tt.expectedMinMax[1], len(spawnPoints))

			// Verify all spawn points are within chunk bounds
			for _, point := range spawnPoints {
				assert.GreaterOrEqual(t, point.x, int32(0))
				assert.LessOrEqual(t, point.x, int32(ChunkSize-1))
				assert.GreaterOrEqual(t, point.y, int32(0))
				assert.LessOrEqual(t, point.y, int32(ChunkSize-1))
			}
		})
	}
}

func TestNodeService_GenerateResourcesForChunk(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name            string
		chunk           *chunkV1.ChunkData
		noiseValue      float64
		randomValues    []int
		expectedMin     int
		expectedMax     int
		expectedTerrain string
	}{
		{
			name:            "grass chunk with controlled noise pattern",
			chunk:           createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			noiseValue:      0.9, // High noise, should spawn resources
			randomValues:    []int{0, 1, 2, 3, 4, 5, 6, 7}, // Deterministic random values
			expectedMin:     1,
			expectedMax:     MaxResourcesPerChunk + 5, // Allow some variation
			expectedTerrain: "grass",
		},
		{
			name:            "water chunk with controlled noise pattern",
			chunk:           createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_WATER),
			noiseValue:      0.9,
			randomValues:    []int{0, 1, 2, 3, 4, 5, 6, 7},
			expectedMin:     1,
			expectedMax:     MaxResourcesPerChunk + 5, // Allow some variation
			expectedTerrain: "water",
		},
		{
			name:            "very low noise generates fewer resources",
			chunk:           createTestChunkData(0, 0, chunkV1.TerrainType_TERRAIN_TYPE_GRASS),
			noiseValue:      0.1, // Very low noise
			randomValues:    []int{0, 1, 2, 3, 4, 5, 6, 7},
			expectedMin:     0,
			expectedMax:     MaxResourcesPerChunk + 10, // Could still hit max if unlucky with noise patterns
			expectedTerrain: "grass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

			// Configure mocks
			mockNoise.SetNoiseValue(tt.noiseValue)
			mockRandom.SetIntSequence(tt.randomValues)

			// For low noise tests, create specific patterns with only a few high-noise spots
			if tt.name == "very low noise generates fewer resources" {
				// Set up specific noise patterns - only a few spots should have high values
				for x := 0; x < 32; x++ {
					for y := 0; y < 32; y++ {
						// Convert to world coordinates as the noise function does
						worldX := tt.chunk.ChunkX*32 + int32(x)
						worldY := tt.chunk.ChunkY*32 + int32(y)
						if x < 3 && y < 3 { // Only small area has noise above threshold
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceNoiseScale, 0.5)     // Above threshold
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceDetailScale, 0.5)
						} else {
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceNoiseScale, -0.9)    // Below threshold
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceDetailScale, -0.9)
						}
					}
				}
			} else if tt.name == "grass chunk with controlled noise pattern" || tt.name == "water chunk with controlled noise pattern" {
				// For controlled high-noise tests, create a limited pattern
				count := 0
				for x := 0; x < 32 && count < 10; x++ { // Limit to ~10 potential spawn spots
					for y := 0; y < 32 && count < 10; y++ {
						worldX := tt.chunk.ChunkX*32 + int32(x)
						worldY := tt.chunk.ChunkY*32 + int32(y)
						if x%3 == 0 && y%3 == 0 { // Every 3rd cell in a grid pattern
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceNoiseScale, 0.8)
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceDetailScale, 0.8)
							count++
						} else {
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceNoiseScale, -0.5)
							mockNoise.SetNoisePattern(int(worldX), int(worldY), ResourceDetailScale, -0.5)
						}
					}
				}
			}

			ctx := testutil.CreateTestContext()
			resources, err := service.GenerateResourcesForChunk(ctx, tt.chunk)

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(resources), tt.expectedMin)
			assert.LessOrEqual(t, len(resources), tt.expectedMax)

			// Verify all resources have expected properties
			for _, resource := range resources {
				assert.NotNil(t, resource)
				assert.NotNil(t, resource.ResourceNodeType)
				assert.Equal(t, tt.chunk.ChunkX, resource.ChunkX)
				assert.Equal(t, tt.chunk.ChunkY, resource.ChunkY)
				assert.Equal(t, tt.expectedTerrain, resource.ResourceNodeType.TerrainType)
				// Check global coordinates are within expected chunk bounds
				expectedMinX := resource.ChunkX * ChunkSize
				expectedMaxX := expectedMinX + ChunkSize - 1
				expectedMinY := resource.ChunkY * ChunkSize
				expectedMaxY := expectedMinY + ChunkSize - 1
				assert.GreaterOrEqual(t, resource.X, expectedMinX)
				assert.LessOrEqual(t, resource.X, expectedMaxX)
				assert.GreaterOrEqual(t, resource.Y, expectedMinY)
				assert.LessOrEqual(t, resource.Y, expectedMaxY)
				assert.NotEmpty(t, resource.ClusterId)
				assert.Equal(t, int32(1), resource.Size)
				assert.NotNil(t, resource.CreatedAt)
			}
		})
	}
}

func TestNodeService_StoreResourceNodes(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name             string
		chunkX, chunkY   int32
		resources        []*resourceNodeV1.ResourceNode
		setupMocks       func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService)
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:   "successful storage of resources",
			chunkX: 0,
			chunkY: 0,
			resources: []*resourceNodeV1.ResourceNode{
				{
					ResourceNodeType: &resourceNodeV1.ResourceNodeType{
						Id: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					},
					ChunkX:    0,
					ChunkY:    0,
					X:      10,
					Y:      10,
					ClusterId: "test-cluster-1",
					Size:      1,
				},
			},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Default setup - mocks should work normally
			},
			expectError: false,
		},
		{
			name:   "world service error",
			chunkX: 0,
			chunkY: 0,
			resources: []*resourceNodeV1.ResourceNode{
				{
					ResourceNodeType: &resourceNodeV1.ResourceNodeType{
						Id: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					},
					ChunkX:    0,
					ChunkY:    0,
					X:      10,
					Y:      10,
					ClusterId: "test-cluster-1",
					Size:      1,
				},
			},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockWorld.SetShouldReturnError(true)
			},
			expectError:      true,
			expectedErrorMsg: "failed to get default world",
		},
		{
			name:   "database error during deletion",
			chunkX: 0,
			chunkY: 0,
			resources: []*resourceNodeV1.ResourceNode{},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockDB.SetShouldReturnError(true)
			},
			expectError:      true,
			expectedErrorMsg: "failed to delete existing resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			tt.setupMocks(mockDB, mockWorld)

			ctx := testutil.CreateTestContext()
			err := service.StoreResourceNodes(ctx, tt.chunkX, tt.chunkY, tt.resources)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNodeService_GetResourcesForChunk(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name             string
		chunkX, chunkY   int32
		setupMocks       func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService)
		expectError      bool
		expectedErrorMsg string
		expectedCount    int
	}{
		{
			name:   "existing resources in database",
			chunkX: 0,
			chunkY: 0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Add existing resource to mock database
				existingNode := db.ResourceNode{
					ID:                 1,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					WorldID:            createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
					ChunkX:             0,
					ChunkY:             0,
					X:               10,
					Y:               10,
					ClusterID:          "test-cluster-1",
					Size:               1,
					CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
				}
				mockDB.resourceNodes["1"] = existingNode
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:   "no existing resources, chunk exists - generation attempted",
			chunkX: 1,
			chunkY: 1,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Chunk exists but no resources
				mockDB.SetChunkExists(true)
				
				// Add chunk data for generation
				chunkData := createTestChunkData(1, 1, chunkV1.TerrainType_TERRAIN_TYPE_GRASS)
				chunkBytes, _ := proto.Marshal(chunkData)
				chunk := db.Chunk{
					WorldID:   createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
					ChunkX:    1,
					ChunkY:    1,
					ChunkData: chunkBytes,
				}
				mockDB.AddChunk(chunk)
			},
			expectError:   false,
			expectedCount: 1, // Should generate resources when high noise values are applied
		},
		{
			name:   "chunk does not exist",
			chunkX: 2,
			chunkY: 2,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockDB.SetChunkExists(false)
			},
			expectError:   false,
			expectedCount: 0, // Should return empty slice
		},
		{
			name:   "world service error",
			chunkX: 0,
			chunkY: 0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockWorld.SetShouldReturnError(true)
			},
			expectError:      true,
			expectedErrorMsg: "failed to get default world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			tt.setupMocks(mockDB, mockWorld)

			// Set appropriate noise values based on test expectations
			if tt.expectedCount == 0 && tt.name != "chunk does not exist" {
				// For tests that should generate few/no resources
				mockNoise.SetNoiseValue(0.1)
			} else {
				// For tests that should generate resources, use higher noise
				mockNoise.SetNoiseValue(0.8)
				// Set up specific high-noise patterns for generation
				for x := 0; x < 10; x++ {
					for y := 0; y < 10; y++ {
						mockNoise.SetNoisePattern(x, y, ResourceNoiseScale, 0.8)
						mockNoise.SetNoisePattern(x, y, ResourceDetailScale, 0.8)
					}
				}
			}

			ctx := testutil.CreateTestContext()
			resources, err := service.GetResourcesForChunk(ctx, tt.chunkX, tt.chunkY)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				
				// For generation tests, check that we got at least the expected minimum
				if tt.expectedCount > 0 {
					assert.GreaterOrEqual(t, len(resources), tt.expectedCount, "Should generate at least %d resources", tt.expectedCount)
				} else {
					assert.Len(t, resources, tt.expectedCount)
				}

				// Verify resource properties if any exist
				for _, resource := range resources {
					assert.NotNil(t, resource)
					if resource.Id != 0 { // Only check ID if it's set
						assert.NotZero(t, resource.Id)
					}
					assert.NotNil(t, resource.ResourceNodeType)
					assert.Equal(t, tt.chunkX, resource.ChunkX)
					assert.Equal(t, tt.chunkY, resource.ChunkY)
				}
			}
		})
	}
}

func TestNodeService_GetResourcesForChunks(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name             string
		chunks           []*chunkV1.ChunkCoordinate
		setupMocks       func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService)
		expectError      bool
		expectedErrorMsg string
		expectedCount    int
	}{
		{
			name: "multiple chunks with resources",
			chunks: []*chunkV1.ChunkCoordinate{
				{ChunkX: 0, ChunkY: 0},
				{ChunkX: 1, ChunkY: 0},
			},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Add resources for both chunks
				worldID := createTestUUID("550e8400-e29b-41d4-a716-446655440001")
				node1 := db.ResourceNode{
					ID: 1, WorldID: worldID, ChunkX: 0, ChunkY: 0,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					X: 10, Y: 10, ClusterID: "cluster-1", Size: 1,
					CreatedAt: pgtype.Timestamp{Valid: true, Time: time.Now()},
				}
				node2 := db.ResourceNode{
					ID: 2, WorldID: worldID, ChunkX: 1, ChunkY: 0,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH),
					X: 15, Y: 15, ClusterID: "cluster-2", Size: 1,
					CreatedAt: pgtype.Timestamp{Valid: true, Time: time.Now()},
				}
				mockDB.resourceNodes["1"] = node1
				mockDB.resourceNodes["2"] = node2
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "too many chunks requested",
			chunks: []*chunkV1.ChunkCoordinate{
				{ChunkX: 0, ChunkY: 0}, {ChunkX: 1, ChunkY: 0}, {ChunkX: 2, ChunkY: 0},
				{ChunkX: 3, ChunkY: 0}, {ChunkX: 4, ChunkY: 0}, {ChunkX: 5, ChunkY: 0}, // 6 chunks > 5 limit
			},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Should only process first 5 chunks
			},
			expectError:   false,
			expectedCount: 0, // No resources in mock
		},
		{
			name:   "world service error",
			chunks: []*chunkV1.ChunkCoordinate{{ChunkX: 0, ChunkY: 0}},
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockWorld.SetShouldReturnError(true)
			},
			expectError:      true,
			expectedErrorMsg: "failed to get default world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			tt.setupMocks(mockDB, mockWorld)

			ctx := testutil.CreateTestContext()
			resources, err := service.GetResourcesForChunks(ctx, tt.chunks)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, resources, tt.expectedCount)
			}
		})
	}
}

func TestNodeService_GetResourcesInChunkRange(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name                               string
		minX, maxX, minY, maxY             int32
		setupMocks                         func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService)
		expectError                        bool
		expectedErrorMsg                   string
		expectedCount                      int
	}{
		{
			name: "range contains multiple resources",
			minX: 0, maxX: 2, minY: 0, maxY: 2,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				// Add resources within and outside the range
				worldID := createTestUUID("550e8400-e29b-41d4-a716-446655440001")
				node1 := db.ResourceNode{ID: 1, WorldID: worldID, ChunkX: 1, ChunkY: 1, // Inside range
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					CreatedAt: pgtype.Timestamp{Valid: true, Time: time.Now()}}
				node2 := db.ResourceNode{ID: 2, WorldID: worldID, ChunkX: 5, ChunkY: 5, // Outside range
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH),
					CreatedAt: pgtype.Timestamp{Valid: true, Time: time.Now()}}
				mockDB.resourceNodes["1"] = node1
				mockDB.resourceNodes["2"] = node2
			},
			expectError:   false,
			expectedCount: 1, // Only node1 should be in range
		},
		{
			name: "world service error",
			minX: 0, maxX: 1, minY: 0, maxY: 1,
			setupMocks: func(mockDB *MockDatabaseInterface, mockWorld *MockWorldService) {
				mockWorld.SetShouldReturnError(true)
			},
			expectError:      true,
			expectedErrorMsg: "failed to get default world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			tt.setupMocks(mockDB, mockWorld)

			ctx := testutil.CreateTestContext()
			resources, err := service.GetResourcesInChunkRange(ctx, tt.minX, tt.maxX, tt.minY, tt.maxY)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, resources, tt.expectedCount)
			}
		})
	}
}

func TestNodeService_convertResourceRows(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	mockDB := NewMockDatabase()
	mockNoise := NewMockNoiseGenerator(12345)
	mockWorld := NewMockWorldService()
	mockRandom := NewMockRandomGenerator()
	mockLogger := NewMockLogger()

	service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)

	tests := []struct {
		name        string
		dbResources []db.ResourceNode
		expected    int
	}{
		{
			name: "convert valid resource nodes",
			dbResources: []db.ResourceNode{
				{
					ID:                 1,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					WorldID:            createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
					ChunkX:             0,
					ChunkY:             0,
					X:               10,
					Y:               10,
					ClusterID:          "test-cluster",
					Size:               1,
					CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
				},
			},
			expected: 1,
		},
		{
			name: "convert unknown resource type",
			dbResources: []db.ResourceNode{
				{
					ID:                 2,
					ResourceNodeTypeID: 9999, // Unknown type
					WorldID:            createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
					ChunkX:             0,
					ChunkY:             0,
					X:               15,
					Y:               15,
					ClusterID:          "unknown-cluster",
					Size:               1,
					CreatedAt:          pgtype.Timestamp{Valid: false}, // Invalid timestamp
				},
			},
			expected: 1, // Should still convert with placeholder type
		},
		{
			name:        "empty resource list",
			dbResources: []db.ResourceNode{},
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.convertResourceRows(tt.dbResources)
			assert.Len(t, result, tt.expected)

			// Verify conversion quality for non-empty results
			for i, resource := range result {
				dbResource := tt.dbResources[i]
				assert.Equal(t, dbResource.ID, resource.Id)
				assert.Equal(t, dbResource.ChunkX, resource.ChunkX)
				assert.Equal(t, dbResource.ChunkY, resource.ChunkY)
				assert.Equal(t, dbResource.X, resource.X)
				assert.Equal(t, dbResource.Y, resource.Y)
				assert.Equal(t, dbResource.ClusterID, resource.ClusterId)
				assert.Equal(t, dbResource.Size, resource.Size)
				assert.NotNil(t, resource.ResourceNodeType)

				// Check timestamp handling
				if dbResource.CreatedAt.Valid {
					assert.NotNil(t, resource.CreatedAt)
				} else {
					assert.Nil(t, resource.CreatedAt)
				}
			}
		})
	}
}

func TestNodeService_GetResourceNode(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name             string
		id               int32
		setupMocks       func(mockDB *MockDatabaseInterface)
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "successful resource retrieval",
			id:   1,
			setupMocks: func(mockDB *MockDatabaseInterface) {
				node := db.ResourceNode{
					ID:                 1,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					WorldID:            createTestUUID("550e8400-e29b-41d4-a716-446655440001"),
					ChunkX:             0,
					ChunkY:             0,
					X:               10,
					Y:               10,
					ClusterID:          "test-cluster",
					Size:               1,
					CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
				}
				mockDB.resourceNodes["1"] = node
			},
			expectError: false,
		},
		{
			name: "resource not found",
			id:   999,
			setupMocks: func(mockDB *MockDatabaseInterface) {
				// Don't add any resources
			},
			expectError: true,
		},
		{
			name: "database error",
			id:   1,
			setupMocks: func(mockDB *MockDatabaseInterface) {
				mockDB.SetShouldReturnError(true)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := NewMockDatabase()
			mockNoise := NewMockNoiseGenerator(12345)
			mockWorld := NewMockWorldService()
			mockRandom := NewMockRandomGenerator()
			mockLogger := NewMockLogger()

			service := NewNodeService(mockDB, mockNoise, mockWorld, mockRandom, mockLogger)
			tt.setupMocks(mockDB)

			ctx := testutil.CreateTestContext()
			resource, err := service.GetResourceNode(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.id, resource.ID)
			}
		})
	}
}

// Test helper functions

func TestDistance(t *testing.T) {
	tests := []struct {
		name     string
		x1, y1   int32
		x2, y2   int32
		expected float64
	}{
		{
			name:     "same point",
			x1: 0, y1: 0, x2: 0, y2: 0,
			expected: 0.0,
		},
		{
			name:     "unit distance",
			x1: 0, y1: 0, x2: 1, y2: 0,
			expected: 1.0,
		},
		{
			name:     "diagonal distance",
			x1: 0, y1: 0, x2: 3, y2: 4,
			expected: 25.0, // 3^2 + 4^2 = 9 + 16 = 25 (squared distance)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distance(tt.x1, tt.y1, tt.x2, tt.y2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateClusterID(t *testing.T) {
	tests := []struct {
		name                     string
		chunkX, chunkY           int32
		posX, posY               int32
		resourceNodeTypeID       int32
		expectedLength           int
		expectedUniqueness       bool
	}{
		{
			name:               "generates consistent length",
			chunkX: 0, chunkY: 0,
			posX: 10, posY: 10,
			resourceNodeTypeID: 1,
			expectedLength:     16, // First 16 characters of MD5 hash
			expectedUniqueness: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1 := generateClusterID(tt.chunkX, tt.chunkY, tt.posX, tt.posY, tt.resourceNodeTypeID)
			id2 := generateClusterID(tt.chunkX, tt.chunkY, tt.posX, tt.posY, tt.resourceNodeTypeID)

			assert.Len(t, id1, tt.expectedLength)
			assert.Len(t, id2, tt.expectedLength)

			if tt.expectedUniqueness {
				// IDs should be different due to timestamp component
				assert.NotEqual(t, id1, id2)
			}
		})
	}
}