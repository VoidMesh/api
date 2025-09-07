package character_actions

import (
	"context"
	"testing"

	"github.com/VoidMesh/api/api/db"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock implementations
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ResourceNode), args.Error(1)
}

type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) AddInventoryItem(ctx context.Context, characterID string, resourceNodeTypeID resourceNodeV1.ResourceNodeTypeId, quantity int32) (*inventoryV1.InventoryItem, error) {
	args := m.Called(ctx, characterID, resourceNodeTypeID, quantity)
	return args.Get(0).(*inventoryV1.InventoryItem), args.Error(1)
}

type MockCharacterService struct {
	mock.Mock
}

func (m *MockCharacterService) GetCharacterByID(ctx context.Context, characterID string) (*db.Character, error) {
	args := m.Called(ctx, characterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Character), args.Error(1)
}

type MockResourceNodeService struct {
	mock.Mock
}

func (m *MockResourceNodeService) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*resourceNodeV1.ResourceNodeType), args.Error(1)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) With(key string, value interface{}) LoggerInterface {
	args := m.Called(key, value)
	return args.Get(0).(LoggerInterface)
}

func TestService_HarvestResource_Success(t *testing.T) {
	// Setup mocks
	mockDB := &MockDatabase{}
	mockInventory := &MockInventoryService{}
	mockCharacter := &MockCharacterService{}
	mockResourceNode := &MockResourceNodeService{}
	mockLogger := &MockLogger{}

	// Setup logger chain
	mockLogger.On("With", "component", "character-actions-service").Return(mockLogger)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()

	service := NewService(mockDB, mockInventory, mockCharacter, mockResourceNode, mockLogger)

	ctx := context.Background()
	characterID := "0123456789abcdef0123456789abcdef" // 32 hex chars
	resourceNodeID := int32(1)

	// Setup test data  
	userID := "12345678-9abc-def0-1234-56789abcdef0" // UUID format string
	// Create a proper UUID for the user - using same UUID
	userUUIDBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	character := &db.Character{
		ID:     pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		UserID: pgtype.UUID{Bytes: userUUIDBytes, Valid: true},
		Name:   "TestCharacter",
		X:      10,
		Y:      10,
		ChunkX: 0,
		ChunkY: 0,
	}

	resourceNode := db.ResourceNode{
		ID:                 1,
		ResourceNodeTypeID: 1,
		WorldID:            pgtype.UUID{Bytes: [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, Valid: true},
		ChunkX:             0,
		ChunkY:             0,
		PosX:               12, // Within range (distance = 2.828)
		PosY:               12,
		Size:               1,
	}

	resourceType := &resourceNodeV1.ResourceNodeType{
		Id:   1,
		Name: "Tree",
		Properties: &resourceNodeV1.ResourceProperties{
			YieldMin: 1,
			YieldMax: 3,
			SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
				{
					Name:      "Stick",
					Chance:    0.3,
					MinAmount: 1,
					MaxAmount: 2,
				},
			},
		},
	}

	inventoryItem := &inventoryV1.InventoryItem{
		Id:                 1,
		CharacterId:        characterID,
		ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId(1),
		Quantity:           2,
	}

	// Setup expectations
	mockCharacter.On("GetCharacterByID", ctx, characterID).Return(character, nil)
	mockDB.On("GetResourceNode", ctx, resourceNodeID).Return(resourceNode, nil)
	mockResourceNode.On("GetResourceNodeTypes", ctx).Return([]*resourceNodeV1.ResourceNodeType{resourceType}, nil)
	mockInventory.On("AddInventoryItem", ctx, characterID, resourceNodeV1.ResourceNodeTypeId(1), mock.AnythingOfType("int32")).Return(inventoryItem, nil)

	// Execute
	results, updatedItem, err := service.HarvestResource(ctx, userID, characterID, resourceNodeID)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, results)
	require.NotNil(t, updatedItem)
	assert.Len(t, results, 1) // Primary yield only (secondary is random)
	assert.Equal(t, "Tree", results[0].ItemName)
	assert.False(t, results[0].IsSecondaryDrop)
	assert.True(t, results[0].Quantity >= 1 && results[0].Quantity <= 3)
	assert.Equal(t, inventoryItem, updatedItem)

	// Verify all mocks were called
	mockCharacter.AssertExpectations(t)
	mockDB.AssertExpectations(t)
	mockResourceNode.AssertExpectations(t)
	mockInventory.AssertExpectations(t)
}

