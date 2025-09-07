package inventory

import (
	"context"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testutil"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/jackc/pgx/v5/pgtype"
)

// MockDatabaseInterface implements DatabaseInterface for testing
type MockDatabaseInterface struct {
	mock.Mock
}

func (m *MockDatabaseInterface) GetCharacterInventory(ctx context.Context, characterID pgtype.UUID) ([]db.CharacterInventory, error) {
	args := m.Called(ctx, characterID)
	return args.Get(0).([]db.CharacterInventory), args.Error(1)
}

func (m *MockDatabaseInterface) InventoryItemExists(ctx context.Context, arg db.InventoryItemExistsParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabaseInterface) AddInventoryItemQuantity(ctx context.Context, arg db.AddInventoryItemQuantityParams) (db.CharacterInventory, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.CharacterInventory), args.Error(1)
}

func (m *MockDatabaseInterface) CreateInventoryItem(ctx context.Context, arg db.CreateInventoryItemParams) (db.CharacterInventory, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.CharacterInventory), args.Error(1)
}

func (m *MockDatabaseInterface) RemoveInventoryItemQuantity(ctx context.Context, arg db.RemoveInventoryItemQuantityParams) (db.CharacterInventory, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.CharacterInventory), args.Error(1)
}

func (m *MockDatabaseInterface) DeleteInventoryItem(ctx context.Context, arg db.DeleteInventoryItemParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDatabaseInterface) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ResourceNode), args.Error(1)
}

// MockCharacterServiceInterface implements CharacterServiceInterface for testing
type MockCharacterServiceInterface struct {
	mock.Mock
}

// MockResourceNodeServiceInterface implements ResourceNodeServiceInterface for testing
type MockResourceNodeServiceInterface struct {
	mock.Mock
}

func (m *MockResourceNodeServiceInterface) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*resourceNodeV1.ResourceNodeType), args.Error(1)
}

// MockLoggerInterface implements LoggerInterface for testing
type MockLoggerInterface struct {
	mock.Mock
}

func (m *MockLoggerInterface) Debug(msg string, keysAndValues ...interface{}) {
	m.Called(append([]interface{}{msg}, keysAndValues...)...)
}

func (m *MockLoggerInterface) Info(msg string, keysAndValues ...interface{}) {
	m.Called(append([]interface{}{msg}, keysAndValues...)...)
}

func (m *MockLoggerInterface) Warn(msg string, keysAndValues ...interface{}) {
	m.Called(append([]interface{}{msg}, keysAndValues...)...)
}

func (m *MockLoggerInterface) Error(msg string, keysAndValues ...interface{}) {
	m.Called(append([]interface{}{msg}, keysAndValues...)...)
}

func (m *MockLoggerInterface) With(keysAndValues ...interface{}) LoggerInterface {
	args := m.Called(keysAndValues...)
	return args.Get(0).(LoggerInterface)
}

// Helper functions
func createTestCharacterUUID(uuidStr string) pgtype.UUID {
	uuidBytes, _ := hex.DecodeString(uuidStr)
	var bytes [16]byte
	copy(bytes[:], uuidBytes)
	return pgtype.UUID{Bytes: bytes, Valid: true}
}

func createTestInventoryItem(id int32, characterID string, resourceType int32, quantity int32) db.CharacterInventory {
	return db.CharacterInventory{
		ID:                 id,
		CharacterID:        createTestCharacterUUID(characterID),
		ResourceNodeTypeID: resourceType,
		Quantity:           quantity,
		CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
		UpdatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
	}
}

func createTestResourceNodeType(id int32, name string) *resourceNodeV1.ResourceNodeType {
	return &resourceNodeV1.ResourceNodeType{
		Id:   id,
		Name: name,
		Properties: &resourceNodeV1.ResourceProperties{
			YieldMin: 1,
			YieldMax: 5,
			SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
				{
					Name:      "secondary_item",
					Chance:    0.2,
					MinAmount: 1,
					MaxAmount: 2,
				},
			},
		},
	}
}

