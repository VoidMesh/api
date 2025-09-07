package character_actions

import (
	"testing"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUUIDNormalizationFix verifies that the ownership validation works with different UUID formats
func TestUUIDNormalizationFix(t *testing.T) {
	tests := []struct {
		name                 string
		jwtUserID            string   // User ID from JWT (could have dashes)
		characterUserID      string   // Character's UserID in database (standard format with dashes)
		expectedValidation   bool     // Should ownership validation pass?
	}{
		{
			name:               "Same UUID - JWT without dashes, DB with dashes",
			jwtUserID:          "0dca793e36ab4a598c492c2980296599", // No dashes (from JWT)
			characterUserID:    "0dca793e-36ab-4a59-8c49-2c2980296599", // With dashes (from DB)
			expectedValidation: true,
		},
		{
			name:               "Same UUID - both with dashes",
			jwtUserID:          "0dca793e-36ab-4a59-8c49-2c2980296599",
			characterUserID:    "0dca793e-36ab-4a59-8c49-2c2980296599",
			expectedValidation: true,
		},
		{
			name:               "Same UUID - both without dashes",
			jwtUserID:          "0dca793e36ab4a598c492c2980296599",
			characterUserID:    "0dca793e36ab4a598c492c2980296599",
			expectedValidation: true,
		},
		{
			name:               "Different UUIDs - should fail",
			jwtUserID:          "0dca793e-36ab-4a59-8c49-2c2980296599",
			characterUserID:    "1dca793e-36ab-4a59-8c49-2c2980296599", // Different UUID
			expectedValidation: false,
		},
		{
			name:               "Real example from logs - should pass",
			jwtUserID:          "0dca793e36ab4a598c492c2980296599", // Format from JWT logs
			characterUserID:    "0dca793e-36ab-4a59-8c49-2c2980296599", // Standard DB format
			expectedValidation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a service with mock logger
			mockLogger := &MockLogger{}
			if !tt.expectedValidation {
				// Expect warning log for failed validation
				mockLogger.On("Warn", "Character ownership validation failed", []interface{}{
					"character_id", "01020304-0506-0708-090a-0b0c0d0e0f10",
					"character_user_id", tt.characterUserID, 
					"requesting_user_id", tt.jwtUserID,
				}).Return()
			}
			service := &Service{logger: mockLogger}

			// Create a character with the specified user ID
			characterPgUUID, err := uuid.StringToPgtype(tt.characterUserID)
			require.NoError(t, err)

			character := &db.Character{
				ID:     pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
				UserID: characterPgUUID,
				Name:   "TestCharacter",
			}

			// Test ownership validation
			err = service.validateCharacterOwnership(character, tt.jwtUserID)

			if tt.expectedValidation {
				assert.NoError(t, err, "Ownership validation should pass for same UUID in different formats")
			} else {
				require.Error(t, err, "Ownership validation should fail for different UUIDs")
				assert.Contains(t, err.Error(), "character not owned by user")
			}

			// Verify mock expectations were met
			mockLogger.AssertExpectations(t)
		})
	}
}

// TestUUIDUtilityFunctions verifies that our UUID utilities work correctly
func TestUUIDUtilityFunctions(t *testing.T) {
	t.Run("Compare function with different formats", func(t *testing.T) {
		uuid1 := "0dca793e-36ab-4a59-8c49-2c2980296599" // With dashes
		uuid2 := "0dca793e36ab4a598c492c2980296599"     // Without dashes

		result := uuid.Compare(uuid1, uuid2)
		assert.True(t, result, "UUIDs with and without dashes should be considered equal")
	})

	t.Run("PgtypeToString and Compare integration", func(t *testing.T) {
		originalUUID := "0dca793e-36ab-4a59-8c49-2c2980296599"
		
		// Convert to pgtype and back to string
		pgUUID, err := uuid.StringToPgtype(originalUUID)
		require.NoError(t, err)
		
		stringFromPg := uuid.PgtypeToString(pgUUID)
		
		// Should be equal when compared
		assert.True(t, uuid.Compare(originalUUID, stringFromPg))
		
		// Should also work with normalized version
		normalizedOriginal := uuid.Normalize(originalUUID)
		assert.True(t, uuid.Compare(normalizedOriginal, stringFromPg))
	})

	t.Run("Real-world JWT scenario", func(t *testing.T) {
		// Simulate JWT providing UUID without dashes
		jwtUserID := "0dca793e36ab4a598c492c2980296599"
		
		// Simulate database UUID with dashes (from pgtype.UUID.String())
		dbUUID, err := uuid.StringToPgtype("0dca793e-36ab-4a59-8c49-2c2980296599")
		require.NoError(t, err)
		
		dbUserIDString := uuid.PgtypeToString(dbUUID)
		
		// This should match using our Compare function
		assert.True(t, uuid.Compare(jwtUserID, dbUserIDString))
	})
}