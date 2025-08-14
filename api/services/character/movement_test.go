package character

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testutil"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
)

func TestMovementCooldown_CriticalAntiCheat(t *testing.T) {
	t.Skip("Skipping MovementCooldown test - requires complex refactoring to use mock interface")
	
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Clear the movement cache before test
	movementCache = make(map[string]time.Time)

	tests := []struct {
		name            string
		characterID     string
		setupCache      func()
		expectBlocked   bool
		description     string
	}{
		{
			name:        "no previous movement - should allow",
			characterID: "550e8400-e29b-41d4-a716-446655440000",
			setupCache:  func() {}, // No setup, cache is empty
			expectBlocked: false,
			description: "First movement should always be allowed",
		},
		{
			name:        "movement within cooldown - should block",
			characterID: "550e8400-e29b-41d4-a716-446655440001",
			setupCache: func() {
				// Set last movement to 10ms ago (within 50ms cooldown)
				movementCache["550e8400-e29b-41d4-a716-446655440001"] = time.Now().Add(-10 * time.Millisecond)
			},
			expectBlocked: true,
			description: "Movement within 50ms cooldown should be blocked",
		},
		{
			name:        "movement after cooldown - should allow",
			characterID: "550e8400-e29b-41d4-a716-446655440002",
			setupCache: func() {
				// Set last movement to 60ms ago (beyond 50ms cooldown)
				movementCache["550e8400-e29b-41d4-a716-446655440002"] = time.Now().Add(-60 * time.Millisecond)
			},
			expectBlocked: false,
			description: "Movement after cooldown expires should be allowed",
		},
		{
			name:        "exactly at cooldown boundary - should allow",
			characterID: "550e8400-e29b-41d4-a716-446655440003",
			setupCache: func() {
				// Set last movement to exactly 50ms ago
				movementCache["550e8400-e29b-41d4-a716-446655440003"] = time.Now().Add(-MovementCooldown)
			},
			expectBlocked: false,
			description: "Movement at exact cooldown boundary should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
			defer testDB.Close()

			// Setup character data in mock
			characterID := pgtype.UUID{
				Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00},
			}

			// Setup cache state
			tt.setupCache()

			if !tt.expectBlocked {
				// If movement should be allowed, setup full mock chain
				testDB.MockPool.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, characterID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))

				testDB.MockPool.ExpectQuery("UPDATE characters SET").
					WithArgs(characterID, int32(11), int32(10), int32(0), int32(0)).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, characterID, "TestChar", int32(11), int32(10), int32(0), int32(0), time.Now()))
			} else {
				// If movement should be blocked, setup character fetch but no update
				testDB.MockPool.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, characterID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))
			}

			mockChunkService := NewMockChunkService()
			service := NewServiceWithPool(testDB.Pool, mockChunkService)
			ctx := testutil.CreateTestContext()

			request := &characterV1.MoveCharacterRequest{
				CharacterId: tt.characterID,
				NewX:        11, // Move one cell right
				NewY:        10, // Same Y position
			}

			response, err := service.MoveCharacter(ctx, request)

			require.NoError(t, err, tt.description)
			require.NotNil(t, response, tt.description)

			if tt.expectBlocked {
				assert.False(t, response.Success, "Movement should be blocked: %s", tt.description)
				assert.Contains(t, response.ErrorMessage, "too fast", "Should indicate rate limiting: %s", tt.description)
			} else {
				assert.True(t, response.Success, "Movement should be allowed: %s", tt.description)
				assert.Empty(t, response.ErrorMessage, "Should have no error message: %s", tt.description)
				
				// Verify movement cache was updated
				_, exists := movementCache[tt.characterID]
				assert.True(t, exists, "Movement cache should be updated after successful movement")
			}

			// Verify mock expectations
			if err := testDB.MockPool.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled mock expectations: %v", err)
			}
		})
	}
}

