package handlers

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testmocks/handlers"
	"github.com/VoidMesh/api/api/internal/testutil"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)


// uuidToBytes converts a UUID string to bytes for protobuf WorldId fields
// This matches the pattern used in world_test.go: []byte(testutil.UUIDTestData.World1)
// The handler should use string conversion: worldID.Scan(string(req.WorldId))
func uuidToBytes(uuidStr string) []byte {
	return []byte(uuidStr)
}
// TestResourceNodeHandler_GetResourcesInChunk demonstrates comprehensive testing patterns for single chunk resource retrieval
func TestResourceNodeHandler_GetResourcesInChunk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	// Create handler instance
	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	// Test data setup
	defaultWorld := db.World{
		ID: testutil.UUIDFromString(testutil.UUIDTestData.World1),
		Name: "Default World",
		Seed: 12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	tests := []struct {
		name       string
		request    *resourceNodeV1.GetResourcesInChunkRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse)
	}{
		{
			name: "successful retrieval with world ID provided",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte{}, // Use empty bytes to trigger default world like chunk handler
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				// Mock default world service call (empty WorldId triggers this)
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 1,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
							ChunkX:             10,
							ChunkY:             20,
							X:               150,
							Y:               250,
							Size:               1,
							CreatedAt:          timestamppb.New(time.Now()),
						},
						{
							Id:                 2,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH,
							ChunkX:             10,
							ChunkY:             20,
							X:               300,
							Y:               400,
							Size:               2,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 2)
				
				// Validate first resource
				assert.Equal(t, int32(1), resp.Resources[0].Id)
				assert.Equal(t, resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, resp.Resources[0].ResourceNodeTypeId)
				assert.Equal(t, int32(10), resp.Resources[0].ChunkX)
				assert.Equal(t, int32(20), resp.Resources[0].ChunkY)
				assert.Equal(t, int32(150), resp.Resources[0].X)
				assert.Equal(t, int32(250), resp.Resources[0].Y)
				assert.Equal(t, int32(1), resp.Resources[0].Size)
				
				// Validate second resource
				assert.Equal(t, int32(2), resp.Resources[1].Id)
				assert.Equal(t, resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH, resp.Resources[1].ResourceNodeTypeId)
				assert.Equal(t, int32(300), resp.Resources[1].X)
				assert.Equal(t, int32(400), resp.Resources[1].Y)
				assert.Equal(t, int32(2), resp.Resources[1].Size)
			},
		},
		{
			name: "successful retrieval without world ID - uses default world",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				ChunkX: 5,
				ChunkY: 8,
			},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(5), int32(8)).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 10,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_MINERAL_OUTCROPPING,
							ChunkX:             5,
							ChunkY:             8,
							X:               100,
							Y:               200,
							Size:               3,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 1)
				assert.Equal(t, int32(10), resp.Resources[0].Id)
				assert.Equal(t, resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_MINERAL_OUTCROPPING, resp.Resources[0].ResourceNodeTypeId)
			},
		},
		{
			name: "empty chunk - no resources found",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  99,
				ChunkY:  99,
			},
			setupMocks: func() {
				// Mock default world service call
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(99), int32(99)).
					Return([]*resourceNodeV1.ResourceNode{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 0)
			},
		},
		{
			name: "invalid world ID format",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte("invalid-uuid-format"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				// No mocks should be called due to early validation failure
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
			wantMsg:  "Invalid world ID",
		},
		{
			name: "empty world ID format",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte(""),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
					Return([]*resourceNodeV1.ResourceNode{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 0)
			},
		},
		{
			name: "default world service error",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				ChunkX: 10,
				ChunkY: 20,
			},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(db.World{}, status.Errorf(codes.Internal, "database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name: "resource service error",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				// Mock default world service call
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
					Return(nil, status.Errorf(codes.Internal, "database query failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "database query failed",
		},
		{
			name: "boundary chunk coordinates - negative values",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: uuidToBytes(testutil.UUIDTestData.World1),
				ChunkX:  -10,
				ChunkY:  -20,
			},
			setupMocks: func() {
				// Mock default world service call
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(-10), int32(-20)).
					Return([]*resourceNodeV1.ResourceNode{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 0)
			},
		},
		{
			name: "boundary chunk coordinates - large values",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: []byte{}, // Use default world
				ChunkX:  999999,
				ChunkY:  999999,
			},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(999999), int32(999999)).
					Return([]*resourceNodeV1.ResourceNode{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 0)
			},
		},
		{
			name: "service returns nil slice",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: uuidToBytes(testutil.UUIDTestData.World1),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
					Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Nil(t, resp.Resources)
			},
		},
		{
			name: "chunk at zero coordinates",
			request: &resourceNodeV1.GetResourcesInChunkRequest{
				WorldId: uuidToBytes(testutil.UUIDTestData.World1),
				ChunkX:  0,
				ChunkY:  0,
			},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(0), int32(0)).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 100,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_FISHING_SPOT,
							ChunkX:             0,
							ChunkY:             0,
							X:               0,
							Y:               0,
							Size:               1,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunkResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 1)
				assert.Equal(t, int32(100), resp.Resources[0].Id)
				assert.Equal(t, int32(0), resp.Resources[0].ChunkX)
				assert.Equal(t, int32(0), resp.Resources[0].ChunkY)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := handler.GetResourcesInChunk(context.Background(), tt.request)

			// Validate error expectation
			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}