func createTestResourceNode(id int32, resourceTypeID int32) db.ResourceNode {
	return db.ResourceNode{
		ID:                 id,
		ResourceNodeTypeID: resourceTypeID,
		WorldID:            createTestCharacterUUID("550e8400e29b41d4a716446655440000"),
		ChunkX:             0,
		ChunkY:             0,
		ClusterID:          "cluster_1",
		PosX:               10,
		PosY:               20,
		Size:               1,
		CreatedAt:          pgtype.Timestamp{Valid: true, Time: time.Now()},
	}
}

func TestNewService(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		setupMocks     func() (DatabaseInterface, CharacterServiceInterface, ResourceNodeServiceInterface, LoggerInterface)
		expectNotNil   []string
		expectPanics   bool
	}{
		{
			name: "successful service creation with all dependencies",
			setupMocks: func() (DatabaseInterface, CharacterServiceInterface, ResourceNodeServiceInterface, LoggerInterface) {
				mockDB := &MockDatabaseInterface{}
				mockCharService := &MockCharacterServiceInterface{}
				mockResourceService := &MockResourceNodeServiceInterface{}
				mockLogger := &MockLoggerInterface{}
				
				// Logger expects With call during service creation
				mockLogger.On("With", "component", "inventory-service").Return(mockLogger)
				mockLogger.On("Debug", "Creating new inventory service").Return()
				
				return mockDB, mockCharService, mockResourceService, mockLogger
			},
			expectNotNil: []string{"db", "characterService", "resourceNodeService", "logger"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, charService, resourceService, logger := tt.setupMocks()
			
			if tt.expectPanics {
				assert.Panics(t, func() {
					NewService(db, charService, resourceService, logger)
				})
			} else {
				service := NewService(db, charService, resourceService, logger)
				require.NotNil(t, service)
				
				for _, field := range tt.expectNotNil {
					switch field {
					case "db":
						assert.NotNil(t, service.db)
					case "characterService":
						assert.NotNil(t, service.characterService)
					case "resourceNodeService":
						assert.NotNil(t, service.resourceNodeService)
					case "logger":
						assert.NotNil(t, service.logger)
					}
				}
			}
		})
	}
}