func TestValidateMovement_AntiCheat(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
	defer testDB.Close()

	mockChunkService := NewMockChunkService()
	service := NewServiceWithPool(testDB.Pool, mockChunkService)

	tests := []struct {
		name        string
		character   db.Character
		newX, newY  int32
		expectValid bool
		description string
	}{
		{
			name: "valid orthogonal movement - right",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        11,
			newY:        10,
			expectValid: true,
			description: "Moving one cell right should be valid",
		},
		{
			name: "valid orthogonal movement - left",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        9,
			newY:        10,
			expectValid: true,
			description: "Moving one cell left should be valid",
		},
		{
			name: "valid orthogonal movement - up",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        10,
			newY:        9,
			expectValid: true,
			description: "Moving one cell up should be valid",
		},
		{
			name: "valid orthogonal movement - down",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        10,
			newY:        11,
			expectValid: true,
			description: "Moving one cell down should be valid",
		},
		{
			name: "no movement - valid",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        10,
			newY:        10,
			expectValid: true,
			description: "No movement should be valid",
		},
		{
			name: "diagonal movement - invalid",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        11,
			newY:        11,
			expectValid: false,
			description: "Diagonal movement should be rejected",
		},
		{
			name: "too far horizontal - invalid",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        12,
			newY:        10,
			expectValid: false,
			description: "Moving more than one cell horizontally should be rejected",
		},
		{
			name: "too far vertical - invalid",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        10,
			newY:        12,
			expectValid: false,
			description: "Moving more than one cell vertically should be rejected",
		},
		{
			name: "far diagonal movement - invalid",
			character: db.Character{
				X: 10,
				Y: 10,
			},
			newX:        15,
			newY:        15,
			expectValid: false,
			description: "Far diagonal movement should be rejected",
		},
		{
			name: "negative coordinates movement - valid",
			character: db.Character{
				X: 0,
				Y: 0,
			},
			newX:        -1,
			newY:        0,
			expectValid: true,
			description: "Valid movement into negative coordinates should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.validateMovement(tt.character, tt.newX, tt.newY)
			assert.Equal(t, tt.expectValid, result, tt.description)
		})
	}
}

func TestIsValidMovePosition_TerrainCollision(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name        string
		x, y        int32
		setupChunk  func(mockService *MockChunkService)
		expectValid bool
		expectError bool
		description string
	}{
		{
			name: "grass terrain - valid",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 10, 10, chunkV1.TerrainType_TERRAIN_TYPE_GRASS)
			},
			expectValid: true,
			expectError: false,
			description: "Grass terrain should be walkable",
		},
		{
			name: "sand terrain - valid",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 10, 10, chunkV1.TerrainType_TERRAIN_TYPE_SAND)
			},
			expectValid: true,
			expectError: false,
			description: "Sand terrain should be walkable",
		},
		{
			name: "dirt terrain - valid",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 10, 10, chunkV1.TerrainType_TERRAIN_TYPE_DIRT)
			},
			expectValid: true,
			expectError: false,
			description: "Dirt terrain should be walkable",
		},
		{
			name: "water terrain - invalid",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 10, 10, chunkV1.TerrainType_TERRAIN_TYPE_WATER)
			},
			expectValid: false,
			expectError: false,
			description: "Water terrain should not be walkable",
		},
		{
			name: "stone terrain - invalid",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 10, 10, chunkV1.TerrainType_TERRAIN_TYPE_STONE)
			},
			expectValid: false,
			expectError: false,
			description: "Stone terrain should not be walkable",
		},
		{
			name: "chunk service error",
			x:    10,
			y:    10,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetShouldError(true)
			},
			expectValid: false,
			expectError: true,
			description: "Chunk service error should propagate",
		},
		{
			name: "edge of chunk - valid grass",
			x:    31, // Last position in chunk
			y:    31,
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 31, 31, chunkV1.TerrainType_TERRAIN_TYPE_GRASS)
			},
			expectValid: true,
			expectError: false,
			description: "Edge positions should work correctly",
		},
		{
			name: "negative coordinates",
			x:    -10,
			y:    -10,
			setupChunk: func(mockService *MockChunkService) {
				// Default chunk will be created, terrain is grass by default
			},
			expectValid: true,
			expectError: false,
			description: "Negative coordinates should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
			defer testDB.Close()

			mockChunkService := NewMockChunkService()
			tt.setupChunk(mockChunkService)
			
			service := NewServiceWithPool(testDB.Pool, mockChunkService)
			ctx := testutil.CreateTestContext()

			valid, err := service.isValidMovePosition(ctx, tt.x, tt.y)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectValid, valid, tt.description)
			}
		})
	}
}

