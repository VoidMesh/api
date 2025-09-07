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
	"github.com/jackc/pgx/v5/pgtype"
)

// MockDatabaseInterface implements DatabaseInterface for testing
type MockDatabaseInterface struct {
	mock.Mock
}

func (m *MockDatabaseInterface) GetCharacterInventory(ctx context.Context, characterID pgtype.UUID) ([]db.GetCharacterInventoryRow, error) {
	args := m.Called(ctx, characterID)
	return args.Get(0).([]db.GetCharacterInventoryRow), args.Error(1)
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

func createTestInventoryItem(id int32, characterID string, itemID int32, quantity int32) db.CharacterInventory {
	return db.CharacterInventory{
		ID:          id,
		CharacterID: createTestCharacterUUID(characterID),
		ItemID:      itemID,
		Quantity:    quantity,
		CreatedAt:   pgtype.Timestamp{Valid: true, Time: time.Now()},
		UpdatedAt:   pgtype.Timestamp{Valid: true, Time: time.Now()},
	}
}

func createTestInventoryRow(id int32, characterID string, itemID int32, quantity int32, itemName string, itemDescription string, itemType string, rarity string, stackSize int32) db.GetCharacterInventoryRow {
	return db.GetCharacterInventoryRow{
		ID:              id,
		CharacterID:     createTestCharacterUUID(characterID),
		ItemID:          itemID,
		Quantity:        quantity,
		CreatedAt:       pgtype.Timestamp{Valid: true, Time: time.Now()},
		UpdatedAt:       pgtype.Timestamp{Valid: true, Time: time.Now()},
		ItemName:        itemName,
		ItemDescription: itemDescription,
		ItemType:        itemType,
		Rarity:          rarity,
		StackSize:       stackSize,
		VisualData:      []byte{},
	}
}



func TestNewService(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name           string
		setupMocks     func() (DatabaseInterface, CharacterServiceInterface, LoggerInterface)
		expectNotNil   []string
		expectPanics   bool
	}{
		{
			name: "successful service creation with all dependencies",
			setupMocks: func() (DatabaseInterface, CharacterServiceInterface, LoggerInterface) {
				mockDB := &MockDatabaseInterface{}
				mockCharService := &MockCharacterServiceInterface{}
				mockLogger := &MockLoggerInterface{}
				
				// Logger expects With call during service creation
				mockLogger.On("With", "component", "inventory-service").Return(mockLogger)
				mockLogger.On("Debug", "Creating new inventory service").Return()
				
				return mockDB, mockCharService, mockLogger
			},
			expectNotNil: []string{"db", "characterService", "logger"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, charService, logger := tt.setupMocks()
			
			if tt.expectPanics {
				assert.Panics(t, func() {
					NewService(db, charService, logger)
				})
			} else {
				service := NewService(db, charService, logger)
				require.NotNil(t, service)
				
				for _, field := range tt.expectNotNil {
					switch field {
					case "db":
						assert.NotNil(t, service.db)
					case "characterService":
						assert.NotNil(t, service.characterService)
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
		setupMocks     func(*MockDatabaseInterface, *MockLoggerInterface)
		expectError    bool
		expectCode     codes.Code
		expectErrorMsg string
		expectItems    int
	}{
		{
			name:        "successful inventory retrieval with items",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				inventoryItems := []db.GetCharacterInventoryRow{
					createTestInventoryRow(1, "550e8400e29b41d4a716446655440000", 1, 10, "Herb Patch", "A valuable herb", "resource", "common", 100),
					createTestInventoryRow(2, "550e8400e29b41d4a716446655440000", 2, 5, "Berry Bush", "Sweet berries", "resource", "common", 50),
				}
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return(inventoryItems, nil)
				
				mockLogger.On("Debug", "Getting character inventory", "character_id", "550e8400e29b41d4a716446655440000")
				mockLogger.On("Debug", "Retrieved character inventory", "character_id", "550e8400e29b41d4a716446655440000", "item_count", 2)
			},
			expectError: false,
			expectItems: 2,
		},
		{
			name:        "empty inventory",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return([]db.GetCharacterInventoryRow{}, nil)
				
				mockLogger.On("Debug", "Getting character inventory", "character_id", "550e8400e29b41d4a716446655440000")
				mockLogger.On("Debug", "Retrieved character inventory", "character_id", "550e8400e29b41d4a716446655440000", "item_count", 0)
			},
			expectError: false,
			expectItems: 0,
		},
		{
			name:        "invalid character ID format",
			characterID: "invalid-uuid",
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				mockLogger.On("Debug", "Getting character inventory", "character_id", "invalid-uuid")
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID format",
		},
		{
			name:        "database error",
			characterID: "550e8400e29b41d4a716446655440000",
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				
				mockDB.On("GetCharacterInventory", mock.Anything, expectedUUID).Return([]db.GetCharacterInventoryRow{}, sql.ErrConnDone)
				
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
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockLogger)
			
			service := &Service{
				db:               mockDB,
				characterService: mockCharService,
				logger:           mockLogger,
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
					// Verify JOIN data is populated
					assert.NotEmpty(t, item.ItemName)
					assert.NotEmpty(t, item.ItemType)
				}
			}
			
			mockDB.AssertExpectations(t)
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
		itemID            int32
		quantity          int32
		setupMocks        func(*MockDatabaseInterface, *MockLoggerInterface)
		expectError       bool
		expectCode        codes.Code
		expectErrorMsg    string
		expectQuantity    int32
	}{
		{
			name:        "successfully add new inventory item",
			characterID: "550e8400e29b41d4a716446655440000",
			itemID:      1,
			quantity:    10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedExistsParams := db.InventoryItemExistsParams{
					CharacterID: expectedUUID,
					ItemID:      1,
				}
				expectedCreateParams := db.CreateInventoryItemParams{
					CharacterID: expectedUUID,
					ItemID:      1,
					Quantity:    10,
				}
				
				createdItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 10)
				
				mockDB.On("InventoryItemExists", mock.Anything, expectedExistsParams).Return(false, nil)
				mockDB.On("CreateInventoryItem", mock.Anything, expectedCreateParams).Return(createdItem, nil)
				
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "quantity", int32(10))
				mockLogger.On("Debug", "Added inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "new_quantity", int32(10))
			},
			expectError:    false,
			expectQuantity: 10,
		},
		{
			name:        "invalid quantity (zero)",
			characterID: "550e8400e29b41d4a716446655440000",
			itemID:      1,
			quantity:    0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				// Mock logger call that occurs before validation
				mockLogger.On("Debug", "Adding inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "quantity", int32(0)).Return()
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "quantity must be positive",
		},
		{
			name:        "invalid character ID format",
			characterID: "invalid-uuid",
			itemID:      1,
			quantity:    10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				// Mock logger call that occurs before validation
				mockLogger.On("Debug", "Adding inventory item", "character_id", "invalid-uuid", "item_id", int32(1), "quantity", int32(10)).Return()
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockLogger)
			
			service := &Service{
				db:               mockDB,
				characterService: mockCharService,
				logger:           mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			item, err := service.AddInventoryItem(ctx, tt.characterID, tt.itemID, tt.quantity)
			
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
				assert.Equal(t, tt.itemID, item.ItemId)
			}
			
			mockDB.AssertExpectations(t)
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
		itemID            int32
		quantity          int32
		setupMocks        func(*MockDatabaseInterface, *MockLoggerInterface)
		expectError       bool
		expectCode        codes.Code
		expectErrorMsg    string
		expectNilItem     bool
		expectQuantity    int32
	}{
		{
			name:        "successfully remove partial quantity",
			characterID: "550e8400e29b41d4a716446655440000",
			itemID:      1,
			quantity:    5,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedRemoveParams := db.RemoveInventoryItemQuantityParams{
					CharacterID: expectedUUID,
					ItemID:      1,
					Quantity:    5,
				}
				
				remainingItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 5)
				
				mockDB.On("RemoveInventoryItemQuantity", mock.Anything, expectedRemoveParams).Return(remainingItem, nil)
				
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "quantity", int32(5))
				mockLogger.On("Debug", "Removed inventory item quantity", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "remaining_quantity", int32(5))
			},
			expectError:    false,
			expectQuantity: 5,
		},
		{
			name:        "successfully remove all quantity (item deleted)",
			characterID: "550e8400e29b41d4a716446655440000",
			itemID:      1,
			quantity:    10,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				expectedUUID := createTestCharacterUUID("550e8400e29b41d4a716446655440000")
				expectedRemoveParams := db.RemoveInventoryItemQuantityParams{
					CharacterID: expectedUUID,
					ItemID:      1,
					Quantity:    10,
				}
				expectedDeleteParams := db.DeleteInventoryItemParams{
					CharacterID: expectedUUID,
					ItemID:      1,
				}
				
				// Item with zero quantity after removal
				removedItem := createTestInventoryItem(1, "550e8400e29b41d4a716446655440000", 1, 0)
				
				mockDB.On("RemoveInventoryItemQuantity", mock.Anything, expectedRemoveParams).Return(removedItem, nil)
				mockDB.On("DeleteInventoryItem", mock.Anything, expectedDeleteParams).Return(nil)
				
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "quantity", int32(10))
			},
			expectError:   false,
			expectNilItem: true,
		},
		{
			name:        "invalid quantity (zero)",
			characterID: "550e8400e29b41d4a716446655440000",
			itemID:      1,
			quantity:    0,
			setupMocks: func(mockDB *MockDatabaseInterface, mockLogger *MockLoggerInterface) {
				// Mock logger call that occurs before validation
				mockLogger.On("Debug", "Removing inventory item", "character_id", "550e8400e29b41d4a716446655440000", "item_id", int32(1), "quantity", int32(0)).Return()
			},
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "quantity must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabaseInterface{}
			mockCharService := &MockCharacterServiceInterface{}
			mockLogger := &MockLoggerInterface{}
			
			tt.setupMocks(mockDB, mockLogger)
			
			service := &Service{
				db:               mockDB,
				characterService: mockCharService,
				logger:           mockLogger,
			}
			
			ctx := testutil.CreateTestContext()
			item, err := service.RemoveInventoryItem(ctx, tt.characterID, tt.itemID, tt.quantity)
			
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
					assert.Equal(t, tt.itemID, item.ItemId)
				}
			}
			
			mockDB.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// NOTE: HarvestResourceNode functionality has been moved to character_actions service
// These tests are commented out as the method no longer exists in this service

// dbInventoryItemToProto and dbInventoryRowToProto tests have been removed
// as these are now private helper functions used internally by the service