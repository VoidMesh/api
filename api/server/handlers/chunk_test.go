package handlers

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testmocks/handlers"
	"github.com/VoidMesh/api/api/internal/testutil"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockLoggerAdapter adapts the mock logger to match interface signature
type mockLoggerAdapter struct {
	mock *mockhandlers.MockLoggerInterface
}

func (m *mockLoggerAdapter) Debug(msg string, keysAndValues ...interface{}) {
	args := make([]any, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = v
	}
	m.mock.Debug(msg, args...)
}

func (m *mockLoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	args := make([]any, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = v
	}
	m.mock.Info(msg, args...)
}

func (m *mockLoggerAdapter) Warn(msg string, keysAndValues ...interface{}) {
	args := make([]any, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = v
	}
	m.mock.Warn(msg, args...)
}

func (m *mockLoggerAdapter) Error(msg string, keysAndValues ...interface{}) {
	args := make([]any, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = v
	}
	m.mock.Error(msg, args...)
}

func (m *mockLoggerAdapter) With(keysAndValues ...interface{}) LoggerInterface {
	args := make([]any, len(keysAndValues))
	for i, v := range keysAndValues {
		args[i] = v
	}
	mockResult := m.mock.With(args...)
	return &mockLoggerAdapter{mock: mockResult.(*mockhandlers.MockLoggerInterface)}
}