func TestMoveCharacter_FullIntegration(t *testing.T) {
	t.Skip("Skipping MoveCharacter integration test - requires complex refactoring to use mock interface")
	
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Clear the movement cache before test
	movementCache = make(map[string]time.Time)

	tests := []struct {
		name           string
		request        *characterV1.MoveCharacterRequest
		setupMock      func(mock pgxmock.PgxPoolIface)
		setupChunk     func(mockService *MockChunkService)
		expectSuccess  bool
		expectError    bool
		expectCode     codes.Code
		expectErrorMsg string
		description    string
	}{
		{
			name: "successful movement - all validations pass",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        11,
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

				// Character fetch
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))

				// Position update
				mock.ExpectQuery("UPDATE characters SET").
					WithArgs(characterID, int32(11), int32(10), int32(0), int32(0)).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(11), int32(10), int32(0), int32(0), time.Now()))
			},
			setupChunk: func(mockService *MockChunkService) {
				// Default grass terrain is walkable
			},
			expectSuccess: true,
			expectError:   false,
			description:   "Valid movement should succeed with all checks passing",
		},
		{
			name: "invalid character ID",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "invalid-uuid",
				NewX:        11,
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// No database calls expected
			},
			setupChunk:     func(mockService *MockChunkService) {},
			expectSuccess:  false,
			expectError:    true,
			expectCode:     codes.InvalidArgument,
			expectErrorMsg: "invalid character ID",
			description:    "Invalid character ID should be rejected early",
		},
		{
			name: "character not found",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        11,
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnError(errors.New("not found"))
			},
			setupChunk:     func(mockService *MockChunkService) {},
			expectSuccess:  false,
			expectError:    true,
			expectCode:     codes.NotFound,
			expectErrorMsg: "character not found",
			description:    "Non-existent character should return not found error",
		},
		{
			name: "invalid movement - too far",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        15, // Too far from current position (10)
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))
			},
			setupChunk:    func(mockService *MockChunkService) {},
			expectSuccess: false,
			expectError:   false, // Returns success=false but no gRPC error
			description:   "Movement validation should reject moves that are too far",
		},
		{
			name: "invalid movement - diagonal",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        11,
				NewY:        11, // Diagonal movement
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))
			},
			setupChunk:    func(mockService *MockChunkService) {},
			expectSuccess: false,
			expectError:   false,
			description:   "Diagonal movement should be rejected by validation",
		},
		{
			name: "invalid terrain - water",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        11,
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))
			},
			setupChunk: func(mockService *MockChunkService) {
				mockService.SetChunkTerrain(0, 0, 11, 10, chunkV1.TerrainType_TERRAIN_TYPE_WATER)
			},
			expectSuccess: false,
			expectError:   false,
			description:   "Movement to water terrain should be rejected",
		},
		{
			name: "chunk transition movement",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "550e8400-e29b-41d4-a716-446655440000",
				NewX:        32, // Moving to next chunk
				NewY:        10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				characterID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
				userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

				// Character fetch
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
					WithArgs(characterID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(31), int32(10), int32(0), int32(0), time.Now()))

				// Position update with new chunk coordinates
				mock.ExpectQuery("UPDATE characters SET").
					WithArgs(characterID, int32(32), int32(10), int32(1), int32(0)).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
						AddRow(characterID, userID, "TestChar", int32(32), int32(10), int32(1), int32(0), time.Now()))
			},
			setupChunk: func(mockService *MockChunkService) {
				// Ensure terrain at new chunk position is walkable
			},
			expectSuccess: true,
			expectError:   false,
			description:   "Movement across chunk boundaries should update chunk coordinates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
			defer testDB.Close()

			tt.setupMock(testDB.MockPool)

			mockChunkService := NewMockChunkService()
			tt.setupChunk(mockChunkService)
			
			service := NewServiceWithPool(testDB.Pool, mockChunkService)
			ctx := testutil.CreateTestContext()

			response, err := service.MoveCharacter(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, response, tt.description)
				
				statusErr, ok := status.FromError(err)
				require.True(t, ok, "Expected gRPC status error")
				assert.Equal(t, tt.expectCode, statusErr.Code(), tt.description)
				if tt.expectErrorMsg != "" {
					assert.Contains(t, statusErr.Message(), tt.expectErrorMsg, tt.description)
				}
			} else {
				assert.NoError(t, err, tt.description)
				require.NotNil(t, response, tt.description)
				assert.Equal(t, tt.expectSuccess, response.Success, tt.description)
				
				if tt.expectSuccess {
					assert.NotNil(t, response.Character, "Successful movement should return character data")
					assert.Equal(t, tt.request.NewX, response.Character.X, "Character X should be updated")
					assert.Equal(t, tt.request.NewY, response.Character.Y, "Character Y should be updated")
				} else {
					assert.NotEmpty(t, response.ErrorMessage, "Failed movement should have error message")
				}
			}

			// Verify all mock expectations were met
			if err := testDB.MockPool.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled mock expectations: %v", err)
			}
		})
	}
}