func TestService_HarvestResource_InvalidCharacterID(t *testing.T) {
	// Setup mocks
	mockDB := &MockDatabase{}
	mockInventory := &MockInventoryService{}
	mockCharacter := &MockCharacterService{}
	mockResourceNode := &MockResourceNodeService{}
	mockLogger := &MockLogger{}

	// Setup logger chain
	mockLogger.On("With", "component", "character-actions-service").Return(mockLogger)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()

	service := NewService(mockDB, mockInventory, mockCharacter, mockResourceNode, mockLogger)

	ctx := context.Background()
	invalidCharacterID := "invalid-hex"
	resourceNodeID := int32(1)

	// Execute
	results, updatedItem, err := service.HarvestResource(ctx, "12345678-9abc-def0-1234-56789abcdef0", invalidCharacterID, resourceNodeID)

	// Verify
	assert.Nil(t, results)
	assert.Nil(t, updatedItem)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid character ID format")
}

func TestService_HarvestResource_CharacterNotFound(t *testing.T) {
	// Setup mocks
	mockDB := &MockDatabase{}
	mockInventory := &MockInventoryService{}
	mockCharacter := &MockCharacterService{}
	mockResourceNode := &MockResourceNodeService{}
	mockLogger := &MockLogger{}

	// Setup logger chain
	mockLogger.On("With", "component", "character-actions-service").Return(mockLogger)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Error", mock.AnythingOfType("string"), mock.Anything).Return()

	service := NewService(mockDB, mockInventory, mockCharacter, mockResourceNode, mockLogger)

	ctx := context.Background()
	characterID := "0123456789abcdef0123456789abcdef"
	resourceNodeID := int32(1)

	// Setup expectations
	mockCharacter.On("GetCharacterByID", ctx, characterID).Return(nil, assert.AnError)

	// Execute
	results, updatedItem, err := service.HarvestResource(ctx, "12345678-9abc-def0-1234-56789abcdef0", characterID, resourceNodeID)

	// Verify
	assert.Nil(t, results)
	assert.Nil(t, updatedItem)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, st.Message(), "character not found")

	mockCharacter.AssertExpectations(t)
}