// TestChunkServiceServer_GetChunk demonstrates comprehensive testing patterns for chunk retrieval
func TestChunkServiceServer_GetChunk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLoggerInterface := mockhandlers.NewMockLoggerInterface(ctrl)
	mockLogger := &mockLoggerAdapter{mock: mockLoggerInterface}

	// Create server instance
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       mockLogger,
	}

	// Test data
	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testChunkData := &chunkV1.ChunkData{
		ChunkX:      10,
		ChunkY:      20,
		Cells:       make([]*chunkV1.TerrainCell, 1024), // 32x32 cells
		Seed:        12345,
		GeneratedAt: timestamppb.New(time.Now()),
		ResourceNodes: []*resourceNodeV1.ResourceNode{
			{
				Id:                  1,
				ResourceNodeTypeId:  resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HARVESTABLE_TREE,
				ChunkX:              10,
				ChunkY:              20,
				X:                15,
				Y:                25,
				Size:                5,
				CreatedAt:           timestamppb.New(time.Now()),
			},
		},
	}
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	tests := []struct {
		name       string
		request    *chunkV1.GetChunkRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *chunkV1.GetChunkResponse)
	}{
		{
			name: "successful chunk retrieval with default world",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Empty world ID to use default
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				// Setup logger expectations
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Info("Successfully retrieved chunk")

				// World service should be called for default world
				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				
				// Chunk service should be called
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(10), int32(20)).Return(testChunkData, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunkResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Chunk)
				assert.Equal(t, int32(10), resp.Chunk.ChunkX)
				assert.Equal(t, int32(20), resp.Chunk.ChunkY)
				assert.Equal(t, int64(12345), resp.Chunk.Seed)
				assert.Len(t, resp.Chunk.Cells, 1024)
				assert.Len(t, resp.Chunk.ResourceNodes, 1)
				
				// Validate resource node details
				rn := resp.Chunk.ResourceNodes[0]
				assert.Equal(t, int32(1), rn.Id)
				assert.Equal(t, resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HARVESTABLE_TREE, rn.ResourceNodeTypeId)
				assert.Equal(t, int32(10), rn.ChunkX)
				assert.Equal(t, int32(20), rn.ChunkY)
				assert.Equal(t, int32(15), rn.X)
				assert.Equal(t, int32(25), rn.Y)
			},
		},
		{
			name: "successful chunk generation - new chunk created",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  30,
				ChunkY:  40,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Info("Successfully retrieved chunk")

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)

				newChunkData := &chunkV1.ChunkData{
					ChunkX:        30,
					ChunkY:        40,
					Cells:         make([]*chunkV1.TerrainCell, 1024),
					Seed:          12345,
					GeneratedAt:   timestamppb.New(time.Now()),
					ResourceNodes: []*resourceNodeV1.ResourceNode{},
				}
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(30), int32(40)).Return(newChunkData, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunkResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Chunk)
				assert.Equal(t, int32(30), resp.Chunk.ChunkX)
				assert.Equal(t, int32(40), resp.Chunk.ChunkY)
				assert.Len(t, resp.Chunk.Cells, 1024)
				assert.Empty(t, resp.Chunk.ResourceNodes)
			},
		},
		{
			name: "chunk at origin coordinates (0,0)",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  0,
				ChunkY:  0,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Info("Successfully retrieved chunk")

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)

				originChunkData := &chunkV1.ChunkData{
					ChunkX:        0,
					ChunkY:        0,
					Cells:         make([]*chunkV1.TerrainCell, 1024),
					Seed:          12345,
					GeneratedAt:   timestamppb.New(time.Now()),
					ResourceNodes: []*resourceNodeV1.ResourceNode{},
				}
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(0), int32(0)).Return(originChunkData, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunkResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Chunk)
				assert.Equal(t, int32(0), resp.Chunk.ChunkX)
				assert.Equal(t, int32(0), resp.Chunk.ChunkY)
			},
		},
		{
			name: "chunk at negative coordinates",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  -10,
				ChunkY:  -20,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Info("Successfully retrieved chunk")

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)

				negativeChunkData := &chunkV1.ChunkData{
					ChunkX:        -10,
					ChunkY:        -20,
					Cells:         make([]*chunkV1.TerrainCell, 1024),
					Seed:          12345,
					GeneratedAt:   timestamppb.New(time.Now()),
					ResourceNodes: []*resourceNodeV1.ResourceNode{},
				}
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(-10), int32(-20)).Return(negativeChunkData, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunkResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Chunk)
				assert.Equal(t, int32(-10), resp.Chunk.ChunkX)
				assert.Equal(t, int32(-20), resp.Chunk.ChunkY)
			},
		},
		{
			name: "invalid world ID format - malformed UUID bytes",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{1, 2, 3}, // Invalid UUID bytes
				ChunkX:  90,
				ChunkY:  100,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Warn(gomock.Any(), gomock.Any())
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
			wantMsg:  "Invalid world ID",
		},
		{
			name: "world service failure - GetDefaultWorld returns error",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Empty world ID triggers default world lookup
				ChunkX:  110,
				ChunkY:  120,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				// World service fails
				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(db.World{}, errors.New("database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name: "chunk service failure - GetOrCreateChunk returns error",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  130,
				ChunkY:  140,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)

				// Chunk service fails
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(130), int32(140)).Return(nil, errors.New("chunk generation failed"))
			},
			wantErr: true,
		},
		{
			name: "context cancellation - request timeout handling",
			request: &chunkV1.GetChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  150,
				ChunkY:  160,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunk request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching or creating chunk")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)

				// Chunk service returns context cancelled error
				mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(150), int32(160)).Return(nil, context.Canceled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for this test case
			tt.setupMocks()

			// Execute the method
			resp, err := server.GetChunk(context.Background(), tt.request)

			// Verify error expectations
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantCode != codes.OK {
					testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				}
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestChunkServiceServer_GetChunks demonstrates comprehensive testing patterns for batch chunk retrieval
func TestChunkServiceServer_GetChunks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLoggerInterface := mockhandlers.NewMockLoggerInterface(ctrl)
	mockLogger := &mockLoggerAdapter{mock: mockLoggerInterface}

	// Create server instance
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       mockLogger,
	}

	// Test data
	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	// Create test chunk data
	createTestChunks := func(minX, maxX, minY, maxY int32) []*chunkV1.ChunkData {
		var chunks []*chunkV1.ChunkData
		for x := minX; x <= maxX; x++ {
			for y := minY; y <= maxY; y++ {
				chunks = append(chunks, &chunkV1.ChunkData{
					ChunkX:        x,
					ChunkY:        y,
					Cells:         make([]*chunkV1.TerrainCell, 1024),
					Seed:          12345,
					GeneratedAt:   timestamppb.New(time.Now()),
					ResourceNodes: []*resourceNodeV1.ResourceNode{},
				})
			}
		}
		return chunks
	}

	tests := []struct {
		name       string
		request    *chunkV1.GetChunksRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *chunkV1.GetChunksResponse)
	}{
		{
			name: "successful range retrieval - 3x3 grid",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Use default world
				MinChunkX: 0,
				MaxChunkX: 2,
				MinChunkY: 0,
				MaxChunkY: 2,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in range")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				expectedChunks := createTestChunks(0, 2, 0, 2)
				mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(0), int32(2), int32(0), int32(2)).Return(expectedChunks, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 9) // 3x3 grid
				
				// Verify all chunks have correct coordinates
				coordMap := make(map[[2]int32]bool)
				for _, chunk := range resp.Chunks {
					coordMap[[2]int32{chunk.ChunkX, chunk.ChunkY}] = true
				}
				
				// Check that all expected coordinates are present
				for x := int32(0); x <= 2; x++ {
					for y := int32(0); y <= 2; y++ {
						assert.True(t, coordMap[[2]int32{x, y}], "Missing chunk at (%d,%d)", x, y)
					}
				}
			},
		},
		{
			name: "single chunk range - min equals max coordinates",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Use default world
				MinChunkX: 5,
				MaxChunkX: 5,
				MinChunkY: 5,
				MaxChunkY: 5,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in range")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				singleChunk := createTestChunks(5, 5, 5, 5)
				mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(5), int32(5), int32(5), int32(5)).Return(singleChunk, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 1)
				assert.Equal(t, int32(5), resp.Chunks[0].ChunkX)
				assert.Equal(t, int32(5), resp.Chunks[0].ChunkY)
			},
		},
		{
			name: "large range - 5x5 grid with negative coordinates",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Use default world
				MinChunkX: -2,
				MaxChunkX: 2,
				MinChunkY: -2,
				MaxChunkY: 2,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in range")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				largeChunkSet := createTestChunks(-2, 2, -2, 2)
				mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(-2), int32(2), int32(-2), int32(2)).Return(largeChunkSet, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 25) // 5x5 grid
				
				// Verify origin chunk is included
				foundOrigin := false
				for _, chunk := range resp.Chunks {
					if chunk.ChunkX == 0 && chunk.ChunkY == 0 {
						foundOrigin = true
						break
					}
				}
				assert.True(t, foundOrigin, "Origin chunk (0,0) should be included in range")
			},
		},
		{
			name: "rectangular range - 2x4 grid",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Use default world
				MinChunkX: 10,
				MaxChunkX: 11,
				MinChunkY: 20,
				MaxChunkY: 23,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in range")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				rectChunks := createTestChunks(10, 11, 20, 23)
				mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(10), int32(11), int32(20), int32(23)).Return(rectChunks, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 8) // 2x4 grid
			},
		},
		{
			name: "world service failure - GetDefaultWorld error",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Empty world ID
				MinChunkX: 20,
				MaxChunkX: 22,
				MinChunkY: 20,
				MaxChunkY: 22,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(db.World{}, errors.New("world service unavailable"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name: "chunk service failure - GetChunksInRange error",
			request: &chunkV1.GetChunksRequest{
				WorldId:   []byte{}, // Use default world
				MinChunkX: 30,
				MaxChunkX: 32,
				MinChunkY: 30,
				MaxChunkY: 32,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunks request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in range")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(30), int32(32), int32(30), int32(32)).Return(nil, errors.New("database timeout"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for this test case
			tt.setupMocks()

			// Execute the method
			resp, err := server.GetChunks(context.Background(), tt.request)

			// Verify error expectations
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantCode != codes.OK {
					testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				}
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestChunkServiceServer_GetChunksInRadius demonstrates comprehensive testing patterns for radius-based chunk retrieval
func TestChunkServiceServer_GetChunksInRadius(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLoggerInterface := mockhandlers.NewMockLoggerInterface(ctrl)
	mockLogger := &mockLoggerAdapter{mock: mockLoggerInterface}

	// Create server instance
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       mockLogger,
	}

	// Test data
	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	// Helper to create test chunks in radius
	createRadiusChunks := func(centerX, centerY, radius int32, pattern string) []*chunkV1.ChunkData {
		var chunks []*chunkV1.ChunkData
		
		switch pattern {
		case "cross":
			// Create cross pattern for radius tests
			chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX, ChunkY: centerY, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
			if radius > 0 {
				chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX - radius, ChunkY: centerY, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
				chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX + radius, ChunkY: centerY, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
				chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX, ChunkY: centerY - radius, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
				chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX, ChunkY: centerY + radius, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
			}
		case "single":
			chunks = append(chunks, &chunkV1.ChunkData{ChunkX: centerX, ChunkY: centerY, Cells: make([]*chunkV1.TerrainCell, 1024), Seed: 12345, GeneratedAt: timestamppb.New(time.Now()), ResourceNodes: []*resourceNodeV1.ResourceNode{}})
		case "large":
			// Create a larger set for big radius tests
			for i := 0; i < 13; i++ {
				chunks = append(chunks, &chunkV1.ChunkData{
					ChunkX: centerX + int32(i%4-1), 
					ChunkY: centerY + int32(i/4-1), 
					Cells: make([]*chunkV1.TerrainCell, 1024), 
					Seed: 12345, 
					GeneratedAt: timestamppb.New(time.Now()), 
					ResourceNodes: []*resourceNodeV1.ResourceNode{},
				})
			}
		}
		
		return chunks
	}

	tests := []struct {
		name       string
		request    *chunkV1.GetChunksInRadiusRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *chunkV1.GetChunksInRadiusResponse)
	}{
		{
			name: "successful radius retrieval - cross pattern around origin",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Use default world
				CenterChunkX: 0,
				CenterChunkY: 0,
				Radius:       1,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in radius")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				radiusChunks := createRadiusChunks(0, 0, 1, "cross")
				mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(0), int32(0), int32(1)).Return(radiusChunks, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksInRadiusResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 5) // Center + 4 adjacent
				
				// Verify center chunk is included
				foundCenter := false
				for _, chunk := range resp.Chunks {
					if chunk.ChunkX == 0 && chunk.ChunkY == 0 {
						foundCenter = true
						break
					}
				}
				assert.True(t, foundCenter, "Center chunk should be included in radius")
			},
		},
		{
			name: "zero radius - single chunk at center",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Use default world
				CenterChunkX: 10,
				CenterChunkY: 10,
				Radius:       0,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in radius")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				centerChunk := createRadiusChunks(10, 10, 0, "single")
				mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(10), int32(10), int32(0)).Return(centerChunk, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksInRadiusResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 1)
				assert.Equal(t, int32(10), resp.Chunks[0].ChunkX)
				assert.Equal(t, int32(10), resp.Chunks[0].ChunkY)
			},
		},
		{
			name: "large radius - multiple chunks in circle",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Use default world
				CenterChunkX: 50,
				CenterChunkY: 50,
				Radius:       10,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in radius")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				largeRadiusChunks := createRadiusChunks(50, 50, 10, "large")
				mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(50), int32(50), int32(10)).Return(largeRadiusChunks, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksInRadiusResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 13)
				
				// Verify chunks have valid coordinates
				for _, chunk := range resp.Chunks {
					assert.NotNil(t, chunk.Cells)
					assert.Len(t, chunk.Cells, 1024)
				}
			},
		},
		{
			name: "negative coordinates center",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Use default world
				CenterChunkX: -25,
				CenterChunkY: -30,
				Radius:       2,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in radius")
				mockLoggerInterface.EXPECT().Info(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				negativeRadiusChunks := createRadiusChunks(-25, -30, 2, "cross")
				mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(-25), int32(-30), int32(2)).Return(negativeRadiusChunks, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *chunkV1.GetChunksInRadiusResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Chunks, 5)
				
				// Verify center chunk with negative coordinates
				foundCenter := false
				for _, chunk := range resp.Chunks {
					if chunk.ChunkX == -25 && chunk.ChunkY == -30 {
						foundCenter = true
						break
					}
				}
				assert.True(t, foundCenter, "Center chunk at negative coordinates should be included")
			},
		},
		{
			name: "world service failure - GetDefaultWorld error",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Empty world ID
				CenterChunkX: 35,
				CenterChunkY: 35,
				Radius:       5,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(db.World{}, errors.New("world not found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name: "chunk service failure - GetChunksInRadius error",
			request: &chunkV1.GetChunksInRadiusRequest{
				WorldId:      []byte{}, // Use default world
				CenterChunkX: 45,
				CenterChunkY: 45,
				Radius:       7,
			},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().With(gomock.Any()).Return(mockLoggerInterface).Times(2)
				mockLoggerInterface.EXPECT().Debug("Received GetChunksInRadius request")
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Debug("Fetching chunks in radius")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())

				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
				mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(45), int32(45), int32(7)).Return(nil, errors.New("radius calculation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for this test case
			tt.setupMocks()

			// Execute the method
			resp, err := server.GetChunksInRadius(context.Background(), tt.request)

			// Verify error expectations
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantCode != codes.OK {
					testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				}
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestChunkServiceServer_resolveWorldID tests the helper method for world ID resolution
func TestChunkServiceServer_resolveWorldID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLoggerInterface := mockhandlers.NewMockLoggerInterface(ctrl)
	mockLogger := &mockLoggerAdapter{mock: mockLoggerInterface}

	// Create server instance
	server := &chunkServiceServer{
		worldService: mockWorldService,
		logger:       mockLogger,
	}

	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	tests := []struct {
		name         string
		worldIDBytes []byte
		setupMocks   func()
		wantErr      bool
		wantCode     codes.Code
		validate     func(t *testing.T, worldID pgtype.UUID)
	}{
		{
			name:         "empty world ID - uses default world",
			worldIDBytes: []byte{},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, worldID pgtype.UUID) {
				assert.Equal(t, testWorldID.Bytes, worldID.Bytes)
				assert.True(t, worldID.Valid)
			},
		},
		{
			name:         "invalid world ID - parsing fails",
			worldIDBytes: []byte{1, 2, 3}, // Invalid UUID bytes
			setupMocks: func() {
				mockLoggerInterface.EXPECT().Warn(gomock.Any(), gomock.Any())
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name:         "default world service failure",
			worldIDBytes: []byte{},
			setupMocks: func() {
				mockLoggerInterface.EXPECT().Debug("World ID not provided, using default world")
				mockLoggerInterface.EXPECT().Error(gomock.Any(), gomock.Any())
				mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(db.World{}, errors.New("world service error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for this test case
			tt.setupMocks()

			// Execute the helper method
			worldID, err := server.resolveWorldID(context.Background(), tt.worldIDBytes, mockLogger)

			// Verify error expectations
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantCode != codes.OK {
					testutil.AssertGRPCError(t, err, tt.wantCode)
				}
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, worldID)
				}
			}
		})
	}
}

// Benchmark tests for performance baseline establishment
func BenchmarkChunkServiceServer_GetChunk(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLogger := log.New(io.Discard) // Use real logger for benchmarks

	// Create server instance with interface wrapper
	loggerWrapper := &loggerWrapper{logger: mockLogger}
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       loggerWrapper,
	}

	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}
	testChunkData := &chunkV1.ChunkData{
		ChunkX:        10,
		ChunkY:        20,
		Cells:         make([]*chunkV1.TerrainCell, 1024),
		Seed:          12345,
		GeneratedAt:   timestamppb.New(time.Now()),
		ResourceNodes: []*resourceNodeV1.ResourceNode{},
	}

	// Setup mock expectations for all benchmark iterations
	mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil).AnyTimes()
	mockChunkService.EXPECT().GetOrCreateChunk(gomock.Any(), int32(10), int32(20)).Return(testChunkData, nil).AnyTimes()

	request := &chunkV1.GetChunkRequest{
		WorldId: []byte{}, // Use default world
		ChunkX:  10,
		ChunkY:  20,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := server.GetChunk(context.Background(), request)
		if err != nil {
			b.Fatalf("Benchmark GetChunk failed: %v", err)
		}
		if resp == nil {
			b.Fatal("Benchmark GetChunk returned nil response")
		}
	}
}