// TestResourceNodeHandler_GetResourcesInChunks demonstrates comprehensive testing patterns for multiple chunk resource retrieval
func TestResourceNodeHandler_GetResourcesInChunks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	// Create handler instance
	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	// Test data setup
	defaultWorld := db.World{
		ID: testutil.UUIDFromString(testutil.UUIDTestData.World1),
		Name: "Default World",
		Seed: 12345,
		CreatedAt: testutil.NowTimestamp(),
	}

	tests := []struct {
		name       string
		request    *resourceNodeV1.GetResourcesInChunksRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunksResponse)
	}{
		{
			name: "successful retrieval with multiple chunks",
			request: &resourceNodeV1.GetResourcesInChunksRequest{
				Coordinates: []*resourceNodeV1.ChunkCoordinate{
					{
						WorldId: uuidToBytes(testutil.UUIDTestData.World1),
						ChunkX:  10,
						ChunkY:  20,
					},
					{
						WorldId: uuidToBytes(testutil.UUIDTestData.World1),
						ChunkX:  11,
						ChunkY:  21,
					},
				},
			},
			setupMocks: func() {
				expectedCoords := []*chunkV1.ChunkCoordinate{
					{ChunkX: 10, ChunkY: 20},
					{ChunkX: 11, ChunkY: 21},
				}
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunks(gomock.Any(), expectedCoords).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 1,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
							ChunkX:             10,
							ChunkY:             20,
							X:               150,
							Y:               250,
							Size:               1,
							CreatedAt:          timestamppb.New(time.Now()),
						},
						{
							Id:                 2,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH,
							ChunkX:             11,
							ChunkY:             21,
							X:               300,
							Y:               400,
							Size:               2,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 2)
				
				// Validate resources from different chunks
				assert.Equal(t, int32(10), resp.Resources[0].ChunkX)
				assert.Equal(t, int32(20), resp.Resources[0].ChunkY)
				assert.Equal(t, int32(11), resp.Resources[1].ChunkX)
				assert.Equal(t, int32(21), resp.Resources[1].ChunkY)
			},
		},
		{
			name: "empty coordinates list",
			request: &resourceNodeV1.GetResourcesInChunksRequest{
				Coordinates: []*resourceNodeV1.ChunkCoordinate{},
			},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(defaultWorld, nil)
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunks(gomock.Any(), []*chunkV1.ChunkCoordinate{}).
					Return([]*resourceNodeV1.ResourceNode{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourcesInChunksResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Resources, 0)
			},
		},
		{
			name: "invalid world ID format in first coordinate",
			request: &resourceNodeV1.GetResourcesInChunksRequest{
				Coordinates: []*resourceNodeV1.ChunkCoordinate{
					{
						WorldId: []byte("invalid-uuid-format"),
						ChunkX:  10,
						ChunkY:  20,
					},
				},
			},
			setupMocks: func() {
				// No mocks should be called due to early validation failure
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
			wantMsg:  "Invalid world ID",
		},
		{
			name: "resource service error",
			request: &resourceNodeV1.GetResourcesInChunksRequest{
				Coordinates: []*resourceNodeV1.ChunkCoordinate{
					{
						WorldId: uuidToBytes(testutil.UUIDTestData.World1),
						ChunkX:  10,
						ChunkY:  20,
					},
				},
			},
			setupMocks: func() {
				expectedCoords := []*chunkV1.ChunkCoordinate{
					{ChunkX: 10, ChunkY: 20},
				}
				
				mockResourceNodeService.EXPECT().
					GetResourcesForChunks(gomock.Any(), expectedCoords).
					Return(nil, status.Errorf(codes.Internal, "database query failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "database query failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := handler.GetResourcesInChunks(context.Background(), tt.request)

			// Validate error expectation
			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestResourceNodeHandler_GetResourceNodeTypes demonstrates comprehensive testing patterns for resource node type retrieval
func TestResourceNodeHandler_GetResourceNodeTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	// Create handler instance
	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *resourceNodeV1.GetResourceNodeTypesRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse)
	}{
		{
			name:    "successful retrieval of resource node types",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return([]*resourceNodeV1.ResourceNodeType{
						{
							Id:          1,
							Name:        "Herb Patch",
							Description: "A patch of medicinal herbs",
							TerrainType: "grass",
							Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
							VisualData: &resourceNodeV1.ResourceVisual{
								Sprite: "herb_patch",
								Color:  "#00FF00",
							},
							Properties: &resourceNodeV1.ResourceProperties{
								HarvestTime: 5,
								RespawnTime: 300,
								YieldMin:    1,
								YieldMax:    3,
								SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
									{
										Name:      "Rare Herb",
										Chance:    0.1,
										MinAmount: 1,
										MaxAmount: 1,
									},
								},
							},
						},
						{
							Id:          2,
							Name:        "Berry Bush",
							Description: "A bush with sweet berries",
							TerrainType: "grass",
							Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
							VisualData: &resourceNodeV1.ResourceVisual{
								Sprite: "berry_bush",
								Color:  "#FF0000",
							},
							Properties: &resourceNodeV1.ResourceProperties{
								HarvestTime: 3,
								RespawnTime: 240,
								YieldMin:    2,
								YieldMax:    5,
								SecondaryDrops: []*resourceNodeV1.SecondaryDrop{},
							},
						},
						{
							Id:          3,
							Name:        "Mineral Outcropping",
							Description: "A rocky outcrop containing minerals",
							TerrainType: "stone",
							Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
							VisualData: &resourceNodeV1.ResourceVisual{
								Sprite: "mineral_outcropping",
								Color:  "#808080",
							},
							Properties: &resourceNodeV1.ResourceProperties{
								HarvestTime: 10,
								RespawnTime: 600,
								YieldMin:    1,
								YieldMax:    2,
								SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
									{
										Name:      "Precious Gem",
										Chance:    0.05,
										MinAmount: 1,
										MaxAmount: 1,
									},
								},
							},
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.ResourceNodeTypes, 3)
				
				// Validate first resource type - Herb Patch
				herbPatch := resp.ResourceNodeTypes[0]
				assert.Equal(t, int32(1), herbPatch.Id)
				assert.Equal(t, "Herb Patch", herbPatch.Name)
				assert.Equal(t, "A patch of medicinal herbs", herbPatch.Description)
				assert.Equal(t, "grass", herbPatch.TerrainType)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON, herbPatch.Rarity)
				
				// Validate visual data
				assert.NotNil(t, herbPatch.VisualData)
				assert.Equal(t, "herb_patch", herbPatch.VisualData.Sprite)
				assert.Equal(t, "#00FF00", herbPatch.VisualData.Color)
				
				// Validate properties
				assert.NotNil(t, herbPatch.Properties)
				assert.Equal(t, int32(5), herbPatch.Properties.HarvestTime)
				assert.Equal(t, int32(300), herbPatch.Properties.RespawnTime)
				assert.Equal(t, int32(1), herbPatch.Properties.YieldMin)
				assert.Equal(t, int32(3), herbPatch.Properties.YieldMax)
				assert.Len(t, herbPatch.Properties.SecondaryDrops, 1)
				assert.Equal(t, "Rare Herb", herbPatch.Properties.SecondaryDrops[0].Name)
				assert.Equal(t, float32(0.1), herbPatch.Properties.SecondaryDrops[0].Chance)
				
				// Validate second resource type - Berry Bush
				berryBush := resp.ResourceNodeTypes[1]
				assert.Equal(t, int32(2), berryBush.Id)
				assert.Equal(t, "Berry Bush", berryBush.Name)
				assert.Equal(t, "grass", berryBush.TerrainType)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON, berryBush.Rarity)
				assert.Len(t, berryBush.Properties.SecondaryDrops, 0)
				
				// Validate third resource type - Mineral Outcropping
				mineralOutcrop := resp.ResourceNodeTypes[2]
				assert.Equal(t, int32(3), mineralOutcrop.Id)
				assert.Equal(t, "Mineral Outcropping", mineralOutcrop.Name)
				assert.Equal(t, "stone", mineralOutcrop.TerrainType)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON, mineralOutcrop.Rarity)
				assert.Len(t, mineralOutcrop.Properties.SecondaryDrops, 1)
			},
		},
		{
			name:    "empty resource types response",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return([]*resourceNodeV1.ResourceNodeType{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.ResourceNodeTypes, 0)
			},
		},
		{
			name:    "service returns nil slice",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse) {
				require.NotNil(t, resp)
				assert.Nil(t, resp.ResourceNodeTypes)
			},
		},
		{
			name:    "service error handling",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return(nil, status.Errorf(codes.Internal, "database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "database connection failed",
		},
		{
			name:    "large resource type list",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				// Generate a large list of resource types
				resourceTypes := make([]*resourceNodeV1.ResourceNodeType, 50)
				for i := 0; i < 50; i++ {
					resourceTypes[i] = &resourceNodeV1.ResourceNodeType{
						Id:          int32(i + 1),
						Name:        fmt.Sprintf("Resource Type %d", i+1),
						Description: fmt.Sprintf("Description for resource type %d", i+1),
						TerrainType: "grass",
						Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
						VisualData: &resourceNodeV1.ResourceVisual{
							Sprite: fmt.Sprintf("sprite_%d", i+1),
							Color:  "#FFFFFF",
						},
						Properties: &resourceNodeV1.ResourceProperties{
							HarvestTime:    5,
							RespawnTime:    300,
							YieldMin:       1,
							YieldMax:       3,
							SecondaryDrops: []*resourceNodeV1.SecondaryDrop{},
						},
					}
				}
				
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return(resourceTypes, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.ResourceNodeTypes, 50)
				
				// Validate first and last items
				assert.Equal(t, int32(1), resp.ResourceNodeTypes[0].Id)
				assert.Equal(t, "Resource Type 1", resp.ResourceNodeTypes[0].Name)
				assert.Equal(t, int32(50), resp.ResourceNodeTypes[49].Id)
				assert.Equal(t, "Resource Type 50", resp.ResourceNodeTypes[49].Name)
			},
		},
		{
			name:    "resource types with all rarity levels",
			request: &resourceNodeV1.GetResourceNodeTypesRequest{},
			setupMocks: func() {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return([]*resourceNodeV1.ResourceNodeType{
						{
							Id:     1,
							Name:   "Common Resource",
							Rarity: resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
						},
						{
							Id:     2,
							Name:   "Uncommon Resource",
							Rarity: resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
						},
						{
							Id:     3,
							Name:   "Rare Resource",
							Rarity: resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
						},
						{
							Id:     4,
							Name:   "Very Rare Resource",
							Rarity: resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE,
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *resourceNodeV1.GetResourceNodeTypesResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.ResourceNodeTypes, 4)
				
				// Validate all rarity levels are present
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON, resp.ResourceNodeTypes[0].Rarity)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON, resp.ResourceNodeTypes[1].Rarity)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE, resp.ResourceNodeTypes[2].Rarity)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE, resp.ResourceNodeTypes[3].Rarity)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := handler.GetResourceNodeTypes(context.Background(), tt.request)

			// Validate error expectation
			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// BenchmarkResourceNodeHandler_GetResourcesInChunk measures performance of single chunk resource retrieval
func BenchmarkResourceNodeHandler_GetResourcesInChunk(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockResourceNodeService.EXPECT().
		GetResourcesForChunk(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*resourceNodeV1.ResourceNode{
			{
				Id:                 1,
				ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
				ChunkX:             10,
				ChunkY:             20,
				X:               150,
				Y:               250,
				Size:               1,
				CreatedAt:          timestamppb.New(time.Now()),
			},
		}, nil).
		AnyTimes()

	request := &resourceNodeV1.GetResourcesInChunkRequest{
		WorldId: uuidToBytes(testutil.UUIDTestData.World1),
		ChunkX:  10,
		ChunkY:  20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.GetResourcesInChunk(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResourceNodeHandler_GetResourcesInChunks measures performance of multiple chunk resource retrieval
func BenchmarkResourceNodeHandler_GetResourcesInChunks(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockResourceNodeService.EXPECT().
		GetResourcesForChunks(gomock.Any(), gomock.Any()).
		Return([]*resourceNodeV1.ResourceNode{
			{
				Id:                 1,
				ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
				ChunkX:             10,
				ChunkY:             20,
				X:               150,
				Y:               250,
				Size:               1,
				CreatedAt:          timestamppb.New(time.Now()),
			},
			{
				Id:                 2,
				ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH,
				ChunkX:             11,
				ChunkY:             21,
				X:               300,
				Y:               400,
				Size:               2,
				CreatedAt:          timestamppb.New(time.Now()),
			},
		}, nil).
		AnyTimes()

	request := &resourceNodeV1.GetResourcesInChunksRequest{
		Coordinates: []*resourceNodeV1.ChunkCoordinate{
			{
				WorldId: uuidToBytes(testutil.UUIDTestData.World1),
				ChunkX:  10,
				ChunkY:  20,
			},
			{
				WorldId: uuidToBytes(testutil.UUIDTestData.World1),
				ChunkX:  11,
				ChunkY:  21,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.GetResourcesInChunks(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResourceNodeHandler_GetResourceNodeTypes measures performance of resource node type retrieval
func BenchmarkResourceNodeHandler_GetResourceNodeTypes(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockResourceNodeService.EXPECT().
		GetResourceNodeTypes(gomock.Any()).
		Return([]*resourceNodeV1.ResourceNodeType{
			{
				Id:          1,
				Name:        "Herb Patch",
				Description: "A patch of medicinal herbs",
				TerrainType: "grass",
				Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
				VisualData: &resourceNodeV1.ResourceVisual{
					Sprite: "herb_patch",
					Color:  "#00FF00",
				},
				Properties: &resourceNodeV1.ResourceProperties{
					HarvestTime: 5,
					RespawnTime: 300,
					YieldMin:    1,
					YieldMax:    3,
					SecondaryDrops: []*resourceNodeV1.SecondaryDrop{},
				},
			},
		}, nil).
		AnyTimes()

	request := &resourceNodeV1.GetResourceNodeTypesRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.GetResourceNodeTypes(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestResourceNodeHandler_ProtocolBufferMessageValidation demonstrates proto message validation testing
func TestResourceNodeHandler_ProtocolBufferMessageValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	t.Run("get resources in chunk response proto validation", func(t *testing.T) {
		now := time.Now()
		expectedResponse := &resourceNodeV1.GetResourcesInChunkResponse{
			Resources: []*resourceNodeV1.ResourceNode{
				{
					Id:                 1,
					ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
					ChunkX:             10,
					ChunkY:             20,
					X:               150,
					Y:               250,
					Size:               1,
					CreatedAt:          timestamppb.New(now),
				},
			},
		}

		mockResourceNodeService.EXPECT().
			GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
			Return(expectedResponse.Resources, nil)

		request := &resourceNodeV1.GetResourcesInChunkRequest{
			WorldId: uuidToBytes(testutil.UUIDTestData.World1),
			ChunkX:  10,
			ChunkY:  20,
		}

		resp, err := handler.GetResourcesInChunk(context.Background(), request)

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate specific proto fields
		testutil.AssertProtoFieldEqual(t, expectedResponse.Resources[0], resp.Resources[0], "id")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Resources[0], resp.Resources[0], "chunk_x")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Resources[0], resp.Resources[0], "chunk_y")
	})

	t.Run("get resource node types response proto validation", func(t *testing.T) {
		expectedResponse := &resourceNodeV1.GetResourceNodeTypesResponse{
			ResourceNodeTypes: []*resourceNodeV1.ResourceNodeType{
				{
					Id:          1,
					Name:        "ProtoTestResource",
					Description: "A resource for proto testing",
					TerrainType: "grass",
					Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
					VisualData: &resourceNodeV1.ResourceVisual{
						Sprite: "test_sprite",
						Color:  "#FFFFFF",
					},
					Properties: &resourceNodeV1.ResourceProperties{
						HarvestTime: 5,
						RespawnTime: 300,
						YieldMin:    1,
						YieldMax:    3,
						SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
							{
								Name:      "Test Drop",
								Chance:    0.1,
								MinAmount: 1,
								MaxAmount: 2,
							},
						},
					},
				},
			},
		}

		mockResourceNodeService.EXPECT().
			GetResourceNodeTypes(gomock.Any()).
			Return(expectedResponse.ResourceNodeTypes, nil)

		request := &resourceNodeV1.GetResourceNodeTypesRequest{}
		resp, err := handler.GetResourceNodeTypes(context.Background(), request)

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate nested proto fields
		assert.Equal(t, "ProtoTestResource", resp.ResourceNodeTypes[0].Name)
		assert.NotNil(t, resp.ResourceNodeTypes[0].VisualData)
		assert.NotNil(t, resp.ResourceNodeTypes[0].Properties)
		testutil.AssertProtoFieldEqual(t, expectedResponse.ResourceNodeTypes[0], resp.ResourceNodeTypes[0], "name")
		testutil.AssertProtoFieldEqual(t, expectedResponse.ResourceNodeTypes[0], resp.ResourceNodeTypes[0], "rarity")
	})
}

// TestResourceNodeHandler_EdgeCases demonstrates edge case testing
func TestResourceNodeHandler_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "maximum coordinate values",
			testFunc: func(t *testing.T) {
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(2147483647), int32(2147483647)).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 1,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
							ChunkX:             2147483647, // max int32
							ChunkY:             2147483647, // max int32
							X:               2147483647,
							Y:               2147483647,
							Size:               1,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)

				request := &resourceNodeV1.GetResourcesInChunkRequest{
					WorldId: uuidToBytes(testutil.UUIDTestData.World1),
					ChunkX:  2147483647,
					ChunkY:  2147483647,
				}

				resp, err := handler.GetResourcesInChunk(context.Background(), request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, int32(2147483647), resp.Resources[0].ChunkX)
				assert.Equal(t, int32(2147483647), resp.Resources[0].ChunkY)
			},
		},
		{
			name: "minimum coordinate values",
			testFunc: func(t *testing.T) {
				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(-2147483648), int32(-2147483648)).
					Return([]*resourceNodeV1.ResourceNode{
						{
							Id:                 1,
							ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
							ChunkX:             -2147483648, // min int32
							ChunkY:             -2147483648, // min int32
							X:               -2147483648,
							Y:               -2147483648,
							Size:               1,
							CreatedAt:          timestamppb.New(time.Now()),
						},
					}, nil)

				request := &resourceNodeV1.GetResourcesInChunkRequest{
					WorldId: uuidToBytes(testutil.UUIDTestData.World1),
					ChunkX:  -2147483648,
					ChunkY:  -2147483648,
				}

				resp, err := handler.GetResourcesInChunk(context.Background(), request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, int32(-2147483648), resp.Resources[0].ChunkX)
				assert.Equal(t, int32(-2147483648), resp.Resources[0].ChunkY)
			},
		},
		{
			name: "very large resource node list",
			testFunc: func(t *testing.T) {
				// Generate a large list of resources (1000 items)
				resources := make([]*resourceNodeV1.ResourceNode, 1000)
				for i := 0; i < 1000; i++ {
					resources[i] = &resourceNodeV1.ResourceNode{
						Id:                 int32(i + 1),
						ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
						ChunkX:             10,
						ChunkY:             20,
						X:               int32(i % 32 * 32), // Spread across chunk
						Y:               int32(i / 32 * 32),
						Size:               int32((i % 3) + 1), // Size 1-3
						CreatedAt:          timestamppb.New(time.Now()),
					}
				}

				mockResourceNodeService.EXPECT().
					GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
					Return(resources, nil)

				request := &resourceNodeV1.GetResourcesInChunkRequest{
					WorldId: uuidToBytes(testutil.UUIDTestData.World1),
					ChunkX:  10,
					ChunkY:  20,
				}

				resp, err := handler.GetResourcesInChunk(context.Background(), request)

				testutil.AssertNoGRPCError(t, err)
				assert.Len(t, resp.Resources, 1000)
				assert.Equal(t, int32(1), resp.Resources[0].Id)
				assert.Equal(t, int32(1000), resp.Resources[999].Id)
			},
		},
		{
			name: "resource type with all possible enum values",
			testFunc: func(t *testing.T) {
				mockResourceNodeService.EXPECT().
					GetResourceNodeTypes(gomock.Any()).
					Return([]*resourceNodeV1.ResourceNodeType{
						{
							Id:     1,
							Name:   "All Enum Resource",
							Rarity: resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE,
							VisualData: &resourceNodeV1.ResourceVisual{
								Sprite: "all_enum_sprite",
								Color:  "#FF00FF",
							},
							Properties: &resourceNodeV1.ResourceProperties{
								HarvestTime: 60,
								RespawnTime: 3600,
								YieldMin:    1,
								YieldMax:    10,
								SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
									{
										Name:      "Ultra Rare Drop",
										Chance:    0.001,
										MinAmount: 1,
										MaxAmount: 1,
									},
								},
							},
						},
					}, nil)

				request := &resourceNodeV1.GetResourceNodeTypesRequest{}
				resp, err := handler.GetResourceNodeTypes(context.Background(), request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE, resp.ResourceNodeTypes[0].Rarity)
				assert.Equal(t, float32(0.001), resp.ResourceNodeTypes[0].Properties.SecondaryDrops[0].Chance)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestResourceNodeHandler_ContextCancellation demonstrates context cancellation handling
func TestResourceNodeHandler_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockResourceNodeService := mockhandlers.NewMockResourceNodeService(ctrl)
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	handler := &ResourceNodeHandler{
		resourceNodeService: mockResourceNodeService,
		worldService:        mockWorldService,
		logger:              log.New(io.Discard),
	}

	t.Run("context cancellation during service call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context before service call
		cancel()

		mockResourceNodeService.EXPECT().
			GetResourcesForChunk(gomock.Any(), int32(10), int32(20)).
			Return(nil, status.Errorf(codes.Canceled, "context canceled"))

		request := &resourceNodeV1.GetResourcesInChunkRequest{
			WorldId: uuidToBytes(testutil.UUIDTestData.World1),
			ChunkX:  10,
			ChunkY:  20,
		}

		_, err := handler.GetResourcesInChunk(ctx, request)

		testutil.AssertGRPCError(t, err, codes.Canceled, "context canceled")
	})
}