func TestService_HarvestResource_OutOfRange(t *testing.T) {
	// Setup mocks
	mockDB := &MockDatabase{}
	mockInventory := &MockInventoryService{}
	mockCharacter := &MockCharacterService{}
	mockResourceNode := &MockResourceNodeService{}
	mockLogger := &MockLogger{}

	// Setup logger chain
	mockLogger.On("With", "component", "character-actions-service").Return(mockLogger)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()

	service := NewService(mockDB, mockInventory, mockCharacter, mockResourceNode, mockLogger)

	ctx := context.Background()
	characterID := "0123456789abcdef0123456789abcdef"
	resourceNodeID := int32(1)

	// Setup test data - character far from resource node but owned by correct user
	userID := "12345678-9abc-def0-1234-56789abcdef0"
	userUUIDBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	character := &db.Character{
		ID:     pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		UserID: pgtype.UUID{Bytes: userUUIDBytes, Valid: true},
		Name:   "TestCharacter",
		X:      0,
		Y:      0,
		ChunkX: 0,
		ChunkY: 0,
	}

	resourceNode := db.ResourceNode{
		ID:                 1,
		ResourceNodeTypeID: 1,
		WorldID:            pgtype.UUID{Bytes: [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, Valid: true},
		ChunkX:             0,
		ChunkY:             0,
		PosX:               10, // Distance = 14.14, > maxHarvestDistance (3)
		PosY:               10,
		Size:               1,
	}

	// Setup expectations
	mockCharacter.On("GetCharacterByID", ctx, characterID).Return(character, nil)
	mockDB.On("GetResourceNode", ctx, resourceNodeID).Return(resourceNode, nil)

	// Execute
	results, updatedItem, err := service.HarvestResource(ctx, userID, characterID, resourceNodeID)

	// Verify
	assert.Nil(t, results)
	assert.Nil(t, updatedItem)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Contains(t, st.Message(), "too far from resource node")

	mockCharacter.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestService_isCharacterInRange(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name         string
		character    *db.Character
		resourceNode *db.ResourceNode
		expected     bool
	}{
		{
			name: "Same position - in range",
			character: &db.Character{
				X: 10, Y: 10, ChunkX: 0, ChunkY: 0,
			},
			resourceNode: &db.ResourceNode{
				PosX: 10, PosY: 10, ChunkX: 0, ChunkY: 0,
			},
			expected: true,
		},
		{
			name: "Within range - distance 2",
			character: &db.Character{
				X: 10, Y: 10, ChunkX: 0, ChunkY: 0,
			},
			resourceNode: &db.ResourceNode{
				PosX: 12, PosY: 10, ChunkX: 0, ChunkY: 0,
			},
			expected: true,
		},
		{
			name: "Out of range - distance 5",
			character: &db.Character{
				X: 10, Y: 10, ChunkX: 0, ChunkY: 0,
			},
			resourceNode: &db.ResourceNode{
				PosX: 15, PosY: 10, ChunkX: 0, ChunkY: 0,
			},
			expected: false,
		},
		{
			name: "Different chunk",
			character: &db.Character{
				X: 10, Y: 10, ChunkX: 0, ChunkY: 0,
			},
			resourceNode: &db.ResourceNode{
				PosX: 10, PosY: 10, ChunkX: 1, ChunkY: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isCharacterInRange(tt.character, tt.resourceNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_HarvestResource_CharacterOwnershipValidation(t *testing.T) {
	// Setup mocks
	mockDB := &MockDatabase{}
	mockInventory := &MockInventoryService{}
	mockCharacter := &MockCharacterService{}
	mockResourceNode := &MockResourceNodeService{}
	mockLogger := &MockLogger{}

	// Setup logger chain
	mockLogger.On("With", "component", "character-actions-service").Return(mockLogger)
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()

	service := NewService(mockDB, mockInventory, mockCharacter, mockResourceNode, mockLogger)

	ctx := context.Background()
	characterID := "0123456789abcdef0123456789abcdef"
	resourceNodeID := int32(1)
	userID := "12345678-9abc-def0-1234-56789abcdef0"

	// Setup test data - character belonging to different user
	character := &db.Character{
		ID:     pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		Name:   "TestCharacter",
		X:      10,
		Y:      10,
		ChunkX: 0,
		ChunkY: 0,
	}
	// Set character to belong to different user
	otherUserUUIDBytes := [16]byte{0x87, 0x65, 0x43, 0x21, 0xfe, 0xdc, 0xba, 0x09, 0x87, 0x65, 0x43, 0x21, 0xfe, 0xdc, 0xba, 0x09}
	character.UserID = pgtype.UUID{Bytes: otherUserUUIDBytes, Valid: true}

	// Setup expectations - character service should be called
	mockCharacter.On("GetCharacterByID", ctx, characterID).Return(character, nil)

	// Execute - try to harvest with different user
	results, updatedItem, err := service.HarvestResource(ctx, userID, characterID, resourceNodeID)

	// Verify
	assert.Nil(t, results)
	assert.Nil(t, updatedItem)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Contains(t, st.Message(), "character not owned by user")

	mockCharacter.AssertExpectations(t)
}

func TestService_validateCharacterOwnership(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()
	service := &Service{logger: mockLogger}

	tests := []struct {
		name            string
		characterUserID string
		requestUserID   string
		expectError     bool
	}{
		{
			name:            "Valid ownership - same user",
			characterUserID: "12345678-9abc-def0-1234-56789abcdef0",
			requestUserID:   "12345678-9abc-def0-1234-56789abcdef0",
			expectError:     false,
		},
		{
			name:            "Invalid ownership - different user",
			characterUserID: "12345678-9abc-def0-1234-56789abcdef0",
			requestUserID:   "87654321-fedc-ba09-8765-4321fedcba09",
			expectError:     true,
		},
		{
			name:            "Empty request user ID",
			characterUserID: "12345678-9abc-def0-1234-56789abcdef0",
			requestUserID:   "",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			character := &db.Character{
				ID:   pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
				Name: "TestCharacter",
			}
			
			// Set character user ID based on the test case
			if tt.characterUserID == "12345678-9abc-def0-1234-56789abcdef0" {
				userUUIDBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
				character.UserID = pgtype.UUID{Bytes: userUUIDBytes, Valid: true}
			} else {
				// For empty or other test cases, use a different UUID or invalid one
				character.UserID = pgtype.UUID{Valid: false}
			}

			err := service.validateCharacterOwnership(character, tt.requestUserID)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.PermissionDenied, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}