func BenchmarkChunkServiceServer_GetChunks(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLogger := log.New(io.Discard) // Use real logger for benchmarks

	// Create server instance with interface wrapper
	loggerWrapper := &loggerWrapper{logger: mockLogger}
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       loggerWrapper,
	}

	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}
	testChunks := make([]*chunkV1.ChunkData, 9) // 3x3 grid
	for i := 0; i < 9; i++ {
		testChunks[i] = &chunkV1.ChunkData{
			ChunkX:        int32(i % 3),
			ChunkY:        int32(i / 3),
			Cells:         make([]*chunkV1.TerrainCell, 1024),
			Seed:          12345,
			GeneratedAt:   timestamppb.New(time.Now()),
			ResourceNodes: []*resourceNodeV1.ResourceNode{},
		}
	}

	// Setup mock expectations for all benchmark iterations
	mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil).AnyTimes()
	mockChunkService.EXPECT().GetChunksInRange(gomock.Any(), int32(0), int32(2), int32(0), int32(2)).Return(testChunks, nil).AnyTimes()

	request := &chunkV1.GetChunksRequest{
		WorldId:   []byte{}, // Use default world
		MinChunkX: 0,
		MaxChunkX: 2,
		MinChunkY: 0,
		MaxChunkY: 2,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := server.GetChunks(context.Background(), request)
		if err != nil {
			b.Fatalf("Benchmark GetChunks failed: %v", err)
		}
		if resp == nil || len(resp.Chunks) != 9 {
			b.Fatal("Benchmark GetChunks returned invalid response")
		}
	}
}