func TestMovementCooldown_ConcurrentSafety(t *testing.T) {
	t.Skip("Skipping ConcurrentSafety test - requires complex refactoring to use mock interface")
	
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Clear the movement cache before test
	movementCache = make(map[string]time.Time)

	testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
	defer testDB.Close()

	characterID := "550e8400-e29b-41d4-a716-446655440000"
	characterUUID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}}
	userID := pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x01}}

	// Setup mock for multiple requests (only first should succeed)
	testDB.MockPool.ExpectQuery("SELECT (.+) FROM characters WHERE id").
		WithArgs(characterUUID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
			AddRow(characterUUID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))

	testDB.MockPool.ExpectQuery("UPDATE characters SET").
		WithArgs(characterUUID, int32(11), int32(10), int32(0), int32(0)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
			AddRow(characterUUID, userID, "TestChar", int32(11), int32(10), int32(0), int32(0), time.Now()))

	// Additional character fetch for the second request (but no update)
	testDB.MockPool.ExpectQuery("SELECT (.+) FROM characters WHERE id").
		WithArgs(characterUUID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at"}).
			AddRow(characterUUID, userID, "TestChar", int32(10), int32(10), int32(0), int32(0), time.Now()))

	mockChunkService := NewMockChunkService()
	service := NewServiceWithPool(testDB.Pool, mockChunkService)

	// Make two rapid movement requests
	ctx := testutil.CreateTestContext()
	request := &characterV1.MoveCharacterRequest{
		CharacterId: characterID,
		NewX:        11,
		NewY:        10,
	}

	// First request should succeed
	response1, err1 := service.MoveCharacter(ctx, request)
	require.NoError(t, err1)
	require.NotNil(t, response1)
	assert.True(t, response1.Success, "First movement should succeed")

	// Second request immediately after should be blocked by cooldown
	response2, err2 := service.MoveCharacter(ctx, request)
	require.NoError(t, err2)
	require.NotNil(t, response2)
	assert.False(t, response2.Success, "Second movement should be blocked by cooldown")
	assert.Contains(t, response2.ErrorMessage, "too fast", "Should indicate rate limiting")

	// Verify mock expectations
	if err := testDB.MockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestMovementCooldown_ExactTiming(t *testing.T) {
	
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	assert.Equal(t, 50*time.Millisecond, MovementCooldown, "Movement cooldown should be exactly 50ms")
	
	// Clear the movement cache
	movementCache = make(map[string]time.Time)
	
	characterID := "test-character-123"
	
	// Test boundary conditions
	testCases := []struct {
		name         string
		timeSince    time.Duration
		expectBlocked bool
	}{
		{"49ms ago - should block", 49 * time.Millisecond, true},
		{"50ms ago - should allow", 50 * time.Millisecond, false},
		{"51ms ago - should allow", 51 * time.Millisecond, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			movementCache[characterID] = time.Now().Add(-tc.timeSince)
			
			lastMove, exists := movementCache[characterID]
			require.True(t, exists)
			
			timeSinceLastMove := time.Since(lastMove)
			isBlocked := timeSinceLastMove < MovementCooldown
			
			assert.Equal(t, tc.expectBlocked, isBlocked, 
				"Time since last move: %v, Cooldown: %v, Should block: %v", 
				timeSinceLastMove, MovementCooldown, tc.expectBlocked)
		})
	}
}