func TestService_GetCharacterInventory(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		characterID    string
		setupMocks     func(*MockDatabaseInterface, *MockResourceNodeServiceInterface, *MockLoggerInterface)
		expectError    bool
		expectCode     codes.Code
		expectErrorMsg string
		expectItems    int
	}{
		{
			name:        "successful inventory retrieval with items",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				inventoryItems := []db.CharacterInventory{
					createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10),
					createTestInventoryItem(2, "550e8400e29b41d4a716446655440000", 2, 5),
				}
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return(inventoryItems, nil)
				
				resourceTypes := []*resourceNodeV1.ResourceNodeType{
					createTestResourceNodeType(1, "Herb"),
					createTestResourceNodeType(2, "Berry"),
				}
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return(resourceTypes, nil)
				
				mockLogger.On("Debug", "Getting character inventory", "character_id", "550e8400e29b41d4a716446655440000")
				mockLogger.On("Debug", "Retrieved character inventory", "character_id", "550e8400e29b41d4a716446655440000", "item_count", 2)
			},
			expectError: false,
			expectItems: 2,
		},
		{
			name:        "empty inventory",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return([]db.CharacterInventory{}, nil)
				
				mockLogger.On("Debug", "Getting character inventory", "character_id", "550e8400e29b41d4a716446655440000")
				mockLogger.On("Debug", "Retrieved character inventory", "character_id", "550e8400e29b41d4a716446655440000", "item_count", 0)
			},
			expectError: false,
			expectItems: 0,
		},
		{
			name:        "invalid character ID format",
			characterID: "invalid-uuid",
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Getting character inventory", "character_id", "invalid-uuid")
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID format",
		},
		{
			name:        "database error",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return([]db.CharacterInventory{}, sql.ErrConnDone)
				
				mockLogger.On("Debug", "Getting character inventory", "character_id", "550e8400e29b41d4a716446655440000")
				mockLogger.On("Error", "Failed to get character inventory", "character_id", "550e8400e29b41d4a716446655440000", "error", sql.ErrConnDone)
			},
			expectError:    true,
			expectCode:     codes.Internal,
			expectErrorMsg: "failed to retrieve inventory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockResourceService := &MockResourceNodeServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockResourceService, mockLogger)
			
			service := &Service{
				db:                  mockDB,
				characterService:    mockCharService,
				resourceNodeService: mockResourceService,
				logger:              mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			items, err := service.GetCharacterInventory(ctx, tt.characterID)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, items)
				
				statusErr, ok := status.FromError(err)
				require.True(t, ok, "Expected gRPC status error")
				assert.Equal(t, tt.expectCode, statusErr.Code())
				if tt.expectErrorMsg != "" {
					assert.Contains(t, statusErr.Message(), tt.expectErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.expectItems)
				
				// Verify each item has expected properties
				for _, item := range items {
					assert.NotEmpty(t, item.CharacterId)
					assert.True(t, item.Quantity > 0)
					assert.NotNil(t, item.CreatedAt)
				}
			}
			
			mockDB.AssertExpectations(t)
			mockResourceService.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestService_AddInventoryItem(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name              string
		characterID       string
		resourceTypeID    resourceNodeV1.ResourceNodeTypeId
		quantity          int32
		setupMocks        func(*MockDatabaseInterface, *MockResourceNodeServiceInterface, *MockLoggerInterface)
		expectError       bool
		expectCode        codes.Code
		expectErrorMsg    string
		expectQuantity    int32
	}{
		{
			name:           "successfully add new inventory item",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedExistsParams := db.InventoryItemExistsParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
				}
				expectedCreateParams := db.CreateInventoryItemParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					Quantity:           10,
				}
				
				createdItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10)
				
				mockDB.On("InventoryItemExists", mock.Anything, expectedExistsParams).Return(false, nil)
				mockDB.On("CreateInventoryItem", mock.Anything, expectedCreateParams).Return(createdItem, nil)
				
				resourceTypes := []*resourceNodeV1.ResourceNodeType{
					createTestResourceNodeType(1, "Herb"),
				}
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return(resourceTypes, nil)
				
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(10))
				mockLogger.On("Debug", "Added inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "new_quantity", int32(10))
			},
			expectError:    false,
			expectQuantity: 10,
		},
		{
			name:           "successfully add to existing inventory item",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       5,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedExistsParams := db.InventoryItemExistsParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
				}
				expectedAddParams := db.AddInventoryItemQuantityParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					Quantity:           5,
				}
				
				updatedItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 15)
				
				mockDB.On("InventoryItemExists", mock.Anything, expectedExistsParams).Return(true, nil)
				mockDB.On("AddInventoryItemQuantity", mock.Anything, expectedAddParams).Return(updatedItem, nil)
				
				resourceTypes := []*resourceNodeV1.ResourceNodeType{
					createTestResourceNodeType(1, "Herb"),
				}
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return(resourceTypes, nil)
				
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(5))
				mockLogger.On("Debug", "Added inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "new_quantity", int32(15))
			},
			expectError:    false,
			expectQuantity: 15,
		},
		{
			name:           "invalid quantity (zero)",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(0))
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "quantity must be positive",
		},
		{
			name:           "invalid quantity (negative)",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       -5,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(-5))
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "quantity must be positive",
		},
		{
			name:           "invalid character ID format",
			characterID:    "invalid-uuid",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Adding inventory item", "character_id", "invalid-uuid", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(10))
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID format",
		},
		{
			name:           "database error on item existence check",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedExistsParams := db.InventoryItemExistsParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
				}
				
				mockDB.On("InventoryItemExists", mock.Anything, expectedExistsParams).Return(false, sql.ErrConnDone)
				
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(10))
				mockLogger.On("Error", "Failed to check inventory item existence", "error", sql.ErrConnDone)
			},
			expectError:    true,
			expectCode:     codes.Internal,
			expectErrorMsg: "failed to check inventory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockResourceService := &MockResourceNodeServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockResourceService, mockLogger)
			
			service := &Service{
				db:                  mockDB,
				characterService:    mockCharService,
				resourceNodeService: mockResourceService,
				logger:              mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			item, err := service.AddInventoryItem(ctx, tt.characterID, tt.resourceTypeID, tt.quantity)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, item)
				
				statusErr, ok := status.FromError(err)
				require.True(t, ok, "Expected gRPC status error")
				assert.Equal(t, tt.expectCode, statusErr.Code())
				if tt.expectErrorMsg != "" {
					assert.Contains(t, statusErr.Message(), tt.expectErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, item)
				assert.Equal(t, tt.expectQuantity, item.Quantity)
				assert.Equal(t, tt.characterID, item.CharacterId)
				assert.Equal(t, tt.resourceTypeID, item.ResourceNodeTypeId)
			}
			
			mockDB.AssertExpectations(t)
			mockResourceService.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestService_RemoveInventoryItem(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name              string
		characterID       string
		resourceTypeID    resourceNodeV1.ResourceNodeTypeId
		quantity          int32
		setupMocks        func(*MockDatabaseInterface, *MockResourceNodeServiceInterface, *MockLoggerInterface)
		expectError       bool
		expectCode        codes.Code
		expectErrorMsg    string
		expectNilItem     bool
		expectQuantity    int32
	}{
		{
			name:           "successfully remove partial quantity",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       5,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedRemoveParams := db.RemoveInventoryItemQuantityParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					Quantity:           5,
				}
				
				remainingItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 5)
				
				mockDB.On("RemoveInventoryItemQuantity", mock.Anything, expectedRemoveParams).Return(remainingItem, nil)
				
				resourceTypes := []*resourceNodeV1.ResourceNodeType{
					createTestResourceNodeType(1, "Herb"),
				}
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return(resourceTypes, nil)
				
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(5))
				mockLogger.On("Debug", "Removed inventory item quantity", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "remaining_quantity", int32(5))
			},
			expectError:    false,
			expectQuantity: 5,
		},
		{
			name:           "successfully remove all quantity (item deleted)",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedRemoveParams := db.RemoveInventoryItemQuantityParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					Quantity:           10,
				}
				expectedDeleteParams := db.DeleteInventoryItemParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
				}
				
				// Item with zero quantity after removal
				removedItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 0)
				
				mockDB.On("RemoveInventoryItemQuantity", mock.Anything, expectedRemoveParams).Return(removedItem, nil)
				mockDB.On("DeleteInventoryItem", mock.Anything, expectedDeleteParams).Return(nil)
				
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(10))
			},
			expectError:   false,
			expectNilItem: true,
		},
		{
			name:           "invalid quantity (zero)",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(0))
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "quantity must be positive",
		},
		{
			name:           "invalid character ID format",
			characterID:    "invalid-uuid",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       5,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Removing inventory item", "character_id", "invalid-uuid", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(5))
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID format",
		},
		{
			name:           "insufficient quantity error",
			characterID:    "550e8400e29b41d4a716446655440000",
			resourceTypeID: resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH,
			quantity:       100,
			setupMocks: func(mockDB *MockDatabaseInterface, mockResourceService *MockResourceNodeServiceInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedRemoveParams := db.RemoveInventoryItemQuantityParams{
					CharacterID:        expectedUUID,
					ResourceNodeTypeID: int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
					Quantity:           100,
				}
				
				mockDB.On("RemoveInventoryItemQuantity", mock.Anything, expectedRemoveParams).Return(db.CharacterInventory{}, sql.ErrNoRows)
				
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "resource_type", resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH, "quantity", int32(100))
				mockLogger.On("Error", "Failed to remove inventory item quantity", "error", sql.ErrNoRows)
			},
			expectError:    true,
			expectCode:     codes.Internal,
			expectErrorMsg: "failed to remove inventory item or insufficient quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockResourceService := &MockResourceNodeServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockResourceService, mockLogger)
			
			service := &Service{
				db:                  mockDB,
				characterService:    mockCharService,
				resourceNodeService: mockResourceService,
				logger:              mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			item, err := service.RemoveInventoryItem(ctx, tt.characterID, tt.resourceTypeID, tt.quantity)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, item)
				
				statusErr, ok := status.FromError(err)
				require.True(t, ok, "Expected gRPC status error")
				assert.Equal(t, tt.expectCode, statusErr.Code())
				if tt.expectErrorMsg != "" {
					assert.Contains(t, statusErr.Message(), tt.expectErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				
				if tt.expectNilItem {
					assert.Nil(t, item)
				} else {
					require.NotNil(t, item)
					assert.Equal(t, tt.expectQuantity, item.Quantity)
					assert.Equal(t, tt.characterID, item.CharacterId)
					assert.Equal(t, tt.resourceTypeID, item.ResourceNodeTypeId)
				}
			}
			
			mockDB.AssertExpectations(t)
			mockResourceService.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// NOTE: HarvestResourceNode functionality has been moved to character_actions service
// These tests are commented out as the method no longer exists in this service

func TestService_dbInventoryItemToProto(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name                  string
		input                 db.CharacterInventory
		includeResourceType   bool
		setupMocks            func(*MockResourceNodeServiceInterface)
		expectError           bool
		expectResourceType    bool
	}{
		{
			name:                "successful conversion with resource type",
			input:               createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10),
			includeResourceType: true,
			setupMocks: func(mockResourceService *MockResourceNodeServiceInterface) {
				resourceTypes := []*resourceNodeV1.ResourceNodeType{
					createTestResourceNodeType(1, "Herb"),
				}
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return(resourceTypes, nil)
			},
			expectError:        false,
			expectResourceType: true,
		},
		{
			name:                "successful conversion without resource type",
			input:               createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10),
			includeResourceType: false,
			setupMocks:          func(mockResourceService *MockResourceNodeServiceInterface) {},
			expectError:         false,
			expectResourceType:  false,
		},
		{
			name:                "resource type service error (non-blocking)",
			input:               createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10),
			includeResourceType: true,
			setupMocks: func(mockResourceService *MockResourceNodeServiceInterface) {
				mockResourceService.On("GetResourceNodeTypes", mock.Anything).Return([]*resourceNodeV1.ResourceNodeType{}, sql.ErrConnDone)
			},
			expectError:        false, // Error is logged but not returned
			expectResourceType: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockResourceService := &MockResourceNodeServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			if tt.includeResourceType && tt.name == "resource type service error (non-blocking)" {
				mockLogger.On("Warn", "Failed to get resource node types", "error", sql.ErrConnDone)
			}
			
			tt.setupMocks(mockResourceService)
			
			service := &Service{
				db:                  mockDB,
				characterService:    mockCharService,
				resourceNodeService: mockResourceService,
				logger:              mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			result, err := service.dbInventoryItemToProto(ctx, tt.input, tt.includeResourceType)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				
				assert.Equal(t, tt.input.ID, result.Id)
				assert.Equal(t, "550e8400e29b41d4a716446655440000", result.CharacterId)
				assert.Equal(t, resourceNodeV1.ResourceNodeTypeId(tt.input.ResourceNodeTypeID), result.ResourceNodeTypeId)
				assert.Equal(t, tt.input.Quantity, result.Quantity)
				
				if tt.expectResourceType {
					assert.NotNil(t, result.ResourceNodeType)
					assert.Equal(t, "Herb", result.ResourceNodeType.Name)
				} else {
					assert.Nil(t, result.ResourceNodeType)
				}
				
				// Check timestamps
				if tt.input.CreatedAt.Valid {
					assert.NotNil(t, result.CreatedAt)
				} else {
					assert.Nil(t, result.CreatedAt)
				}
				
				if tt.input.UpdatedAt.Valid {
					assert.NotNil(t, result.UpdatedAt)
				} else {
					assert.Nil(t, result.UpdatedAt)
				}
			}
			
			mockResourceService.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}