func BenchmarkChunkServiceServer_GetChunksInRadius(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	// Create mocks
	mockChunkService := mockhandlers.NewMockChunkService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)
	mockLogger := log.New(io.Discard) // Use real logger for benchmarks

	// Create server instance with interface wrapper
	loggerWrapper := &loggerWrapper{logger: mockLogger}
	server := &chunkServiceServer{
		chunkService: mockChunkService,
		worldService: mockWorldService,
		logger:       loggerWrapper,
	}

	testWorldID := testutil.UUIDFromString(testutil.UUIDTestData.World1)
	testWorld := db.World{
		ID:        testWorldID,
		Name:      "Test World",
		Seed:      12345,
		CreatedAt: testutil.NowTimestamp(),
	}
	testChunks := make([]*chunkV1.ChunkData, 5) // Radius 1 around center
	positions := [][2]int32{{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	for i, pos := range positions {
		testChunks[i] = &chunkV1.ChunkData{
			ChunkX:        pos[0],
			ChunkY:        pos[1],
			Cells:         make([]*chunkV1.TerrainCell, 1024),
			Seed:          12345,
			GeneratedAt:   timestamppb.New(time.Now()),
			ResourceNodes: []*resourceNodeV1.ResourceNode{},
		}
	}

	// Setup mock expectations for all benchmark iterations
	mockWorldService.EXPECT().GetDefaultWorld(gomock.Any()).Return(testWorld, nil).AnyTimes()
	mockChunkService.EXPECT().GetChunksInRadius(gomock.Any(), int32(0), int32(0), int32(1)).Return(testChunks, nil).AnyTimes()

	request := &chunkV1.GetChunksInRadiusRequest{
		WorldId:      []byte{}, // Use default world
		CenterChunkX: 0,
		CenterChunkY: 0,
		Radius:       1,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := server.GetChunksInRadius(context.Background(), request)
		if err != nil {
			b.Fatalf("Benchmark GetChunksInRadius failed: %v", err)
		}
		if resp == nil || len(resp.Chunks) != 5 {
			b.Fatal("Benchmark GetChunksInRadius returned invalid response")
		}
	}
}